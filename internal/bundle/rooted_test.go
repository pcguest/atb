// SPDX-License-Identifier: MIT
package bundle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRootedBundleRoundTrip(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	b, err := NewWithOptions(NewOptions{ManifestVersion: ManifestVersionV2})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Append("test.event", map[string]any{"ok": true}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("nested", "bundle.atb")
	if err := b.SaveRooted(context.Background(), root, path); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadRooted(root, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Records) != len(b.Records) {
		t.Fatalf("record count = %d, want %d", len(loaded.Records), len(b.Records))
	}
	if info, err := os.Stat(filepath.Join(dir, path)); err != nil {
		t.Fatal(err)
	} else if info.Mode().Perm() != bundleFileMode {
		t.Fatalf("mode = %o, want %o", info.Mode().Perm(), bundleFileMode)
	}
}

func TestRootedBundleRejectsCancelledSaveAndMalformedLoad(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	b, err := New()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := b.SaveRooted(ctx, root, "cancelled.atb"); !errors.Is(err, context.Canceled) {
		t.Fatalf("SaveRooted error = %v, want context.Canceled", err)
	}
	if err := root.WriteFile("empty.atb", nil, bundleFileMode); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRooted(root, "empty.atb"); !errors.Is(err, ErrNoManifest) {
		t.Fatalf("LoadRooted error = %v, want ErrNoManifest", err)
	}
}

func TestWriteAtomicRootedDoesNotFollowFinalSymlink(t *testing.T) {
	dir, outside := t.TempDir(), t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	outsidePath := filepath.Join(outside, "value")
	if err := os.WriteFile(outsidePath, []byte("outside sentinel"), bundleFileMode); err != nil {
		t.Fatal(err)
	}
	insidePath := filepath.Join(dir, "value")
	if err := os.Symlink(outsidePath, insidePath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := WriteAtomicRooted(root, "value", []byte("inside replacement")); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(outsidePath); err != nil || string(got) != "outside sentinel" {
		t.Fatalf("outside file = %q, %v", got, err)
	}
	if got, err := os.ReadFile(insidePath); err != nil || string(got) != "inside replacement" {
		t.Fatalf("inside file = %q, %v", got, err)
	}
}

func TestRootedBundleLockContention(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	release, err := lockPathRooted(root, "bundle.atb")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, err := lockPathRooted(root, "bundle.atb"); !errors.Is(err, ErrBundleLocked) {
		t.Fatalf("second lock error = %v, want ErrBundleLocked", err)
	}
}

func TestRootedRelativePath(t *testing.T) {
	dir := t.TempDir()
	want := filepath.Join("nested", "bundle.atb")
	for _, path := range []string{want, filepath.Join(dir, want)} {
		got, err := RootedRelativePath(dir, path)
		if err != nil || got != want {
			t.Fatalf("RootedRelativePath(%q) = %q, %v; want %q", path, got, err, want)
		}
	}
	for _, path := range []string{".", "../outside.atb", filepath.Join(filepath.Dir(dir), "outside.atb")} {
		if got, err := RootedRelativePath(dir, path); err == nil {
			t.Fatalf("RootedRelativePath(%q) = %q, want error", path, got)
		}
	}
}
