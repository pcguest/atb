//go:build !windows

// SPDX-License-Identifier: MIT
package capturejournal

import "os"

// syncDir fsyncs a directory so that a newly created journal file entry is
// durable. Journal state is operational, not evidence.
func syncDir(path string) error {
	dirFile, err := os.Open(path) // #nosec G304 -- dir derived from caller-selected local journal path
	if err != nil {
		return err
	}
	if err := dirFile.Sync(); err != nil {
		_ = dirFile.Close()
		return err
	}
	return dirFile.Close()
}
