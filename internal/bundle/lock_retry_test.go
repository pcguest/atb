// SPDX-License-Identifier: MIT

package bundle

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestAcquireWithRetrySucceedsAfterContention(t *testing.T) {
	original := lockPathForRetry
	t.Cleanup(func() {
		lockPathForRetry = original
	})

	attempts := 0
	lockPathForRetry = func(path string) (func() error, error) {
		attempts++
		if attempts <= 3 {
			return nil, fmt.Errorf("%w: %s", ErrBundleLocked, path+".lock")
		}
		return func() error { return nil }, nil
	}

	release, err := AcquireWithRetry(
		context.Background(),
		"bundle.atb",
		time.Second,
		time.Millisecond,
	)
	if err != nil {
		t.Fatalf("AcquireWithRetry() error = %v", err)
	}
	if release == nil {
		t.Fatal("AcquireWithRetry() release func is nil")
	}
	if attempts != 4 {
		t.Fatalf("attempts = %d, want 4", attempts)
	}
}

func TestAcquireWithRetryTimeoutReturnsBundleLocked(t *testing.T) {
	original := lockPathForRetry
	t.Cleanup(func() {
		lockPathForRetry = original
	})

	attempts := 0
	lockPathForRetry = func(path string) (func() error, error) {
		attempts++
		return nil, fmt.Errorf("%w: %s", ErrBundleLocked, path+".lock")
	}

	_, err := AcquireWithRetry(
		context.Background(),
		"bundle.atb",
		10*time.Millisecond,
		time.Millisecond,
	)
	if !errors.Is(err, ErrBundleLocked) {
		t.Fatalf("AcquireWithRetry() error = %v, want ErrBundleLocked", err)
	}
	if attempts < 2 {
		t.Fatalf("attempts = %d, want at least 2", attempts)
	}
}
