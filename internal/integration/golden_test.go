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
	appendEvent(t, bundlePath, event.TypeAIActionExecuted, map[string]any{
		"action_id":           actionID,
		"execution_outcome":   "success",
		"tool_receipt_digest": "sha256:tool-receipt-golden-001",
	})
	appendEvent(t, bundlePath, event.TypeAIHumanApproval, map[string]any{
		"approval_id":          "approval-golden-001",
		"approver_id_hash":     "approver-golden-001",
		"approval_outcome":     "approved",
		"justification_digest": "sha256:justification-golden-001",
		"action_id":            actionID,
	})
	appendEvent(t, bundlePath, event.TypeAIActionCommitted, map[string]any{
		"action_id":           actionID,
		"commit_outcome":      "success",
		"sink_receipt_digest": "sha256:sink-receipt-golden-001",
	})

	b := loadBundle(t, bundlePath)
	profile := mustProfile(t, profileIDPrivilegedToolAction)

	unsignedResult := verify.VerifyWithProfile(b, bundlePath, profile)
	if unsignedResult.BundlePath != bundlePath {
		t.Fatalf("unexpected bundle path: got %q want %q", unsignedResult.BundlePath, bundlePath)
	}
	if !unsignedResult.Integrity.ChainValid {
		t.Fatalf("expected chain_valid=true, got false with error %q", unsignedResult.Integrity.Error)
	}
	if unsignedResult.Integrity.Canonicalization != "rfc8785" {
		t.Fatalf("unexpected canonicalization: got %q want %q", unsignedResult.Integrity.Canonicalization, "rfc8785")
	}
	if unsignedResult.Integrity.HashAlgo != "sha256" {
		t.Fatalf("unexpected hash algorithm: got %q want %q", unsignedResult.Integrity.HashAlgo, "sha256")
	}
	if unsignedResult.Integrity.FirstSeq != 0 {
		t.Fatalf("unexpected first seq: got %d want %d", unsignedResult.Integrity.FirstSeq, 0)
	}
	if unsignedResult.Integrity.LastSeq != 6 {
		t.Fatalf("unexpected last seq: got %d want %d", unsignedResult.Integrity.LastSeq, 6)
	}
	if unsignedResult.Anchoring.AnchorRequired {
		t.Fatalf("expected anchor_required=false")
	}
	if unsignedResult.Anchoring.AnchorPresent {
		t.Fatalf("expected anchor_present=false")
	}
	if unsignedResult.Anchoring.TSAVerified {
		t.Fatalf("expected tsa_verified=false")
	}
	if unsignedResult.Anchoring.AnchorHash != "" {
		t.Fatalf("expected empty anchor hash, got %q", unsignedResult.Anchoring.AnchorHash)
	}
	if len(unsignedResult.Profiles) != 1 {
		t.Fatalf("expected 1 profile result, got %d", len(unsignedResult.Profiles))
	}

	profileResult := unsignedResult.Profiles[0]
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
	if unsignedResult.CAS == nil {
		t.Fatalf("expected CAS result")
	}
	if grade := unsignedResult.CAS.Grade; grade != "High" && grade != "Medium" {
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
		appendEvent(t, bundlePath, event.TypeAIActionExecuted, map[string]any{
			"action_id":           actionID,
			"execution_outcome":   "success",
			"tool_receipt_digest": "sha256:tool-receipt-golden-signed-001",
		})
		appendEvent(t, bundlePath, event.TypeAIHumanApproval, map[string]any{
			"approval_id":          "approval-golden-signed-001",
			"approver_id_hash":     "approver-golden-signed-001",
			"approval_outcome":     "approved",
			"justification_digest": "sha256:justification-golden-signed-001",
			"action_id":            actionID,
		})
		appendEvent(t, bundlePath, event.TypeAIActionCommitted, map[string]any{
			"action_id":           actionID,
			"commit_outcome":      "success",
			"sink_receipt_digest": "sha256:sink-receipt-golden-signed-001",
		})

		b := loadBundle(t, bundlePath)
		profile := mustProfile(t, profileIDPrivilegedToolAction)

		signedResult := verify.VerifyWithProfile(b, bundlePath, profile)
		if len(signedResult.Profiles) != 1 {
			t.Fatalf("expected 1 profile result, got %d", len(signedResult.Profiles))
		}
		if !containsString(signedResult.Profiles[0].InformationalNotes, "ai.policy.decision: signature verified") {
			t.Fatalf("expected signature verified note, got %v", signedResult.Profiles[0].InformationalNotes)
		}
		if signedResult.CAS == nil {
			t.Fatalf("expected CAS result")
		}
		if signedResult.CAS.SubScores["SC"] != unsignedResult.CAS.SubScores["SC"] {
			t.Fatalf("expected policy signatures to leave SC unchanged, got %.3f want %.3f", signedResult.CAS.SubScores["SC"], unsignedResult.CAS.SubScores["SC"])
		}
	})

	t.Run("missing_approval_after_execution", func(t *testing.T) {
		bundlePath := newTempBundle(t)

		const actionID = "act-golden-missing-approval-001"

		appendEvent(t, bundlePath, event.TypeAIRequestReceived, map[string]any{
			"request_id":    "req-golden-missing-approval-001",
			"actor_id_hash": "actor-golden-missing-approval-001",
			"purpose_tag":   "privileged_tool_action",
		})
		appendEvent(t, bundlePath, event.TypeAIPolicyDecision, map[string]any{
			"policy_id":             "policy-pta-missing-approval-001",
			"policy_version":        "2026-04",
			"decision":              "allow",
			"decision_reason_codes": []string{"ticket_present"},
			"subject_id_hash":       "subject-golden-missing-approval-001",
			"action_id":             actionID,
		})
		appendEvent(t, bundlePath, event.TypeAIActionPrecommit, map[string]any{
			"action_id":                actionID,
			"action_type":              "deploy_change",
			"action_parameters_digest": "sha256:params-golden-missing-approval-001",
			"target_resource_id":       "svc-prod-001",
			"intended_effect":          "deploy build 44",
		})
		appendEvent(t, bundlePath, event.TypeAIActionExecuted, map[string]any{
			"action_id":           actionID,
			"execution_outcome":   "success",
			"tool_receipt_digest": "sha256:tool-receipt-golden-missing-approval-001",
		})
		appendEvent(t, bundlePath, event.TypeAIActionCommitted, map[string]any{
			"action_id":           actionID,
			"commit_outcome":      "success",
			"sink_receipt_digest": "sha256:sink-receipt-golden-missing-approval-001",
		})

		b := loadBundle(t, bundlePath)
		profile := mustProfile(t, profileIDPrivilegedToolAction)

		result := verify.VerifyWithProfile(b, bundlePath, profile)
		if len(result.Profiles) != 1 {
			t.Fatalf("expected 1 profile result, got %d", len(result.Profiles))
		}
		if result.Profiles[0].Pass {
			t.Fatalf("expected profile failure, got warnings %v", result.Profiles[0].RequiredWarnings)
		}
		if !hasCriticalFailure(result.Profiles[0].CriticalFailures, "missing_event", "ai.human.approval required when actions execute") {
			t.Fatalf("expected missing approval critical failure, got %+v", result.Profiles[0].CriticalFailures)
		}
		if result.CAS == nil {
			t.Fatalf("expected CAS result for privileged profile")
		}

		report := trust.BuildReport("", bundlePath, profileIDPrivilegedToolAction)
		if report.Status != trust.StatusFail {
			t.Fatalf("expected failing trust report status, got %q", report.Status)
		}
		if report.Gate.Status != trust.StatusFail {
			t.Fatalf("expected failing trust gate status, got %q", report.Gate.Status)
		}
		if !hasTrustCheckDetail(report, "obligation_profile", "missing_event: ai.human.approval required when actions execute") {
			t.Fatalf("expected trust report missing approval failure, got %+v", report.Categories)
		}
		if report.CAS == nil {
			t.Fatalf("expected trust report CAS section")
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
	appendEvent(t, bundlePath, event.TypeAIHumanApproval, map[string]any{
		"approval_id":          "approval-export-001",
		"approver_id_hash":     "approver-export-001",
		"approval_outcome":     "approved",
		"justification_digest": "sha256:justification-export-001",
		"action_id":            actionID,
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
	if result.Integrity.LastSeq != 6 {
		t.Fatalf("unexpected last seq: got %d want %d", result.Integrity.LastSeq, 6)
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
	if len(profileResult.RequiredWarnings) != 1 {
		t.Fatalf("expected 1 required warning, got %v", profileResult.RequiredWarnings)
	}
	if profileResult.RequiredWarnings[0] != "ai.policy.decision: policy_signature absent" {
		t.Fatalf("unexpected required warning: got %q", profileResult.RequiredWarnings[0])
	}
	if result.CAS == nil {
		t.Fatalf("expected verify CAS for %q", profileIDDataExport)
	}
	if requireSubScore(t, result.CAS.SubScores, "SC") <= 0 {
		t.Fatalf("expected positive SC sub-score, got %.3f", result.CAS.SubScores["SC"])
	}

	report := trust.BuildReport("", bundlePath, profileIDDataExport)
	if report.BundlePath != bundlePath {
		t.Fatalf("unexpected trust bundle path: got %q want %q", report.BundlePath, bundlePath)
	}
	if report.CAS == nil {
		t.Fatalf("expected trust-report CAS for %q", profileIDDataExport)
	}
	if report.ChainLength != 7 {
		t.Fatalf("unexpected chain length: got %d want %d", report.ChainLength, 7)
	}
	if report.HeadHash == "" {
		t.Fatalf("expected non-empty head hash")
	}

	t.Run("missing_approval_after_execution", func(t *testing.T) {
		bundlePath := newTempBundle(t)

		const actionID = "act-export-missing-approval-001"

		appendEvent(t, bundlePath, event.TypeAIRequestReceived, map[string]any{
			"request_id":    "req-export-missing-approval-001",
			"actor_id_hash": "actor-export-missing-approval-001",
			"purpose_tag":   "data_export",
		})
		appendEvent(t, bundlePath, event.TypeAIPolicyDecision, map[string]any{
			"policy_id":             "policy-export-missing-approval-001",
			"policy_version":        "2026-04",
			"decision":              "allow",
			"decision_reason_codes": []string{"export_allowed"},
			"subject_id_hash":       "subject-export-missing-approval-001",
			"action_id":             actionID,
		})
		appendEvent(t, bundlePath, event.TypeAIActionPrecommit, map[string]any{
			"action_id":                actionID,
			"action_type":              "export_data",
			"action_parameters_digest": "sha256:params-export-missing-approval-001",
			"target_resource_id":       "customer-dataset-001",
			"intended_effect":          "export approved customer dataset",
		})
		appendEvent(t, bundlePath, event.TypeAIActionExecuted, map[string]any{
			"action_id":           actionID,
			"execution_outcome":   "success",
			"tool_receipt_digest": "sha256:tool-receipt-export-missing-approval-001",
		})
		appendEvent(t, bundlePath, event.TypeAIActionCommitted, map[string]any{
			"action_id":           actionID,
			"commit_outcome":      "success",
			"sink_receipt_digest": "sha256:sink-receipt-export-missing-approval-001",
		})

		b := loadBundle(t, bundlePath)
		profile := mustProfile(t, profileIDDataExport)

		result := verify.VerifyWithProfile(b, bundlePath, profile)
		if len(result.Profiles) != 1 {
			t.Fatalf("expected 1 profile result, got %d", len(result.Profiles))
		}
		if result.Profiles[0].Pass {
			t.Fatalf("expected profile failure, got warnings %v", result.Profiles[0].RequiredWarnings)
		}
		if !hasCriticalFailure(result.Profiles[0].CriticalFailures, "missing_event", "ai.human.approval required when data exports execute") {
			t.Fatalf("expected missing approval critical failure, got %+v", result.Profiles[0].CriticalFailures)
		}
		if result.CAS == nil {
			t.Fatalf("expected verify CAS for %q on violation path", profileIDDataExport)
		}

		violationReport := trust.BuildReport("", bundlePath, profileIDDataExport)
		if violationReport.Status != trust.StatusFail {
			t.Fatalf("expected failing trust report status, got %q", violationReport.Status)
		}
		if violationReport.Gate.Status != trust.StatusFail {
			t.Fatalf("expected failing trust gate status, got %q", violationReport.Gate.Status)
		}
		if !hasTrustCheckDetail(violationReport, "obligation_profile", "missing_event: ai.human.approval required when data exports execute") {
			t.Fatalf("expected trust report missing approval failure, got %+v", violationReport.Categories)
		}
		if violationReport.CAS == nil {
			t.Fatalf("expected trust-report CAS for %q on violation path", profileIDDataExport)
		}
	})
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

func hasCriticalFailure(failures []verify.CriticalFailure, kind string, detail string) bool {
	for _, failure := range failures {
		if failure.Kind == kind && failure.Detail == detail {
			return true
		}
	}
	return false
}

func hasTrustCheckDetail(report trust.Report, categoryKey string, detail string) bool {
	for _, category := range report.Categories {
		if category.Key != categoryKey {
			continue
		}
		for _, check := range category.Checks {
			if check.Details == detail {
				return true
			}
		}
	}
	return false
}
