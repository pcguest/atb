//go:build integration

// Run with: go test -tags integration -count=1 ./cmd/atb/... -run TestE2EPipeline
//
// End-to-end regression harness for the Phase 1–8 happy path. Drives the
// compiled binary via real subprocesses (no internal package calls) so that
// any breakage in the wiring between commands surfaces here.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func runATB(t *testing.T, bin string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if err == nil {
		return stdout, stderr, 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return stdout, stderr, ee.ExitCode()
	}
	t.Logf("atb %v failed to start: %v\nstdout=%s\nstderr=%s", args, err, stdout, stderr)
	return stdout, stderr, -1
}

func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	out := filepath.Join(dir, "atb")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	repoRoot := repoRootFromTest(t)
	build := exec.Command("go", "build", "-o", out, "./cmd/atb")
	build.Dir = repoRoot
	var buildErr bytes.Buffer
	build.Stderr = &buildErr
	if err := build.Run(); err != nil {
		t.Fatalf("go build ./cmd/atb failed: %v\nstderr=%s", err, buildErr.String())
	}
	return out
}

func repoRootFromTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// Tests run inside cmd/atb; the module root is two levels up.
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func writeChatlogFixture(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "chatlog.jsonl")
	body := strings.Join([]string{
		`{"role":"user","content":"hello","timestamp":"2026-04-26T12:00:00Z"}`,
		`{"role":"assistant","content":"hi there","timestamp":"2026-04-26T12:00:01Z","model":"gpt-4o"}`,
		`{"role":"user","content":"what's the weather?","timestamp":"2026-04-26T12:00:02Z"}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write chatlog fixture: %v", err)
	}
	return path
}

func TestE2EPipeline(t *testing.T) {
	bin := buildBinary(t)

	work := t.TempDir()
	bundlePath := filepath.Join(work, "run.atb", "bundle.atb")

	// Generate an Ed25519 signing key via the CLI.
	if _, _, code := runATB(t, bin, "keygen", "--out-dir", work); code != 0 {
		t.Fatalf("keygen failed (exit %d)", code)
	}
	keyPath := filepath.Join(work, "atb-key.pem")
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("keygen did not produce %s: %v", keyPath, err)
	}

	// STEP 1 — capture run wraps a trivial child command and creates the bundle.
	{
		stdout, stderr, code := runATB(t, bin,
			"capture", "run",
			"--bundle", bundlePath,
			"--snapshot", "pre-import",
			"--", "echo", "hello",
		)
		if code != 0 {
			t.Fatalf("capture run exit=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
		}
		if _, err := os.Stat(bundlePath); err != nil {
			t.Fatalf("bundle not created at %s: %v", bundlePath, err)
		}
		// Sanity: capture should announce that it created a fresh bundle.
		if !strings.Contains(stderr, "created new bundle") {
			t.Logf("note: capture stderr did not mention 'created new bundle': %q", stderr)
		}
	}

	// STEP 2 — import a chatlog fixture and request a JSON status report.
	fixture := writeChatlogFixture(t, work)
	{
		stdout, stderr, code := runATB(t, bin,
			"import", "chatlog",
			"--from", "generic-jsonl",
			"--input", fixture,
			"--bundle", bundlePath,
			"--snapshot", "post-import",
			"--format", "json",
		)
		if code != 0 {
			t.Fatalf("import chatlog exit=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
		}
		var got struct {
			EventsWritten    int  `json:"events_written"`
			SnapshotAppended bool `json:"snapshot_appended"`
		}
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("decode import json: %v\nstdout=%s", err, stdout)
		}
		if got.EventsWritten < 3 {
			t.Fatalf("events_written = %d, want >= 3", got.EventsWritten)
		}
		if !got.SnapshotAppended {
			t.Fatalf("snapshot_appended = false, want true")
		}
	}

	// STEP 3 — append a manual snapshot and inspect the JSON envelope.
	{
		stdout, stderr, code := runATB(t, bin,
			"snapshot", "manual-checkpoint",
			"--bundle", bundlePath,
			"--format", "json",
		)
		if code != 0 {
			t.Fatalf("snapshot exit=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
		}
		// The CLI returns a generic mutationResult envelope rather than the
		// raw snapshotEventData; assert on the fields that are stable across
		// future event-payload changes.
		var got struct {
			Status    string `json:"status"`
			Action    string `json:"action"`
			EventType string `json:"event_type"`
			Hash      string `json:"hash"`
		}
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("decode snapshot json: %v\nstdout=%s", err, stdout)
		}
		if got.Status != "ok" {
			t.Fatalf("snapshot status = %q, want ok", got.Status)
		}
		if got.Action != "snapshot" {
			t.Fatalf("snapshot action = %q, want snapshot", got.Action)
		}
		if got.EventType != "atb.snapshot" {
			t.Fatalf("snapshot event_type = %q, want atb.snapshot", got.EventType)
		}
		if len(got.Hash) == 0 {
			t.Fatalf("snapshot hash is empty")
		}
	}

	// STEP 4 — verify before signing.
	{
		stdout, stderr, code := runATB(t, bin, "verify", "--bundle", bundlePath)
		if code != 0 {
			t.Fatalf("pre-sign verify exit=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
		}
	}

	preSignSize := fileSize(t, bundlePath)

	// STEP 5 — sign the bundle with the local Ed25519 key.
	{
		stdout, stderr, code := runATB(t, bin,
			"sign",
			"--bundle", bundlePath,
			"--key", keyPath,
		)
		if code != 0 {
			t.Fatalf("sign exit=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
		}
		if got := fileSize(t, bundlePath); got <= preSignSize {
			t.Fatalf("bundle size did not grow after sign: pre=%d post=%d", preSignSize, got)
		}
	}

	// STEP 6 — verify after signing.
	{
		stdout, stderr, code := runATB(t, bin, "verify", "--bundle", bundlePath)
		if code != 0 {
			t.Fatalf("post-sign verify exit=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
		}
	}
}

func fileSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Size()
}

func TestE2EPipelineConcurrentImport(t *testing.T) {
	bin := buildBinary(t)

	work := t.TempDir()
	bundlePath := filepath.Join(work, "run.atb", "bundle.atb")

	// Seed: one capture run to materialise the bundle, then a single import
	// to mirror the steady-state shape callers will hit concurrently.
	if _, stderr, code := runATB(t, bin,
		"capture", "run",
		"--bundle", bundlePath,
		"--snapshot", "seed",
		"--", "echo", "seed",
	); code != 0 {
		t.Fatalf("seed capture run exit=%d stderr=%s", code, stderr)
	}

	fixture := writeChatlogFixture(t, work)

	// Three concurrent imports against the same bundle — the advisory file
	// lock should serialise them; some may legitimately fail with a lock
	// contention exit, but the bundle must remain verifiable afterwards.
	var wg sync.WaitGroup
	const goroutines = 3
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, _, _ = runATB(t, bin,
				"import", "chatlog",
				"--from", "generic-jsonl",
				"--input", fixture,
				"--bundle", bundlePath,
			)
		}()
	}
	wg.Wait()

	stdout, stderr, code := runATB(t, bin, "verify", "--bundle", bundlePath)
	if code != 0 {
		t.Fatalf("verify after concurrent imports exit=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}
}
