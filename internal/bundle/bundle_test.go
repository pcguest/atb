// SPDX-License-Identifier: MIT
package bundle_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/hash"
)

type slowJSON struct {
	Value int
}

func (s slowJSON) MarshalJSON() ([]byte, error) {
	time.Sleep(100 * time.Microsecond)
	return []byte(fmt.Sprintf(`{"value":%d}`, s.Value)), nil
}

func newTestBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b, err := bundle.New()
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	return b
}

func TestNewBundleID(t *testing.T) {
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if b == nil {
		t.Fatalf("New() returned nil bundle")
	}
	if len(b.Records) == 0 {
		t.Fatalf("New() returned bundle without manifest record")
	}
	if b.Records[0].Event.Type != bundle.ManifestEventType {
		t.Fatalf("record 0 event type = %q, want %q", b.Records[0].Event.Type, bundle.ManifestEventType)
	}
	manifest := b.Manifest()
	if manifest == nil {
		t.Fatalf("expected manifest on new bundle")
	}
	if manifest.BundleID == "" {
		t.Fatalf("expected non-empty manifest bundle_id")
	}
}

func TestLoadCancelled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")
	b := newTestBundle(t)
	if err := b.Save(context.Background(), path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := bundle.Load(ctx, path)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Load() error = %v, want context.Canceled", err)
	}
}

func TestSaveCancelled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")
	b := newTestBundle(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := b.Save(ctx, path)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Save() error = %v, want context.Canceled", err)
	}
}

func TestSaveContextCancelledMidEncode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")
	existing := newTestBundle(t)
	if err := existing.Save(context.Background(), path); err != nil {
		t.Fatalf("save existing bundle: %v", err)
	}

	records := make([]bundle.Record, 1001)
	for i := range records {
		eventType := "ai.tool.exec"
		seq := i + 1
		if i == 0 {
			eventType = bundle.ManifestEventType
			seq = 0
		}
		records[i] = bundle.Record{
			Event: hash.Event{
				Sequence: seq,
				PrevHash: strings.Repeat("0", 64),
				Type:     eventType,
				HashAlgo: "sha256",
				Data:     slowJSON{Value: i},
			},
			Hash: strings.Repeat("a", 64),
		}
	}
	slowBundle := &bundle.Bundle{Records: records}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- slowBundle.Save(ctx, path)
	}()

	time.Sleep(time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Save() error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Save() did not return after context cancellation")
	}

	loaded, err := bundle.Load(context.Background(), path)
	if err != nil && !errors.Is(err, bundle.ErrMalformed) {
		t.Fatalf("post-cancel Load() error = %v, want nil or ErrMalformed", err)
	}
	if loaded != nil {
		if err := loaded.Verify(); err != nil {
			t.Fatalf("post-cancel bundle should remain verifiable when load succeeds: %v", err)
		}
	}
}

func TestAppendRejectsInvalidEventType(t *testing.T) {
	b := newTestBundle(t)

	if err := b.AppendWithOptions("INVALID", nil, &bundle.AppendOptions{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}); err == nil {
		t.Fatalf("expected invalid event type to return an error")
	}
}

