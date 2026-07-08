// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

func TestRunCorroborateRejectsInvalidArguments(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "invalid mutation flag", args: []string{"--format", "yaml"}, wantErr: "expected text|json"},
		{name: "missing source value", args: []string{"--source"}, wantErr: "missing value for --source"},
		{name: "missing URL value", args: []string{"--url"}, wantErr: "missing value for --url"},
		{name: "missing ref value", args: []string{"--ref"}, wantErr: "missing value for --ref"},
		{name: "missing bundle value", args: []string{"--bundle"}, wantErr: "missing value for --bundle"},
		{name: "missing path value", args: []string{"--path"}, wantErr: "missing value for --path"},
		{name: "unknown argument", args: []string{"--wat"}, wantErr: "unknown argument"},
		{name: "missing required fields", args: nil, wantErr: "Usage: atb corroborate"},
		{name: "HTTP source needs URL", args: []string{"--source", "http-gateway", "--ref", "event-1"}, wantErr: "--url is required"},
		{name: "file source needs path", args: []string{"--source", "file-receipt", "--ref", "event-1"}, wantErr: "--path is required"},
		{name: "unknown source", args: []string{"--source", "ledger", "--ref", "event-1"}, wantErr: "unknown source"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := runCorroborate(tc.args, &stdout, &stderr); code != exitUserError {
				t.Fatalf("exit code = %d, want %d", code, exitUserError)
			}
			if !strings.Contains(stderr.String(), tc.wantErr) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tc.wantErr)
			}
		})
	}
}

func TestRunCorroborateFileReceiptDryRunAndAppend(t *testing.T) {
	dir := t.TempDir()
	receiptPath := filepath.Join(dir, "receipt.json")
	if err := os.WriteFile(receiptPath, []byte(`{"receipt_id":"receipt-1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	bundlePath := filepath.Join(dir, "evidence.atb")
	baseArgs := []string{
		"--source", "file-receipt",
		"--path", receiptPath,
		"--ref", "event-hash-1",
		"--bundle", bundlePath,
	}

	var stdout, stderr bytes.Buffer
	dryArgs := append(append([]string(nil), baseArgs...), "--dry-run")
	if code := runCorroborate(dryArgs, &stdout, &stderr); code != exitSuccess {
		t.Fatalf("dry-run exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Dry run") {
		t.Fatalf("dry-run stdout = %q", stdout.String())
	}
	if _, err := os.Stat(bundlePath); !os.IsNotExist(err) {
		t.Fatalf("dry run created bundle: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := runCorroborate(baseArgs, &stdout, &stderr); code != exitSuccess {
		t.Fatalf("append exit = %d, stderr = %q", code, stderr.String())
	}
	b, err := bundle.Load(t.Context(), bundlePath)
	if err != nil {
		t.Fatalf("load appended bundle: %v", err)
	}
	if len(b.Records) != 2 || b.Records[len(b.Records)-1].Event.Type != event.TypeCorroborationExternal {
		t.Fatalf("records = %+v", b.Records)
	}
}

func TestRunCorroborateHTTPGatewayAndAdapterFailures(t *testing.T) {
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"verified"}`))
	}))
	t.Cleanup(okServer.Close)

	var stdout, stderr bytes.Buffer
	bundlePath := filepath.Join(t.TempDir(), "gateway.atb")
	code := runCorroborate([]string{
		"--source", "http-gateway",
		"--url", okServer.URL,
		"--ref", "event-hash-2",
		"--bundle", bundlePath,
	}, &stdout, &stderr)
	if code != exitSuccess {
		t.Fatalf("HTTP corroborate exit = %d, stderr = %q", code, stderr.String())
	}

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(failServer.Close)
	stdout.Reset()
	stderr.Reset()
	code = runCorroborate([]string{
		"--source", "http-gateway",
		"--url", failServer.URL,
		"--ref", "event-hash-3",
		"--bundle", filepath.Join(t.TempDir(), "failed.atb"),
	}, &stdout, &stderr)
	if code != exitSystemError || !strings.Contains(stderr.String(), "unexpected status 503") {
		t.Fatalf("failure exit = %d, stderr = %q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runCorroborate([]string{
		"--source", "file-receipt",
		"--path", filepath.Join(t.TempDir(), "missing.json"),
		"--ref", "event-hash-4",
	}, &stdout, &stderr)
	if code != exitSystemError || !strings.Contains(stderr.String(), "read file") {
		t.Fatalf("missing receipt exit = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunCorroborateClassifiesInvalidBundle(t *testing.T) {
	dir := t.TempDir()
	receiptPath := filepath.Join(dir, "receipt.json")
	bundlePath := filepath.Join(dir, "invalid.atb")
	if err := os.WriteFile(receiptPath, []byte(`{"receipt_id":"receipt-2"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bundlePath, []byte("not a bundle"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runCorroborate([]string{
		"--source", "file-receipt",
		"--path", receiptPath,
		"--ref", "event-hash-5",
		"--bundle", bundlePath,
	}, &stdout, &stderr)
	if code == exitSuccess || !strings.Contains(stderr.String(), "load bundle") {
		t.Fatalf("invalid bundle exit = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunCorroborateJSONUsesProvidedWriters(t *testing.T) {
	dir := t.TempDir()
	receiptPath := filepath.Join(dir, "receipt.json")
	if err := os.WriteFile(receiptPath, []byte(`{"receipt_id":"receipt-json"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runCorroborate([]string{
		"--source", "file-receipt",
		"--path", receiptPath,
		"--ref", "event-hash-json",
		"--bundle", filepath.Join(dir, "json.atb"),
		"--dry-run",
		"--format", "json",
	}, &stdout, &stderr)
	if code != exitSuccess {
		t.Fatalf("JSON exit = %d, stderr = %q", code, stderr.String())
	}
	var result mutationResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode stdout %q: %v", stdout.String(), err)
	}
	if result.Status != "ok" || result.Action != "preview_corroborate" || !result.DryRun {
		t.Fatalf("result = %+v", result)
	}
}
