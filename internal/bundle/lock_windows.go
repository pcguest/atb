//go:build windows

// SPDX-License-Identifier: MIT

package bundle

import (
	"fmt"
)

// lockPath is a no-op placeholder on Windows. Advisory locking is not yet
// implemented on Windows in this task; callers still execute their critical
// section through withBundleLock, but no OS-level exclusion is provided here.
func lockPath(path string) (release func() error, err error) {
	if path == "" {
		return nil, fmt.Errorf("bundle: lock: empty path")
	}
	return func() error { return nil }, nil
}