func TestAppendAcceptsUnderscoreEventTypeSegments(t *testing.T) {
	b := newTestBundle(t)

	if err := b.AppendWithOptions("atb.event.rag_index", map[string]any{"ok": true}, &bundle.AppendOptions{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("expected underscore-bearing event type to append cleanly: %v", err)
	}
}

func TestAppendWithOptionsRoundTripsCanonicalFields(t *testing.T) {
	b := newTestBundle(t)
	path := filepath.Join(t.TempDir(), "bundle.atb")

	opts := &bundle.AppendOptions{
		Timestamp:    "2026-03-27T00:00:00Z",
		TraceID:      "0123456789abcdef0123456789abcdef",
		SpanID:       "0123456789abcdef",
		ParentSpanID: "fedcba9876543210",
	}
	if err := b.AppendWithOptions("ai.tool.exec", map[string]any{"ok": true}, opts); err != nil {
		t.Fatalf("append with options: %v", err)
	}
	if err := b.Save(context.Background(), path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	loaded, err := bundle.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if len(loaded.Records) != 2 {
		t.Fatalf("expected 2 loaded records, got %d", len(loaded.Records))
	}

	got := loaded.Records[1].Event
	if got.Timestamp != opts.Timestamp {
		t.Fatalf("timestamp: got %q want %q", got.Timestamp, opts.Timestamp)
	}
	if got.TraceID != opts.TraceID {
		t.Fatalf("trace_id: got %q want %q", got.TraceID, opts.TraceID)
	}
	if got.SpanID != opts.SpanID {
		t.Fatalf("span_id: got %q want %q", got.SpanID, opts.SpanID)
	}
	if got.ParentSpanID != opts.ParentSpanID {
		t.Fatalf("parent_span_id: got %q want %q", got.ParentSpanID, opts.ParentSpanID)
	}
}

func TestNewBundleHasManifest(t *testing.T) {
	b := newTestBundle(t)

	manifest := b.Manifest()
	if manifest == nil {
		t.Fatalf("expected manifest on new bundle")
	}
	if manifest.Version != bundle.ManifestVersion {
		t.Fatalf("manifest version: got %q want %q", manifest.Version, bundle.ManifestVersion)
	}
	if manifest.CreatedAt == "" {
		t.Fatalf("expected non-empty manifest created_at")
	}
	if len(manifest.BundleID) != 32 {
		t.Fatalf("manifest bundle_id length: got %d want 32", len(manifest.BundleID))
	}
	if strings.Trim(manifest.BundleID, "0123456789abcdef") != "" {
		t.Fatalf("manifest bundle_id should be lowercase hex, got %q", manifest.BundleID)
	}
	t.Logf("manifest=%+v", *manifest)
}

func TestManifestCaptureRunIDRoundTrips(t *testing.T) {
	b, err := bundle.NewWithOptions(bundle.NewOptions{CaptureRunID: "cap-test-123"})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	manifest := b.Manifest()
	if manifest == nil {
		t.Fatalf("expected manifest")
	}
	if manifest.CaptureRunID != "cap-test-123" {
		t.Fatalf("CaptureRunID = %q, want %q", manifest.CaptureRunID, "cap-test-123")
	}

	path := filepath.Join(t.TempDir(), "bundle.atb")
	if err := b.Save(context.Background(), path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}
	loaded, err := bundle.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	loadedManifest := loaded.Manifest()
	if loadedManifest == nil {
		t.Fatalf("expected loaded manifest")
	}
	if loadedManifest.CaptureRunID != "cap-test-123" {
		t.Fatalf("loaded CaptureRunID = %q, want %q", loadedManifest.CaptureRunID, "cap-test-123")
	}
}

func TestManifestOmitsEmptyCaptureRunIDAndLoads(t *testing.T) {
	b := newTestBundle(t)
	path := filepath.Join(t.TempDir(), "bundle.atb")
	if err := b.Save(context.Background(), path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	if strings.Contains(string(raw), "capture_run_id") {
		t.Fatalf("manifest unexpectedly contains capture_run_id: %s", raw)
	}

	loaded, err := bundle.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("load bundle without capture_run_id: %v", err)
	}
	manifest := loaded.Manifest()
	if manifest == nil {
		t.Fatalf("expected loaded manifest")
	}
	if manifest.CaptureRunID != "" {
		t.Fatalf("CaptureRunID = %q, want empty", manifest.CaptureRunID)
	}
}

func TestLoadLegacyBundleNoManifest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.atb")
	event := hash.Event{
		Sequence: 1,
		PrevHash: hash.GenesisHash,
		Type:     "ai.tool.exec",
		HashAlgo: "sha256",
		Data: map[string]any{
			"ok": true,
		},
	}
	sum, err := hash.Compute(event)
	if err != nil {
		t.Fatalf("compute legacy event hash: %v", err)
	}
	record := bundle.Record{
		Event: event,
		Hash:  sum,
	}
	line, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal legacy record: %v", err)
	}
	if err := os.WriteFile(path, append(line, '\n'), 0o600); err != nil {
		t.Fatalf("write legacy bundle: %v", err)
	}

	loaded, err := bundle.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("load legacy bundle: %v", err)
	}
	if loaded.Manifest() != nil {
		t.Fatalf("expected nil manifest for legacy bundle")
	}
}

