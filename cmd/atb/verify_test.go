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

func writeVerifyTestBundle(t testing.TB, b *bundle.Bundle) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "bundle.atb")
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
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
