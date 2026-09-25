//go:build windows

package main

import (
	"encoding/binary"
	"strings"
)

func resolveFMSMessage(h processHandle, linear uintptr, selectorOffset uint64, messages map[uint32]uint64) (string, bool, error) {
	if selectorOffset == 0 || len(messages) == 0 {
		return "", false, nil
	}

	b, err := h.Read(linear+uintptr(selectorOffset), 4)
	if err != nil {
		return "", false, err
	}
	selector := binary.LittleEndian.Uint32(b)
	msgOffset, ok := lookupFMSMessageOffset(messages, selector)
	if !ok {
		return "", false, nil
	}

	raw, err := h.Read(linear+uintptr(msgOffset), 0x60)
	if err != nil {
		return "", false, err
	}
	msg := decodeWasmString(raw, 0, 24)
	if strings.TrimSpace(msg) == "" {
		return "", false, nil
	}
	return msg, true, nil
}