func TestVerifyDetectsSequenceTampering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")
	b := newTestBundle(t)

	if err := b.Append("ai.tool.exec", map[string]any{"step": 1}); err != nil {
		t.Fatalf("append first event: %v", err)
	}
	if err := b.Append("ai.tool.exec", map[string]any{"step": 2}); err != nil {
		t.Fatalf("append second event: %v", err)
	}
	if err := b.Save(context.Background(), path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	loaded, err := bundle.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}

	loaded.Records[2].Event.Sequence = 99

	err = loaded.Verify()
	if err == nil {
		t.Fatal("expected sequence tampering to be detected")
	}
	if !strings.Contains(err.Error(), "sequence mismatch") {
		t.Fatalf("expected sequence mismatch error, got %v", err)
	}
}

func TestSaveAtomic(t *testing.T) {
	t.Run("saves 5 events produces valid file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bundle.atb")
		b, err := bundle.New()
		if err != nil {
			t.Fatalf("new bundle: %v", err)
		}
		for i := 0; i < 5; i++ {
			if err := b.Append("ai.tool.exec", map[string]any{"step": i}); err != nil {
				t.Fatalf("append event %d: %v", i, err)
			}
		}
		if err := b.Save(context.Background(), path); err != nil {
			t.Fatalf("save: %v", err)
		}
		loaded, err := bundle.Load(context.Background(), path)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		// 1 manifest + 5 events = 6 records
		if len(loaded.Records) != 6 {
			t.Fatalf("expected 6 records, got %d", len(loaded.Records))
		}
		if err := loaded.Verify(); err != nil {
			t.Fatalf("verify: %v", err)
		}
	})

	t.Run("original file not truncated when save fails", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("root ignores file permission checks")
		}
		if runtime.GOOS == "windows" {
			t.Skip("Windows chmod does not restrict directory writes")
		}
		dir := t.TempDir()
		path := filepath.Join(dir, "bundle.atb")

		// Save the original bundle.
		orig, err := bundle.New()
		if err != nil {
			t.Fatalf("new original bundle: %v", err)
		}
		if err := orig.Append("ai.tool.exec", map[string]any{"original": true}); err != nil {
			t.Fatalf("append: %v", err)
		}
		if err := orig.Save(context.Background(), path); err != nil {
			t.Fatalf("save original: %v", err)
		}
		origStat, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat original: %v", err)
		}

		// Make the directory read-only so CreateTemp fails before touching the target file.
		if err := os.Chmod(dir, 0555); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0755) })

		// Attempt to save a new bundle to the same path — must fail.
		newB, err := bundle.New()
		if err != nil {
			t.Fatalf("new bundle: %v", err)
		}
		if err := newB.Save(context.Background(), path); err == nil {
			t.Fatal("expected Save to fail with read-only directory")
		}

		// Original file must still exist with its original size (not truncated).
		newStat, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat after failed save: %v", err)
		}
		if newStat.Size() != origStat.Size() {
			t.Fatalf("file size changed: was %d bytes, now %d bytes — original was truncated", origStat.Size(), newStat.Size())
		}
	})
}

