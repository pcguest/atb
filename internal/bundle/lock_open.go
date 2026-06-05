// SPDX-License-Identifier: MIT

package bundle

import (
	"errors"
	"os"
)

// openLockFile creates or opens the bundle sidecar lock file. When created is
// true, the caller created the sidecar and should remove it after unlocking;
// pre-existing lock files (created false) are left in place. The helper is
// platform-independent so both the Unix (flock) and Windows (LockFileEx) lock
// implementations share identical create/open semantics.
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
