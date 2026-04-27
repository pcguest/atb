// SPDX-License-Identifier: MIT
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

const lockWaitEnv = "ATB_LOCK_WAIT"

func parseLockWaitDuration(value string) (time.Duration, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	wait, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, err
	}
	if wait < 0 {
		return 0, fmt.Errorf("must be non-negative")
	}
	return wait, nil
}

func lockWaitFromEnv() (time.Duration, error) {
	value, ok := os.LookupEnv(lockWaitEnv)
	if !ok {
		return 0, nil
	}
	wait, err := parseLockWaitDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", lockWaitEnv, err)
	}
	return wait, nil
}

type lockWaitTimeoutError struct {
	waited time.Duration
}

func (e lockWaitTimeoutError) Error() string {
	return fmt.Sprintf(
		"bundle locked after waiting %s; try again or check for a stale .lock file",
		e.waited,
	)
}

func (e lockWaitTimeoutError) Unwrap() error {
	return bundle.ErrBundleLocked
}

func lockWaitError(waited time.Duration) error {
	return lockWaitTimeoutError{waited: waited}
}