func TestSaveConcurrentWritersSerialiseOrReturnLocked(t *testing.T) {
	requireAdvisoryLocking(t)
	t.Parallel()

	const goroutines = 5
	const rounds = 5

	for round := 0; round < rounds; round++ {
		round := round
		t.Run(fmt.Sprintf("round_%d", round), func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "bundle.atb")
			start := make(chan struct{})
			results := make([]error, goroutines)
			var wg sync.WaitGroup

			for i := 0; i < goroutines; i++ {
				i := i
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start

					b, err := bundle.New()
					if err != nil {
						results[i] = err
						return
					}
					if err := b.Append("ai.tool.exec", map[string]any{
						"round":  round,
						"writer": i,
					}); err != nil {
						results[i] = err
						return
					}
					results[i] = b.Save(context.Background(), path)
				}()
			}

			close(start)
			wg.Wait()

			successes := 0
			for _, err := range results {
				switch {
				case err == nil:
					successes++
				case errors.Is(err, bundle.ErrBundleLocked):
				default:
					t.Fatalf("unexpected Save error: %v", err)
				}
			}
			if successes == 0 {
				t.Fatal("expected at least one Save to succeed")
			}

			loaded, err := bundle.Load(context.Background(), path)
			if err != nil {
				t.Fatalf("load final bundle: %v", err)
			}
			if err := loaded.Verify(); err != nil {
				t.Fatalf("verify final bundle: %v", err)
			}
		})
	}
}

func TestVerifyDetectsHashTampering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")
	b := newTestBundle(t)

	if err := b.Append("ai.tool.exec", map[string]any{"step": 1}); err != nil {
		t.Fatalf("append event: %v", err)
	}
	if err := b.Save(context.Background(), path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	loaded, err := bundle.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}

	loaded.Records[1].Hash = strings.Repeat("0", 64)

	err = loaded.Verify()
	if err == nil {
		t.Fatal("expected hash tampering to be detected")
	}
	if !strings.Contains(err.Error(), "tamper detected") {
		t.Fatalf("expected tamper detected error, got %v", err)
	}
}

func TestManifestV1OpensAndVerifies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v1.atb")
	b, err := bundle.New() // default v1
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := b.Append("ai.tool.exec", map[string]any{"k": "v"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := b.Save(context.Background(), path); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := bundle.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("load v1: %v", err)
	}
	if err := loaded.Verify(); err != nil {
		t.Fatalf("verify v1: %v", err)
	}
	if m := loaded.Manifest(); m == nil || m.Version != bundle.ManifestVersion {
		t.Fatalf("v1 manifest = %+v, want version %s", m, bundle.ManifestVersion)
	}
}

func TestManifestV2OpensAndVerifies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v2.atb")
	b, err := bundle.NewWithOptions(bundle.NewOptions{ManifestVersion: bundle.ManifestVersionV2})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := b.Append("ai.tool.exec", map[string]any{"k": "v"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := b.Save(context.Background(), path); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := bundle.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("load v2: %v", err)
	}
	if err := loaded.Verify(); err != nil {
		t.Fatalf("verify v2: %v", err)
	}
	if m := loaded.Manifest(); m == nil || m.Version != "2" {
		t.Fatalf("v2 manifest = %+v, want version 2", m)
	}
	if loaded.Records[0].Event.Data == nil {
		t.Fatal("v2 manifest data unexpectedly nil")
	}
	if _, ok := loaded.Records[0].Event.Data.(string); ok {
		t.Fatal("v2 manifest must not be a JSON-encoded string")
	}
}

func TestManifestV2RoundTrip(t *testing.T) {
	b, err := bundle.NewWithOptions(bundle.NewOptions{ManifestVersion: bundle.ManifestVersionV2})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	original := b.Manifest()
	if original == nil {
		t.Fatal("expected manifest")
	}

	path := filepath.Join(t.TempDir(), "rt.atb")
	if err := b.Save(context.Background(), path); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := bundle.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	got := loaded.Manifest()
	if got == nil {
		t.Fatal("loaded manifest nil")
	}
	if got.Version != original.Version || got.CreatedAt != original.CreatedAt || got.BundleID != original.BundleID {
		t.Fatalf("v2 round-trip drift: original=%+v loaded=%+v", *original, *got)
	}
}

