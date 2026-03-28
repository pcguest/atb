package bundle_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/hash"
)

func TestAppendRejectsInvalidEventType(t *testing.T) {
	b := bundle.New()

	if err := b.AppendWithOptions("INVALID", nil, &bundle.AppendOptions{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}); err == nil {
		t.Fatalf("expected invalid event type to return an error")
	}
}

func TestAppendWithOptionsRoundTripsCanonicalFields(t *testing.T) {
	b := bundle.New()
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
	b := bundle.New()

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
