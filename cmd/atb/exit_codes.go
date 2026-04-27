// SPDX-License-Identifier: MIT
package main

import (
	"errors"
	"os"
	"strings"
)

const (
	exitSuccess          = 0
	exitUserError        = 1
	exitIntegrityFailure = 2
	exitVerifyFailure    = 3
	exitSystemError      = 3
	// exitLockContention is returned when the bundle file is locked by another
	// process. Downstream automation should retry after a short delay.
	exitLockContention = 9
)

func classifyBundleLoadError(err error) int {
	switch {
	case isBundleLocked(err):
		return exitLockContention
	case errors.Is(err, os.ErrNotExist):
		return exitUserError
	case errors.Is(err, os.ErrPermission):
		return exitSystemError
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unmarshal") || strings.Contains(msg, "scan") {
		return exitIntegrityFailure
	}
	return exitSystemError
}