func TestManifestUnknownVersionWrapsErrMalformed(t *testing.T) {
	// Hand-craft a bundle file whose manifest declares an unsupported version 99.
	createdAt := time.Now().UTC().Format(time.RFC3339)
	manifestData := map[string]any{
		"version":    99,
		"created_at": createdAt,
		"bundle_id":  "00112233445566778899aabbccddeeff",
	}
	manifestEvent := map[string]any{
		"seq":       0,
		"prev_hash": hash.GenesisHash,
		"type":      bundle.ManifestEventType,
		"hash_algo": "sha256",
		"timestamp": createdAt,
		"data":      manifestData,
	}
	record := map[string]any{
		"event": manifestEvent,
		"hash":  strings.Repeat("0", 64),
	}
	line, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	path := filepath.Join(t.TempDir(), "v99.atb")
	if err := os.WriteFile(path, append(line, '\n'), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err = bundle.Load(context.Background(), path)
	if err == nil {
		t.Fatal("expected error loading bundle with manifest version 99")
	}
	if !errors.Is(err, bundle.ErrMalformed) {
		t.Fatalf("expected error wrapping ErrMalformed, got %v", err)
	}
	if !strings.Contains(err.Error(), "unsupported manifest version") {
		t.Fatalf("expected 'unsupported manifest version' in error, got %v", err)
	}
}

func TestLegacyCompatibilityRefusesWithoutRewriting(t *testing.T) {
	t.Parallel()

	createdAt := "2026-04-01T00:00:00Z"
	futureManifestData := map[string]any{
		"version":    bundle.ManifestVersionMax + 1,
		"created_at": createdAt,
		"bundle_id":  "00112233445566778899aabbccddeeff",
	}
	futureManifestEvent := map[string]any{
		"seq":       0,
		"prev_hash": hash.GenesisHash,
		"type":      bundle.ManifestEventType,
		"hash_algo": "sha256",
		"timestamp": createdAt,
		"data":      futureManifestData,
	}
	futureRecord, err := json.Marshal(map[string]any{
		"event": futureManifestEvent,
		"hash":  strings.Repeat("0", 64),
	})
	if err != nil {
		t.Fatalf("marshal future manifest: %v", err)
	}

	cases := []struct {
		name        string
		contents    []byte
		want        error
		wantMessage string
	}{
		{
			name:        "future manifest version",
			contents:    append(futureRecord, '\n'),
			want:        bundle.ErrMalformed,
			wantMessage: "unsupported manifest version",
		},
		{
			name:        "truncated JSON record",
			contents:    []byte(`{"event":{"seq":0,"type":"atb.bundle.manifest"}`),
			want:        bundle.ErrMalformed,
			wantMessage: "unmarshal",
		},
		{
			name:        "legacy JSON array is not NDJSON",
			contents:    []byte(`[{"event":{"seq":0}}]` + "\n"),
			want:        bundle.ErrMalformed,
			wantMessage: "unmarshal",
		},
		{
			name:        "pre-manifest event is not verified as current evidence",
			contents:    []byte(`{"event":{"seq":1,"prev_hash":"0000000000000000000000000000000000000000000000000000000000000000","type":"ai.tool.exec","hash_algo":"sha256","data":{"step":1}},"hash":"0000000000000000000000000000000000000000000000000000000000000000"}` + "\n"),
			want:        bundle.ErrNotABundle,
			wantMessage: "record 0 type",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "legacy.atb")
			if err := os.WriteFile(path, tc.contents, 0o600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read before: %v", err)
			}

			_, err = bundle.LoadVerified(path)
			if !errors.Is(err, tc.want) {
				t.Fatalf("LoadVerified error = %v, want errors.Is(..., %v)", err, tc.want)
			}
			if tc.wantMessage != "" && !strings.Contains(err.Error(), tc.wantMessage) {
				t.Fatalf("LoadVerified error = %v, want message containing %q", err, tc.wantMessage)
			}

			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read after: %v", err)
			}
			if string(after) != string(before) {
				t.Fatalf("legacy refusal rewrote source evidence")
			}
		})
	}
}

