// Package hash implements the ATB SHA-256 hash-chaining algorithm.
// Each event's hash is computed over the canonical JSON of the event
// prepended with the previous event's hash, forming a tamper-evident chain.
package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/pcguest/atb/internal/canonicalize"
	"github.com/pcguest/atb/internal/event"
)

const (
	// GenesisHash is the sentinel previous-hash used for the first event in a bundle.
	GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"
)

// Event is the canonical ATB event model used by hashing and bundle records.
type Event = event.Event

// Compute returns the hex-encoded SHA-256 hash for the given event.
// The hash is computed as: SHA256(prevHash || canonicalJSON(event))
// NOTE: canonical hash output changed in v1.1.2 for float values >= 1e21
// or < 1e-6. Bundles containing such values written before this version
// will not re-verify against this implementation.
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
