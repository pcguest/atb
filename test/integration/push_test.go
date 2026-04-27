// SPDX-License-Identifier: MIT
package integration_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

type queueEnvelope struct {
	BundleID      string `json:"bundle_id"`
	Digest        string `json:"digest"`
	SealTimestamp string `json:"seal_timestamp"`
	ProfileID     string `json:"profile_id"`
	ATBVersion    string `json:"atb_version"`
}

func TestPushQueueReceivesSignedEnvelope(t *testing.T) {
	bundlePath, wantEnvelope, key := newPushTestBundle(t)

	var gotBody []byte
	var gotSignature string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method: got %q want %q", r.Method, http.MethodPost)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		gotBody = body
		gotSignature = r.Header.Get("X-ATB-Signature")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	stdout, stderr, exitCode := runATB(t,
		"push",
		"--bundle", bundlePath,
		"--queue", server.URL,
		"--hmac-key", hex.EncodeToString(key),
	)
	if exitCode != 0 {
		t.Fatalf("exit code: got %d want 0 (stdout=%q stderr=%q)", exitCode, stdout, stderr)
	}

	var gotEnvelope queueEnvelope
	if err := json.Unmarshal(gotBody, &gotEnvelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	if gotEnvelope != wantEnvelope {
		t.Fatalf("envelope: got %+v want %+v", gotEnvelope, wantEnvelope)
	}

	mac := hmac.New(sha256.New, key)
	mac.Write(gotBody)
	wantSignature := hex.EncodeToString(mac.Sum(nil))
	if gotSignature != wantSignature {
		t.Fatalf("signature: got %q want %q", gotSignature, wantSignature)
	}
}

func TestPushQueueRequiresHMACKey(t *testing.T) {
	bundlePath, _, _ := newPushTestBundle(t)

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	stdout, stderr, exitCode := runATB(t,
		"push",
		"--bundle", bundlePath,
		"--queue", server.URL,
	)
	if exitCode != 1 {
		t.Fatalf("exit code: got %d want 1 (stdout=%q stderr=%q)", exitCode, stdout, stderr)
	}
	if requests.Load() != 0 {
		t.Fatalf("queue server should not be called, got %d requests", requests.Load())
	}
	if !bytes.Contains([]byte(stderr), []byte("--hmac-key is required when --queue is set")) {
		t.Fatalf("stderr %q missing expected message", stderr)
	}
}

func TestPushQueueDryRunDoesNotSend(t *testing.T) {
	bundlePath, wantEnvelope, key := newPushTestBundle(t)

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	stdout, stderr, exitCode := runATB(t,
		"push",
		"--bundle", bundlePath,
		"--queue", server.URL,
		"--hmac-key", hex.EncodeToString(key),
		"--dry-run",
	)
	if exitCode != 0 {
		t.Fatalf("exit code: got %d want 0 (stdout=%q stderr=%q)", exitCode, stdout, stderr)
	}
	if requests.Load() != 0 {
		t.Fatalf("queue server should not be called, got %d requests", requests.Load())
	}
	if !bytes.Contains([]byte(stdout), []byte(server.URL)) {
		t.Fatalf("stdout %q missing queue endpoint", stdout)
	}
	if !bytes.Contains([]byte(stdout), []byte(`"bundle_id":"`+wantEnvelope.BundleID+`"`)) {
		t.Fatalf("stdout %q missing envelope JSON", stdout)
	}
}

func newPushTestBundle(t *testing.T) (string, queueEnvelope, []byte) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.atb")

	b, err := bundle.New()
	if err != nil {
		t.Fatalf("bundle.New: %v", err)
	}

	if err := b.AppendWithOptions("dev.session", map[string]any{
		"step": "queue",
	}, &bundle.AppendOptions{
		Timestamp: "2026-04-20T03:04:05Z",
	}); err != nil {
		t.Fatalf("append dev.session: %v", err)
	}

	if err := b.AppendWithOptions("atb.snapshot", map[string]any{
		"name": "push-test",
	}, &bundle.AppendOptions{
		Timestamp: "2026-04-20T03:05:06Z",
	}); err != nil {
		t.Fatalf("append atb.snapshot: %v", err)
	}

	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	sum := sha256.Sum256(raw)

	manifest := b.Manifest()
	if manifest == nil {
		t.Fatal("expected manifest")
	}

	return path, queueEnvelope{
		BundleID:      manifest.BundleID,
		Digest:        hex.EncodeToString(sum[:]),
		SealTimestamp: "2026-04-20T03:05:06Z",
		ProfileID:     "",
		ATBVersion:    "1.10.0",
	}, []byte("push-queue-secret")
}

func runATB(t *testing.T, args ...string) (string, string, int) {
	t.Helper()

	repoRoot := repoRootForPushIntegration(t)
	binaryPath := buildPushBinary(t, repoRoot)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Dir = t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return stdout.String(), stderr.String(), exitErr.ExitCode()
	}

	t.Fatalf("run atb: %v (stdout=%q stderr=%q)", err, stdout.String(), stderr.String())
	return "", "", 0
}

func repoRootForPushIntegration(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func buildPushBinary(t *testing.T, repoRoot string) string {
	t.Helper()

	binaryPath := filepath.Join(t.TempDir(), "atb")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, "./cmd/atb")
	cmd.Dir = repoRoot

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build installed binary: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	return binaryPath
}
