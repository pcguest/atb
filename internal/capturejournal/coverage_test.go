// SPDX-License-Identifier: MIT
package capturejournal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestJournalAccessorsAndReadMissing(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "j.ndjson")
	j, err := Open(p, "stream-1")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if j.Path() != p {
		t.Fatalf("Path = %q, want %q", j.Path(), p)
	}
	if j.StreamID() != "stream-1" {
		t.Fatalf("StreamID = %q", j.StreamID())
	}
	if len(j.Entries()) != 0 {
		t.Fatalf("fresh Entries = %d, want 0", len(j.Entries()))
	}
	if _, err := j.Append(Entry{
		ObservationID:         "s:1",
		SourceSystem:          "x",
		RepresentationVersion: "v",
		RepresentationDigest:  "d",
		ObservationType:       "t",
		Payload:               json.RawMessage(`{}`),
		ObservedAt:            "now",
		Adapter:               "a",
		AdapterVersion:        "1",
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if len(j.Entries()) != 1 {
		t.Fatalf("Entries = %d, want 1", len(j.Entries()))
	}
	if err := j.Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	_ = j.Close()

	if entries, repaired, err := Read(filepath.Join(dir, "missing.ndjson")); err != nil || repaired || entries != nil {
		t.Fatalf("Read(missing) = (%v, %v, %v)", entries, repaired, err)
	}
}

func TestReadReportsRepairMarker(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "j.ndjson")
	j, err := Open(p, "stream")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := j.Append(Entry{
		ObservationID:         "s:1",
		SourceSystem:          "x",
		RepresentationVersion: "v",
		RepresentationDigest:  "d",
		ObservationType:       "t",
		Payload:               json.RawMessage(`{}`),
		ObservedAt:            "now",
		Adapter:               "a",
		AdapterVersion:        "1",
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	_ = j.Close()

	// Torn tail, then reopen to repair (writes the marker).
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatalf("open append: %v", err)
	}
	if _, err := f.WriteString(`{"format_version":1,"position":2,"observation`); err != nil {
		t.Fatalf("write torn tail: %v", err)
	}
	_ = f.Close()

	j2, err := Open(p, "stream")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if !j2.Repaired() {
		t.Fatal("reopened journal should report repaired")
	}
	_ = j2.Close()

	entries, repaired, err := Read(p)
	if err != nil || !repaired {
		t.Fatalf("Read after repair = (repaired %v, err %v), want repaired", repaired, err)
	}
	if len(entries) != 1 {
		t.Fatalf("Read after repair returned %d entries, want 1", len(entries))
	}
}

func TestReadMissingWithStaleMarker(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "j.ndjson")
	// A marker with no journal must not be reported as a repair.
	if err := os.WriteFile(repairMarkerPath(p), []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write marker: %v", err)
	}
	entries, repaired, err := Read(p)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if entries != nil || repaired {
		t.Fatalf("Read(missing journal, stale marker) = (%v, %v), want (nil, false)", entries, repaired)
	}
}
