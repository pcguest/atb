// SPDX-License-Identifier: MIT
package event_test

import (
	"bytes"
	"testing"

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

func TestCanonicalBackwardCompatWithUnsetOptionalFields(t *testing.T) {
	old := legacyEvent{
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
	oldCanonical, err := canonicalize.Marshal(old)
	if err != nil {
		t.Fatalf("canonicalize legacy event: %v", err)
	}
	newCanonical, err := canonicalize.Marshal(newEvent)
	if err != nil {
		t.Fatalf("canonicalize new event: %v", err)
	}
	if !bytes.Equal(oldCanonical, newCanonical) {
		t.Fatalf("canonical mismatch\ngot:  %s\nwant: %s", string(newCanonical), string(oldCanonical))
	}
}

func TestCanonicalIncludesOptionalFieldsWhenSet(t *testing.T) {
	actorID := "actor-123"
	base := event.Event{
		Sequence: 1,
		PrevHash: hash.GenesisHash,
		Type:     "schema.compat",
		Data:     map[string]any{"x": 1},
	}
	withActor := base
	withActor.ActorID = &actorID

	baseCanonical, err := canonicalize.Marshal(base)
	if err != nil {
		t.Fatalf("canonicalize base event: %v", err)
	}
	withActorCanonical, err := canonicalize.Marshal(withActor)
	if err != nil {
		t.Fatalf("canonicalize with actor event: %v", err)
	}
	if bytes.Equal(baseCanonical, withActorCanonical) {
		t.Fatalf("expected canonical JSON to differ when optional field is set")
	}
}
