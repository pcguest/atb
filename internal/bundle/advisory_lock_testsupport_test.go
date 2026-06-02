// SPDX-License-Identifier: MIT
package bundle_test

import (
	"runtime"
	"testing"
)

func requireAdvisoryLocking(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("advisory bundle locking is a no-op on Windows; see internal/bundle/lock_windows.go")
	}
}
