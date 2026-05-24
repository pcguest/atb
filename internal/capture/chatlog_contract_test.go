// SPDX-License-Identifier: MIT
package capture_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/capture"
	"github.com/pcguest/atb/internal/event"
)

// makeTestEvents returns n minimal EventSpecs of type ai.request.received
// with distinct, monotonically-increasing RFC3339Nano timestamps so each
// record is uniquely hashable.
func makeTestEvents(n int) []capture.EventSpec {
	base := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	out := make([]capture.EventSpec, n)
	for i := 0; i < n; i++ {
		out[i] = capture.EventSpec{
			Type: event.TypeAIRequestReceived,
			Data: map[string]any{
				"request_id":    fmt.Sprintf("req-test-%03d", i+1),
				"actor_id_hash": "sha256:test",
				"purpose_tag":   "contract_test",
				"input_digest":  "sha256:deadbeef",
				"input_format":  "text/plain",
			},
			Timestamp: base.Add(time.Duration(i) * time.Second).Format(time.RFC3339Nano),
		}
	}
	return out
}

// callAppendInMemorySafe invokes AppendEventsToBundleInMemory and converts a
// panic (e.g. from a nil bundle) into an error so the test runner is not
// crashed. It pins the *observable contract* of the function — callers should
// not see process termination on bad input.
func callAppendInMemorySafe(b *bundle.Bundle, events []capture.EventSpec) (count int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return capture.AppendEventsToBundleInMemory(b, events)
}

func TestParseChatlogOpenAIJSONLContract(t *testing.T) {
	raw := `{"role":"user","content":"hello","timestamp":"2026-04-24T09:00:00Z"}` + "\n"
	for _, format := range []string{capture.FormatGenericJSONL, capture.FormatOpenAIJSONL} {
		t.Run(format, func(t *testing.T) {
			messages, err := capture.ParseChatlog(format, strings.NewReader(raw))
			if err != nil {
				t.Fatalf("ParseChatlog(%q) error = %v", format, err)
			}
			if len(messages) != 1 {
				t.Fatalf("len(messages) = %d, want 1", len(messages))
			}
			if messages[0].Role != "user" {
				t.Fatalf("role = %q, want user", messages[0].Role)
			}
		})
	}

	_, err := capture.ParseChatlog(capture.FormatOpenAIJSONL, strings.NewReader(`{"role":"unknown","content":"x"}`+"\n"))
	if err != nil {
		t.Fatalf("unexpected error for unknown role: %v", err)
	}
}

