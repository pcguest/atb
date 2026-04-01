//go:build integration

package integration

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	signpkg "github.com/pcguest/atb/internal/sign"
	"github.com/pcguest/atb/internal/trust"
	"github.com/pcguest/atb/internal/verify"
)

const (
	profileIDPrivilegedToolAction = "atb.profile.privileged_tool_action"
	profileIDDataExport           = "atb.profile.data_export"
)

func TestGoldenPath_PrivilegedToolAction(t *testing.T) {
	bundlePath := newTempBundle(t)

	const actionID = "act-golden-001"

	appendEvent(t, bundlePath, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-golden-001",
		"actor_id_hash": "actor-golden-001",
		"purpose_tag":   "privileged_tool_action",
	})
	appendEvent(t, bundlePath, event.TypeAIPolicyDecision, map[string]any{
		"policy_id":             "policy-pta-001",
		"policy_version":        "2026-04",
		"decision":              "allow",
		"decision_reason_codes": []string{"ticket_present"},
		"subject_id_hash":       "subject-golden-001",
		"action_id":             actionID,
	})
	appendEvent(t, bundlePath, event.TypeAIActionPrecommit, map[string]any{
		"action_id":                actionID,
		"action_type":              "deploy_change",
		"action_parameters_digest": "sha256:params-golden-001",
		"target_resource_id":       "svc-prod-001",
		"intended_effect":          "deploy build 42",
	})
	appendEvent(t, bundlePath, event.TypeAIHumanApproval, map[string]any{
		"approval_id":          "approval-golden-001",
		"approver_id_hash":     "approver-golden-001",
		"approval_outcome":     "approved",
		"justification_digest": "sha256:justification-golden-001",
		"action_id":            actionID,
	})
	appendEvent(t, bundlePath, event.TypeAIActionExecuted, map[string]any{
		"action_id":           actionID,
		"execution_outcome":   "success",
		"tool_receipt_digest": "sha256:tool-receipt-golden-001",
	})
	appendEvent(t, bundlePath, event.TypeAIActionCommitted, map[string]any{
		"action_id":           actionID,
		"commit_outcome":      "success",
		"sink_receipt_digest": "sha256:sink-receipt-golden-001",
	})

	b := loadBundle(t, bundlePath)
	profile := mustProfile(t, profileIDPrivilegedToolAction)

	result := verify.VerifyWithProfile(b, bundlePath, profile)
	if result.BundlePath != bundlePath {
		t.Fatalf("unexpected bundle path: got %q want %q", result.BundlePath, bundlePath)
	}
	if !result.Integrity.ChainValid {
		t.Fatalf("expected chain_valid=true, got false with error %q", result.Integrity.Error)
	}
	if result.Integrity.Canonicalization != "rfc8785" {
		t.Fatalf("unexpected canonicalization: got %q want %q", result.Integrity.Canonicalization, "rfc8785")
	}
	if result.Integrity.HashAlgo != "sha256" {
		t.Fatalf("unexpected hash algorithm: got %q want %q", result.Integrity.HashAlgo, "sha256")
	}
	if result.Integrity.FirstSeq != 0 {
		t.Fatalf("unexpected first seq: got %d want %d", result.Integrity.FirstSeq, 0)
	}
	if result.Integrity.LastSeq != 6 {
		t.Fatalf("unexpected last seq: got %d want %d", result.Integrity.LastSeq, 6)
	}
	if result.Anchoring.AnchorRequired {
		t.Fatalf("expected anchor_required=false")
	}
	if result.Anchoring.AnchorPresent {
		t.Fatalf("expected anchor_present=false")
	}
	if result.Anchoring.TSAVerified {
		t.Fatalf("expected tsa_verified=false")
	}
	if result.Anchoring.AnchorHash != "" {
		t.Fatalf("expected empty anchor hash, got %q", result.Anchoring.AnchorHash)
	}
	if len(result.Profiles) != 1 {
		t.Fatalf("expected 1 profile result, got %d", len(result.Profiles))
	}

	profileResult := result.Profiles[0]
	if profileResult.ProfileID != profileIDPrivilegedToolAction {
		t.Fatalf("unexpected profile id: got %q want %q", profileResult.ProfileID, profileIDPrivilegedToolAction)
	}
	if profileResult.Version != 1 {
		t.Fatalf("unexpected profile version: got %d want %d", profileResult.Version, 1)
	}
	if profileResult.WorkflowClass != "privileged_tool_action" {
		t.Fatalf("unexpected workflow class: got %q want %q", profileResult.WorkflowClass, "privileged_tool_action")
	}
	if !profileResult.Pass {
		t.Fatalf("expected profile pass, got failures %+v", profileResult.CriticalFailures)
	}
	if len(profileResult.CriticalFailures) != 0 {
		t.Fatalf("expected no critical failures, got %+v", profileResult.CriticalFailures)
	}
	if len(profileResult.RequiredWarnings) != 1 {
		t.Fatalf("expected 1 required warning, got %v", profileResult.RequiredWarnings)
	}
	if profileResult.RequiredWarnings[0] != "ai.policy.decision: policy_signature absent" {
		t.Fatalf("unexpected required warning: got %q", profileResult.RequiredWarnings[0])
	}
	if result.CAS == nil {
		t.Fatalf("expected CAS result")
	}
	if grade := result.CAS.Grade; grade != "High" && grade != "Medium" {
		t.Fatalf("unexpected CAS grade: got %q want High or Medium", grade)
	}

	report := trust.BuildReport("", bundlePath, profileIDPrivilegedToolAction)
	if report.BundlePath != bundlePath {
		t.Fatalf("unexpected trust bundle path: got %q want %q", report.BundlePath, bundlePath)
	}
	if report.CAS == nil {
		t.Fatalf("expected CAS section")
	}
	if report.CAS.ProfileID != profileIDPrivilegedToolAction {
		t.Fatalf("unexpected CAS profile id: got %q want %q", report.CAS.ProfileID, profileIDPrivilegedToolAction)
	}
	if report.CAS.WorkflowClass != "privileged_tool_action" {
		t.Fatalf("unexpected CAS workflow class: got %q want %q", report.CAS.WorkflowClass, "privileged_tool_action")
	}
	if report.CAS.AnchorQuality.Label != "absent" {
		t.Fatalf("unexpected anchor quality label: got %q want %q", report.CAS.AnchorQuality.Label, "absent")
	}
	if report.ChainLength != 7 {
		t.Fatalf("unexpected chain length: got %d want %d", report.ChainLength, 7)
	}
	if report.HeadHash == "" {
		t.Fatalf("expected non-empty head hash")
	}
	if report.Status != trust.StatusPass && report.Status != trust.StatusWarn {
		t.Fatalf("unexpected report status: got %q want pass or warn", report.Status)
	}

	t.Run("signed_policy_decision", func(t *testing.T) {
		bundlePath := newTempBundle(t)

		const actionID = "act-golden-signed-001"

		appendEvent(t, bundlePath, event.TypeAIRequestReceived, map[string]any{
			"request_id":    "req-golden-signed-001",
			"actor_id_hash": "actor-golden-signed-001",
			"purpose_tag":   "privileged_tool_action",
		})

		policyFields := map[string]any{
			"policy_id":             "policy-pta-signed-001",
			"policy_version":        "2026-04",
			"decision":              "allow",
			"decision_reason_codes": []string{"ticket_present"},
			"subject_id_hash":       "subject-golden-signed-001",
			"action_id":             actionID,
		}
		signPolicyDecision(t, policyFields)
		appendEvent(t, bundlePath, event.TypeAIPolicyDecision, policyFields)

		appendEvent(t, bundlePath, event.TypeAIActionPrecommit, map[string]any{
			"action_id":                actionID,
			"action_type":              "deploy_change",
			"action_parameters_digest": "sha256:params-golden-signed-001",
			"target_resource_id":       "svc-prod-001",
			"intended_effect":          "deploy build 43",
		})
		appendEvent(t, bundlePath, event.TypeAIHumanApproval, map[string]any{
			"approval_id":          "approval-golden-signed-001",
			"approver_id_hash":     "approver-golden-signed-001",
			"approval_outcome":     "approved",
			"justification_digest": "sha256:justification-golden-signed-001",
			"action_id":            actionID,
		})
		appendEvent(t, bundlePath, event.TypeAIActionExecuted, map[string]any{
			"action_id":           actionID,
			"execution_outcome":   "success",
			"tool_receipt_digest": "sha256:tool-receipt-golden-signed-001",
		})
		appendEvent(t, bundlePath, event.TypeAIActionCommitted, map[string]any{
			"action_id":           actionID,
			"commit_outcome":      "success",
			"sink_receipt_digest": "sha256:sink-receipt-golden-signed-001",
		})

		b := loadBundle(t, bundlePath)
		profile := mustProfile(t, profileIDPrivilegedToolAction)

		result := verify.VerifyWithProfile(b, bundlePath, profile)
		if len(result.Profiles) != 1 {
			t.Fatalf("expected 1 profile result, got %d", len(result.Profiles))
		}
		if !containsString(result.Profiles[0].InformationalNotes, "ai.policy.decision: signature verified") {
			t.Fatalf("expected signature verified note, got %v", result.Profiles[0].InformationalNotes)
		}
	})
}

