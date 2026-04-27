// SPDX-License-Identifier: MIT
package bundle

import (
	"context"
	"errors"
	"time"
)

// ErrBundleLocked is returned when an exclusive advisory lock on the bundle
// cannot be acquired because another process holds it.
var ErrBundleLocked = errors.New("bundle: locked by another process")

// lockSuffix is appended to the bundle path to form the lock file path.
const lockSuffix = ".lock"

// DefaultLockRetryInterval is the sleep between lock acquisition attempts
// when callers opt into bounded lock waiting.
const DefaultLockRetryInterval = 200 * time.Millisecond

var lockPathForRetry = lockPath

// withBundleLock acquires an exclusive advisory lock on path+".lock", calls fn,
// then releases the lock. It creates the lock file if absent (mode 0600).
// Returns ErrBundleLocked if the lock is held (non-blocking attempt).
func withBundleLock(path string, fn func() error) error {
	release, err := lockPath(path)
	if err != nil {
		return err
	}

	fnErr := fn()
	releaseErr := release()
	if fnErr != nil {
		return fnErr
	}
	return releaseErr
}

// AcquireWithRetry calls lockPath repeatedly until it succeeds, the context
// is cancelled, or timeout elapses. interval is the sleep between retries.
// If interval is zero or negative, DefaultLockRetryInterval is used. A zero
// timeout preserves the default fail-fast lock behaviour.
func AcquireWithRetry(ctx context.Context, path string, timeout, interval time.Duration) (func() error, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		return lockPathForRetry(path)
	}
	if interval <= 0 {
		interval = DefaultLockRetryInterval
	}

	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		release, err := lockPathForRetry(path)
		if err == nil {
			return release, nil
		}
		if !errors.Is(err, ErrBundleLocked) {
			return nil, err
		}
		lastErr = err

		sleepFor := time.Until(deadline)
		if sleepFor <= 0 {
			return nil, lastErr
		}
		if sleepFor > interval {
			sleepFor = interval
		}

		timer := time.NewTimer(sleepFor)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