func TestManifestEValidBundle(t *testing.T) {
	b := newTestBundle(t)

	m, err := b.ManifestE()
	if err != nil {
		t.Fatalf("ManifestE on new bundle: unexpected error %v", err)
	}
	if m == nil {
		t.Fatal("ManifestE returned nil ManifestData")
	}
	if m.Version != bundle.ManifestVersion {
		t.Fatalf("manifest version: got %q want %q", m.Version, bundle.ManifestVersion)
	}
}

func TestManifestEEmptyBundle(t *testing.T) {
	b := &bundle.Bundle{}

	m, err := b.ManifestE()
	if m != nil {
		t.Fatalf("expected nil manifest, got %+v", m)
	}
	if !errors.Is(err, bundle.ErrNoManifest) {
		t.Fatalf("expected ErrNoManifest, got %v", err)
	}
}

func TestManifestEFirstRecordNotManifest(t *testing.T) {
	b := &bundle.Bundle{
		Records: []bundle.Record{
			{
				Event: hash.Event{
					Sequence: 1,
					PrevHash: hash.GenesisHash,
					Type:     "ai.tool.exec",
					HashAlgo: "sha256",
					Data:     map[string]any{"step": 1},
				},
			},
		},
	}

	m, err := b.ManifestE()
	if m != nil {
		t.Fatalf("expected nil manifest, got %+v", m)
	}
	if !errors.Is(err, bundle.ErrNoManifest) {
		t.Fatalf("expected ErrNoManifest, got %v", err)
	}
}

func TestManifestEManifestRecordWithInvalidJSON(t *testing.T) {
	b := &bundle.Bundle{
		Records: []bundle.Record{
			{
				Event: hash.Event{
					Sequence: 0,
					PrevHash: hash.GenesisHash,
					Type:     bundle.ManifestEventType,
					HashAlgo: "sha256",
					Data:     "{not valid json",
				},
			},
		},
	}

	m, err := b.ManifestE()
	if m != nil {
		t.Fatalf("expected nil manifest, got %+v", m)
	}
	if !errors.Is(err, bundle.ErrMalformed) {
		t.Fatalf("expected error wrapping ErrMalformed, got %v", err)
	}
}

