package bundle_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
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
	if len(loaded.Records) != 1 {
		t.Fatalf("expected 1 loaded record, got %d", len(loaded.Records))
	}

	got := loaded.Records[0].Event
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
