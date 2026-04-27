// SPDX-License-Identifier: MIT

//go:build integration

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

var lockContentionBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "atb-lock-contention-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	lockContentionBinary = filepath.Join(dir, "atb")
	build := exec.Command("go", "build", "-o", lockContentionBinary, "./cmd/atb")
	build.Dir = "../.."
	if out, err := build.CombinedOutput(); err != nil {
		panic("build atb test binary: " + err.Error() + "\n" + string(out))
	}

	os.Exit(m.Run())
}

func TestLockContentionTwoSnapshotProcesses(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "bundle.atb")
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	payload := bytes.Repeat([]byte("x"), 8192)
	for i := 0; i < 1200; i++ {
		if err := b.Append("ai.tool.exec", map[string]any{
			"index":   i,
			"payload": string(payload),
		}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("Save: %v", err)
	}

	start := make(chan struct{})
	results := make(chan int, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			cmd := exec.Command(
				lockContentionBinary,
				"snapshot",
				"--bundle",
				bundlePath,
				"--lock-wait",
				"0",
				"concurrent_snap",
			)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			err := cmd.Run()
			if err == nil {
				results <- exitSuccess
				return
			}
			exitErr, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("snapshot failed without exit status: %v stderr=%q", err, stderr.String())
			}
			results <- exitErr.ExitCode()
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	counts := map[int]int{}
	for code := range results {
		counts[code]++
	}
	if counts[exitSuccess] != 1 || counts[exitLockContention] != 1 {
		t.Fatalf("exit code counts = %+v, want one success and one lock contention", counts)
	}

	verify := exec.Command(lockContentionBinary, "verify", "--bundle", bundlePath)
	if out, err := verify.CombinedOutput(); err != nil {
		t.Fatalf("verify after contention: %v\n%s", err, string(out))
	}
}