func TestLoadVerified(t *testing.T) {
	t.Parallel()

	writeBundle := func(t *testing.T, b *bundle.Bundle, path string) {
		t.Helper()
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		enc := json.NewEncoder(f)
		for _, r := range b.Records {
			if err := enc.Encode(r); err != nil {
				_ = f.Close()
				t.Fatalf("encode: %v", err)
			}
		}
		if err := f.Close(); err != nil {
			t.Fatalf("close: %v", err)
		}
	}

	makeValid := func(t *testing.T, dir string) string {
		t.Helper()
		b := newTestBundle(t)
		if err := b.AppendWithOptions("ai.tool.exec", map[string]any{"step": 1}, &bundle.AppendOptions{
			Timestamp: "2026-04-01T00:00:00Z",
		}); err != nil {
			t.Fatalf("append: %v", err)
		}
		path := filepath.Join(dir, "valid.atb")
		writeBundle(t, b, path)
		return path
	}

	makeEmpty := func(t *testing.T, dir string) string {
		t.Helper()
		path := filepath.Join(dir, "empty.atb")
		if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
			t.Fatalf("write empty: %v", err)
		}
		return path
	}

	makeNoManifest := func(t *testing.T, dir string) string {
		t.Helper()
		event := hash.Event{
			Sequence: 1,
			PrevHash: hash.GenesisHash,
			Type:     "ai.tool.exec",
			HashAlgo: "sha256",
			Data:     map[string]any{"x": 1},
		}
		h, err := hash.Compute(event)
		if err != nil {
			t.Fatalf("compute hash: %v", err)
		}
		rec := bundle.Record{Event: event, Hash: h}
		buf, err := json.Marshal(rec)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		path := filepath.Join(dir, "no_manifest.atb")
		if err := os.WriteFile(path, append(buf, '\n'), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		return path
	}

	makeTampered := func(t *testing.T, dir string) string {
		t.Helper()
		b := newTestBundle(t)
		if err := b.AppendWithOptions("ai.tool.exec", map[string]any{"step": 1}, &bundle.AppendOptions{
			Timestamp: "2026-04-01T00:00:00Z",
		}); err != nil {
			t.Fatalf("append: %v", err)
		}
		path := filepath.Join(dir, "tampered.atb")
		writeBundle(t, b, path)

		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		// Locate the second record's hash field and flip one hex char.
		lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
		if len(lines) < 2 {
			t.Fatalf("expected at least 2 records, got %d", len(lines))
		}
		var rec bundle.Record
		if err := json.Unmarshal([]byte(lines[1]), &rec); err != nil {
			t.Fatalf("unmarshal record 1: %v", err)
		}
		// Flip first hex character of the stored hash.
		flipped := []byte(rec.Hash)
		if flipped[0] == '0' {
			flipped[0] = '1'
		} else {
			flipped[0] = '0'
		}
		rec.Hash = string(flipped)
		buf, err := json.Marshal(rec)
		if err != nil {
			t.Fatalf("marshal tampered: %v", err)
		}
		lines[1] = string(buf)
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
			t.Fatalf("rewrite: %v", err)
		}
		return path
	}

	cases := []struct {
		name    string
		make    func(t *testing.T, dir string) string
		wantErr error // sentinel that errors.Is must match; nil means expect nil error
		assert  func(t *testing.T, b *bundle.Bundle, err error)
	}{
		{
			name:    "valid bundle",
			make:    makeValid,
			wantErr: nil,
			assert: func(t *testing.T, b *bundle.Bundle, err error) {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				if b == nil {
					t.Fatalf("expected non-nil bundle")
				}
			},
		},
		{
			name:    "empty file",
			make:    makeEmpty,
			wantErr: bundle.ErrNoManifest,
		},
		{
			name:    "ndjson but no manifest",
			make:    makeNoManifest,
			wantErr: bundle.ErrNotABundle,
		},
		{
			name:    "tampered chain",
			make:    makeTampered,
			wantErr: bundle.ErrTamper,
		},
		{
			name: "file does not exist",
			make: func(t *testing.T, dir string) string {
				return filepath.Join(dir, "missing.atb")
			},
			assert: func(t *testing.T, b *bundle.Bundle, err error) {
				if err == nil {
					t.Fatalf("expected error for missing file")
				}
				if errors.Is(err, bundle.ErrNotABundle) {
					t.Fatalf("missing file should not match ErrNotABundle: %v", err)
				}
				if errors.Is(err, bundle.ErrTamper) {
					t.Fatalf("missing file should not match ErrTamper: %v", err)
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := tc.make(t, dir)
			b, err := bundle.LoadVerified(path)
			if tc.assert != nil {
				tc.assert(t, b, err)
				return
			}
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				if b == nil {
					t.Fatalf("expected non-nil bundle")
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected errors.Is(err, %v); got %v", tc.wantErr, err)
			}
		})
	}
}

func TestTypedErrors(t *testing.T) {
	t.Parallel()

	sentinels := map[string]error{
		"ErrTamper":       bundle.ErrTamper,
		"ErrMalformed":    bundle.ErrMalformed,
		"ErrNoManifest":   bundle.ErrNoManifest,
		"ErrNotABundle":   bundle.ErrNotABundle,
		"ErrBundleLocked": bundle.ErrBundleLocked,
	}

	for nameA, a := range sentinels {
		for nameB, b := range sentinels {
			if nameA == nameB {
				if !errors.Is(a, b) {
					t.Errorf("%s should match itself via errors.Is", nameA)
				}
				continue
			}
			if errors.Is(a, b) {
				t.Errorf("%s and %s must be distinct sentinels (errors.Is returned true)", nameA, nameB)
			}
		}
	}
}

