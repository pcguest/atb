//go:build unix

// SPDX-License-Identifier: MIT

package bundle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// lockPath acquires an exclusive, non-blocking advisory lock on a sidecar
// file at path + lockSuffix. The returned release function releases the
// lock and closes the underlying descriptor; it MUST be invoked (typically
// via defer) on the success path.
//
// Contract:
//   - path is the bundle file path, not a directory.
//   - The lock file is created in the bundle's parent directory if missing
//     (mode 0o600). If this process created the sidecar, release removes it
//     after unlocking; pre-existing lock files are left in place.
//   - On contention, lockPath returns ErrBundleLocked wrapped with context.
//     It does NOT block.
//
// The lock is process-level (flock(2)). Multiple goroutines in the same
// process do not block each other through the kernel: flock on a duplicate
// fd does not block, but each call here opens a fresh fd, so they DO
// serialise. The contention test in lock_test.go relies on that.
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

	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return nil, fmt.Errorf("%w: %s", ErrBundleLocked, lockFile)
		}
		return nil, fmt.Errorf("bundle: lock: flock: %w", err)
	}

	released := false
	release = func() error {
		if released {
			return nil
		}
		released = true
		flockErr := unix.Flock(int(f.Fd()), unix.LOCK_UN)
		closeErr := f.Close()
		var removeErr error
		if created {
			removeErr = os.Remove(lockFile)
			if errors.Is(removeErr, os.ErrNotExist) {
				removeErr = nil
			}
		}
		if flockErr != nil {
			return fmt.Errorf("bundle: unlock: flock: %w", flockErr)
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

func openLockFile(lockFile string) (*os.File, bool, error) {
	for {
		f, err := os.OpenFile(lockFile, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600) // #nosec G304 -- derived from caller path
		if err == nil {
			return f, true, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, false, err
		}

		f, err = os.OpenFile(lockFile, os.O_RDWR, 0o600) // #nosec G304 -- derived from caller path
		if err == nil {
			return f, false, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, false, err
		}
	}
}
