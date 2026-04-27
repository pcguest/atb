// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	verifypkg "github.com/pcguest/atb/internal/verify"
)

func TestRunProfilesValidateBuiltIns(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runProfiles([]string{"validate"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}

	output := stdout.String()
	for _, profile := range verifypkg.AllProfiles() {
		line := "OK " + profile.ID()
		if !strings.Contains(output, line) {
			t.Fatalf("expected %q in output, got:\n%s", line, output)
		}
	}
}

func TestRunProfilesValidateFailures(t *testing.T) {
	dir := t.TempDir()

	writeProfilesTestFile(t, dir, "bad-weights.yaml", `
id: "org.example.bad_weights"
version: 1
workflow_class: "custom_workflow"
blind_spots:
  - "queue receipt not recorded"
weights:
  EC: 0.50
  FC: 0.30
  RC: 0.10
  TC: 0.10
  SC: 0.10
  XC: 0.05
  AC: 0.05
  GC: 0.05
supports_cas: true
required:
  - type: "ai.request.received"
    severity: "critical"
optional: []
relations: []
`)

	writeProfilesTestFile(t, dir, "missing-workflow.yaml", `
id: "org.example.missing_workflow"
version: 1
blind_spots:
  - "sink receipt not recorded"
weights:
  EC: 0.20
  FC: 0.15
  RC: 0.15
  TC: 0.10
  SC: 0.10
  XC: 0.10
  AC: 0.10
  GC: 0.10
supports_cas: true
required:
  - type: "ai.request.received"
    severity: "critical"
optional: []
relations: []
`)

	writeProfilesTestFile(t, dir, "duplicate-a.yaml", `
id: "org.example.duplicate_profile"
version: 1
workflow_class: "duplicate_a"
blind_spots:
  - "operator ticket not recorded"
weights:
  EC: 0.20
  FC: 0.15
  RC: 0.15
  TC: 0.10
  SC: 0.10
  XC: 0.10
  AC: 0.10
  GC: 0.10
supports_cas: true
required:
  - type: "ai.request.received"
    severity: "critical"
optional: []
relations: []
`)

	writeProfilesTestFile(t, dir, "duplicate-b.yaml", `
id: "org.example.duplicate_profile"
version: 1
workflow_class: "duplicate_b"
blind_spots:
  - "operator ticket not recorded"
weights:
  EC: 0.20
  FC: 0.15
  RC: 0.15
  TC: 0.10
  SC: 0.10
  XC: 0.10
  AC: 0.10
  GC: 0.10
supports_cas: true
required:
  - type: "ai.request.received"
    severity: "critical"
optional: []
relations: []
`)

	t.Run("text", func(t *testing.T) {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		exitCode := runProfiles([]string{"validate", "--dir", dir}, &stdout, &stderr)
		if exitCode != exitUserError {
			t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitUserError, stderr.String())
		}
		if stderr.Len() != 0 {
			t.Fatalf("unexpected stderr: %q", stderr.String())
		}

		output := stdout.String()
		for _, want := range []string{
			"FAILED org.example.bad_weights",
			"FAILED org.example.missing_workflow",
			"FAILED org.example.duplicate_profile",
			"weights sum to",
			"workflow_class is required",
			"duplicate profile ID",
		} {
			if !strings.Contains(output, want) {
				t.Fatalf("expected %q in output, got:\n%s", want, output)
			}
		}
	})

	t.Run("json", func(t *testing.T) {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		exitCode := runProfiles([]string{"validate", "--dir", dir, "--format", "json"}, &stdout, &stderr)
		if exitCode != exitUserError {
			t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitUserError, stderr.String())
		}
		if stderr.Len() != 0 {
			t.Fatalf("unexpected stderr: %q", stderr.String())
		}

		var report map[string]struct {
			ID     string   `json:"id"`
			Valid  bool     `json:"valid"`
			Errors []string `json:"errors"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatalf("unmarshal validation report: %v\noutput=%s", err, stdout.String())
		}

		for _, key := range []string{
			"org.example.bad_weights",
			"org.example.missing_workflow",
			"org.example.duplicate_profile",
		} {
			entry, ok := report[key]
			if !ok {
				t.Fatalf("missing report entry for %q", key)
			}
			if entry.Valid {
				t.Fatalf("expected %q to be invalid", key)
			}
			if len(entry.Errors) == 0 {
				t.Fatalf("expected %q to include errors", key)
			}
		}
	})
}

func TestRunProfilesValidateBuiltInsJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runProfiles([]string{"validate", "--format", "json"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}

	var report map[string]struct {
		ID     string   `json:"id"`
		Valid  bool     `json:"valid"`
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal JSON output: %v\noutput=%s", err, stdout.String())
	}

	profiles := verifypkg.AllProfiles()
	if len(report) == 0 {
		t.Fatal("expected non-empty JSON report for built-in profiles")
	}

	for _, profile := range profiles {
		entry, ok := report[profile.ID()]
		if !ok {
			t.Errorf("missing report entry for built-in profile %q", profile.ID())
			continue
		}
		if !entry.Valid {
			t.Errorf("expected profile %q to be valid, errors: %v", profile.ID(), entry.Errors)
		}
		if entry.ID != profile.ID() {
			t.Errorf("profile %q: id field mismatch: got %q", profile.ID(), entry.ID)
		}
		if len(entry.Errors) != 0 {
			t.Errorf("profile %q: unexpected errors: %v", profile.ID(), entry.Errors)
		}
	}
}

func writeProfilesTestFile(t testing.TB, dir, name, content string) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o600); err != nil {
		t.Fatalf("write profile fixture %s: %v", path, err)
	}
}
