// SPDX-License-Identifier: MIT
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchiveFileHelpers(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.atb")
	content := []byte("archive-content")
	if err := os.WriteFile(source, content, 0o600); err != nil {
		t.Fatal(err)
	}
	hash, err := computeFileSHA256(source)
	if err != nil || len(hash) != 64 {
		t.Fatalf("hash=%q err=%v", hash, err)
	}
	if _, err := computeFileSHA256(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing file hash succeeded")
	}

	copied := filepath.Join(dir, "copied.atb")
	if err := copyFile(source, copied); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	got, err := os.ReadFile(copied)
	if err != nil || string(got) != string(content) {
		t.Fatalf("copied=%q err=%v", got, err)
	}
	if err := copyFile(filepath.Join(dir, "missing"), copied); err == nil {
		t.Fatal("missing source copied")
	}
	if err := copyFile(source, dir); err == nil {
		t.Fatal("directory destination accepted")
	}

	moved := filepath.Join(dir, "nested", "moved.atb")
	if err := moveFile(source, moved); err != nil {
		t.Fatalf("moveFile: %v", err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source remains after move: %v", err)
	}
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := moveFile(copied, filepath.Join(blocker, "moved.atb")); err == nil ||
		!strings.Contains(err.Error(), "mkdir archive destination") {
		t.Fatalf("blocked destination error=%v", err)
	}

	if !isPathWithin(dir, dir) || !isPathWithin(moved, dir) || isPathWithin(dir, moved) {
		t.Fatal("path containment checks failed")
	}
}
