// SPDX-License-Identifier: MIT
package capturejournal

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func newTestEntry(id string) Entry {
	return Entry{
		ObservationID:         id,
		SourceSystem:          "atb.proxy",
		SourceIncarnation:     "inc-1",
		RepresentationVersion: "atb.intercept.event.v1",
		RepresentationDigest:  "digest-" + id,
		ObservationType:       "atb.llm.request",
		Payload:               json.RawMessage(`{"session_id":"s1"}`),
		ObservedAt:            "2026-01-01T00:00:00Z",
		Adapter:               "atb.intercept",
		AdapterVersion:        "1.0.0",
	}
}

func TestAppendReopenAndReplay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "capture.journal.ndjson")

	j, err := Open(path, "stream-1")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	for i, id := range []string{"obs-1", "obs-2", "obs-3"} {
		e, err := j.Append(newTestEntry(id))
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
		if e.Position != int64(i+1) {
			t.Fatalf("position = %d, want %d", e.Position, i+1)
		}
		if e.Hash == "" {
			t.Fatalf("hash empty")
		}
	}
	if err := j.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	reopened, err := Open(path, "stream-1")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	if reopened.Len() != 3 {
		t.Fatalf("len = %d, want 3", reopened.Len())
	}
	if reopened.LastPosition() != 3 {
		t.Fatalf("last = %d, want 3", reopened.LastPosition())
	}
	if got := reopened.EntriesFrom(1); len(got) != 2 || got[0].ObservationID != "obs-2" {
		t.Fatalf("entries from 1 = %+v", got)
	}

	// Appending after reopen continues the chain.
	e, err := reopened.Append(newTestEntry("obs-4"))
	if err != nil {
		t.Fatalf("append after reopen: %v", err)
	}
	if e.Position != 4 {
		t.Fatalf("position = %d, want 4", e.Position)
	}
}

func TestOpenRepairsTornTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "capture.journal.ndjson")

	j, err := Open(path, "stream-1")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := j.Append(newTestEntry("obs-1")); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := j.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Simulate a crash mid-append: a partial line with no trailing newline.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatalf("open append: %v", err)
	}
	if _, err := f.WriteString(`{"format_version":1,"position":2,"observ`); err != nil {
		t.Fatalf("write partial: %v", err)
	}
	_ = f.Close()

	reopened, err := Open(path, "stream-1")
	if err != nil {
		t.Fatalf("reopen after torn tail: %v", err)
	}
	defer reopened.Close()
	if !reopened.Repaired() {
		t.Fatalf("expected torn tail to be reported as repaired")
	}
	if reopened.Len() != 1 {
		t.Fatalf("len = %d, want 1 after repair", reopened.Len())
	}
	// The journal must be appendable again at position 2.
	e, err := reopened.Append(newTestEntry("obs-2"))
	if err != nil {
		t.Fatalf("append after repair: %v", err)
	}
	if e.Position != 2 {
		t.Fatalf("position = %d, want 2", e.Position)
	}
}

func TestOpenFailsClosedOnInteriorCorruption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "capture.journal.ndjson")

	j, err := Open(path, "stream-1")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := j.Append(newTestEntry("obs-1")); err != nil {
		t.Fatalf("append 1: %v", err)
	}
	if _, err := j.Append(newTestEntry("obs-2")); err != nil {
		t.Fatalf("append 2: %v", err)
	}
	if err := j.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Corrupt the first line's hash while leaving valid newlines.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var e Entry
	firstLineEnd := 0
	for i, b := range data {
		if b == '\n' {
			firstLineEnd = i
			break
		}
	}
	if err := json.Unmarshal(data[:firstLineEnd], &e); err != nil {
		t.Fatalf("unmarshal first: %v", err)
	}
	e.Hash = "deadbeef"
	corrupt, _ := json.Marshal(e)
	out := append(append(corrupt, '\n'), data[firstLineEnd+1:]...)
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatalf("write corrupt: %v", err)
	}

	if _, err := Open(path, "stream-1"); !errors.Is(err, ErrJournalCorrupted) {
		t.Fatalf("expected ErrJournalCorrupted, got %v", err)
	}
}

func TestAppendRejectsMissingIdentity(t *testing.T) {
	dir := t.TempDir()
	j, err := Open(filepath.Join(dir, "capture.journal.ndjson"), "stream-1")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer j.Close()
	if _, err := j.Append(Entry{}); err == nil {
		t.Fatalf("expected error for missing observation id")
	}
}

func TestAppendRejectsFutureVersion(t *testing.T) {
	dir := t.TempDir()
	j, err := Open(filepath.Join(dir, "capture.journal.ndjson"), "stream-1")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer j.Close()
	e := newTestEntry("obs-1")
	e.FormatVersion = JournalFormatVersion + 1
	if _, err := j.Append(e); !errors.Is(err, ErrJournalVersionUnsupported) {
		t.Fatalf("expected ErrJournalVersionUnsupported, got %v", err)
	}
}
