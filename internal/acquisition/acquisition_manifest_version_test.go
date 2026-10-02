// SPDX-License-Identifier: MIT
package acquisition

import (
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

// TestImportCreatesV3Bundle pins the acquisition canonical-profile declaration:
// a fresh chatlog import must produce a bundle that declares manifest version 3,
// carries acquisition provenance on its events, and verifies under the current
// build.
func TestImportCreatesV3Bundle(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "import.atb")

	importPass(t, bundlePath, fixturePath(t, "pass1.jsonl"), false)

	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	m := b.Manifest()
	if m == nil || m.Version != "3" {
		t.Fatalf("manifest = %+v, want version 3", m)
	}
	hasAcq := false
	for _, r := range b.Records {
		if r.Event.Acquisition != nil {
			hasAcq = true
			break
		}
	}
	if !hasAcq {
		t.Fatal("expected acquisition-bearing events in imported bundle")
	}
	if _, err := bundle.LoadVerified(bundlePath); err != nil {
		t.Fatalf("LoadVerified: %v", err)
	}
}

// TestReconcileOntoV3BundleAllowed pins that reconciling a further import onto
// an already-v3 bundle is permitted and keeps the bundle valid.
func TestReconcileOntoV3BundleAllowed(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "import.atb")

	importPass(t, bundlePath, fixturePath(t, "pass1.jsonl"), false)
	res := importPass(t, bundlePath, fixturePath(t, "pass2.jsonl"), true)
	if res.NewCount == 0 {
		t.Fatalf("expected new reconciled records, got %+v", res)
	}
	if _, err := bundle.LoadVerified(bundlePath); err != nil {
		t.Fatalf("LoadVerified after v3 reconcile: %v", err)
	}
	if m := mustManifest(t, bundlePath); m.Version != "3" {
		t.Fatalf("manifest version after reconcile = %q, want 3", m.Version)
	}
}

func mustManifest(t *testing.T, path string) *bundle.ManifestData {
	t.Helper()
	b, err := bundle.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	m := b.Manifest()
	if m == nil {
		t.Fatal("manifest missing")
	}
	return m
}
