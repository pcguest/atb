package verify_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	signpkg "github.com/pcguest/atb/internal/sign"
	"github.com/pcguest/atb/internal/verify"
)

const privilegedToolActionProfileID = "atb.profile.privileged_tool_action"

func TestTamperDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		tamper     bool
		expectLoad bool
	}{
		{
			name:       "valid bundle passes verification",
			expectLoad: true,
		},
		{
			name:   "tampered bundle fails verification",
			tamper: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bundlePath := writePrivilegedToolActionBundle(t)
			if tc.tamper {
				corruptBundleFile(t, bundlePath)
			}

			report, err := evaluateBundle(bundlePath)
			if tc.expectLoad {
				if err != nil {
					t.Fatalf("evaluate bundle: %v", err)
				}
				if report.CASScore <= 0 {
					t.Fatalf("CAS score = %f, want > 0", report.CASScore)
				}
				if len(report.Failures) != 0 {
					t.Fatalf("critical failures = %v, want none", report.Failures)
				}
				return
			}

			if err != nil {
				return
			}

			if len(report.Failures) == 0 {
				t.Fatalf("tampered bundle returned no error and no critical failures")
			}
			for _, failure := range report.Failures {
				kind := strings.ToLower(failure.Kind)
				if strings.Contains(kind, "tamper") || strings.Contains(kind, "hash") || strings.Contains(kind, "chain") {
					return
				}
			}
			t.Fatalf("tampered bundle failures = %v, want kind containing tamper, hash, or chain", report.Failures)
		})
	}
}

func writePrivilegedToolActionBundle(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	bundlePath := filepath.Join(root, bundle.BundleDir, bundle.BundleFile)

	b, err := bundle.New()
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	appendRecord(t, bundlePath, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-1",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "privileged_tool_action",
	}, "2026-03-27T12:00:00Z")
	appendRecord(t, bundlePath, event.TypeAILLMCall, map[string]any{
		"call_id":    "call-1",
		"model_id":   "gpt-5.4",
		"request_id": "req-1",
	}, "2026-03-27T12:00:30Z")
	appendRecord(t, bundlePath, event.TypeAIActionPrecommit, map[string]any{
		"action_id":                "act-1",
		"action_type":              "deploy_change",
		"action_parameters_digest": "params-digest",
		"target_resource_id":       "svc-1",
		"intended_effect":          "deploy build 42",
	}, "2026-03-27T12:01:00Z")

	policyFields := map[string]any{
		"policy_id":             "pol-1",
		"policy_version":        "2026-03",
		"decision":              "allow",
		"decision_reason_codes": []any{"ticket_present"},
		"subject_id_hash":       "subject-hash",
		"action_id":             "act-1",
	}
	signPolicyDecision(t, policyFields)
	appendRecord(t, bundlePath, event.TypeAIPolicyDecision, policyFields, "2026-03-27T12:02:00Z")
	appendRecord(t, bundlePath, event.TypeAIToolExec, map[string]any{
		"action_id":  "act-1",
		"tool_name":  "deployctl",
		"exit_code":  0,
		"request_id": "req-1",
	}, "2026-03-27T12:02:30Z")
	appendRecord(t, bundlePath, event.TypeAIActionExecuted, map[string]any{
		"action_id":           "act-1",
		"execution_outcome":   "success",
		"tool_receipt_digest": "tool-digest",
	}, "2026-03-27T12:03:00Z")
	appendRecord(t, bundlePath, event.TypeAIHumanApproval, map[string]any{
		"approval_id":          "appr-1",
		"approver_id_hash":     "approver-hash",
		"approval_outcome":     "approved",
		"justification_digest": "just-digest",
		"action_id":            "act-1",
	}, "2026-03-27T12:03:30Z")
	appendRecord(t, bundlePath, event.TypeAIActionCommitted, map[string]any{
		"action_id":           "act-1",
		"commit_outcome":      "success",
		"sink_receipt_digest": "sink-digest",
	}, "2026-03-27T12:04:00Z")

	return bundlePath
}

func appendRecord(t *testing.T, bundlePath string, eventType string, data interface{}, timestamp string) {
	t.Helper()

	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load bundle %q: %v", bundlePath, err)
	}
	if err := b.AppendWithOptions(eventType, data, &bundle.AppendOptions{Timestamp: timestamp}); err != nil {
		t.Fatalf("append %s: %v", eventType, err)
	}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle %q: %v", bundlePath, err)
	}
}

func signPolicyDecision(t *testing.T, fields map[string]any) {
	t.Helper()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate Ed25519 keypair: %v", err)
	}

	signature, err := signpkg.SignPolicyDecision(fields, privateKey)
	if err != nil {
		t.Fatalf("sign policy decision: %v", err)
	}

	fields[event.FieldPolicySignature] = signature
	fields[event.FieldPolicySignerPubKey] = base64.StdEncoding.EncodeToString(publicKey)
}

func corruptBundleFile(t *testing.T, bundlePath string) {
	t.Helper()

	data, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle %q: %v", bundlePath, err)
	}
	if len(data) == 0 {
		t.Fatalf("bundle %q is empty", bundlePath)
	}

	data[len(data)/2] = 0
	if err := os.WriteFile(bundlePath, data, 0o600); err != nil {
		t.Fatalf("write corrupted bundle %q: %v", bundlePath, err)
	}
}

func evaluateBundle(bundlePath string) (verify.VerifierReport, error) {
	b, err := bundle.Load(bundlePath)
	if err != nil {
		return verify.VerifierReport{}, err
	}

	profile := verify.ProfileByID(privilegedToolActionProfileID)
	report := verify.VerifyWithProfile(b, bundlePath, profile)
	return verify.ReportFromVerify(report), nil
}