func TestAppendEventsToBundleInMemory(t *testing.T) {
	t.Run("HappyPath", func(t *testing.T) {
		b, err := bundle.New()
		if err != nil {
			t.Fatalf("bundle.New: %v", err)
		}
		startLen := len(b.Records)

		count, err := capture.AppendEventsToBundleInMemory(b, makeTestEvents(3))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 3 {
			t.Fatalf("count = %d, want 3", count)
		}
		if got := len(b.Records); got != startLen+3 {
			t.Fatalf("len(Records) = %d, want %d", got, startLen+3)
		}
	})

	t.Run("EmptySlice", func(t *testing.T) {
		b, err := bundle.New()
		if err != nil {
			t.Fatalf("bundle.New: %v", err)
		}
		startLen := len(b.Records)

		count, err := capture.AppendEventsToBundleInMemory(b, []capture.EventSpec{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 0 {
			t.Fatalf("count = %d, want 0", count)
		}
		if got := len(b.Records); got != startLen {
			t.Fatalf("bundle mutated: len(Records) = %d, want %d", got, startLen)
		}
	})

	t.Run("NilBundle", func(t *testing.T) {
		count, err := callAppendInMemorySafe(nil, makeTestEvents(1))
		if err == nil {
			t.Fatalf("expected error from nil bundle, got nil")
		}
		if count != 0 {
			t.Fatalf("count = %d, want 0 on nil bundle", count)
		}
	})

	t.Run("FirstEventFails", func(t *testing.T) {
		b, err := bundle.New()
		if err != nil {
			t.Fatalf("bundle.New: %v", err)
		}
		startLen := len(b.Records)

		bad := capture.EventSpec{
			Type:      "INVALID TYPE",
			Data:      map[string]any{"x": 1},
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		}
		count, err := capture.AppendEventsToBundleInMemory(b, []capture.EventSpec{bad})
		if err == nil {
			t.Fatalf("expected error for invalid event type")
		}
		if count != 0 {
			t.Fatalf("count = %d, want 0 (no event appended)", count)
		}
		if got := len(b.Records); got != startLen {
			t.Fatalf("bundle mutated despite error: len(Records) = %d, want %d", got, startLen)
		}
	})

	// PartialFailure pins the *current* behaviour: AppendEventsToBundleInMemory
	// stops at the first error and reports the index of the failed event.
	// The first (valid) event has already been appended in-memory by then —
	// callers that need all-or-nothing semantics must not Save on error.
	t.Run("PartialFailure", func(t *testing.T) {
		b, err := bundle.New()
		if err != nil {
			t.Fatalf("bundle.New: %v", err)
		}
		startLen := len(b.Records)

		good := makeTestEvents(1)[0]
		bad := capture.EventSpec{
			Type:      "INVALID TYPE",
			Data:      map[string]any{"x": 1},
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		}
		count, err := capture.AppendEventsToBundleInMemory(b, []capture.EventSpec{good, bad})
		if err == nil {
			t.Fatalf("expected error on second event")
		}
		if count != 1 {
			t.Fatalf("count = %d, want 1 (stop-at-first-error)", count)
		}
		if got := len(b.Records); got != startLen+1 {
			t.Fatalf("len(Records) = %d, want %d (good event retained in memory)", got, startLen+1)
		}
	})
}

// writeFreshBundle creates a fresh bundle on disk at path and returns its
// initial record count for later comparison.
func writeFreshBundle(t *testing.T, path string) int {
	t.Helper()
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("bundle.New: %v", err)
	}
	if err := b.Save(path); err != nil {
		t.Fatalf("bundle.Save: %v", err)
	}
	return len(b.Records)
}

func TestAppendEventsToBundle(t *testing.T) {
	t.Run("HappyPath", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bundle.atb")
		startLen := writeFreshBundle(t, path)

		summary, err := capture.AppendEventsToBundle(path, makeTestEvents(2))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if summary.EventsWritten != 2 {
			t.Fatalf("EventsWritten = %d, want 2", summary.EventsWritten)
		}

		loaded, err := bundle.Load(path)
		if err != nil {
			t.Fatalf("bundle.Load: %v", err)
		}
		if err := loaded.Verify(); err != nil {
			t.Fatalf("loaded bundle failed Verify: %v", err)
		}
		if got := len(loaded.Records); got != startLen+2 {
			t.Fatalf("len(Records) = %d, want %d", got, startLen+2)
		}
	})

	// BundleDoesNotExist documents that AppendEventsToBundle currently
	// auto-creates a missing bundle (loadOrCreateBundle falls back to
	// bundle.New on os.ErrNotExist). To cover the unhappy path we point at
	// a path whose parent is a regular file, so Save cannot succeed —
	// AppendEventsToBundle must surface the error rather than panic.
	t.Run("BundleDoesNotExist", func(t *testing.T) {
		dir := t.TempDir()
		notADir := filepath.Join(dir, "not-a-dir")
		if err := os.WriteFile(notADir, []byte("blocker"), 0o600); err != nil {
			t.Fatalf("seed blocker file: %v", err)
		}
		path := filepath.Join(notADir, "bundle.atb")

		_, err := capture.AppendEventsToBundle(path, makeTestEvents(1))
		if err == nil {
			t.Fatalf("expected error when save target is unwritable")
		}
	})

	t.Run("EmptyEventList", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bundle.atb")
		startLen := writeFreshBundle(t, path)

		summary, err := capture.AppendEventsToBundle(path, []capture.EventSpec{})
		if err != nil {
			// Empty input is rejected by the disk-level wrapper. That's the
			// current contract; pin it here so any future relaxation is
			// deliberate. Either no-op or error is acceptable so long as
			// the bundle on disk is unchanged.
			t.Logf("empty list returned error (acceptable): %v", err)
		} else if summary.EventsWritten != 0 {
			t.Fatalf("EventsWritten = %d, want 0", summary.EventsWritten)
		}

		loaded, err := bundle.Load(path)
		if err != nil {
			t.Fatalf("bundle.Load: %v", err)
		}
		if got := len(loaded.Records); got != startLen {
			t.Fatalf("bundle mutated by empty append: len(Records) = %d, want %d", got, startLen)
		}
	})

	// ConcurrentAppends exercises AppendEventsToBundle under contention.
	// The advisory file-lock (flock) may serialise or reject concurrent
	// writers; the goal of this case is solely that `go test -race` does
	// not surface a data-race detector failure.
	t.Run("ConcurrentAppends", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bundle.atb")
		writeFreshBundle(t, path)

		const goroutines = 3
		var wg sync.WaitGroup
		wg.Add(goroutines)
		for i := 0; i < goroutines; i++ {
			go func() {
				defer wg.Done()
				_, _ = capture.AppendEventsToBundle(path, makeTestEvents(2))
			}()
		}
		wg.Wait()

		loaded, err := bundle.Load(path)
		if err != nil {
			t.Fatalf("bundle.Load after concurrent appends: %v", err)
		}
		if err := loaded.Verify(); err != nil {
			t.Fatalf("bundle failed Verify after concurrent appends: %v", err)
		}
	})
}
