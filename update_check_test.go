package main

import "testing"

func TestParseAndCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0-rc3", "1.0.0-rc4", -1},
		{"1.0.0-rc4", "1.0.0", -1},
		{"1.0.0", "1.0.1-alpha1", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.0.0-beta2", "1.0.0-rc1", -1},
	}
	for _, tt := range tests {
		a, ok := parseSemVersion(tt.a)
		if !ok {
			t.Fatalf("parse %q failed", tt.a)
		}
		b, ok := parseSemVersion(tt.b)
		if !ok {
			t.Fatalf("parse %q failed", tt.b)
		}
		got := compareSemVersion(a, b)
		if got < 0 {
			got = -1
		} else if got > 0 {
			got = 1
		}
		if got != tt.want {
			t.Fatalf("compare %s %s = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
