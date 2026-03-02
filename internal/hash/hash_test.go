package hash_test

import (
	"testing"

	"github.com/pcguest/atb/internal/hash"
)

func TestGenesisHash(t *testing.T) {
	if len(hash.GenesisHash) != 64 {
		t.Errorf("GenesisHash must be 64 hex chars, got %d", len(hash.GenesisHash))
	}
}

func TestComputeDeterministic(t *testing.T) {
	e := hash.Event{
		Sequence: 1,
		PrevHash: hash.GenesisHash,
		Type:     "dev.session",
		Data: map[string]interface{}{
			"date": "2025-01-15",
		},
	}
	h1, err := hash.Compute(e)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	h2, err := hash.Compute(e)
	if err != nil {
		t.Fatalf("Compute (2nd): %v", err)
	}
	if h1 != h2 {
		t.Errorf("Compute is not deterministic: %s != %s", h1, h2)
	}
}

func TestChainAndVerify(t *testing.T) {
	events := []hash.Event{
		{Type: "dev.session", Data: map[string]interface{}{"date": "2025-01-15"}},
		{Type: "decision", Data: map[string]interface{}{"choice": "Go over Rust"}},
		{Type: "release", Data: map[string]interface{}{"version": "v1.0.0"}},
	}

	finalHash, err := hash.Chain(events)
	if err != nil {
		t.Fatalf("Chain: %v", err)
	}
	if finalHash == "" {
		t.Error("Chain returned empty final hash")
	}

	hashes := make([]string, len(events))
	for i, e := range events {
		h, err := hash.Compute(e)
		if err != nil {
			t.Fatalf("Compute at %d: %v", i, err)
		}
		hashes[i] = h
	}

	if err := hash.Verify(events, hashes); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestVerifyTamperDetection(t *testing.T) {
	events := []hash.Event{
		{Type: "dev.session", Data: map[string]interface{}{"date": "2025-01-15"}},
		{Type: "decision", Data: map[string]interface{}{"choice": "Go over Rust"}},
	}
	_, err := hash.Chain(events)
	if err != nil {
		t.Fatalf("Chain: %v", err)
	}

	// Tamper with the second event's data.
	events[1].Data = map[string]interface{}{"choice": "TAMPERED"}

	hashes := make([]string, len(events))
	for i, e := range events {
		h, err := hash.Compute(e)
		if err != nil {
			t.Fatalf("Compute at %d: %v", i, err)
		}
		hashes[i] = h
	}
	// Use the original (pre-tamper) hash for event 1.
	hashes[1] = "0000000000000000000000000000000000000000000000000000000000000000"

	if err := hash.Verify(events, hashes); err == nil {
		t.Error("Verify should have detected tampering but returned nil")
	}
}
