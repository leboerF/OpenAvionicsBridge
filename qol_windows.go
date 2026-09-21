package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

var (
	pCreateMutexW     = k32.NewProc("CreateMutexW")
	pGlobalAlloc      = k32.NewProc("GlobalAlloc")
	pGlobalLock       = k32.NewProc("GlobalLock")
	pGlobalUnlock     = k32.NewProc("GlobalUnlock")
	pGlobalFree       = k32.NewProc("GlobalFree")
	pOpenClipboard    = user32.NewProc("OpenClipboard")
	pEmptyClipboard   = user32.NewProc("EmptyClipboard")
	pSetClipboardData = user32.NewProc("SetClipboardData")
	pCloseClipboard   = user32.NewProc("CloseClipboard")
)

func acquireSingleInstance() (uintptr, bool, error) {
	name, err := syscall.UTF16PtrFromString(`Local\OpenAvionicsBridge.SingleInstance`)
	if err != nil {
		return 0, false, err
	}
	h, _, callErr := pCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(name)))
	if h == 0 {
		return 0, false, fmt.Errorf("CreateMutexW failed: %v", callErr)
	}
	if callErr == syscall.Errno(183) { // ERROR_ALREADY_EXISTS
		closeHandle(h)
		return 0, true, nil
	}
	return h, false, nil
}

func copyTextToClipboard(owner uintptr, text string) error {
	utf16, err := syscall.UTF16FromString(text)
	if err != nil {
		return err
	}
	bytes := uintptr(len(utf16) * 2)
	hmem, _, e1 := pGlobalAlloc.Call(gmemMoveable, bytes)
	if hmem == 0 {
		return fmt.Errorf("GlobalAlloc failed: %v", e1)
	}
	locked, _, e2 := pGlobalLock.Call(hmem)
	if locked == 0 {
		pGlobalFree.Call(hmem)
		return fmt.Errorf("GlobalLock failed: %v", e2)
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(locked)), len(utf16))
	copy(dst, utf16)
	pGlobalUnlock.Call(hmem)

	ok, _, e3 := pOpenClipboard.Call(owner)
	if ok == 0 {
		pGlobalFree.Call(hmem)
		return fmt.Errorf("OpenClipboard failed: %v", e3)
	}
	defer pCloseClipboard.Call()
	if r, _, e4 := pEmptyClipboard.Call(); r == 0 {
		pGlobalFree.Call(hmem)
		return fmt.Errorf("EmptyClipboard failed: %v", e4)
	}
	if r, _, e5 := pSetClipboardData.Call(cfUnicodeText, hmem); r == 0 {
		pGlobalFree.Call(hmem)
		return fmt.Errorf("SetClipboardData failed: %v", e5)
	}
	// Ownership of hmem transfers to the clipboard after SetClipboardData.
	return nil
}
