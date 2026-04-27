// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

func TestRunVerify_JSONOutput(t *testing.T) {
	path := writeVerifyTestBundle(t, buildCLIPrivilegedToolActionBundle(t))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runVerify([]string{"--bundle", path, "--json"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	var report verifypkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal verify report: %v", err)
	}
	if !report.Integrity.ChainValid {
		t.Fatalf("expected integrity pass, got %+v", report.Integrity)
	}
}

func TestRunVerify_ExitCodeContract(t *testing.T) {
	tests := []struct {
		name      string
		setupPath func(t *testing.T) string
		wantCode  int
	}{
		{
			name: "success",
			setupPath: func(t *testing.T) string {
				b, err := bundle.New()
				if err != nil {
					t.Fatalf("bundle.New() error = %v", err)
				}
				return writeVerifyTestBundle(t, b)
			},
			wantCode: exitSuccess,
		},
		{
			name: "missing file is user error",
			setupPath: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "missing.atb")
			},
			wantCode: exitUserError,
		},
		{
			name: "bad json is user error",
			setupPath: func(t *testing.T) string {
				return writeRawVerifyFile(t, "{not json}\n")
			},
			wantCode: exitUserError,
		},
		{
			name: "valid json missing manifest is user error",
			setupPath: func(t *testing.T) string {
				return writeRawVerifyFile(t, "{}\n")
			},
			wantCode: exitUserError,
		},
		{
			name: "broken chain is tamper error",
			setupPath: func(t *testing.T) string {
				b := newTestBundle(t)
				appendTestBundleEvent(t, b, event.TypeDevSession, map[string]any{"event_id": "evt-1"})
				b.Records[1].Event.Type = "dev.session.tampered"
				return writeVerifyTestBundle(t, b)
			},
			wantCode: exitIntegrityFailure,
		},
		{
			name: "read failure is system error",
			setupPath: func(t *testing.T) string {
				if runtime.GOOS == "windows" {
					t.Skip("permission-bit read failure is not portable on Windows")
				}
				path := writeRawVerifyFile(t, "")
				if err := os.Chmod(path, 0000); err != nil {
					t.Fatalf("chmod unreadable bundle: %v", err)
				}
				t.Cleanup(func() { _ = os.Chmod(path, 0600) })
				return path
			},
			wantCode: exitSystemError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.setupPath(t)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := runVerify([]string{"--bundle", path}, &stdout, &stderr)
			if exitCode != tc.wantCode {
				t.Fatalf("runVerify() exit code = %d, want %d (stdout=%q stderr=%q)", exitCode, tc.wantCode, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunVerify_JSONOutput_AllBuiltInProfilesEmitCAS(t *testing.T) {
	for _, tc := range snapshotProfileCases() {
		tc := tc
		t.Run(tc.profileID, func(t *testing.T) {
			bundlePath := buildSnapshotBundle(t, tc.events)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := runVerify([]string{"--bundle", bundlePath, "--profile", tc.profileID, "--json"}, &stdout, &stderr)
			if exitCode != exitSuccess {
				t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
			}

			var report verifypkg.Report
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatalf("unmarshal verify report: %v", err)
			}
			if len(report.Profiles) != 1 {
				t.Fatalf("expected one evaluated profile, got %d", len(report.Profiles))
			}
			if report.Profiles[0].ProfileID != tc.profileID {
				t.Fatalf("ProfileID = %q, want %q", report.Profiles[0].ProfileID, tc.profileID)
			}
			if report.CAS == nil {
				t.Fatalf("expected CAS output for %q", tc.profileID)
			}
		})
	}
}

func TestRunVerify_ProfileNotFoundExitsUserError(t *testing.T) {
	path := writeVerifyTestBundle(t, buildCLIPrivilegedToolActionBundle(t))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runVerify([]string{"--bundle", path, "--profile=nonexistent"}, &stdout, &stderr)
	if exitCode != exitUserError {
		t.Fatalf("runVerify() exit code = %d, want %d (stderr=%q)", exitCode, exitUserError, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	want := "verify: profile \"nonexistent\" not found in config\n"
	if stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestRunVerify_FormatJSONOutput(t *testing.T) {
	path := writeVerifyTestBundle(t, buildCLIPrivilegedToolActionBundle(t))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runVerify([]string{"--bundle", path, "--format", "json"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	var report verifypkg.VerifierReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal verifier report: %v", err)
	}
	if report.ProfileID != "atb.profile.privileged_tool_action" {
		t.Fatalf("unexpected profile ID: got %q", report.ProfileID)
	}
	if !report.Pass {
		t.Fatalf("expected verifier report pass, got %+v", report)
	}
	if report.CASGrade == "" {
		t.Fatalf("expected CAS grade, got %+v", report)
	}
	if report.ResidualRisk == "" {
		t.Fatalf("expected residual risk, got %+v", report)
	}
}

func TestRunVerify_DryRunOutputsVerifierReport(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runVerify([]string{"--dry-run"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	var report verifypkg.VerifierReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal dry-run verifier report: %v", err)
	}
	if len(report.Notes) != 1 || report.Notes[0] != "dry-run: no evaluation performed" {
		t.Fatalf("unexpected dry-run notes: %+v", report.Notes)
	}
}

func TestVerifyCaptureRunRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("bundle.New() error = %v", err)
	}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("bundle.Save() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	captureExit := runCaptureRun([]string{
		"--bundle", bundlePath,
		"--snapshot", "after_capture",
		"--",
		"sh", "-c", "true",
	}, bytes.NewBuffer(nil), &stdout, &stderr)
	if captureExit != exitSuccess {
		t.Fatalf("runCaptureRun() exit code = %d, want %d (stdout=%q stderr=%q)", captureExit, exitSuccess, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	verifyExit := runVerify([]string{"--bundle", bundlePath}, &stdout, &stderr)
	if verifyExit != exitSuccess {
		t.Fatalf("runVerify() exit code = %d, want %d (stdout=%q stderr=%q)", verifyExit, exitSuccess, stdout.String(), stderr.String())
	}
}

func TestRunVerify_InvalidBundleSignatureFails(t *testing.T) {
	bundlePath := writeVerifyTestBundle(t, buildCLIPrivilegedToolActionBundle(t))
	keyPath := writeSignTestPrivateKey(t, t.TempDir())

	var signStdout bytes.Buffer
	var signStderr bytes.Buffer
	exitCode := runSign([]string{"--bundle", bundlePath, "--key", keyPath}, &signStdout, &signStderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected sign exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, signStderr.String())
	}

	signedBundle, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load signed bundle: %v", err)
	}
	for i := range signedBundle.Records {
		if signedBundle.Records[i].Event.Type != event.TypeBundleSignature {
			continue
		}
		data, ok := signedBundle.Records[i].Event.Data.(map[string]any)
		if !ok {
			t.Fatalf("signature record data type = %T, want map[string]any", signedBundle.Records[i].Event.Data)
		}
		data["signature"] = base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x7f}, ed25519.SignatureSize))
		break
	}
	rehashTestBundle(t, signedBundle)
	if err := signedBundle.Save(bundlePath); err != nil {
		t.Fatalf("save tampered signed bundle: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode = runVerify([]string{"--bundle", bundlePath, "--json"}, &stdout, &stderr)
	if exitCode != exitIntegrityFailure {
		t.Fatalf("unexpected verify exit code: got %d want %d (stderr=%q)", exitCode, exitIntegrityFailure, stderr.String())
	}

	var report verifypkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal verify report: %v", err)
	}
	if report.BundleSignature == nil {
		t.Fatalf("expected bundle signature result")
	}
	if report.BundleSignature.Verified {
		t.Fatalf("expected invalid bundle signature, got %+v", *report.BundleSignature)
	}
	if !strings.Contains(report.BundleSignature.Error, "signature mismatch") {
		t.Fatalf("expected signature mismatch error, got %+v", *report.BundleSignature)
	}
}

func TestRunVerify_IntegrityFail_ExitCode(t *testing.T) {
	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "dev.session", map[string]any{"event_id": "evt-1"})
	appendTestBundleEvent(t, b, "dev.session", map[string]any{"event_id": "evt-2"})
	b.Records[1].Event.Type = "dev.session.tampered"

	path := writeVerifyTestBundle(t, b)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runVerify([]string{"--bundle", path}, &stdout, &stderr)
	if exitCode != exitIntegrityFailure {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, exitIntegrityFailure)
	}
}

func TestRunVerify_WithSnapshotCheckDetectsTamperedPrefix(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if exitCode := runInit(nil, &stdout, &stderr); exitCode != exitSuccess {
		t.Fatalf("runInit() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if exitCode := runAppend([]string{"dev.session", `{"state":"before-snapshot","message":"original"}`}, &stdout, &stderr); exitCode != exitSuccess {
		t.Fatalf("runAppend() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if exitCode := runSnapshot([]string{"before-export"}, &stdout, &stderr); exitCode != exitSuccess {
		t.Fatalf("runSnapshot() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if exitCode := runAppend([]string{"dev.session", `{"state":"after-snapshot"}`}, &stdout, &stderr); exitCode != exitSuccess {
		t.Fatalf("runAppend() after snapshot exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	b, err := bundle.Load(bundle.DefaultPath())
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	fields, ok := b.Records[1].Event.Data.(map[string]any)
	if !ok {
		t.Fatalf("event data type = %T, want map[string]any", b.Records[1].Event.Data)
	}
	fields["message"] = "tampered-after-snapshot"
	rehashTestBundle(t, b)
	if err := b.Save(bundle.DefaultPath()); err != nil {
		t.Fatalf("save tampered bundle: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	exitCode := runVerify([]string{"--bundle", bundle.DefaultPath(), "--with-snapshot-check", "--json"}, &stdout, &stderr)
	if exitCode != exitIntegrityFailure {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitIntegrityFailure, stderr.String())
	}

	var report verifypkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal verify report: %v", err)
	}
	if !report.Integrity.ChainValid {
		t.Fatalf("expected chain integrity to remain valid, got %+v", report.Integrity)
	}
	if !containsVerifyNote(report.InformationalNotes, "snapshot_hash_mismatch at seq 2") {
		t.Fatalf("expected snapshot mismatch note, got %+v", report.InformationalNotes)
	}
	if report.ResidualRisk.Level != "Critical" {
		t.Fatalf("ResidualRisk.Level = %q, want %q", report.ResidualRisk.Level, "Critical")
	}

	stdout.Reset()
	stderr.Reset()
	exitCode = runVerify([]string{"--bundle", bundle.DefaultPath(), "--json"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("verify without --with-snapshot-check should exit 0, got %d (stderr=%q)", exitCode, stderr.String())
	}
}

func TestRunVerify_ProfileFail_ExitCode(t *testing.T) {
	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "ai.request.received", map[string]any{
		"request_id":    "req-1",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "approve-change",
	})
	appendTestBundleEvent(t, b, "ai.action.precommit", map[string]any{
		"action_id":                "act-1",
		"action_type":              "deploy_change",
		"action_parameters_digest": "params-digest",
		"target_resource_id":       "svc-1",
		"intended_effect":          "deploy build 42",
	})

	path := writeVerifyTestBundle(t, b)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runVerify([]string{"--bundle", path}, &stdout, &stderr)
	if exitCode != exitVerifyFailure {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, exitVerifyFailure)
	}
}

func TestRunVerify_ProfilePath_JSONOutput(t *testing.T) {
	bundlePath := writeVerifyTestBundle(t, buildCLIPrivilegedToolActionBundle(t))
	profilePath := filepath.Join(t.TempDir(), "profile.yaml")
	profileYAML := `
id: "org.example.my_profile"
version: 1
workflow_class: "custom_profile"
weights:
  EC: 0.20
  FC: 0.15
  RC: 0.20
  TC: 0.05
  SC: 0.10
  XC: 0.10
  AC: 0.10
  GC: 0.10
required:
  - type: "ai.action.precommit"
    fields:
      - "action_id"
    message: "Pre-commit record required"
    severity: "critical"
  - type: "ai.action.executed"
    fields:
      - "action_id"
    message: "Execution record required"
    severity: "critical"
optional:
  - type: "atb.bundle.anchor"
    fields: []
    message: "Anchor required"
    severity: "warning"
relations:
  - name: "precommit_to_execution"
    from: "ai.action.executed"
    to: "ai.action.precommit"
    field: "action_id"
    message: "Precommit must precede execution"
`
	if err := os.WriteFile(profilePath, []byte(profileYAML), 0600); err != nil {
		t.Fatalf("write profile fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runVerify([]string{"--bundle", bundlePath, "--profile", profilePath, "--json"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	var report verifypkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal verify report: %v", err)
	}
	if len(report.Profiles) != 1 {
		t.Fatalf("expected one evaluated profile, got %d", len(report.Profiles))
	}
	if report.Profiles[0].ProfileID != "org.example.my_profile" {
		t.Fatalf("unexpected profile ID: got %q want %q", report.Profiles[0].ProfileID, "org.example.my_profile")
	}
	if !report.Profiles[0].Pass {
		t.Fatalf("expected custom profile pass, got failures %+v", report.Profiles[0].CriticalFailures)
	}
}

func TestRunVerify_ProfilePath_JSONOutput_LegacyFallback(t *testing.T) {
	bundlePath := writeVerifyTestBundle(t, buildCLIPrivilegedToolActionBundle(t))
	profilePath := filepath.Join(t.TempDir(), "profile.yaml")
	profileYAML := `
id: "org.example.legacy_profile"
display_name: "My Legacy Profile"
description: "Custom obligations for our deployment"
detect:
  event_types:
    - "ai.action.precommit"
obligations:
  critical:
    - event_type: "ai.action.precommit"
      message: "Pre-commit record required"
    - event_type: "ai.action.executed"
      message: "Execution record required"
  required:
    - event_type: "atb.bundle.anchor"
      message: "Anchor required"
relations:
  - from: "ai.action.precommit"
    to: "ai.action.executed"
    message: "Precommit must precede execution"
`
	if err := os.WriteFile(profilePath, []byte(profileYAML), 0600); err != nil {
		t.Fatalf("write profile fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runVerify([]string{"--bundle", bundlePath, "--profile", profilePath, "--json"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	var report verifypkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal verify report: %v", err)
	}
	if len(report.Profiles) != 1 {
		t.Fatalf("expected one evaluated profile, got %d", len(report.Profiles))
	}
	if report.Profiles[0].ProfileID != "org.example.legacy_profile" {
		t.Fatalf("unexpected profile ID: got %q want %q", report.Profiles[0].ProfileID, "org.example.legacy_profile")
	}
	if !report.Profiles[0].Pass {
		t.Fatalf("expected legacy custom profile pass, got failures %+v", report.Profiles[0].CriticalFailures)
	}
}

func TestRunVerify_ProfileEqualsPath_FileNotFound(t *testing.T) {
	bundlePath := writeVerifyTestBundle(t, buildCLIPrivilegedToolActionBundle(t))
	profilePath := filepath.Join(t.TempDir(), "missing-profile.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runVerify([]string{"--bundle", bundlePath, "--profile=" + profilePath}, &stdout, &stderr)
	if exitCode != exitUserError {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitUserError, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "load profile") {
		t.Fatalf("expected load profile error, got %q", stderr.String())
	}
}

func TestRunVerify_DSLProfile_WithCAS_JSONOutput(t *testing.T) {
	bundlePath := writeVerifyTestBundle(t, buildCLIPrivilegedToolActionBundle(t))
	profilePath := filepath.Join(t.TempDir(), "custom-support.yaml")
	profileYAML := `
id: "org.example.dsl_custom"
description: "DSL v1 test profile"
required_events:
  - "ai.action.precommit"
  - "ai.action.executed"
warning_events:
  - "ai.action.committed"
cas_weights:
  EC: 0.50
  FC: 0.25
  RC: 0.00
  TC: 0.00
  SC: 0.00
  XC: 0.10
  AC: 0.10
  GC: 0.05
`
	if err := os.WriteFile(profilePath, []byte(strings.TrimSpace(profileYAML)+"\n"), 0600); err != nil {
		t.Fatalf("write DSL profile: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runVerify([]string{"--bundle", bundlePath, "--profile", profilePath, "--json"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	var report verifypkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal verify report: %v", err)
	}
	if len(report.Profiles) != 1 {
		t.Fatalf("expected one evaluated profile, got %d", len(report.Profiles))
	}
	if report.Profiles[0].ProfileID != "org.example.dsl_custom" {
		t.Fatalf("profile ID = %q, want %q", report.Profiles[0].ProfileID, "org.example.dsl_custom")
	}
	if !report.Profiles[0].Pass {
		t.Fatalf("expected DSL profile pass, got failures %+v", report.Profiles[0].CriticalFailures)
	}
	if report.CAS == nil {
		t.Fatalf("expected CAS to be present for profile with cas_weights")
	}
	if report.CAS.Overall <= 0 {
		t.Fatalf("CAS overall = %f, want > 0", report.CAS.Overall)
	}
}

func TestRunVerify_DSLProfile_NoCAS_JSONOutput(t *testing.T) {
	bundlePath := writeVerifyTestBundle(t, buildCLIPrivilegedToolActionBundle(t))
	profilePath := filepath.Join(t.TempDir(), "no-cas.yaml")
	profileYAML := `
id: "org.example.dsl_nocas"
required_events:
  - "ai.action.precommit"
  - "ai.action.executed"
`
	if err := os.WriteFile(profilePath, []byte(strings.TrimSpace(profileYAML)+"\n"), 0600); err != nil {
		t.Fatalf("write DSL profile: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runVerify([]string{"--bundle", bundlePath, "--profile", profilePath, "--json"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	var report verifypkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal verify report: %v", err)
	}
	if report.CAS != nil {
		t.Fatalf("expected CAS to be nil for profile without cas_weights, got %+v", report.CAS)
	}
}

func TestRunVerify_DSLProfile_InvalidWeights(t *testing.T) {
	bundlePath := writeVerifyTestBundle(t, buildCLIPrivilegedToolActionBundle(t))
	profilePath := filepath.Join(t.TempDir(), "bad-weights.yaml")
	profileYAML := `
id: "org.example.dsl_bad"
required_events:
  - "ai.action.precommit"
cas_weights:
  EC: 0.50
  FC: 0.30
`
	if err := os.WriteFile(profilePath, []byte(strings.TrimSpace(profileYAML)+"\n"), 0600); err != nil {
		t.Fatalf("write DSL profile: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runVerify([]string{"--bundle", bundlePath, "--profile", profilePath}, &stdout, &stderr)
	if exitCode == exitSuccess {
		t.Fatalf("expected non-zero exit for invalid DSL profile")
	}
	if !strings.Contains(stderr.String(), "cas_weights must sum to 1.0") {
		t.Fatalf("expected sum-to-1 error, got stderr=%q", stderr.String())
	}
}

func TestRunVerify_DSLProfile_CollidesWithBuiltIn(t *testing.T) {
	bundlePath := writeVerifyTestBundle(t, buildCLIPrivilegedToolActionBundle(t))
	profilePath := filepath.Join(t.TempDir(), "collision.yaml")
	profileYAML := `
id: "atb.profile.privileged_tool_action"
required_events:
  - "ai.action.precommit"
`
	if err := os.WriteFile(profilePath, []byte(strings.TrimSpace(profileYAML)+"\n"), 0600); err != nil {
		t.Fatalf("write DSL profile: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runVerify([]string{"--bundle", bundlePath, "--profile", profilePath}, &stdout, &stderr)
	if exitCode == exitSuccess {
		t.Fatalf("expected non-zero exit for ID collision with built-in")
	}
	if !strings.Contains(stderr.String(), "collides with a built-in profile") {
		t.Fatalf("expected collision error, got stderr=%q", stderr.String())
	}
}

func TestVerifyWithAnchorUsesCustomRoots(t *testing.T) {
	bundlePath, originalHash := writeAnchorTestBundle(t)
	fixture, _, err := buildCLISignedAnchorFixture(originalHash)
	if err != nil {
		t.Fatalf("buildCLISignedAnchorFixture: %v", err)
	}
	stubTSATransport(t, fixture)

	if _, err := runAnchor(anchorConfig{
		BundlePath: bundlePath,
		TSAURL:     "http://tsa.example.test",
	}); err != nil {
		t.Fatalf("runAnchor: %v", err)
	}

	rootsPath := filepath.Join(t.TempDir(), "tsa-roots.pem")
	if err := os.WriteFile(rootsPath, buildUnrelatedRootPEM(t), 0600); err != nil {
		t.Fatalf("write roots pem: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runVerify([]string{"--bundle", bundlePath, "--with-anchor", "--roots", rootsPath}, &stdout, &stderr)
	if exitCode == exitSuccess {
		t.Fatalf("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "certificate") {
		t.Fatalf("expected certificate failure, got %q", stderr.String())
	}
}

func buildCLIPrivilegedToolActionBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b := newTestBundle(t)
	appendTestBundleEventWithOptions(t, b, "ai.request.received", map[string]any{
		"request_id":    "req-1",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "approve-change",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:00:00Z"})
	appendTestBundleEventWithOptions(t, b, "ai.action.precommit", map[string]any{
		"action_id":                "act-1",
		"action_type":              "deploy_change",
		"action_parameters_digest": "params-digest",
		"target_resource_id":       "svc-1",
		"intended_effect":          "deploy build 42",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:01:00Z"})
	appendTestBundleEventWithOptions(t, b, "ai.policy.decision", map[string]any{
		"policy_id":             "pol-1",
		"policy_version":        "2026-03",
		"decision":              "allow",
		"decision_reason_codes": []any{"ticket_present"},
		"subject_id_hash":       "subject-hash",
		"action_id":             "act-1",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:02:00Z"})
	appendTestBundleEventWithOptions(t, b, "ai.action.executed", map[string]any{
		"action_id":           "act-1",
		"execution_outcome":   "success",
		"tool_receipt_digest": "tool-digest",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:05:00Z"})
	appendTestBundleEventWithOptions(t, b, "ai.human.approval", map[string]any{
		"approval_id":          "appr-1",
		"approver_id_hash":     "approver-hash",
		"approval_outcome":     "approve",
		"justification_digest": "just-digest",
		"action_id":            "act-1",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:05:30Z"})
	appendTestBundleEventWithOptions(t, b, "ai.action.committed", map[string]any{
		"action_id":           "act-1",
		"commit_outcome":      "success",
		"sink_receipt_digest": "sink-digest",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:06:00Z"})
	appendTestBundleEventWithOptions(
		t,
		b,
		bundle.AnchorEventType,
		`{"bundle_hash":"bundle-hash","tsr_hash":"tsr-hash","certified_time":"2026-03-27T12:06:30Z"}`,
		&bundle.AppendOptions{Timestamp: "2026-03-27T12:06:30Z"},
	)
	return b
}

// TestRunVerify_ByteCorruptionDetected writes a valid bundle to disk, corrupts
// a single byte in the event payload, and asserts that verify reports an
// integrity failure and identifies the corrupted record by sequence number.
func TestRunVerify_ByteCorruptionDetected(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if exitCode := runInit(nil, &stdout, &stderr); exitCode != exitSuccess {
		t.Fatalf("runInit() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if exitCode := runAppend([]string{"ai.request.received", `{"request_id":"req-1","actor_id_hash":"h1","purpose_tag":"test"}`}, &stdout, &stderr); exitCode != exitSuccess {
		t.Fatalf("runAppend() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	bundlePath := bundle.DefaultPath()
	content, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle file: %v", err)
	}

	// Flip one byte inside the event type string of the appended record (seq 1).
	// The stored hash will no longer match the recomputed hash of the modified event.
	corrupted := bytes.Replace(content, []byte(`"ai.request.received"`), []byte(`"Xi.request.received"`), 1)
	if bytes.Equal(corrupted, content) {
		t.Fatal("byte replacement had no effect; event type string not found in bundle file")
	}
	if err := os.WriteFile(bundlePath, corrupted, 0600); err != nil {
		t.Fatalf("write corrupted bundle: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	exitCode := runVerify([]string{"--bundle", bundlePath, "--json"}, &stdout, &stderr)
	if exitCode != exitIntegrityFailure {
		t.Fatalf("runVerify() exit code = %d, want %d (stdout=%q stderr=%q)", exitCode, exitIntegrityFailure, stdout.String(), stderr.String())
	}

	var report verifypkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal verify report: %v", err)
	}
	if report.Integrity.ChainValid {
		t.Fatalf("expected chain integrity failure, got %+v", report.Integrity)
	}
	// The error must identify the corrupted record by sequence number.
	if !strings.Contains(report.Integrity.Error, "seq 1") {
		t.Fatalf("expected error to identify seq 1, got %q", report.Integrity.Error)
	}
}

func containsVerifyNote(notes []string, wantSubstring string) bool {
	for _, note := range notes {
		if strings.Contains(note, wantSubstring) {
			return true
		}
	}
	return false
}

// TestRunVerify_ManifestOnlyBundle asserts that a bundle containing only a
// manifest record (atb bundle new, no appended events) reports chain_valid=true,
// has exactly one record, and matches no profiles. This documents the behaviour
// for empty-ish bundles that have a valid chain but no workflow events.
func TestRunVerify_ManifestOnlyBundle(t *testing.T) {
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("new bundle: %v", err)
	}
	// Only the manifest record — no events appended.
	if len(b.Records) != 1 {
		t.Fatalf("expected exactly 1 manifest record, got %d", len(b.Records))
	}
	path := writeVerifyTestBundle(t, b)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runVerify([]string{"--bundle", path, "--json"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	var report verifypkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal verify report: %v", err)
	}
	if !report.Integrity.ChainValid {
		t.Fatalf("expected chain_valid=true for manifest-only bundle, got %+v", report.Integrity)
	}
	if len(report.Profiles) != 0 {
		t.Fatalf("expected no matched profiles for manifest-only bundle, got %d", len(report.Profiles))
	}
}

func writeVerifyTestBundle(t testing.TB, b *bundle.Bundle) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "bundle.atb")
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}
	return path
}

func writeRawVerifyFile(t testing.TB, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "bundle.atb")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write raw verify file: %v", err)
	}
	return path
}

func buildUnrelatedRootPEM(t testing.TB) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate root key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(100),
		Subject:               pkix.Name{CommonName: "ATB Unrelated TSA Root"},
		NotBefore:             time.Date(2026, 3, 27, 0, 0, 0, 0, time.UTC),
		NotAfter:              time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create unrelated root cert: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
