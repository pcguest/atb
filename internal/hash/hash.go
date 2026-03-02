// Package hash implements the ATB SHA-256 hash-chaining algorithm.
// Each event's hash is computed over the canonical JSON of the event
// prepended with the previous event's hash, forming a tamper-evident chain.
package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/pcguest/atb/internal/canonicalize"
)

const (
	// GenesisHash is the sentinel previous-hash used for the first event in a bundle.
	GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"
)

// Event represents a single auditable event in an ATB bundle.
type Event struct {
	// Sequence is the 1-based position of this event in the bundle.
	Sequence int `json:"seq"`
	// PrevHash is the hex-encoded SHA-256 hash of the preceding event.
	// For the first event this MUST equal GenesisHash.
	PrevHash string `json:"prev_hash"`
	// Type is the event type identifier (e.g. "dev.session", "decision").
	Type string `json:"type"`
	// Data is the arbitrary payload associated with this event.
	Data interface{} `json:"data"`
}

// Compute returns the hex-encoded SHA-256 hash for the given event.
// The hash is computed as: SHA256(prevHash || canonicalJSON(event))
func Compute(e Event) (string, error) {
	canonical, err := canonicalize.Marshal(e)
	if err != nil {
		return "", fmt.Errorf("hash: canonicalize event: %w", err)
	}
	h := sha256.New()
	h.Write([]byte(e.PrevHash))
	h.Write(canonical)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Chain computes and assigns hashes for a slice of events in sequence.
// It mutates each event's PrevHash field and returns the final hash.
func Chain(events []Event) (string, error) {
	prev := GenesisHash
	for i := range events {
		events[i].PrevHash = prev
		events[i].Sequence = i + 1
		h, err := Compute(events[i])
		if err != nil {
			return "", fmt.Errorf("hash: chain at index %d: %w", i, err)
		}
		prev = h
	}
	return prev, nil
}

// Verify checks the integrity of a chain of events.
// It returns an error if any hash in the chain is invalid.
func Verify(events []Event, hashes []string) error {
	if len(events) != len(hashes) {
		return fmt.Errorf("hash: verify: event count (%d) != hash count (%d)", len(events), len(hashes))
	}
	prev := GenesisHash
	for i, e := range events {
		e.PrevHash = prev
		e.Sequence = i + 1
		computed, err := Compute(e)
		if err != nil {
			return fmt.Errorf("hash: verify at index %d: %w", i, err)
		}
		if computed != hashes[i] {
			return fmt.Errorf("hash: verify: tamper detected at event %d (seq %d): expected %s, got %s",
				i, e.Sequence, hashes[i], computed)
		}
		prev = computed
	}
	return nil
}
