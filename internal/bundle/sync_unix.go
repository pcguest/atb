//go:build !windows

// SPDX-License-Identifier: MIT
package bundle

import "os"

func syncDir(path string) error {
	dirFile, err := os.Open(path) // #nosec G304 -- dir derived from caller path
	if err != nil {
		return err
	}
	if err := dirFile.Sync(); err != nil {
		_ = dirFile.Close()
		return err
	}
	return dirFile.Close()
}
