package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
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

func TestRunVerify_IntegrityFail_ExitCode(t *testing.T) {
	b := bundle.New()
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

func TestRunVerify_ProfileFail_ExitCode(t *testing.T) {
	b := bundle.New()
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
display_name: "My Custom Profile"
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
	if report.Profiles[0].ProfileID != "org.example.my_profile" {
		t.Fatalf("unexpected profile ID: got %q want %q", report.Profiles[0].ProfileID, "org.example.my_profile")
	}
	if !report.Profiles[0].Pass {
		t.Fatalf("expected custom profile pass, got failures %+v", report.Profiles[0].CriticalFailures)
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

func buildCLIPrivilegedToolActionBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b := bundle.New()
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
	appendTestBundleEventWithOptions(t, b, "ai.human.approval", map[string]any{
		"approval_id":          "appr-1",
		"approver_id_hash":     "approver-hash",
		"approval_outcome":     "approve",
		"justification_digest": "just-digest",
		"action_id":            "act-1",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:03:00Z"})
	appendTestBundleEventWithOptions(t, b, "ai.action.executed", map[string]any{
		"action_id":           "act-1",
		"execution_outcome":   "success",
		"tool_receipt_digest": "tool-digest",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:05:00Z"})
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

func writeVerifyTestBundle(t testing.TB, b *bundle.Bundle) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "bundle.atb")
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}
	return path
}
