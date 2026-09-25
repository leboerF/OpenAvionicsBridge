package main

import (
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

type aircraftAttachment struct {
	Profile       *aircraftProfile
	Module        winModule
	Linear        uintptr
	ModuleHash    string
	KnownHash     bool
	LinearExport  string
	LinearRVA     uint32
	ValidationMsg string
}

type aircraftAdapter interface {
	ID() string
	TryAttach(h processHandle, p *aircraftProfile, mods []winModule) (*aircraftAttachment, string, error)
}

type wasmLinearMemoryAdapter struct {
	exportCache map[string]map[string]uint32
}

func newWasmLinearMemoryAdapter() *wasmLinearMemoryAdapter {
	return &wasmLinearMemoryAdapter{exportCache: make(map[string]map[string]uint32)}
}

func (a *wasmLinearMemoryAdapter) ID() string { return wasmCDUAdapterID }

func moduleCacheKey(m winModule) string {
	return fmt.Sprintf("%d:%X:%X:%s", m.PID, m.Base, m.Size, strings.ToLower(m.Path))
}

func orderModulesByHints(mods []winModule, hints []string) []winModule {
	if len(hints) == 0 {
		return append([]winModule(nil), mods...)
	}
	hint := make(map[string]bool, len(hints))
	for _, h := range hints {
		hint[strings.ToLower(strings.TrimSpace(h))] = true
	}
	out := append([]winModule(nil), mods...)
	rank := func(m winModule) int {
		name := strings.ToLower(m.Name)
		if hint[name] {
			return 0
		}
		if strings.HasPrefix(name, "m") && strings.HasSuffix(name, ".dll") {
			return 1
		}
		return 2
	}
	sort.SliceStable(out, func(i, j int) bool { return rank(out[i]) < rank(out[j]) })
	return out
}

func exportBySuffix(exports map[string]uint32, suffix string) (string, uint32, bool) {
	for name, rva := range exports {
		if strings.HasSuffix(name, suffix) {
			return name, rva, true
		}
	}
	return "", 0, false
}

func hasAllExportSuffixes(exports map[string]uint32, suffixes []string) bool {
	for _, suffix := range suffixes {
		if _, _, ok := exportBySuffix(exports, suffix); !ok {
			return false
		}
	}
	return true
}

func (a *wasmLinearMemoryAdapter) moduleExports(h processHandle, m winModule) (map[string]uint32, error) {
	key := moduleCacheKey(m)
	if v, ok := a.exportCache[key]; ok {
		return v, nil
	}
	exports, err := h.remotePEExports(m)
	if err != nil {
		return nil, err
	}
	a.exportCache[key] = exports
	return exports, nil
}

func (a *wasmLinearMemoryAdapter) TryAttach(h processHandle, p *aircraftProfile, mods []winModule) (*aircraftAttachment, string, error) {
	ordered := orderModulesByHints(mods, p.ModuleNames)
	var closest string
	for _, m := range ordered {
		exports, err := a.moduleExports(h, m)
		if err != nil {
			continue
		}
		if !hasAllExportSuffixes(exports, p.RequiredExportSuffixes) {
			continue
		}

		linearName, linearRVA, ok := exportBySuffix(exports, p.LinearMemoryExportSuffix)
		if !ok {
			continue
		}
		linear64, err := h.ReadU64(m.Base + uintptr(linearRVA))
		if err != nil {
			closest = fmt.Sprintf("%s matched the export signature but its linear-memory pointer could not be read: %v", m.Name, err)
			continue
		}
		if linear64 < 0x10000 {
			closest = fmt.Sprintf("%s matched the export signature but returned an implausible linear-memory pointer 0x%X", m.Name, linear64)
			continue
		}

		linear := uintptr(linear64)
		if err := validateProfileMemory(h, linear, p); err != nil {
			closest = fmt.Sprintf("%s matched the FMS export signature but the configured CDU layout failed runtime validation: %v", m.Name, err)
			continue
		}

		hash := ""
		if strings.TrimSpace(m.Path) != "" {
			hash, _ = sha256File(m.Path)
		}
		known := knownHash(p, hash)
		msg := "runtime export + memory-layout validation passed"
		if known {
			msg = "known module hash + runtime validation passed"
		}
		return &aircraftAttachment{
			Profile:       p,
			Module:        m,
			Linear:        linear,
			ModuleHash:    hash,
			KnownHash:     known,
			LinearExport:  linearName,
			LinearRVA:     linearRVA,
			ValidationMsg: msg,
		}, "", nil
	}
	return nil, closest, nil
}

func validateProfileMemory(h processHandle, linear uintptr, p *aircraftProfile) error {
	if err := validateCDULayoutMemory(h, linear, p.Captain); err != nil {
		return fmt.Errorf("captain CDU: %w", err)
	}
	if err := validateCDULayoutMemory(h, linear, p.Copilot); err != nil {
		return fmt.Errorf("first-officer CDU: %w", err)
	}
	return nil
}

func validateCDULayoutMemory(h processHandle, linear uintptr, l cduLayout) error {
	checks := []struct {
		name string
		off  uint64
		size int
	}{
		{"title-large", l.TitleLarge, 0x60},
		{"title-small", l.TitleSmall, 0x60},
		{"labels", l.LabelLayer, 0x480},
		{"large", l.LargeLayer, 0x480},
		{"small", l.SmallLayer, 0x480},
		{"scratchpad", l.Scratchpad, 0x60},
	}
	for _, c := range checks {
		b, err := h.Read(linear+uintptr(c.off), c.size)
		if err != nil {
			return fmt.Errorf("%s at 0x%X is not readable: %v", c.name, c.off, err)
		}
		if err := validateUTF32DisplayBlock(b); err != nil {
			return fmt.Errorf("%s at 0x%X is not a plausible CDU text block: %v", c.name, c.off, err)
		}
	}
	if l.MessageSelector != 0 {
		if _, err := h.Read(linear+uintptr(l.MessageSelector), 4); err != nil {
			return fmt.Errorf("message selector at 0x%X is not readable: %v", l.MessageSelector, err)
		}
	}
	return nil
}

func validateUTF32DisplayBlock(b []byte) error {
	if len(b)%4 != 0 {
		return fmt.Errorf("unaligned UTF-32 block")
	}
	invalid := 0
	for i := 0; i < len(b); i += 4 {
		cp := binary.LittleEndian.Uint32(b[i : i+4])
		if cp == 0 || cp == 0x20 || cp == 0xA0 || cp == 0x2B26 || cp == 0x25A1 {
			continue
		}
		if cp > 0x10FFFF || (cp >= 0xD800 && cp <= 0xDFFF) {
			invalid++
			continue
		}
		r := rune(cp)
		if unicode.IsControl(r) && r != '\t' {
			invalid++
			continue
		}
		if !unicode.IsGraphic(r) && r != '\t' {
			invalid++
		}
	}
	if invalid != 0 {
		return fmt.Errorf("%d invalid code point(s)", invalid)
	}
	return nil
}

func defaultAircraftAdapters() map[string]aircraftAdapter {
	a := newWasmLinearMemoryAdapter()
	return map[string]aircraftAdapter{a.ID(): a}
}
