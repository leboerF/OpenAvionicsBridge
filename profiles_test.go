package main

import "testing"

func TestNormalizeLegacyHashAndDefaults(t *testing.T) {
	p := normalizeProfile(aircraftProfile{
		ID:           "test",
		DisplayName:  "Test Aircraft",
		ModuleSHA256: "abc",
	})
	if p.Adapter != wasmCDUAdapterID {
		t.Fatalf("adapter=%q", p.Adapter)
	}
	if len(p.KnownModuleSHA256) != 1 || p.KnownModuleSHA256[0] != "abc" {
		t.Fatalf("hash migration failed: %#v", p.KnownModuleSHA256)
	}
	if p.LinearMemoryExportSuffix != "_WASM_linearmemory0" {
		t.Fatalf("linear-memory suffix=%q", p.LinearMemoryExportSuffix)
	}
	if len(p.RequiredExportSuffixes) != 3 {
		t.Fatalf("required exports=%#v", p.RequiredExportSuffixes)
	}
}

func TestMergeProfileJSONOverridesByID(t *testing.T) {
	base := defaultProfiles()
	json := []byte(`{"profiles":[{"id":"justflight-f70-f100-1.3","adapter":"wasm-linear-memory-cdu-v1","displayName":"Override","requiredExportSuffixes":["_WASM_linearmemory0"],"linearMemoryExportSuffix":"_WASM_linearmemory0","captain":{"titleLarge":1,"titleSmall":2,"labelLayer":3,"largeLayer":4,"smallLayer":5,"scratchpad":6,"url":"ws://localhost/a"},"copilot":{"titleLarge":7,"titleSmall":8,"labelLayer":9,"largeLayer":10,"smallLayer":11,"scratchpad":12,"url":"ws://localhost/b"}}]}`)
	got, err := mergeProfileJSON(base, json)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(base) {
		t.Fatalf("profiles=%d want %d", len(got), len(base))
	}
	if got[0].DisplayName != "Override" {
		t.Fatalf("override not applied: %q", got[0].DisplayName)
	}
}

func TestKnownHashCaseInsensitive(t *testing.T) {
	p := aircraftProfile{KnownModuleSHA256: []string{"ABCDEF"}}
	if !knownHash(&p, "abcdef") {
		t.Fatal("known hash should be case-insensitive")
	}
}
