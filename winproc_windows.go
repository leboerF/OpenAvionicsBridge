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
		mods = append(mods, winModule{Base: me.ModBaseAddr, Size: me.ModBaseSize, Name: syscall.UTF16ToString(me.SzModule[:]), Path: syscall.UTF16ToString(me.SzExePath[:])})
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
