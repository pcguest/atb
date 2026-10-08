// SPDX-License-Identifier: MIT
package capturejournal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAppendWriteFailurePersistsDegradedMarker covers the honesty gap: a journal
// write failure that leaves the journal file byte-identical to the last commit
// (no torn tail) must still leave a durable degradation signal, so a restart
// cannot report healthy over the dropped observation.
func TestAppendWriteFailurePersistsDegradedMarker(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "j.ndjson")

	j, err := Open(path, "stream")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(Entry{ObservationID: "a", ObservationType: "x", Payload: []byte(`{}`)}); err != nil {
		t.Fatalf("first append: %v", err)
	}

	// Swap the fd for a read-only handle so Write fails with zero bytes written
	// (models ENOSPC-before-write / a lost fsync), leaving no torn tail.
	ro, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ro.Close() }()
	j.f = ro
	if _, err := j.Append(Entry{ObservationID: "b", ObservationType: "x", Payload: []byte(`{}`)}); err == nil {
		t.Fatal("expected append to fail after the fd became read-only")
	}
	j.f = nil

	degraded, reason := Degraded(path)
	if !degraded {
		t.Fatal("expected a durable degradation marker after a write failure")
	}
	if !strings.Contains(reason, "write failed") {
		t.Fatalf("reason = %q, want a write failure reason", reason)
	}

	// Reopening from disk must disclose the degradation even though the file has
	// no torn tail (so Repaired() is false).
	j2, err := Open(path, "stream")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = j2.Close() }()
	if j2.Repaired() {
		t.Fatalf("journal unexpectedly reported a torn-tail repair")
	}
	if d, _ := Degraded(path); !d {
		t.Fatal("reopened journal lost its degradation marker")
	}
}

// TestLostJournalKeepsDegradedMarker verifies that a degradation marker is not
// cleared merely because the journal file is absent: a marker beside a missing
// journal means the journal was lost, which must stay disclosed rather than be
// reported healthy on a fresh start.
func TestLostJournalKeepsDegradedMarker(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "j.ndjson")
	if err := writeDegradedMarker(path, "journal lost"); err != nil {
		t.Fatal(err)
	}
	j, err := Open(path, "stream")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = j.Close() }()
	d, reason := Degraded(path)
	if !d {
		t.Fatal("a lost journal must keep its degradation marker")
	}
	if !strings.Contains(reason, "journal lost") {
		t.Fatalf("reason = %q, want the recorded reason", reason)
	}
}

// TestOversizedEntryPersistsDegradedMarkerAndJournalRemainsUsable verifies that a
// rejected oversized entry is recorded as a durable gap while the journal stays
// usable for later observations.
func TestOversizedEntryPersistsDegradedMarkerAndJournalRemainsUsable(t *testing.T) {
	if testing.Short() {
		t.Skip("allocates an entry larger than the per-line limit")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "j.ndjson")
	j, err := Open(path, "stream")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = j.Close() }()

	big := make([]byte, maxLineBytes+1024)
	for i := range big {
		big[i] = 'a'
	}
	payload := append([]byte(`"`), append(big, '"')...)

	if _, err := j.Append(Entry{ObservationID: "big", ObservationType: "x", Payload: payload}); err == nil {
		t.Fatal("expected oversized append to be rejected")
	}
	degraded, reason := Degraded(path)
	if !degraded || !strings.Contains(reason, "oversized") {
		t.Fatalf("degraded=%v reason=%q, want an oversized rejection marker", degraded, reason)
	}

	// The journal must remain usable after a rejection.
	if _, err := j.Append(Entry{ObservationID: "small", ObservationType: "x", Payload: []byte(`{}`)}); err != nil {
		t.Fatalf("journal should remain usable after a rejection: %v", err)
	}
}
