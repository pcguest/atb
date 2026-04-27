// SPDX-License-Identifier: MIT
package golden

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/canonicalize"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/hash"
)

type legacyEvent struct {
	Sequence int    `json:"seq"`
	PrevHash string `json:"prev_hash"`
	Type     string `json:"type"`
	Data     any    `json:"data"`
}

type legacyRecord struct {
	Event legacyEvent `json:"event"`
	Hash  string      `json:"hash"`
}

func TestCanonicalJSON_BackwardCompat(t *testing.T) {
	oldEvent := legacyEvent{
		Sequence: 1,
		PrevHash: hash.GenesisHash,
		Type:     "schema.compat",
		Data:     map[string]any{"x": 1},
	}
	newEvent := event.Event{
		Sequence: 1,
		PrevHash: hash.GenesisHash,
		Type:     "schema.compat",
		Data:     map[string]any{"x": 1},
	}

	oldJSON, err := canonicalize.Marshal(oldEvent)
	if err != nil {
		t.Fatalf("canonicalize legacy event: %v", err)
	}
	newJSON, err := canonicalize.Marshal(newEvent)
	if err != nil {
		t.Fatalf("canonicalize new event: %v", err)
	}
	if !bytes.Equal(oldJSON, newJSON) {
		t.Fatalf("canonical JSON mismatch\nold: %s\nnew: %s", oldJSON, newJSON)
	}
}

func TestLegacyBundleVerifiesWithNewSDK(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "legacy.atb")

	ev := legacyEvent{
		Sequence: 1,
		PrevHash: hash.GenesisHash,
		Type:     "legacy.test",
		Data:     map[string]any{"x": 1},
	}
	record := legacyRecord{
		Event: ev,
		Hash:  computeLegacyHash(t, hash.GenesisHash, ev),
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal legacy record: %v", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write legacy bundle: %v", err)
	}

	loaded, err := bundle.Load(path)
	if err != nil {
		t.Fatalf("load legacy bundle: %v", err)
	}
	if err := loaded.Verify(); err != nil {
		t.Fatalf("verify legacy bundle: %v", err)
	}
	if got := loaded.Records[0].Event.ActorID; got != nil {
		t.Fatalf("expected actor_id to be nil for legacy event")
	}
}

func computeLegacyHash(t *testing.T, prevHash string, ev legacyEvent) string {
	t.Helper()
	canonical, err := canonicalize.Marshal(ev)
	if err != nil {
		t.Fatalf("canonicalize legacy event for hash: %v", err)
	}
	h := sha256.New()
	h.Write([]byte(prevHash))
	h.Write(canonical)
	return hex.EncodeToString(h.Sum(nil))
}
