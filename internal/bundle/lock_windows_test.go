// SPDX-License-Identifier: MIT

//go:build windows

package bundle

import (
	"path/filepath"
	"testing"
)

func TestLockPathWindowsReturnsReleaseFunc(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")

	release, err := lockPath(path)
	if err != nil {
		t.Fatalf("lockPath() error = %v", err)
	}
	if release == nil {
		t.Fatal("lockPath() release func is nil")
	}
	if err := release(); err != nil {
		t.Fatalf("release() error = %v", err)
	}
}
