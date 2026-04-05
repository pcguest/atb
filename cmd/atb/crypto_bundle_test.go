package main

import (
	"testing"

	"github.com/pcguest/atb/internal/canonicalize"
	"github.com/pcguest/atb/internal/hash"
)

func TestBundlePayloadRoundTripVerifies(t *testing.T) {
	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "dev.session", map[string]any{"ok": true})

	raw, err := canonicalPayloadFromBundle(b)
	if err != nil {
		t.Fatalf("canonical payload: %v", err)
	}
	out, err := bundleFromCanonicalPayload(raw)
	if err != nil {
		t.Fatalf("bundle from payload: %v", err)
	}
	if len(out.Records) != 2 {
		t.Fatalf("unexpected record count: got %d want 2", len(out.Records))
	}
}

func TestBundlePayloadHeadHashMismatchFails(t *testing.T) {
	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "dev.session", map[string]any{"ok": true})
	payload := encryptedBundlePayload{
		HeadHash: hash.GenesisHash,
		Records:  b.Records,
	}
	raw, err := canonicalize.Marshal(payload)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}
	if _, err := bundleFromCanonicalPayload(raw); err == nil {
		t.Fatalf("expected head hash mismatch error")
	}
}