func TestGoldenPath_DataExport(t *testing.T) {
	bundlePath := newTempBundle(t)

	const actionID = "act-export-001"

	appendEvent(t, bundlePath, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-export-001",
		"actor_id_hash": "actor-export-001",
		"purpose_tag":   "data_export",
	})
	appendEvent(t, bundlePath, event.TypeAIPolicyDecision, map[string]any{
		"policy_id":             "policy-export-001",
		"policy_version":        "2026-04",
		"decision":              "allow",
		"decision_reason_codes": []string{"export_allowed"},
		"subject_id_hash":       "subject-export-001",
		"action_id":             actionID,
	})
	appendEvent(t, bundlePath, event.TypeAIActionPrecommit, map[string]any{
		"action_id":                actionID,
		"action_type":              "export_data",
		"action_parameters_digest": "sha256:params-export-001",
		"target_resource_id":       "customer-dataset-001",
		"intended_effect":          "export approved customer dataset",
	})
	appendEvent(t, bundlePath, event.TypeAIActionExecuted, map[string]any{
		"action_id":           actionID,
		"execution_outcome":   "success",
		"tool_receipt_digest": "sha256:tool-receipt-export-001",
	})
	appendEvent(t, bundlePath, event.TypeAIActionCommitted, map[string]any{
		"action_id":           actionID,
		"commit_outcome":      "success",
		"sink_receipt_digest": "sha256:sink-receipt-export-001",
	})

	b := loadBundle(t, bundlePath)
	profile := mustProfile(t, profileIDDataExport)

	result := verify.VerifyWithProfile(b, bundlePath, profile)
	if result.BundlePath != bundlePath {
		t.Fatalf("unexpected bundle path: got %q want %q", result.BundlePath, bundlePath)
	}
	if !result.Integrity.ChainValid {
		t.Fatalf("expected chain_valid=true, got false with error %q", result.Integrity.Error)
	}
	if result.Integrity.Canonicalization != "rfc8785" {
		t.Fatalf("unexpected canonicalization: got %q want %q", result.Integrity.Canonicalization, "rfc8785")
	}
	if result.Integrity.HashAlgo != "sha256" {
		t.Fatalf("unexpected hash algorithm: got %q want %q", result.Integrity.HashAlgo, "sha256")
	}
	if result.Integrity.FirstSeq != 0 {
		t.Fatalf("unexpected first seq: got %d want %d", result.Integrity.FirstSeq, 0)
	}
	if result.Integrity.LastSeq != 5 {
		t.Fatalf("unexpected last seq: got %d want %d", result.Integrity.LastSeq, 5)
	}
	if len(result.Profiles) != 1 {
		t.Fatalf("expected 1 profile result, got %d", len(result.Profiles))
	}

	profileResult := result.Profiles[0]
	if profileResult.ProfileID != profileIDDataExport {
		t.Fatalf("unexpected profile id: got %q want %q", profileResult.ProfileID, profileIDDataExport)
	}
	if profileResult.Version != 1 {
		t.Fatalf("unexpected profile version: got %d want %d", profileResult.Version, 1)
	}
	if profileResult.WorkflowClass != "data_export" {
		t.Fatalf("unexpected workflow class: got %q want %q", profileResult.WorkflowClass, "data_export")
	}
	if !profileResult.Pass {
		t.Fatalf("expected profile pass, got failures %+v", profileResult.CriticalFailures)
	}
	if len(profileResult.CriticalFailures) != 0 {
		t.Fatalf("expected no critical failures, got %+v", profileResult.CriticalFailures)
	}
	if len(profileResult.RequiredWarnings) != 2 {
		t.Fatalf("expected 2 required warnings, got %v", profileResult.RequiredWarnings)
	}
	if profileResult.RequiredWarnings[0] != "ai.human.approval required for data export workflows" {
		t.Fatalf("unexpected first required warning: got %q", profileResult.RequiredWarnings[0])
	}
	if profileResult.RequiredWarnings[1] != "ai.policy.decision: policy_signature absent" {
		t.Fatalf("unexpected second required warning: got %q", profileResult.RequiredWarnings[1])
	}
	if result.CAS != nil {
		t.Fatalf("expected verify CAS to be nil for %q, got %+v", profileIDDataExport, result.CAS)
	}

	report := trust.BuildReport("", bundlePath, profileIDDataExport)
	if report.BundlePath != bundlePath {
		t.Fatalf("unexpected trust bundle path: got %q want %q", report.BundlePath, bundlePath)
	}
	if report.CAS == nil {
		t.Fatalf("expected CAS section")
	}
	if report.CAS.ProfileID != profileIDDataExport {
		t.Fatalf("unexpected CAS profile id: got %q want %q", report.CAS.ProfileID, profileIDDataExport)
	}
	if report.CAS.WorkflowClass != "data_export" {
		t.Fatalf("unexpected CAS workflow class: got %q want %q", report.CAS.WorkflowClass, "data_export")
	}
	if report.CAS.AnchorQuality.Label != "absent" {
		t.Fatalf("unexpected anchor quality label: got %q want %q", report.CAS.AnchorQuality.Label, "absent")
	}
	if report.ChainLength != 6 {
		t.Fatalf("unexpected chain length: got %d want %d", report.ChainLength, 6)
	}
	if report.HeadHash == "" {
		t.Fatalf("expected non-empty head hash")
	}
}

func loadBundle(t *testing.T, bundlePath string) *bundle.Bundle {
	t.Helper()

	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load bundle %q: %v", bundlePath, err)
	}
	return b
}

func mustProfile(t *testing.T, id string) verify.Profile {
	t.Helper()

	profile := verify.ProfileByID(id)
	if profile == nil {
		t.Fatalf("profile %q not registered", id)
	}
	return profile
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

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
