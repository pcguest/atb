// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	evidencepack "github.com/pcguest/atb/internal/evidencepack"
)

func TestRunEvidencePackMissingPaths(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runEvidencePack(nil, &stdout, &stderr)
	if exitCode != exitUserError {
		t.Fatalf("runEvidencePack() exit code = %d, want %d", exitCode, exitUserError)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "at least one bundle path is required") {
		t.Fatalf("stderr = %q, want missing paths error", stderr.String())
	}
}

func TestRunEvidencePackNonexistentFileStillEmitsJSON(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.atb")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runEvidencePack([]string{missing}, &stdout, &stderr)
	if exitCode != exitUserError {
		t.Fatalf("runEvidencePack() exit code = %d, want %d (stderr=%q)", exitCode, exitUserError, stderr.String())
	}

	var pack evidencepack.Pack
	if err := json.Unmarshal(stdout.Bytes(), &pack); err != nil {
		t.Fatalf("unmarshal evidence pack: %v\noutput=%s", err, stdout.String())
	}
	if len(pack.Bundles) != 1 {
		t.Fatalf("bundles len = %d, want 1", len(pack.Bundles))
	}
	if pack.Bundles[0].Error == "" {
		t.Fatal("expected error field for missing bundle")
	}
	if pack.Bundles[0].BundlePath != missing {
		t.Fatalf("bundle_path = %q, want %q", pack.Bundles[0].BundlePath, missing)
	}
}

func TestRunEvidencePackMixedSuccessAndError(t *testing.T) {
	passPath := profileFixturePath(t, "rag_answer-pass.atb")
	missing := filepath.Join(t.TempDir(), "missing.atb")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runEvidencePack([]string{passPath, missing}, &stdout, &stderr)
	if exitCode != exitUserError {
		t.Fatalf("runEvidencePack() exit code = %d, want %d (stderr=%q)", exitCode, exitUserError, stderr.String())
	}

	var pack evidencepack.Pack
	if err := json.Unmarshal(stdout.Bytes(), &pack); err != nil {
		t.Fatalf("unmarshal evidence pack: %v\noutput=%s", err, stdout.String())
	}
	if len(pack.Bundles) != 2 {
		t.Fatalf("bundles len = %d, want 2", len(pack.Bundles))
	}
	if pack.Bundles[0].Error != "" {
		t.Fatalf("pass bundle error = %q, want empty", pack.Bundles[0].Error)
	}
	if !pack.Bundles[0].IntegrityPass || !pack.Bundles[0].ProfilePass {
		t.Fatalf("pass bundle summary = %+v", pack.Bundles[0])
	}
	if pack.Bundles[1].Error == "" {
		t.Fatal("expected error for missing bundle entry")
	}
}

func TestRunEvidencePackPassFixture(t *testing.T) {
	passPath := profileFixturePath(t, "rag_answer-pass.atb")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runEvidencePack([]string{passPath}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runEvidencePack() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	var pack evidencepack.Pack
	if err := json.Unmarshal(stdout.Bytes(), &pack); err != nil {
		t.Fatalf("unmarshal evidence pack: %v\noutput=%s", err, stdout.String())
	}
	if len(pack.Bundles) != 1 {
		t.Fatalf("bundles len = %d, want 1", len(pack.Bundles))
	}
	entry := pack.Bundles[0]
	if entry.ProfileID != "atb.profile.rag_answer" {
		t.Fatalf("profile_id = %q, want rag_answer profile", entry.ProfileID)
	}
	if !entry.IntegrityPass {
		t.Fatal("integrity_pass = false, want true")
	}
	if !entry.ProfilePass {
		t.Fatal("profile_pass = false, want true")
	}
	if entry.CASGrade == "" {
		t.Fatal("cas_grade is empty")
	}
	if entry.HeadHash == "" {
		t.Fatal("head_hash is empty")
	}
}

func TestRunEvidencePackMarkdownOutput(t *testing.T) {
	passPath := profileFixturePath(t, "rag_answer-pass.atb")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runEvidencePack([]string{passPath, "--output", "markdown"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runEvidencePack() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}
	rendered := stdout.String()
	if rendered == "" {
		t.Fatal("markdown output is empty")
	}
	for _, want := range []string{
		"# ATB evidence pack",
		"## Bundle:",
		"- Integrity: PASS",
		"## Notes for AI governance / AI Act Article 12",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("markdown output missing %q:\n%s", want, rendered)
		}
	}
}

func TestRunEvidencePackFailFixture(t *testing.T) {
	failPath := profileFixturePath(t, "rag_answer-fail.atb")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runEvidencePack([]string{failPath}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runEvidencePack() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	var pack evidencepack.Pack
	if err := json.Unmarshal(stdout.Bytes(), &pack); err != nil {
		t.Fatalf("unmarshal evidence pack: %v\noutput=%s", err, stdout.String())
	}
	entry := pack.Bundles[0]
	if !entry.IntegrityPass {
		t.Fatal("integrity_pass = false, want true for -fail fixture")
	}
	if entry.ProfilePass {
		t.Fatal("profile_pass = true, want false for -fail fixture")
	}
	if entry.ResidualRisk == nil {
		t.Fatal("expected residual_risk object for -fail fixture")
	}
	if len(entry.Exclusions) == 0 && len(entry.ResidualRisk.Drivers) == 0 {
		t.Fatalf("expected residual risk drivers or exclusions, got %+v", entry)
	}
}

func TestParseEvidencePackArgsWorkspace(t *testing.T) {
	cfg, err := parseEvidencePackArgs([]string{"--workspace", "/tmp/foo"})
	if err != nil {
		t.Fatalf("parseEvidencePackArgs() error = %v", err)
	}
	if cfg.WorkspaceDir != "/tmp/foo" {
		t.Fatalf("WorkspaceDir = %q, want /tmp/foo", cfg.WorkspaceDir)
	}
	if len(cfg.Paths) != 0 {
		t.Fatalf("Paths = %v, want empty", cfg.Paths)
	}
}

func TestParseEvidencePackArgsWorkspaceWithPaths(t *testing.T) {
	_, err := parseEvidencePackArgs([]string{"--workspace", "/tmp/foo", "bundle.atb"})
	if err == nil {
		t.Fatal("parseEvidencePackArgs() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "--workspace cannot be used with explicit bundle paths") {
		t.Fatalf("error = %v, want workspace/path conflict", err)
	}
}

func TestRunEvidencePackEmptyWorkspace(t *testing.T) {
	root := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runEvidencePack([]string{"--workspace", root}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runEvidencePack() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	var pack evidencepack.Pack
	if err := json.Unmarshal(stdout.Bytes(), &pack); err != nil {
		t.Fatalf("unmarshal evidence pack: %v\noutput=%s", err, stdout.String())
	}
	if len(pack.Bundles) != 0 {
		t.Fatalf("bundles len = %d, want 0", len(pack.Bundles))
	}
}

func TestRunEvidencePackEmptyWorkspaceMarkdown(t *testing.T) {
	root := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runEvidencePack([]string{"--workspace", root, "--output", "markdown"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runEvidencePack() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}
	rendered := stdout.String()
	if !strings.Contains(rendered, "# ATB evidence pack") {
		t.Fatalf("markdown output missing title:\n%s", rendered)
	}
	if strings.Contains(rendered, "## Bundle:") {
		t.Fatalf("markdown output should not list bundles:\n%s", rendered)
	}
}

func profileFixturePath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	return filepath.Join(root, "examples", "bundles", "profiles", name)
}
