//go:build windows

// SPDX-License-Identifier: MIT
package bundle

func syncDir(string) error {
	// Windows does not support directory fsync via os.File.Sync.
	return nil
}
