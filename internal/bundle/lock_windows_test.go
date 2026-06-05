// SPDX-License-Identifier: MIT

//go:build windows

package bundle

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestLockPathWindowsExclusiveLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")

	release1, err := lockPath(path)
	if err != nil {
		t.Fatalf("first lockPath() error = %v", err)
	}
	t.Cleanup(func() { _ = release1() })

	if _, err := lockPath(path); !errors.Is(err, ErrBundleLocked) {
		t.Fatalf("second lockPath() err = %v, want ErrBundleLocked", err)
	}
}

func TestLockPathWindowsReleaseAllowsReacquire(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")

	release1, err := lockPath(path)
	if err != nil {
		t.Fatalf("first lockPath() error = %v", err)
	}
	if err := release1(); err != nil {
		t.Fatalf("release1() error = %v", err)
	}

	release2, err := lockPath(path)
	if err != nil {
		t.Fatalf("re-acquire lockPath() after release error = %v", err)
	}
	if err := release2(); err != nil {
		t.Fatalf("release2() error = %v", err)
	}
}

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
