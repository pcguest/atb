// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

func TestRunPushQueueOnlyDryRunAndDelivery(t *testing.T) {
	t.Chdir(t.TempDir())
	bundlePath, _ := newTestBundleFile(t)
	key := strings.Repeat("ab", 32)

	var stdout, stderr bytes.Buffer
	code := runPush([]string{
		"--queue", "https://queue.example.test/ingest",
		"--hmac-key", key,
		"--bundle", bundlePath,
		"--dry-run",
		"--format", "json",
	}, &stdout, &stderr)
	if code != exitSuccess {
		t.Fatalf("dry-run exit=%d stderr=%q", code, stderr.String())
	}
	var preview pushResult
	if err := json.Unmarshal(stdout.Bytes(), &preview); err != nil {
		t.Fatalf("decode preview %q: %v", stdout.String(), err)
	}
	if preview.Action != "preview_push" || preview.Envelope == nil || preview.QueueURL == "" {
		t.Fatalf("preview = %+v", preview)
	}

	var delivered []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		delivered, _ = io.ReadAll(r.Body)
		if r.Header.Get("X-ATB-Signature") == "" {
			t.Error("missing queue signature")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	stdout.Reset()
	stderr.Reset()
	code = runPush([]string{
		"--queue=" + server.URL,
		"--hmac-key=" + key,
		"--bundle=" + bundlePath,
	}, &stdout, &stderr)
	if code != exitSuccess {
		t.Fatalf("delivery exit=%d stderr=%q", code, stderr.String())
	}
	if len(delivered) == 0 || !strings.Contains(stdout.String(), server.URL) {
		t.Fatalf("delivered=%q stdout=%q", delivered, stdout.String())
	}
}

func TestRunPushQueueValidationAndRemoteFailure(t *testing.T) {
	t.Chdir(t.TempDir())
	bundlePath, _ := newTestBundleFile(t)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "missing key",
			args: []string{"--queue", "https://queue.example.test", "--bundle", bundlePath},
			want: "--hmac-key is required",
		},
		{
			name: "invalid key",
			args: []string{"--queue", "https://queue.example.test", "--hmac-key", "xyz", "--bundle", bundlePath},
			want: "valid hex",
		},
		{
			name: "empty key",
			args: []string{"--queue", "https://queue.example.test", "--hmac-key=", "--bundle", bundlePath},
			want: "required",
		},
		{
			name: "invalid target",
			args: []string{"https://not-s3.example", "--bundle", bundlePath},
			want: "s3://",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := runPush(tc.args, &stdout, &stderr); code != exitUserError {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("stderr=%q want=%q", stderr.String(), tc.want)
			}
		})
	}

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(failServer.Close)
	var stdout, stderr bytes.Buffer
	code := runPush([]string{
		"--queue", failServer.URL,
		"--hmac-key", strings.Repeat("cd", 32),
		"--bundle", bundlePath,
		"--format=json",
	}, &stdout, &stderr)
	if code != exitSystemError {
		t.Fatalf("remote failure exit=%d stderr=%q", code, stderr.String())
	}
	var result pushResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode failure %q: %v", stdout.String(), err)
	}
	if result.Status != "error" || result.ExitCode != exitSystemError {
		t.Fatalf("result=%+v", result)
	}
}

func TestParsePushArgsErrorContracts(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "bundle value", args: []string{"--bundle"}, want: "missing value"},
		{name: "duplicate bundle", args: []string{"--bundle=a", "--bundle=b"}, want: "already set"},
		{name: "lock value", args: []string{"--lock-until"}, want: "missing value"},
		{name: "queue value", args: []string{"--queue"}, want: "missing value"},
		{name: "key value", args: []string{"--hmac-key"}, want: "missing value"},
		{name: "format value", args: []string{"--format"}, want: "missing value"},
		{name: "format invalid", args: []string{"--format=yaml"}, want: "expected text|json"},
		{name: "flag unknown", args: []string{"--wat"}, want: "unknown flag"},
		{name: "extra target", args: []string{"s3://one", "s3://two"}, want: "target already set"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parsePushArgs(tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v want=%q", err, tc.want)
			}
		})
	}

	cfg, err := parsePushArgs([]string{
		"s3://bucket/prefix",
		"-b", "bundle.atb",
		"--lock-until=2028-01-01",
		"--queue=https://queue.example",
		"--hmac-key=abcd",
		"--dry-run",
		"--format=json",
	})
	if err != nil || !cfg.DryRun || cfg.QueueEndpoint == "" || cfg.HMACKeyHex != "abcd" {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
}

func TestPushMetadataFallbacksAndMessages(t *testing.T) {
	key, err := parseHMACKey(" abcd ")
	if err != nil || len(key) != 2 {
		t.Fatalf("key=%x err=%v", key, err)
	}
	for _, value := range []string{"", "xyz"} {
		if _, err := parseHMACKey(value); err == nil {
			t.Fatalf("parseHMACKey(%q) succeeded", value)
		}
	}

	if got := successMessage("", "", "https://queue.example"); !strings.Contains(got, "queue envelope") {
		t.Fatalf("queue message=%q", got)
	}
	if got := successMessage("s3://bucket/key", "key", ""); !strings.Contains(got, "bundle pushed") {
		t.Fatalf("S3 message=%q", got)
	}
	if got := successMessage("s3://bucket/key", "key", " https://queue.example "); !strings.Contains(got, "; queue") {
		t.Fatalf("combined message=%q", got)
	}

	path := filepath.Join(t.TempDir(), "bundle.atb")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	modTime := time.Date(2026, 6, 30, 1, 2, 3, 0, time.UTC)
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
	if got := inferSealTimestamp(&bundle.Bundle{}, path); !got.Equal(modTime) {
		t.Fatalf("file fallback=%v want=%v", got, modTime)
	}
	now := time.Now()
	if got := inferSealTimestamp(&bundle.Bundle{}, filepath.Join(t.TempDir(), "missing")); got.Before(now) {
		t.Fatalf("clock fallback=%v before %v", got, now)
	}
}
