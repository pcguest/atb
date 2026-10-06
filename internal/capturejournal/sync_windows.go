//go:build windows

// SPDX-License-Identifier: MIT
package capturejournal

// syncDir is a no-op on Windows, which does not support directory fsync via
// os.File.Sync.
func syncDir(string) error {
	return nil
}
