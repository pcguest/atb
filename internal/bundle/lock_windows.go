//go:build windows

// SPDX-License-Identifier: MIT

package bundle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// lockPath acquires an exclusive, non-blocking byte-range lock on a sidecar
// file at path + lockSuffix using LockFileEx. The returned release function
// unlocks and closes the handle (and removes the sidecar if this call created
// it); it MUST be invoked on the success path. The semantics mirror the Unix
// flock implementation in lock_unix.go.
//
// Contract:
//   - path is the bundle file path, not a directory.
//   - The lock file is created in the bundle's parent directory if missing
//     (mode 0o600). If this process created the sidecar, release removes it
//     after unlocking; pre-existing lock files are left in place.
//   - On contention, lockPath returns ErrBundleLocked wrapped with context.
//     It does NOT block (LOCKFILE_FAIL_IMMEDIATELY).
//
// Windows byte-range locks are mandatory and enforced per handle, so a second
// lockPath on the same file — even within this process — fails with
// ErrBundleLocked. The contention tests rely on that.
func lockPath(path string) (release func() error, err error) {
	if path == "" {
		return nil, fmt.Errorf("bundle: lock: empty path")
	}

	lockFile := path + lockSuffix
	dir := filepath.Dir(lockFile)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil { // #nosec G301 -- bundle parent dir
			return nil, fmt.Errorf("bundle: lock: mkdir parent: %w", err)
		}
	}

	f, created, err := openLockFile(lockFile)
	if err != nil {
		return nil, fmt.Errorf("bundle: lock: open lock file: %w", err)
	}

	if err := lockFileEx(f); err != nil {
		_ = f.Close()
		if errors.Is(err, ErrBundleLocked) {
			return nil, fmt.Errorf("%w: %s", ErrBundleLocked, lockFile)
		}
		return nil, err
	}

	released := false
	release = func() error {
		if released {
			return nil
		}
		released = true
		unlockErr := unlockFileEx(f)
		closeErr := f.Close()
		var removeErr error
		if created {
			removeErr = os.Remove(lockFile)
			if errors.Is(removeErr, os.ErrNotExist) {
				removeErr = nil
			}
		}
		if unlockErr != nil {
			return unlockErr
		}
		if closeErr != nil {
			return fmt.Errorf("bundle: unlock: close: %w", closeErr)
		}
		if removeErr != nil {
			return fmt.Errorf("bundle: unlock: remove lock file: %w", removeErr)
		}
		return nil
	}
	return release, nil
}

func lockFileEx(f *os.File) error {
	var ol windows.Overlapped
	flags := uint32(windows.LOCKFILE_EXCLUSIVE_LOCK | windows.LOCKFILE_FAIL_IMMEDIATELY)
	err := windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, &ol)
	if err == nil {
		return nil
	}
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return ErrBundleLocked
	}
	return fmt.Errorf("bundle: lock: LockFileEx: %w", err)
}

func unlockFileEx(f *os.File) error {
	var ol windows.Overlapped
	if err := windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &ol); err != nil {
		return fmt.Errorf("bundle: unlock: UnlockFileEx: %w", err)
	}
	return nil
}
