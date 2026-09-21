package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

const (
	th32csSnapProcess  = 0x00000002
	th32csSnapModule   = 0x00000008
	th32csSnapModule32 = 0x00000010
	processVMRead      = 0x0010
	processQueryInfo   = 0x0400
)

var (
	k32                       = syscall.NewLazyDLL("kernel32.dll")
	pCreateToolhelp32Snapshot = k32.NewProc("CreateToolhelp32Snapshot")
	pProcess32FirstW          = k32.NewProc("Process32FirstW")
	pProcess32NextW           = k32.NewProc("Process32NextW")
	pModule32FirstW           = k32.NewProc("Module32FirstW")
	pModule32NextW            = k32.NewProc("Module32NextW")
	pOpenProcess              = k32.NewProc("OpenProcess")
	pReadProcessMemory        = k32.NewProc("ReadProcessMemory")
	pCloseHandle              = k32.NewProc("CloseHandle")
)

type processEntry32 struct {
	Size            uint32
	CntUsage        uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	CntThreads      uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

type moduleEntry32 struct {
	Size         uint32
	ModuleID     uint32
	ProcessID    uint32
	GlblcntUsage uint32
	ProccntUsage uint32
	ModBaseAddr  uintptr
	ModBaseSize  uint32
	Module       uintptr
	SzModule     [256]uint16
	SzExePath    [260]uint16
}

type winProcess struct {
	PID  uint32
	Name string
}
type winModule struct {
	PID        uint32
	Base       uintptr
	Size       uint32
	Name, Path string
}

type processHandle uintptr

func closeHandle(h uintptr) {
	if h != 0 && h != ^uintptr(0) {
		_, _, _ = pCloseHandle.Call(h)
	}
}

func findMSFS() (*winProcess, error) {
	snap, _, e := pCreateToolhelp32Snapshot.Call(th32csSnapProcess, 0)
	if snap == ^uintptr(0) {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot(process): %v", e)
	}
	defer closeHandle(snap)
	var pe processEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	r, _, _ := pProcess32FirstW.Call(snap, uintptr(unsafe.Pointer(&pe)))
	if r == 0 {
		return nil, nil
	}
	wanted := map[string]bool{"flightsimulator2024.exe": true, "flightsimulator.exe": true, "microsoftflightsimulator.exe": true}
	for {
		name := strings.ToLower(syscall.UTF16ToString(pe.ExeFile[:]))
		if wanted[name] {
			return &winProcess{PID: pe.ProcessID, Name: name}, nil
		}
		r, _, _ = pProcess32NextW.Call(snap, uintptr(unsafe.Pointer(&pe)))
		if r == 0 {
			break
		}
	}
	return nil, nil
}

func listModules(pid uint32) ([]winModule, error) {
	snap, _, e := pCreateToolhelp32Snapshot.Call(th32csSnapModule|th32csSnapModule32, uintptr(pid))
	if snap == ^uintptr(0) {
		return nil, fmt.Errorf("Cannot enumerate MSFS modules (Win32: %v). If MSFS runs as Administrator, run this app as Administrator too", e)
	}
	defer closeHandle(snap)
	var me moduleEntry32
	me.Size = uint32(unsafe.Sizeof(me))
	r, _, _ := pModule32FirstW.Call(snap, uintptr(unsafe.Pointer(&me)))
	if r == 0 {
		return nil, fmt.Errorf("Module32FirstW failed")
	}
	mods := make([]winModule, 0, 256)
	for {
		mods = append(mods, winModule{PID: pid, Base: me.ModBaseAddr, Size: me.ModBaseSize, Name: syscall.UTF16ToString(me.SzModule[:]), Path: syscall.UTF16ToString(me.SzExePath[:])})
		me.Size = uint32(unsafe.Sizeof(me))
		r, _, _ = pModule32NextW.Call(snap, uintptr(unsafe.Pointer(&me)))
		if r == 0 {
			break
		}
	}
	return mods, nil
}

func openReadProcess(pid uint32) (processHandle, error) {
	h, _, e := pOpenProcess.Call(processVMRead|processQueryInfo, 0, uintptr(pid))
	if h == 0 {
		return 0, fmt.Errorf("OpenProcess failed: %v", e)
	}
	return processHandle(h), nil
}
func (h processHandle) Close() { closeHandle(uintptr(h)) }

func (h processHandle) Read(address uintptr, size int) ([]byte, error) {
	if size < 0 {
		return nil, fmt.Errorf("invalid read size")
	}
	b := make([]byte, size)
	if size == 0 {
		return b, nil
	}
	var read uintptr
	r, _, e := pReadProcessMemory.Call(uintptr(h), address, uintptr(unsafe.Pointer(&b[0])), uintptr(size), uintptr(unsafe.Pointer(&read)))
	if r == 0 || read != uintptr(size) {
		return nil, fmt.Errorf("ReadProcessMemory failed at 0x%X, requested=%d read=%d: %v", address, size, read, e)
	}
	return b, nil
}
func (h processHandle) ReadU64(address uintptr) (uint64, error) {
	b, err := h.Read(address, 8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}


// remotePEExports returns exported names and their RVAs directly from the
// loaded module image inside the simulator process. Reading the in-memory PE
// export directory avoids depending on the module's on-disk filename or path.
func (h processHandle) remotePEExports(mod winModule) (map[string]uint32, error) {
	headerSize := 0x2000
	if mod.Size > 0 && int(mod.Size) < headerSize {
		headerSize = int(mod.Size)
	}
	if headerSize < 0x100 {
		return nil, fmt.Errorf("module %s is too small to contain a PE header", mod.Name)
	}
	header, err := h.Read(mod.Base, headerSize)
	if err != nil {
		return nil, err
	}
	if len(header) < 0x100 || header[0] != 'M' || header[1] != 'Z' {
		return nil, fmt.Errorf("module %s has no valid DOS header", mod.Name)
	}
	peOff := int(binary.LittleEndian.Uint32(header[0x3c:0x40]))
	if peOff < 0 || peOff+0x200 > len(header) {
		return nil, fmt.Errorf("module %s has an invalid PE header offset", mod.Name)
	}
	if string(header[peOff:peOff+4]) != "PE\x00\x00" {
		return nil, fmt.Errorf("module %s has no valid PE signature", mod.Name)
	}
	optional := peOff + 24
	if optional+2 > len(header) {
		return nil, fmt.Errorf("module %s has a truncated optional header", mod.Name)
	}
	magic := binary.LittleEndian.Uint16(header[optional : optional+2])
	var dataDir int
	switch magic {
	case 0x20b:
		dataDir = optional + 112
	case 0x10b:
		dataDir = optional + 96
	default:
		return nil, fmt.Errorf("module %s has unsupported PE optional-header magic 0x%X", mod.Name, magic)
	}
	if dataDir+8 > len(header) {
		return nil, fmt.Errorf("module %s has a truncated export data directory", mod.Name)
	}
	exportRVA := binary.LittleEndian.Uint32(header[dataDir : dataDir+4])
	exportSize := binary.LittleEndian.Uint32(header[dataDir+4 : dataDir+8])
	if exportRVA == 0 || exportSize == 0 {
		return map[string]uint32{}, nil
	}
	if exportSize > 16*1024*1024 {
		return nil, fmt.Errorf("module %s has an implausible export directory size", mod.Name)
	}
	blob, err := h.Read(mod.Base+uintptr(exportRVA), int(exportSize))
	if err != nil {
		return nil, err
	}
	if len(blob) < 40 {
		return nil, fmt.Errorf("module %s has a truncated export directory", mod.Name)
	}
	ed := blob[:40]
	numFunctions := binary.LittleEndian.Uint32(ed[20:24])
	numNames := binary.LittleEndian.Uint32(ed[24:28])
	functionsRVA := binary.LittleEndian.Uint32(ed[28:32])
	namesRVA := binary.LittleEndian.Uint32(ed[32:36])
	ordinalsRVA := binary.LittleEndian.Uint32(ed[36:40])
	if numNames == 0 || numFunctions == 0 {
		return map[string]uint32{}, nil
	}
	if numNames > 100000 || numFunctions > 100000 {
		return nil, fmt.Errorf("module %s has implausible export counts", mod.Name)
	}

	readRVA := func(rva uint32, size int) ([]byte, error) {
		if rva >= exportRVA {
			rel := uint64(rva - exportRVA)
			if rel+uint64(size) <= uint64(len(blob)) {
				return blob[rel : rel+uint64(size)], nil
			}
		}
		return h.Read(mod.Base+uintptr(rva), size)
	}
	names, err := readRVA(namesRVA, int(numNames)*4)
	if err != nil {
		return nil, err
	}
	ords, err := readRVA(ordinalsRVA, int(numNames)*2)
	if err != nil {
		return nil, err
	}
	funcs, err := readRVA(functionsRVA, int(numFunctions)*4)
	if err != nil {
		return nil, err
	}

	out := make(map[string]uint32)
	for i := uint32(0); i < numNames; i++ {
		nameRVA := binary.LittleEndian.Uint32(names[i*4 : i*4+4])
		var name string
		if nameRVA >= exportRVA {
			rel := int(nameRVA - exportRVA)
			if rel >= 0 && rel < len(blob) {
				end := rel
				limit := rel + 512
				if limit > len(blob) {
					limit = len(blob)
				}
				for end < limit && blob[end] != 0 {
					end++
				}
				name = string(blob[rel:end])
			}
		}
		if name == "" {
			name, _ = h.readCString(mod.Base+uintptr(nameRVA), 512)
		}
		if !strings.Contains(name, "_WASM_") {
			continue
		}
		ord := binary.LittleEndian.Uint16(ords[i*2 : i*2+2])
		if uint32(ord) >= numFunctions {
			continue
		}
		funcRVA := binary.LittleEndian.Uint32(funcs[uint32(ord)*4 : uint32(ord)*4+4])
		out[name] = funcRVA
	}
	return out, nil
}

func (h processHandle) readCString(address uintptr, max int) (string, error) {
	if max <= 0 {
		return "", nil
	}
	b, err := h.Read(address, max)
	if err != nil {
		return "", err
	}
	for i, c := range b {
		if c == 0 {
			return string(b[:i]), nil
		}
	}
	return string(b), nil
}
