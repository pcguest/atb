//go:build !windows

// SPDX-License-Identifier: MIT
package acquisition

import "os"

// syncDir fsyncs a directory so that a rename into it is durable. Used by
// Checkpoint.Save; checkpoint state is operational, not evidence.
func syncDir(path string) error {
	dirFile, err := os.Open(path) // #nosec G304 -- dir derived from caller-selected local checkpoint path
	if err != nil {
		return err
	}
	if err := dirFile.Sync(); err != nil {
		_ = dirFile.Close()
		return err
	}
	return dirFile.Close()
}
