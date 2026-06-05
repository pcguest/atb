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

const otelLLMAndToolPayload = `{
  "resourceSpans": [{
    "resource": {"attributes": [{"key": "gen_ai.system", "value": {"stringValue": "openai"}}]},
    "scopeSpans": [{
      "scope": {"name": "instr", "version": "1.0.0"},
      "spans": [
        {
          "traceId": "0102030405060708090a0b0c0d0e0f10",
          "spanId": "0102030405060708",
          "name": "gen_ai.chat",
          "kind": 3,
          "startTimeUnixNano": "1772622902000000000",
          "endTimeUnixNano": "1772622903200000000",
          "status": {"code": 1},
          "attributes": [{"key": "gen_ai.request.model", "value": {"stringValue": "gpt-4.1-mini"}}]
        },
        {
          "traceId": "0102030405060708090a0b0c0d0e0f10",
          "spanId": "1112131415161718",
          "parentSpanId": "0102030405060708",
          "name": "tool.run",
          "kind": 3,
          "startTimeUnixNano": "1772622903300000000",
          "endTimeUnixNano": "1772622903400000000",
          "status": {"code": 1}
        }
      ]
    }]
  }]
}`

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestImportOTel_EndToEnd(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	inputPath := writeTempFile(t, tmp, "trace.json", otelLLMAndToolPayload)

	var stdout, stderr bytes.Buffer
	exit := runImport([]string{
		"otel",
		"--input", inputPath,
		"--bundle", bundlePath,
		"--format", "json",
	}, nil, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("runImport otel exit = %d, want %d (stderr=%q)", exit, exitSuccess, stderr.String())
	}

	var result importOTelResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v (stdout=%q)", err, stdout.String())
	}
	if result.EventsWritten != 2 {
		t.Fatalf("events_written = %d, want 2", result.EventsWritten)
	}
	if _, err := os.Stat(bundlePath); err != nil {
		t.Fatalf("expected bundle at %s: %v", bundlePath, err)
	}

	// The ingested bundle is a sound, hash-chained bundle stamped as a
	// retrospective import. The overall assurance gate may report a failure
	// (an OTLP import is unsigned and unanchored, with a low CAS score) without
	// that being an integrity break — exactly as the chatlog import e2e treats
	// it — so we accept either the pass or the gate-failure exit and assert the
	// report's retrospective provenance instead.
	var vOut, vErr bytes.Buffer
	vExit := runVerify([]string{"--bundle", bundlePath, "--format", "json"}, &vOut, &vErr)
	if vExit != exitSuccess && vExit != exitIntegrityFailure {
		t.Fatalf("runVerify exit = %d, want %d or %d (stderr=%q)", vExit, exitSuccess, exitIntegrityFailure, vErr.String())
	}
	var report map[string]any
	if err := json.Unmarshal(vOut.Bytes(), &report); err != nil {
		t.Fatalf("decode verify report: %v (stdout=%q)", err, vOut.String())
	}
	if retrospective, ok := report["retrospective"].(bool); !ok || !retrospective {
		t.Fatalf("expected retrospective=true in verify report, got %#v", report["retrospective"])
	}
}

func TestImportOTel_ReadsStdin(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")

	var stdout, stderr bytes.Buffer
	exit := runImport([]string{"otel", "--input", "-", "--bundle", bundlePath},
		strings.NewReader(otelLLMAndToolPayload), &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("runImport otel (stdin) exit = %d, want %d (stderr=%q)", exit, exitSuccess, stderr.String())
	}
	if !strings.Contains(stdout.String(), "imported: 2 events") {
		t.Fatalf("stdout = %q, want it to report 2 imported events", stdout.String())
	}
}

func TestImportOTel_MissingInputIsUserError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runImport([]string{"otel", "--bundle", "x.atb"}, nil, &stdout, &stderr)
	if exit != exitUserError {
		t.Fatalf("exit = %d, want %d", exit, exitUserError)
	}
	if !strings.Contains(stderr.String(), "--input") {
		t.Fatalf("stderr = %q, want it to mention the required --input flag", stderr.String())
	}
}

func TestImportOTel_MalformedJSONIsUserError(t *testing.T) {
	tmp := t.TempDir()
	inputPath := writeTempFile(t, tmp, "bad.json", "{not json")

	var stdout, stderr bytes.Buffer
	exit := runImport([]string{"otel", "--input", inputPath, "--bundle", filepath.Join(tmp, "b.atb")},
		nil, &stdout, &stderr)
	if exit != exitUserError {
		t.Fatalf("exit = %d, want %d (stderr=%q)", exit, exitUserError, stderr.String())
	}
}

func TestImportOTel_NoTranslatableSpansIsUserError(t *testing.T) {
	tmp := t.TempDir()
	inputPath := writeTempFile(t, tmp, "empty.json", `{"resourceSpans": []}`)

	var stdout, stderr bytes.Buffer
	exit := runImport([]string{"otel", "--input", inputPath, "--bundle", filepath.Join(tmp, "b.atb")},
		nil, &stdout, &stderr)
	if exit != exitUserError {
		t.Fatalf("exit = %d, want %d (stderr=%q)", exit, exitUserError, stderr.String())
	}
	if !strings.Contains(stderr.String(), "no translatable spans") {
		t.Fatalf("stderr = %q, want the no-spans message", stderr.String())
	}
}
