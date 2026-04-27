// SPDX-License-Identifier: MIT

//go:build unix

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

func TestRunSignLockedBundleReturnsExitLockContention(t *testing.T) {
	bundlePath := writeVerifyTestBundle(t, buildSignSCBundle(t))
	keyPath := writeSignTestPrivateKey(t, t.TempDir())
	release := holdCLIBundleLock(t, bundlePath)
	defer release()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runSign([]string{"--bundle", bundlePath, "--key", keyPath}, &stdout, &stderr)
	if exitCode != exitLockContention {
		t.Fatalf("runSign() exit code = %d, want %d (stderr=%q)", exitCode, exitLockContention, stderr.String())
	}
}

func TestRunSignLockWaitReturnsExitLockContention(t *testing.T) {
	bundlePath := writeVerifyTestBundle(t, buildSignSCBundle(t))
	keyPath := writeSignTestPrivateKey(t, t.TempDir())
	release := holdCLIBundleLock(t, bundlePath)
	defer release()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runSign([]string{
		"--bundle",
		bundlePath,
		"--key",
		keyPath,
		"--lock-wait",
		"300ms",
	}, &stdout, &stderr)
	if exitCode != exitLockContention {
		t.Fatalf("runSign() exit code = %d, want %d (stderr=%q)", exitCode, exitLockContention, stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("locked after waiting")) {
		t.Fatalf("stderr missing lock-wait message: %q", stderr.String())
	}
}

func TestRunSnapshotLockedBundleReturnsExitLockContention(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}
	release := holdCLIBundleLock(t, path)
	defer release()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runSnapshot([]string{"locked", "--bundle", path}, &stdout, &stderr)
	if exitCode != exitLockContention {
		t.Fatalf("runSnapshot() exit code = %d, want %d (stderr=%q)", exitCode, exitLockContention, stderr.String())
	}
}

func holdCLIBundleLock(t testing.TB, path string) func() {
	t.Helper()

	lockFile := path + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockFile), 0o750); err != nil {
		t.Fatalf("create lock parent: %v", err)
	}
	f, err := os.OpenFile(lockFile, os.O_RDWR|os.O_CREATE, 0o600) // #nosec G304 -- test path is controlled
	if err != nil {
		t.Fatalf("open lock file: %v", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		t.Fatalf("flock lock file: %v", err)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}
}
