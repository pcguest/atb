//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

var integrationBaseTime = time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

func newTempBundle(t *testing.T) string {
	t.Helper()

	root, err := os.MkdirTemp("", "atb-integration-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(root)
	})

	bundlePath := filepath.Join(root, bundle.BundleDir, bundle.BundleFile)
	b := bundle.New()
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}
	return bundlePath
}

func appendEvent(t *testing.T, bundlePath string, eventType string, fields map[string]any) {
	t.Helper()

	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load bundle %q: %v", bundlePath, err)
	}

	offset := len(b.Records) - 1
	if offset < 0 {
		offset = 0
	}
	timestamp := integrationBaseTime.Add(time.Duration(offset) * time.Minute).Format(time.RFC3339)
	if err := b.AppendWithOptions(eventType, fields, &bundle.AppendOptions{Timestamp: timestamp}); err != nil {
		t.Fatalf("append %s: %v", eventType, err)
	}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle %q: %v", bundlePath, err)
	}
}
