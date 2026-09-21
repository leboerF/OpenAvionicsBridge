package main

import (
	"encoding/binary"
	"strings"
	"unicode/utf8"
)

type cell struct {
	Ch   string
	Size int // 0=large, 1=small
}

type displayFrame struct {
	Rows  [14]string
	Cells [336]cell
}

func mapCodePoint(cp uint32) string {
	switch cp {
	case 0, 0x20, 0xA0:
		return " "
	case 0x7E:
		return "☐"
	case 0x7D:
		return "→"
	case 0x7B:
		return "↑"
	case 0x7C:
		return "↓"
	case 0x2B26:
		return "°"
	case 0x25A1:
		return "☐"
	}
	if cp > utf8.MaxRune || (cp >= 0xD800 && cp <= 0xDFFF) {
		return " "
	}
	return string(rune(cp))
}

func decodeWasmString(b []byte, byteOffset, maxChars int) string {
	var sb strings.Builder
	for i := 0; i < maxChars; i++ {
		off := byteOffset + i*4
		if off+4 > len(b) {
			break
		}
		cp := binary.LittleEndian.Uint32(b[off : off+4])
		if cp == 0 {
			break
		}
		sb.WriteString(mapCodePoint(cp))
	}
	return sb.String()
}

func decodeWasmLayer(b []byte) [12]string {
	var out [12]string
	for i := range out {
		out[i] = decodeWasmString(b, i*0x60, 24)
	}
	return out
}

func overlay(row *[24]string, sizes *[24]int, text string, size int, right bool) {
	if text == "" {
		return
	}
	text = strings.TrimRight(text, " \t\r\n")
	if right {
		text = strings.TrimSpace(text)
	}
	rr := []rune(text)
	if len(rr) > 24 {
		rr = rr[:24]
	}
	start := 0
	if right {
		start = 24 - len(rr)
		if start < 0 {
			start = 0
		}
	}
	for i, r := range rr {
		idx := start + i
		if idx >= 24 {
			break
		}
		if r != ' ' {
			row[idx] = string(r)
			cellSize := size
			// The F100 uses a square glyph as a compact selectable-field marker.
			// WinCtrl looks much closer to the aircraft CDU when this symbol is
			// rendered with its small font, independent of the source layer.
			if r == '☐' {
				cellSize = 1
			}
			sizes[idx] = cellSize
		}
	}
}

func buildDisplay(titleLarge, titleSmall string, labels, large, small [12]string, scratch string) displayFrame {
	var rows [14][24]string
	var sizes [14][24]int
	for r := 0; r < 14; r++ {
		for c := 0; c < 24; c++ {
			rows[r][c] = " "
			sizes[r][c] = 0
		}
	}

	// Verified against the Just Flight F100 pages used during PoC testing:
	// the whole title row is rendered with the small WinCtrl font.
	overlay(&rows[0], &sizes[0], titleLarge, 1, false)
	overlay(&rows[0], &sizes[0], titleSmall, 1, false)

	for i := 0; i < 6; i++ {
		labelRow := 1 + i*2
		dataRow := 2 + i*2
		overlay(&rows[labelRow], &sizes[labelRow], labels[i], 1, false)
		overlay(&rows[labelRow], &sizes[labelRow], labels[i+6], 1, true)
		overlay(&rows[dataRow], &sizes[dataRow], large[i], 0, false)
		overlay(&rows[dataRow], &sizes[dataRow], large[i+6], 0, true)
		overlay(&rows[dataRow], &sizes[dataRow], small[i], 1, false)
		overlay(&rows[dataRow], &sizes[dataRow], small[i+6], 1, true)
	}
	overlay(&rows[13], &sizes[13], scratch, 0, false)

	var out displayFrame
	cellIndex := 0
	for r := 0; r < 14; r++ {
		var sb strings.Builder
		for c := 0; c < 24; c++ {
			sb.WriteString(rows[r][c])
			out.Cells[cellIndex] = cell{Ch: rows[r][c], Size: sizes[r][c]}
			cellIndex++
		}
		out.Rows[r] = sb.String()
	}
	return out
}
