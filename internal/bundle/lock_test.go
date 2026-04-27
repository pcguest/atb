// SPDX-License-Identifier: MIT
package bundle

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/signer"
)

func TestLockPath_ContentionReturnsErrBundleLocked(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.atb")

	release1, err := lockPath(path)
	if err != nil {
		t.Fatalf("first lockPath: %v", err)
	}
	t.Cleanup(func() { _ = release1() })

	if _, err := lockPath(path); !errors.Is(err, ErrBundleLocked) {
		t.Fatalf("second lockPath: err = %v, want ErrBundleLocked", err)
	}

	if err := release1(); err != nil {
		t.Fatalf("release1: %v", err)
	}

	release2, err := lockPath(path)
	if err != nil {
		t.Fatalf("third lockPath after release: %v", err)
	}
	if err := release2(); err != nil {
		t.Fatalf("release2: %v", err)
	}
}

func TestLockPath_DifferentPathsDoNotConflict(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.atb")
	pathB := filepath.Join(dir, "b.atb")

	releaseA, err := lockPath(pathA)
	if err != nil {
		t.Fatalf("lock A: %v", err)
	}
	defer func() { _ = releaseA() }()

	releaseB, err := lockPath(pathB)
	if err != nil {
		t.Fatalf("lock B (different path) should not conflict: %v", err)
	}
	defer func() { _ = releaseB() }()
}

func TestSave_ContentionSurfacesErrBundleLocked(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.atb")

	b, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Hold the lock manually so any concurrent Save must surface ErrBundleLocked.
	release, err := lockPath(path)
	if err != nil {
		t.Fatalf("hold lock: %v", err)
	}
	defer func() { _ = release() }()

	saveErr := b.Save(path)
	if !errors.Is(saveErr, ErrBundleLocked) {
		t.Fatalf("Save while locked: err = %v, want ErrBundleLocked", saveErr)
	}
}

func TestSave_SerialisesConcurrentWritersWithoutLoss(t *testing.T) {
	// Two goroutines call Save on the same path. One gets the lock;
	// the other gets ErrBundleLocked or succeeds after the first releases.
	// Either way, no save returns nil while the other returns a non-lock error,
	// and the file on disk must be a valid bundle.
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.atb")

	const goroutines = 16
	var wg sync.WaitGroup
	results := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			b, err := New()
			if err != nil {
				results[idx] = err
				return
			}
			results[idx] = b.Save(path)
		}(i)
	}
	wg.Wait()

	successes := 0
	locked := 0
	for _, err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrBundleLocked):
			locked++
		default:
			t.Fatalf("unexpected Save error: %v", err)
		}
	}
	if successes == 0 {
		t.Fatalf("expected at least one Save to succeed; %d returned ErrBundleLocked", locked)
	}

	// Final file on disk must be loadable and verifiable.
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load after concurrent Saves: %v", err)
	}
	if err := loaded.Verify(); err != nil {
		t.Fatalf("Verify after concurrent Saves: %v", err)
	}
}

func TestSignToWithSigner_HonoursLock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.atb")

	// Create and save a bundle so SignToWithSigner has something to read.
	b, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := b.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	s := signer.NewLocalSigner(priv)

	// Hold the lock; sign must surface ErrBundleLocked.
	release, err := lockPath(path)
	if err != nil {
		t.Fatalf("hold lock: %v", err)
	}
	defer func() { _ = release() }()

	_, signErr := SignToWithSigner(context.Background(), path, path, s)
	if !errors.Is(signErr, ErrBundleLocked) {
		t.Fatalf("Sign while locked: err = %v, want ErrBundleLocked", signErr)
	}
	if !strings.Contains(signErr.Error(), "sign bundle") {
		t.Fatalf("Sign error should be wrapped with 'sign bundle' context: %v", signErr)
	}
}

func TestAcquireWithRetrySucceedsAfterContentionReleases(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.atb")

	release, err := lockPath(path)
	if err != nil {
		t.Fatalf("hold lock: %v", err)
	}

	result := make(chan error, 1)
	go func() {
		retryRelease, err := AcquireWithRetry(context.Background(), path, 500*time.Millisecond, 25*time.Millisecond)
		if err != nil {
			result <- err
			return
		}
		result <- retryRelease()
	}()

	time.Sleep(200 * time.Millisecond)
	if err := release(); err != nil {
		t.Fatalf("release: %v", err)
	}

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("AcquireWithRetry: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("AcquireWithRetry did not return")
	}
}
