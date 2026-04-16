package bundle_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/hash"
)

func newTestBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b, err := bundle.New()
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	return b
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
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	loaded, err := bundle.Load(path)
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

	loaded, err := bundle.Load(path)
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
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	loaded, err := bundle.Load(path)
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
		if err := b.Save(path); err != nil {
			t.Fatalf("save: %v", err)
		}
		loaded, err := bundle.Load(path)
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
		if err := orig.Save(path); err != nil {
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
		if err := newB.Save(path); err == nil {
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

func TestVerifyDetectsHashTampering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")
	b := newTestBundle(t)

	if err := b.Append("ai.tool.exec", map[string]any{"step": 1}); err != nil {
		t.Fatalf("append event: %v", err)
	}
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	loaded, err := bundle.Load(path)
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
