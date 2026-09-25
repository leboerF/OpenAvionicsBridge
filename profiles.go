package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

const wasmCDUAdapterID = "wasm-linear-memory-cdu-v1"

type cduLayout struct {
	TitleLarge      uint64 `json:"titleLarge"`
	TitleSmall      uint64 `json:"titleSmall"`
	LabelLayer      uint64 `json:"labelLayer"`
	LargeLayer      uint64 `json:"largeLayer"`
	SmallLayer      uint64 `json:"smallLayer"`
	Scratchpad      uint64 `json:"scratchpad"`
	MessageSelector uint64 `json:"messageSelector,omitempty"`
	URL             string `json:"url"`
}

// aircraftProfile contains only aircraft/build-specific information. The engine
// and output transport do not need to know which vendor produced the aircraft.
// Profiles using the same adapter can therefore be added without changing the
// bridge core.
type aircraftProfile struct {
	ID           string `json:"id"`
	Adapter      string `json:"adapter"`
	Manufacturer string `json:"manufacturer,omitempty"`
	Aircraft     string `json:"aircraft,omitempty"`
	DisplayName  string `json:"displayName"`

	// ModuleNames are discovery hints only. They are never a compatibility
	// requirement. This matters because MSFS may generate a different runtime
	// DLL name on another machine.
	ModuleNames []string `json:"moduleNames,omitempty"`

	// ModuleSHA256 is retained for backwards compatibility with early profile
	// files. KnownModuleSHA256 is the preferred multi-hash form.
	ModuleSHA256      string   `json:"moduleSha256,omitempty"`
	KnownModuleSHA256 []string `json:"knownModuleSha256,omitempty"`

	// The WASM adapter identifies a module by exported symbols rather than by
	// filename. Suffix matching deliberately ignores MSFS' generated prefix.
	RequiredExportSuffixes   []string `json:"requiredExportSuffixes,omitempty"`
	LinearMemoryExportSuffix string   `json:"linearMemoryExportSuffix,omitempty"`

	// FMSMessages maps selector values to UTF-32 string offsets inside the
	// aircraft WASM linear memory. A zero selector always means "use the normal
	// scratchpad"; unknown selectors deliberately fall back to that path too.
	FMSMessages map[uint32]uint64 `json:"fmsMessages,omitempty"`

	Captain cduLayout `json:"captain"`
	Copilot cduLayout `json:"copilot"`
}

type profileFile struct {
	Profiles []aircraftProfile `json:"profiles"`
}

func defaultProfiles() []aircraftProfile {
	return []aircraftProfile{{
		ID:           "justflight-f70-f100-1.3",
		Adapter:      wasmCDUAdapterID,
		Manufacturer: "Just Flight",
		Aircraft:     "F70/F100 Professional",
		DisplayName:  "Just Flight F70/F100 Professional - 1.3 compatible layout",
		ModuleNames:  []string{"m14f2f2d272d86b96_0.dll"},
		KnownModuleSHA256: []string{
			"04ec747e90bb07e88a55e977856d47d5dd0cfc5a7cbee0e899ae2770b6cace87",
		},
		RequiredExportSuffixes: []string{
			"_WASM_linearmemory0",
			"_WASM_CDU_DISP_gauge_callback",
			"_WASM_CDU_DISP_2_gauge_callback",
		},
		LinearMemoryExportSuffix: "_WASM_linearmemory0",
		FMSMessages: map[uint32]uint64{
			1: 0x21E80, 2: 0x21EC0,
			1001: 0x2C050, 1002: 0x2C094, 1003: 0x2C0CC,
			1004: 0x2C120, 1005: 0x2C17C, 1006: 0x2C1B4,
			1007: 0x2C204, 1008: 0x2C25C, 1009: 0x2C2B4,
		},
		Captain: cduLayout{
			TitleLarge: 0x1805F0, TitleSmall: 0x1806B0,
			LabelLayer: 0x180770, LargeLayer: 0x181070, SmallLayer: 0x181970,
			Scratchpad: 0x182270, MessageSelector: 0x182398,
			URL: "ws://localhost:8320/winwing/cdu-captain",
		},
		Copilot: cduLayout{
			TitleLarge: 0x180650, TitleSmall: 0x180710,
			LabelLayer: 0x180BF0, LargeLayer: 0x1814F0, SmallLayer: 0x181DF0,
			Scratchpad: 0x1822D0, MessageSelector: 0x18239C,
			URL: "ws://localhost:8320/winwing/cdu-co-pilot",
		},
	}}
}

