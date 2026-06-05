// SPDX-License-Identifier: MIT
package bundle_test

import (
	"runtime"
	"testing"
)

func requireAdvisoryLocking(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		// Windows advisory locking is implemented (lock_windows.go, LockFileEx)
		// and validated directly by lock_windows_test.go. These cross-platform
		// scenarios are skipped on Windows because they exercise sidecar removal
		// under contention, where mandatory locks plus share-mode semantics make
		// os.Remove racy; the single-threaded Windows tests cover the contract.
		t.Skip("validated separately on Windows; see lock_windows_test.go")
	}
}