func TestSaveConcurrent(t *testing.T) {
	requireAdvisoryLocking(t)
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent.atb")

	const goroutines = 10
	bundles := make([]*bundle.Bundle, goroutines)
	for i := 0; i < goroutines; i++ {
		b := newTestBundle(t)
		if err := b.AppendWithOptions("ai.tool.exec", map[string]any{"writer": i}, &bundle.AppendOptions{
			Timestamp: "2026-04-01T00:00:00Z",
		}); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
		bundles[i] = b
	}

	results := make([]error, goroutines)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i] = bundles[i].Save(context.Background(), path)
		}(i)
	}
	close(start)
	wg.Wait()

	successes := 0
	for _, err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, bundle.ErrBundleLocked):
			// Expected: contention surfaces ErrBundleLocked.
		default:
			t.Fatalf("unexpected Save error: %v", err)
		}
	}
	if successes == 0 {
		t.Fatal("expected at least one Save to succeed")
	}

	// Whatever winner landed last must be a complete, verifiable bundle —
	// not a torn write. With fsync+rename, no partial bytes can be observed.
	loaded, err := bundle.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("load final bundle: %v", err)
	}
	if err := loaded.Verify(); err != nil {
		t.Fatalf("verify final bundle: %v", err)
	}
	// The final bundle must be exactly one of the writers' bundles, not a
	// concatenation. Each test bundle has 2 records (manifest + 1 event).
	if len(loaded.Records) != 2 {
		t.Fatalf("expected 2 records in winning bundle, got %d (suggests a torn write)", len(loaded.Records))
	}
}

func TestSaveAcquiresLock(t *testing.T) {
	requireAdvisoryLocking(t)
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "locked.atb")

	// Pre-acquire the lock so any concurrent Save must surface ErrBundleLocked.
	release, err := bundle.AcquireWithRetry(context.TODO(), path, 0, 0)
	if err != nil {
		t.Fatalf("pre-acquire lock: %v", err)
	}

	b := newTestBundle(t)
	if err := b.AppendWithOptions("ai.tool.exec", map[string]any{"x": 1}, &bundle.AppendOptions{
		Timestamp: "2026-04-01T00:00:00Z",
	}); err != nil {
		t.Fatalf("append: %v", err)
	}

	// First Save must fail-fast with ErrBundleLocked because the lock is held.
	saveErr := b.Save(context.Background(), path)
	if !errors.Is(saveErr, bundle.ErrBundleLocked) {
		_ = release()
		t.Fatalf("Save while locked: err = %v, want ErrBundleLocked", saveErr)
	}

	// Release and try again — must succeed.
	if err := release(); err != nil {
		t.Fatalf("release lock: %v", err)
	}
	if err := b.Save(context.Background(), path); err != nil {
		t.Fatalf("Save after release: %v", err)
	}

	loaded, err := bundle.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := loaded.Verify(); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

// TestSaveLeavesNoTempFile confirms that a successful Save leaves the bundle
// directory clean: only the bundle path (and its lock sidecar, if any) remain.
// This regression-tests the writeAtomic temp-file cleanup path.
func TestSaveLeavesNoTempFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "clean.atb")

	b := newTestBundle(t)
	if err := b.AppendWithOptions("ai.tool.exec", map[string]any{"x": 1}, &bundle.AppendOptions{
		Timestamp: "2026-04-01T00:00:00Z",
	}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := b.Save(context.Background(), path); err != nil {
		t.Fatalf("save: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if strings.Contains(name, ".tmp") {
			t.Errorf("temp file left behind: %s", name)
		}
	}
}
