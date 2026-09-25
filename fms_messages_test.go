package main

import "testing"

func TestFokkerFMSMessageMappings(t *testing.T) {
	p := defaultProfiles()[0]

	cases := map[uint32]uint64{
		1: 0x21E80,
		2: 0x21EC0,
		1001: 0x2C050,
		1004: 0x2C120,
		1009: 0x2C2B4,
	}
	for selector, want := range cases {
		got, ok := lookupFMSMessageOffset(p.FMSMessages, selector)
		if !ok || got != want {
			t.Fatalf("selector %d: got 0x%X ok=%t, want 0x%X", selector, got, ok, want)
		}
	}
}

func TestFokkerMessageSelectors(t *testing.T) {
	p := defaultProfiles()[0]
	if p.Captain.MessageSelector != 0x182398 {
		t.Fatalf("captain selector=0x%X", p.Captain.MessageSelector)
	}
	if p.Copilot.MessageSelector != 0x18239C {
		t.Fatalf("copilot selector=0x%X", p.Copilot.MessageSelector)
	}
}

func TestUnknownOrZeroSelectorFallsBack(t *testing.T) {
	p := defaultProfiles()[0]
	if _, ok := lookupFMSMessageOffset(p.FMSMessages, 0); ok {
		t.Fatal("selector 0 must use normal scratchpad")
	}
	if _, ok := lookupFMSMessageOffset(p.FMSMessages, 999999); ok {
		t.Fatal("unknown selector must fall back to normal scratchpad")
	}
}
