// SPDX-License-Identifier: MIT
package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/compliancepack"
)

func TestRunCompliancePackWritesZip(t *testing.T) {
	b, err := bundle.New()
	if err != nil {
		t.Fatal(err)
	}
	add := func(eventType string, data map[string]any) {
		if err := b.AppendWithOptions(eventType, data, &bundle.AppendOptions{
			Timestamp: "2026-06-15T00:00:00Z",
		}); err != nil {
			t.Fatalf("append %s: %v", eventType, err)
		}
	}
	add("ai.request.received", map[string]any{
		"request_id": "req-1", "actor_id_hash": "sha256:actor", "purpose_tag": "policy_decision",
	})
	add("ai.action.precommit", map[string]any{
		"action_id": "act-1", "action_type": "deploy",
		"action_parameters_digest": "sha256:params",
	})
	add("ai.policy.decision", map[string]any{
		"action_id": "act-1", "policy_id": "policy", "policy_version": "v1",
		"decision": "deny", "decision_reason_codes": []any{"review"},
		"subject_id_hash": "sha256:actor",
	})
	bundlePath := filepath.Join(t.TempDir(), "bundle.atb")
	if err := b.Save(bundlePath); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "pack.zip")
	var stdout, stderr bytes.Buffer
	code := runCompliance([]string{
		"pack",
		"--bundle", bundlePath,
		"--profile", "atb.profile.policy_decision",
		"--regime", "eu-ai-act",
		"--out", output,
	}, &stdout, &stderr)
	if code != exitSuccess {
		t.Fatalf("runCompliance exit=%d stderr=%s", code, stderr.String())
	}
	reader, err := zip.OpenReader(output)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	seen := map[string]bool{}
	for _, file := range reader.File {
		seen[file.Name] = true
	}
	for _, name := range []string{"bundle.atb", "MANIFEST.json", "reports/verify.report.json"} {
		if !seen[name] {
			t.Errorf("zip missing %q", name)
		}
	}
}

func TestWriteComplianceDirectoryValidatesPaths(t *testing.T) {
	output := filepath.Join(t.TempDir(), "pack")
	pack := compliancepack.Pack{
		GeneratedAt: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		Files: []compliancepack.File{
			{Name: "MANIFEST.json", Content: []byte(`{"pack_version":"1"}`)},
			{Name: "reports/verify.json", Content: []byte(`{"integrity_pass":true}`)},
		},
	}
	if err := writeComplianceDirectory(output, pack); err != nil {
		t.Fatalf("writeComplianceDirectory: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(output, "reports", "verify.json"))
	if err != nil {
		t.Fatalf("read nested artifact: %v", err)
	}
	if !bytes.Contains(body, []byte(`"integrity_pass":true`)) {
		t.Fatalf("nested artifact = %q", body)
	}

	unsafe := compliancepack.Pack{Files: []compliancepack.File{
		{Name: "../escape.json", Content: []byte("{}")},
	}}
	if err := writeComplianceDirectory(filepath.Join(t.TempDir(), "unsafe"), unsafe); err == nil ||
		!strings.Contains(err.Error(), "unsafe pack path") {
		t.Fatalf("unsafe path error = %v", err)
	}

	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("block"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeComplianceDirectory(filepath.Join(blocker, "pack"), pack); err == nil {
		t.Fatal("expected output-directory creation error")
	}
}

func TestParseComplianceArgsContracts(t *testing.T) {
	valid, err := parseComplianceArgs([]string{
		"pack",
		"--bundle=bundle.atb",
		"--profile=atb.profile.policy_decision",
		"--regime=EU-AI-ACT",
		"--out=pack",
	})
	if err != nil {
		t.Fatalf("parse valid equals args: %v", err)
	}
	if valid.Regime != compliancepack.RegimeEUAIAct || valid.Output != "pack" {
		t.Fatalf("config = %+v", valid)
	}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "subcommand", args: nil, want: "expected subcommand"},
		{name: "bundle value", args: []string{"pack", "--bundle"}, want: "missing value"},
		{name: "profile value", args: []string{"pack", "--profile"}, want: "missing value"},
		{name: "regime value", args: []string{"pack", "--regime"}, want: "missing value"},
		{name: "out value", args: []string{"pack", "--out"}, want: "missing value"},
		{name: "unknown", args: []string{"pack", "--wat"}, want: "unknown argument"},
		{name: "bundle required", args: []string{"pack", "--profile", "p", "--out", "o"}, want: "--bundle is required"},
		{name: "profile required", args: []string{"pack", "--bundle", "b", "--out", "o"}, want: "--profile is required"},
		{name: "out required", args: []string{"pack", "--bundle", "b", "--profile", "p"}, want: "--out is required"},
		{name: "regime", args: []string{"pack", "--bundle", "b", "--profile", "p", "--out", "o", "--regime", "nist"}, want: "unsupported"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseComplianceArgs(tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}

	_, err = parseComplianceArgs([]string{"pack", "--help"})
	if !errors.Is(err, errComplianceHelp) {
		t.Fatalf("help error = %v", err)
	}
}

func TestRunComplianceHelpAndBuildFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runCompliance([]string{"pack", "--help"}, &stdout, &stderr); code != exitSuccess {
		t.Fatalf("help exit = %d", code)
	}
	if !strings.Contains(stdout.String(), "Usage: atb compliance pack") {
		t.Fatalf("help stdout = %q", stdout.String())
	}

	stdout.Reset()
	if code := runCompliance(nil, &stdout, &stderr); code != exitUserError {
		t.Fatalf("invalid args exit = %d", code)
	}
	if !strings.Contains(stderr.String(), "expected subcommand") {
		t.Fatalf("invalid args stderr = %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code := runCompliance([]string{
		"pack",
		"--bundle", filepath.Join(t.TempDir(), "missing.atb"),
		"--profile", "atb.profile.policy_decision",
		"--out", filepath.Join(t.TempDir(), "pack"),
	}, &stdout, &stderr)
	if code != exitUserError || !strings.Contains(stderr.String(), "compliance pack") {
		t.Fatalf("build failure exit = %d, stderr = %q", code, stderr.String())
	}
}
