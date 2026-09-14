// SPDX-License-Identifier: MIT
package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBundleManagerPathBoundary(t *testing.T) {
	for _, name := range []string{"nested", "absolute inside", "absolute outside", "parent escape", "sibling prefix", "intermediate symlink", "final symlink"} {
		t.Run(name, func(t *testing.T) {
			parent := t.TempDir()
			root, outside := filepath.Join(parent, "data"), filepath.Join(parent, "data-other")
			if err := os.MkdirAll(root, 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(outside, 0o750); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(outside, "bundle.atb")
			if err := os.WriteFile(target, []byte("outside sentinel"), 0o600); err != nil {
				t.Fatal(err)
			}
			path, valid := "missing/nested/bundle.atb", false
			switch name {
			case "nested":
				valid = true
			case "absolute inside":
				path, valid = filepath.Join(root, path), true
			case "absolute outside", "sibling prefix":
				path = target
			case "parent escape":
				path = "../data-other/bundle.atb"
			case "intermediate symlink":
				if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
				path = "link/bundle.atb"
			case "final symlink":
				if err := os.Symlink(target, filepath.Join(root, "link.atb")); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
				path = "link.atb"
			}
			manager := NewBundleFileManager(root)
			defer manager.Shutdown(context.Background())
			id, err := manager.OpenSession(context.Background(), OpenParams{BundlePath: path})
			if valid {
				if err != nil {
					t.Fatal(err)
				}
				if err := manager.AppendEvent(context.Background(), id, PendingEvent{EventType: "test.event"}); err != nil {
					t.Fatal(err)
				}
				if _, err := manager.CloseSession(context.Background(), id, CloseSessionOpts{}); err != nil {
					t.Fatal(err)
				}
			} else if err == nil {
				t.Fatal("escaping path accepted")
			}
			assertOutsideUntouched(t, outside)
		})
	}
}

func assertOutsideUntouched(t *testing.T, outside string) {
	t.Helper()
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "bundle.atb" {
		t.Fatalf("outside directory modified: %v", entries)
	}
	raw, err := os.ReadFile(filepath.Join(outside, "bundle.atb"))
	if err != nil || string(raw) != "outside sentinel" {
		t.Fatalf("outside file modified: %q %v", raw, err)
	}
}

func TestBundleManagerSymlinkSwapAtUse(t *testing.T) {
	for _, operation := range []string{"create", "append", "close"} {
		t.Run(operation, func(t *testing.T) {
			root, outside := t.TempDir(), t.TempDir()
			if err := os.WriteFile(filepath.Join(outside, "bundle.atb"), []byte("outside sentinel"), 0o600); err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, "nested")
			if err := os.Mkdir(dir, 0o750); err != nil {
				t.Fatal(err)
			}
			requested, err := resolveAgentBundlePath(root, "nested/bundle.atb")
			if err != nil {
				t.Fatal(err)
			}
			manager := NewBundleFileManager(root)
			defer manager.Shutdown(context.Background())
			var id SessionID
			if operation != "create" {
				id, err = manager.OpenSession(context.Background(), OpenParams{BundlePath: requested})
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Rename(dir, filepath.Join(root, "original")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, dir); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			switch operation {
			case "create":
				_, err = manager.OpenSession(context.Background(), OpenParams{BundlePath: requested})
			case "append":
				err = manager.AppendEvent(context.Background(), id, PendingEvent{EventType: "test.event"})
			case "close":
				_, err = manager.CloseSession(context.Background(), id, CloseSessionOpts{})
			}
			if err == nil {
				t.Fatal("operation through replaced ancestor succeeded")
			}
			assertOutsideUntouched(t, outside)
		})
	}
}

func TestBundleManagerFinalSymlinkSwap(t *testing.T) {
	for _, operation := range []string{"append", "close", "metadata"} {
		t.Run(operation, func(t *testing.T) {
			root, outside := t.TempDir(), t.TempDir()
			if err := os.WriteFile(filepath.Join(outside, "bundle.atb"), []byte("outside sentinel"), 0o600); err != nil {
				t.Fatal(err)
			}
			manager := NewBundleFileManager(root)
			defer manager.Shutdown(context.Background())
			id, err := manager.OpenSession(context.Background(), OpenParams{})
			if err != nil {
				t.Fatal(err)
			}
			target := sessionBundlePath(root, id)
			if operation == "metadata" {
				target = sessionMetaPath(root, id)
			} else {
				if err := os.Rename(target, target+".original"); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(filepath.Join(outside, "bundle.atb"), target); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			if operation == "append" {
				err = manager.AppendEvent(context.Background(), id, PendingEvent{EventType: "test.event"})
			} else {
				_, err = manager.CloseSession(context.Background(), id, CloseSessionOpts{})
			}
			// Atomic replacement may replace the symlink itself; following it must
			// never alter the outside target.
			assertOutsideUntouched(t, outside)
			if err != nil {
				t.Logf("operation rejected: %v", err)
			}
		})
	}
}

func TestWorkspaceIndexRejectsEscapingMetadata(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "sessions"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "meta.json"), []byte(`{"session_id":"outside"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "sessions", "escape")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	got, _ := NewWorkspaceIndex(root).ListBundles(context.Background())
	if len(got) != 0 {
		t.Fatalf("outside metadata exposed: %+v", got)
	}
}
