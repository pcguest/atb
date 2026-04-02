//go:build integration

package integration

import (
	"testing"

	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/trust"
	"github.com/pcguest/atb/internal/verify"
)

type acpControlPlaneProfile struct {
	name             string
	profileID        string
	purposeTag       string
	actionType       string
	targetResourceID string
	intendedEffect   string
}

func TestACPControlPlaneGate(t *testing.T) {
	profiles := []acpControlPlaneProfile{
		{
			name:             "privileged_tool_action",
			profileID:        profileIDPrivilegedToolAction,
			purposeTag:       "privileged_tool_action",
			actionType:       "deploy_change",
			targetResourceID: "svc-prod-001",
			intendedEffect:   "deploy approved build",
		},
		{
			name:             "data_export",
			profileID:        profileIDDataExport,
			purposeTag:       "data_export",
			actionType:       "export_data",
			targetResourceID: "customer-dataset-001",
			intendedEffect:   "export approved customer dataset",
		},
	}

	for _, profile := range profiles {
		profile := profile
		t.Run(profile.name, func(t *testing.T) {
			happyBundlePath := newTempBundle(t)
			appendACPControlPlaneEvents(t, happyBundlePath, profile, "happy", true)

			happyVerifyResult := verifyACPControlPlaneBundle(t, happyBundlePath, profile.profileID)
			happyProfile := requireSingleProfileResult(t, happyVerifyResult)
			if !happyProfile.Pass {
				t.Fatalf("expected profile pass, got failures %+v", happyProfile.CriticalFailures)
			}
			if len(happyProfile.CriticalFailures) != 0 {
				t.Fatalf("expected no critical failures, got %+v", happyProfile.CriticalFailures)
			}

			happyTrustReport := trust.BuildReport("", happyBundlePath, profile.profileID)
			if happyTrustReport.Status == trust.StatusFail {
				t.Fatalf("expected non-failing trust report status, got %q", happyTrustReport.Status)
			}
			if happyTrustReport.Gate.Status == trust.StatusFail {
				t.Fatalf("expected non-failing trust gate status, got %q", happyTrustReport.Gate.Status)
			}
			if happyTrustReport.CAS == nil {
				t.Fatalf("expected trust report CAS section")
			}
			happyGateScore := requireSubScore(t, happyTrustReport.CAS.SubScores, "GC")
			if happyGateScore <= 0 {
				t.Fatalf("expected positive happy-path GC sub-score, got %.3f", happyGateScore)
			}

			switch profile.profileID {
			case profileIDPrivilegedToolAction:
				if happyVerifyResult.CAS == nil {
					t.Fatalf("expected verify CAS result for %q", profile.profileID)
				}
				if happyVerifyResult.CAS.Overall <= 0 {
					t.Fatalf("expected positive happy-path CAS overall, got %.3f", happyVerifyResult.CAS.Overall)
				}
				if requireSubScore(t, happyVerifyResult.CAS.SubScores, "GC") <= 0 {
					t.Fatalf("expected positive happy-path verify GC sub-score, got %.3f", happyVerifyResult.CAS.SubScores["GC"])
				}
			case profileIDDataExport:
				if happyVerifyResult.CAS != nil {
					t.Fatalf("expected verify CAS to remain nil for %q, got %+v", profile.profileID, happyVerifyResult.CAS)
				}
			default:
				t.Fatalf("unhandled profile id %q", profile.profileID)
			}

			t.Run("executed_without_precommit", func(t *testing.T) {
				violationBundlePath := newTempBundle(t)
				appendACPControlPlaneEvents(t, violationBundlePath, profile, "missing-precommit", false)

				violationVerifyResult := verifyACPControlPlaneBundle(t, violationBundlePath, profile.profileID)
				violationProfile := requireSingleProfileResult(t, violationVerifyResult)
				if violationProfile.Pass {
					t.Fatalf("expected profile failure, got warnings %v", violationProfile.RequiredWarnings)
				}
				if !hasCriticalFailure(violationProfile.CriticalFailures, "missing_event", "ai.action.precommit missing required fields") {
					t.Fatalf("expected missing precommit critical failure, got %+v", violationProfile.CriticalFailures)
				}

				violationTrustReport := trust.BuildReport("", violationBundlePath, profile.profileID)
				if violationTrustReport.Status != trust.StatusFail {
					t.Fatalf("expected failing trust report status, got %q", violationTrustReport.Status)
				}
				if violationTrustReport.Gate.Status != trust.StatusFail {
					t.Fatalf("expected failing trust gate status, got %q", violationTrustReport.Gate.Status)
				}
				if !hasTrustCheckDetail(violationTrustReport, "obligation_profile", "missing_event: ai.action.precommit missing required fields") {
					t.Fatalf("expected trust report missing precommit failure, got %+v", violationTrustReport.Categories)
				}
				if violationTrustReport.CAS == nil {
					t.Fatalf("expected trust report CAS section")
				}
				if violationTrustReport.CAS.Overall >= happyTrustReport.CAS.Overall {
					t.Fatalf(
						"expected ACP violation trust CAS overall %.3f to be lower than happy path %.3f",
						violationTrustReport.CAS.Overall,
						happyTrustReport.CAS.Overall,
					)
				}
				if violationGateScore := requireSubScore(t, violationTrustReport.CAS.SubScores, "GC"); violationGateScore >= happyGateScore {
					t.Fatalf(
						"expected ACP violation GC sub-score %.3f to be lower than happy path %.3f",
						violationGateScore,
						happyGateScore,
					)
				}

				switch profile.profileID {
				case profileIDPrivilegedToolAction:
					if violationVerifyResult.CAS == nil {
						t.Fatalf("expected verify CAS result for %q", profile.profileID)
					}
					if violationVerifyResult.CAS.Overall >= happyVerifyResult.CAS.Overall {
						t.Fatalf(
							"expected ACP violation verify CAS overall %.3f to be lower than happy path %.3f",
							violationVerifyResult.CAS.Overall,
							happyVerifyResult.CAS.Overall,
						)
					}
				case profileIDDataExport:
					if violationVerifyResult.CAS != nil {
						t.Fatalf("expected verify CAS to remain nil for %q, got %+v", profile.profileID, violationVerifyResult.CAS)
					}
				}
			})
		})
	}
}

