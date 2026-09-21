package main

import (
	"encoding/binary"
	"testing"
)

func encodeW(s string, chars int) []byte {
	b := make([]byte, chars*4)
	i := 0
	for _, r := range s {
		if i >= chars {
			break
		}
		binary.LittleEndian.PutUint32(b[i*4:], uint32(r))
		i++
	}
	return b
}

func TestGlyphMapping(t *testing.T) {
	cases := map[uint32]string{0x7E: "☐", 0x7D: "→", 0x7B: "↑", 0x7C: "↓", 0x2B26: "°"}
	for in, want := range cases {
		if got := mapCodePoint(in); got != want {
			t.Fatalf("%x: got %q want %q", in, got, want)
		}
	}
}

func TestDecodeStopsAtNull(t *testing.T) {
	b := make([]byte, 24*4)
	binary.LittleEndian.PutUint32(b[0:], 'A')
	binary.LittleEndian.PutUint32(b[4:], 'B')
	binary.LittleEndian.PutUint32(b[8:], 0)
	binary.LittleEndian.PutUint32(b[12:], 'X')
	if got := decodeWasmString(b, 0, 24); got != "AB" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildDisplayShape(t *testing.T) {
	var labels, large, small [12]string
	labels[0] = "LABEL"
	labels[6] = "RIGHT"
	large[0] = "EDDC"
	large[6] = "EDDF"
	f := buildDisplay("INIT", "", labels, large, small, "SCRATCH")
	if len(f.Cells) != 336 {
		t.Fatalf("cells=%d", len(f.Cells))
	}
	if len([]rune(f.Rows[0])) != 24 {
		t.Fatalf("row0 width=%d", len([]rune(f.Rows[0])))
	}
	if f.Cells[0].Size != 1 {
		t.Fatalf("title not small")
	}
	if f.Rows[13][:7] != "SCRATCH" {
		t.Fatalf("scratch row %q", f.Rows[13])
	}
}

func TestDecodeWasmStringSpecial(t *testing.T) {
	b := encodeW("A~}|{", 24)
	got := decodeWasmString(b, 0, 24)
	if got != "A☐→↓↑" {
		t.Fatalf("got %q", got)
	}
}

func TestSquareAlwaysUsesSmallFont(t *testing.T) {
	var labels, large, small [12]string
	large[0] = "A☐B"
	f := buildDisplay("", "", labels, large, small, "")
	// First data row starts at cell 2*24. A is large, the square must be small, B large.
	base := 2 * 24
	if f.Cells[base].Size != 0 || f.Cells[base+1].Ch != "☐" || f.Cells[base+1].Size != 1 || f.Cells[base+2].Size != 0 {
		t.Fatalf("unexpected sizes around square: %#v %#v %#v", f.Cells[base], f.Cells[base+1], f.Cells[base+2])
	}
}
