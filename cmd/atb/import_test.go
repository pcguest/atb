// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
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