func appendACPControlPlaneEvents(
	t *testing.T,
	bundlePath string,
	profile acpControlPlaneProfile,
	suffix string,
	includePrecommit bool,
) {
	t.Helper()

	actionID := "act-acp-" + profile.name + "-" + suffix

	appendEvent(t, bundlePath, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-acp-" + profile.name + "-" + suffix,
		"actor_id_hash": "actor-acp-" + profile.name + "-" + suffix,
		"purpose_tag":   profile.purposeTag,
	})
	if includePrecommit {
		appendEvent(t, bundlePath, event.TypeAIActionPrecommit, map[string]any{
			"action_id":                actionID,
			"action_type":              profile.actionType,
			"action_parameters_digest": "sha256:params-acp-" + profile.name + "-" + suffix,
			"target_resource_id":       profile.targetResourceID,
			"intended_effect":          profile.intendedEffect,
		})
	}
	appendEvent(t, bundlePath, event.TypeAIPolicyDecision, map[string]any{
		"policy_id":             "policy-acp-" + profile.name + "-" + suffix,
		"policy_version":        "2026-04",
		"decision":              "allow",
		"decision_reason_codes": []string{"approved"},
		"subject_id_hash":       "subject-acp-" + profile.name + "-" + suffix,
		"action_id":             actionID,
	})
	appendEvent(t, bundlePath, event.TypeAIActionExecuted, map[string]any{
		"action_id":           actionID,
		"execution_outcome":   "success",
		"tool_receipt_digest": "sha256:tool-receipt-acp-" + profile.name + "-" + suffix,
	})
	appendEvent(t, bundlePath, event.TypeAIHumanApproval, map[string]any{
		"approval_id":          "approval-acp-" + profile.name + "-" + suffix,
		"approver_id_hash":     "approver-acp-" + profile.name + "-" + suffix,
		"approval_outcome":     "approved",
		"justification_digest": "sha256:justification-acp-" + profile.name + "-" + suffix,
		"action_id":            actionID,
	})
	appendEvent(t, bundlePath, event.TypeAIActionCommitted, map[string]any{
		"action_id":           actionID,
		"commit_outcome":      "success",
		"sink_receipt_digest": "sha256:sink-receipt-acp-" + profile.name + "-" + suffix,
	})
}

func verifyACPControlPlaneBundle(t *testing.T, bundlePath string, profileID string) verify.Report {
	t.Helper()

	b := loadBundle(t, bundlePath)
	profile := mustProfile(t, profileID)
	return verify.VerifyWithProfile(b, bundlePath, profile)
}

func requireSingleProfileResult(t *testing.T, result verify.Report) verify.ProfileResult {
	t.Helper()

	if len(result.Profiles) != 1 {
		t.Fatalf("expected 1 profile result, got %d", len(result.Profiles))
	}
	return result.Profiles[0]
}

func requireSubScore(t *testing.T, subScores map[string]float64, key string) float64 {
	t.Helper()

	score, ok := subScores[key]
	if !ok {
		t.Fatalf("expected %s sub-score to be present", key)
	}
	return score
}