func normalizeProfile(p aircraftProfile) aircraftProfile {
	p.ID = strings.TrimSpace(p.ID)
	p.Adapter = strings.TrimSpace(p.Adapter)
	p.DisplayName = strings.TrimSpace(p.DisplayName)
	if p.Adapter == "" {
		p.Adapter = wasmCDUAdapterID
	}
	if len(p.KnownModuleSHA256) == 0 && strings.TrimSpace(p.ModuleSHA256) != "" {
		p.KnownModuleSHA256 = []string{strings.TrimSpace(p.ModuleSHA256)}
	}
	if p.Adapter == wasmCDUAdapterID {
		if p.LinearMemoryExportSuffix == "" {
			p.LinearMemoryExportSuffix = "_WASM_linearmemory0"
		}
		if len(p.RequiredExportSuffixes) == 0 {
			p.RequiredExportSuffixes = []string{
				p.LinearMemoryExportSuffix,
				"_WASM_CDU_DISP_gauge_callback",
				"_WASM_CDU_DISP_2_gauge_callback",
			}
		}
	}
	if p.DisplayName == "" {
		p.DisplayName = strings.TrimSpace(strings.TrimSpace(p.Manufacturer + " " + p.Aircraft))
	}
	return p
}

func validateProfile(p aircraftProfile) error {
	if p.ID == "" {
		return fmt.Errorf("profile id is empty")
	}
	if p.Adapter == "" {
		return fmt.Errorf("profile %s has no adapter", p.ID)
	}
	if p.DisplayName == "" {
		return fmt.Errorf("profile %s has no display name", p.ID)
	}
	if p.Adapter == wasmCDUAdapterID {
		if p.LinearMemoryExportSuffix == "" || len(p.RequiredExportSuffixes) == 0 {
			return fmt.Errorf("profile %s has incomplete WASM export configuration", p.ID)
		}
		for side, l := range map[string]cduLayout{"captain": p.Captain, "copilot": p.Copilot} {
			if l.TitleLarge == 0 || l.TitleSmall == 0 || l.LabelLayer == 0 || l.LargeLayer == 0 || l.SmallLayer == 0 || l.Scratchpad == 0 {
				return fmt.Errorf("profile %s has incomplete %s CDU layout", p.ID, side)
			}
			if strings.TrimSpace(l.URL) == "" {
				return fmt.Errorf("profile %s has no %s output URL", p.ID, side)
			}
		}
	}
	return nil
}

func mergeProfileJSON(base []aircraftProfile, b []byte) ([]aircraftProfile, error) {
	var pf profileFile
	if err := json.Unmarshal(b, &pf); err != nil {
		return base, err
	}
	out := append([]aircraftProfile(nil), base...)
	index := make(map[string]int, len(out))
	for i := range out {
		out[i] = normalizeProfile(out[i])
		index[out[i].ID] = i
	}
	for _, raw := range pf.Profiles {
		p := normalizeProfile(raw)
		if err := validateProfile(p); err != nil {
			return base, err
		}
		if i, ok := index[p.ID]; ok {
			out[i] = p
		} else {
			index[p.ID] = len(out)
			out = append(out, p)
		}
	}
	return out, nil
}

func knownHash(p *aircraftProfile, hash string) bool {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return false
	}
	for _, h := range p.KnownModuleSHA256 {
		if strings.EqualFold(strings.TrimSpace(h), hash) {
			return true
		}
	}
	if strings.EqualFold(strings.TrimSpace(p.ModuleSHA256), hash) && hash != "" {
		return true
	}
	return false
}
