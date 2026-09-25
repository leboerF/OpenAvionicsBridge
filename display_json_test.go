package main

import (
	"encoding/json"
	"testing"
)

func TestNormalizeDisplayColor(t *testing.T) {
	supported := []displayColor{
		displayColorAmber, displayColorWhite, displayColorCyan, displayColorGreen,
		displayColorMagenta, displayColorRed, displayColorYellow, displayColorBlue,
		displayColorGrey, displayColorKhaki,
	}
	for _, color := range supported {
		if got := normalizeDisplayColor(color); got != color {
			t.Fatalf("normalizeDisplayColor(%q)=%q", color, got)
		}
	}
	if got := normalizeDisplayColor(displayColor("?")); got != displayColorGreen {
		t.Fatalf("unknown color fallback=%q, want %q", got, displayColorGreen)
	}
	if got := normalizeDisplayColor(""); got != displayColorGreen {
		t.Fatalf("empty color fallback=%q, want %q", got, displayColorGreen)
	}
}

func TestFrameJSONColorAndInverse(t *testing.T) {
	var f displayFrame
	f.Cells[0] = cell{Ch: "M", Size: 0, Color: displayColorMagenta}
	f.Cells[1] = cell{Ch: "A", Size: 1, Color: displayColorAmber, Inverse: true}
	f.Cells[2] = cell{Ch: "G", Size: 0, Color: displayColor("invalid")}

	var payload struct {
		Target string            `json:"Target"`
		Data   []json.RawMessage `json:"Data"`
	}
	if err := json.Unmarshal([]byte(frameJSON(f)), &payload); err != nil {
		t.Fatalf("frameJSON produced invalid JSON: %v", err)
	}
	if payload.Target != "Display" {
		t.Fatalf("Target=%q", payload.Target)
	}
	if len(payload.Data) != 336 {
		t.Fatalf("Data cells=%d, want 336", len(payload.Data))
	}

	var normal []any
	if err := json.Unmarshal(payload.Data[0], &normal); err != nil {
		t.Fatal(err)
	}
	if len(normal) != 3 || normal[0] != "M" || normal[1] != "m" || normal[2] != float64(0) {
		t.Fatalf("magenta cell=%#v", normal)
	}

	var inverse []any
	if err := json.Unmarshal(payload.Data[1], &inverse); err != nil {
		t.Fatal(err)
	}
	if len(inverse) != 4 || inverse[0] != "A" || inverse[1] != "a" || inverse[2] != float64(1) || inverse[3] != float64(1) {
		t.Fatalf("inverse cell=%#v", inverse)
	}

	var fallback []any
	if err := json.Unmarshal(payload.Data[2], &fallback); err != nil {
		t.Fatal(err)
	}
	if len(fallback) != 3 || fallback[1] != "g" {
		t.Fatalf("fallback cell=%#v", fallback)
	}
}

func TestFokkerBuildDisplayDefaultsToGreen(t *testing.T) {
	var labels, large, small [12]string
	large[0] = "EDDC"
	f := buildDisplay("IDENT", "", labels, large, small, "")
	for i, c := range f.Cells {
		if c.Color != displayColorGreen {
			t.Fatalf("cell %d color=%q, want green", i, c.Color)
		}
		if c.Inverse {
			t.Fatalf("cell %d unexpectedly inverse", i)
		}
	}
}
