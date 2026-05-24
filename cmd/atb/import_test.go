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

func TestImport_RequiresChatlogSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runImport([]string{"--source", "dummy.atb"}, nil, &stdout, &stderr)
	if exitCode != exitUserError {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitUserError, stderr.String())
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "supported: chatlog") {
		t.Fatalf("expected chatlog-only guidance, got %q", combined)
	}
}

func TestImport_MissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runImport(nil, nil, &stdout, &stderr)
	if exitCode != exitUserError {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, exitUserError)
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "import chatlog") {
		t.Fatalf("expected import chatlog usage, got stdout=%q stderr=%q", stdout.String(), stderr.String())
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
