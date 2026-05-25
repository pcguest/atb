// SPDX-License-Identifier: MIT
package evidencepack

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscoverWorkspaceBundles(t *testing.T) {
	root := t.TempDir()
	sessions := filepath.Join(root, "sessions")

	mkBundle := func(sessionID string) string {
		dir := filepath.Join(sessions, sessionID)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir session %s: %v", sessionID, err)
		}
		path := filepath.Join(dir, "bundle.atb")
		if err := os.WriteFile(path, []byte("bundle"), 0o644); err != nil {
			t.Fatalf("write bundle %s: %v", path, err)
		}
		return path
	}

	wantAlpha := mkBundle("alpha")
	wantBeta := mkBundle("beta")
	wantGamma := mkBundle("gamma")

	// Session without bundle.atb should be skipped.
	if err := os.MkdirAll(filepath.Join(sessions, "empty"), 0o755); err != nil {
		t.Fatalf("mkdir empty session: %v", err)
	}

	got, err := DiscoverWorkspaceBundles(root)
	if err != nil {
		t.Fatalf("DiscoverWorkspaceBundles() error = %v", err)
	}
	want := []string{wantAlpha, wantBeta, wantGamma}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DiscoverWorkspaceBundles() = %v, want %v", got, want)
	}
}

func TestDiscoverWorkspaceBundlesEmptyWorkspace(t *testing.T) {
	t.Run("no sessions directory", func(t *testing.T) {
		root := t.TempDir()
		got, err := DiscoverWorkspaceBundles(root)
		if err != nil {
			t.Fatalf("DiscoverWorkspaceBundles() error = %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("DiscoverWorkspaceBundles() = %v, want empty slice", got)
		}
	})

	t.Run("sessions directory with no bundles", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "sessions", "only-meta"), 0o755); err != nil {
			t.Fatalf("mkdir sessions: %v", err)
		}
		got, err := DiscoverWorkspaceBundles(root)
		if err != nil {
			t.Fatalf("DiscoverWorkspaceBundles() error = %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("DiscoverWorkspaceBundles() = %v, want empty slice", got)
		}
	})
}
