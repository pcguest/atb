// SPDX-License-Identifier: MIT
package main

import (
	"fmt"
	"io"
	"runtime"
	"sync"
)

var warnWindowsLockOnce sync.Once

func maybeWarnWindowsBundleLock(stderr io.Writer) {
	if runtime.GOOS != "windows" || stderr == nil {
		return
	}
	warnWindowsLockOnce.Do(func() {
		fmt.Fprintln(stderr, "atb: warning: advisory bundle locking is not enforced on Windows; avoid concurrent writers against the same bundle")
	})
}
