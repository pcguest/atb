// SPDX-License-Identifier: MIT
package bundle

import (
	"fmt"
	"os"
	"path/filepath"
)

// bundleFileMode is the standard file permission applied to bundle files
// produced by Save and Sign.
const bundleFileMode os.FileMode = 0o600

// writeAtomic writes data to path using a temp-file + fsync + rename strategy
// so that a crash mid-write cannot leave a truncated or partially-written file.
//
// It writes the temp file in filepath.Dir(path), fsyncs the temp file before
// rename, fsyncs the parent directory after rename, and removes the temp file
// on any error path. perm is the permission applied to the temp file before
// rename. If perm is zero, bundleFileMode is used.
func writeAtomic(path string, data []byte, perm os.FileMode) error {
	if perm == 0 {
		perm = bundleFileMode
	}

	cleanPath := filepath.Clean(path)
	dir := filepath.Dir(cleanPath)
	tmp, err := os.CreateTemp(dir, filepath.Base(cleanPath)+".*.atb.tmp") // #nosec G304 -- dir derived from caller path
	if err != nil {
		return fmt.Errorf("bundle: write atomic: create temp: %w", err)
	}
	tmpPath := tmp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(perm.Perm()); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("bundle: write atomic: chmod temp: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("bundle: write atomic: write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("bundle: write atomic: fsync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("bundle: write atomic: close temp: %w", err)
	}
	if err := os.Rename(tmpPath, cleanPath); err != nil {
		return fmt.Errorf("bundle: write atomic: rename: %w", err)
	}
	removeTemp = false

	if err := syncDir(dir); err != nil {
		return fmt.Errorf("bundle: write atomic: fsync parent: %w", err)
	}
	return nil
}
