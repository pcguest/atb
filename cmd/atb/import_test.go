// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImport_NotImplemented(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runImport([]string{"--source", "dummy.atb"}, nil, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}
	if !strings.Contains(stdout.String(), "import: not yet implemented (roadmap Q3 2026)") {
		t.Fatalf("expected not-implemented message, got %q", stdout.String())
	}
}

func TestImport_MissingSource(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runImport(nil, nil, &stdout, &stderr)
	if exitCode != exitUserError {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, exitUserError)
	}
	if !strings.Contains(stdout.String()+stderr.String(), "--source is required") {
		t.Fatalf("expected missing source message, got stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestImportChatlog_EndToEnd(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	sourcePath := filepath.Join("testdata", "import_e2e_chatlog.jsonl")

	var importStdout bytes.Buffer
	var importStderr bytes.Buffer
	importExitCode := runImport([]string{
		"chatlog",
		"--source", sourcePath,
		"--bundle", bundlePath,
	}, nil, &importStdout, &importStderr)
	if importExitCode != exitSuccess {
		t.Fatalf("runImport() exit code = %d, want %d (stderr=%q)", importExitCode, exitSuccess, importStderr.String())
	}
	if _, err := os.Stat(bundlePath); err != nil {
		t.Fatalf("expected bundle at %s: %v", bundlePath, err)
	}

	var verifyStdout bytes.Buffer
	var verifyStderr bytes.Buffer
	verifyExitCode := runVerify([]string{
		"--bundle", bundlePath,
		"--format", "json",
	}, &verifyStdout, &verifyStderr)
	if verifyExitCode != exitSuccess && verifyExitCode != exitIntegrityFailure {
		t.Fatalf("runVerify() exit code = %d, want %d or %d (stderr=%q)", verifyExitCode, exitSuccess, exitIntegrityFailure, verifyStderr.String())
	}

	var report map[string]any
	if err := json.Unmarshal(verifyStdout.Bytes(), &report); err != nil {
		t.Fatalf("decode verify report: %v (stdout=%q)", err, verifyStdout.String())
	}
	retrospective, ok := report["retrospective"].(bool)
	if !ok || !retrospective {
		t.Fatalf("expected retrospective=true in verify report, got %#v", report["retrospective"])
	}
}
