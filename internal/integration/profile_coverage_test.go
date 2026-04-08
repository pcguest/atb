//go:build integration

package integration

import (
	"testing"

	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/trust"
	"github.com/pcguest/atb/internal/verify"
)

const (
	profileIDRAGAnswer            = "atb.profile.rag_answer"
	profileIDPolicyDecision       = "atb.profile.policy_decision"
	profileIDHumanOverride        = "atb.profile.human_override"
	profileIDBackgroundAutomation = "atb.profile.background_automation"
)

func TestIntegrationProfiles_RAGAnswer(t *testing.T) {
	bundlePath := newTempBundle(t)

	appendEvent(t, bundlePath, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-rag-001",
		"actor_id_hash": "actor-rag-001",
		"purpose_tag":   "rag_answer",
	})
	appendEvent(t, bundlePath, event.TypeAIPolicyDecision, map[string]any{
		"policy_id":             "policy-rag-001",
		"policy_version":        "2026-04",
		"decision":              "allow",
		"decision_reason_codes": []string{"approved"},
		"subject_id_hash":       "subject-rag-001",
		"action_id":             "rag-answer-001",
	})
	appendEvent(t, bundlePath, event.TypeAIRetrievalExecuted, map[string]any{
		"retrieval_query_hash":     "sha256:query-rag-001",
		"retrieval_corpus_id":      "corpus-001",
		"retrieval_corpus_version": "2026-04",
		"top_k":                    5,
		"result_set_digest":        "sha256:results-rag-001",
	})
	appendEvent(t, bundlePath, event.TypeAIModelInvoked, map[string]any{
		"model_provider":          "openai",
		"model_id":                "gpt-4o-mini",
		"model_parameters_digest": "sha256:model-params-rag-001",
		"prompt_digest":           "sha256:prompt-rag-001",
	})
	appendEvent(t, bundlePath, event.TypeAIModelOutput, map[string]any{
		"output_digest": "sha256:output-rag-001",
		"output_format": "text/plain",
	})
	appendEvent(t, bundlePath, event.TypeAIResponseSent, map[string]any{
		"request_id":    "req-rag-001",
		"output_digest": "sha256:output-rag-001",
	})

	b := loadBundle(t, bundlePath)
	profile := mustProfile(t, profileIDRAGAnswer)
	result := verify.VerifyWithProfile(b, bundlePath, profile)
	requireSingleProfileResult(t, result)
	if !result.Profiles[0].Pass {
		t.Fatalf("expected profile pass, got failures %+v", result.Profiles[0].CriticalFailures)
	}
	if result.CAS == nil {
		t.Fatalf("expected CAS result")
	}
	if result.CAS.SubScores["SC"] <= 0 {
		t.Fatalf("expected positive SC sub-score, got %.3f", result.CAS.SubScores["SC"])
	}

	report := trust.BuildReport("", bundlePath, profileIDRAGAnswer)
	if report.Status == trust.StatusFail {
		t.Fatalf("expected non-failing trust report status, got %q", report.Status)
	}
	if report.CAS == nil {
		t.Fatalf("expected trust report CAS section")
	}

	t.Run("response_request_mismatch", func(t *testing.T) {
		bundlePath := newTempBundle(t)

		appendEvent(t, bundlePath, event.TypeAIRequestReceived, map[string]any{
			"request_id":    "req-rag-mismatch-001",
			"actor_id_hash": "actor-rag-mismatch-001",
			"purpose_tag":   "rag_answer",
		})
		appendEvent(t, bundlePath, event.TypeAIModelInvoked, map[string]any{
			"model_provider":          "openai",
			"model_id":                "gpt-4o-mini",
			"model_parameters_digest": "sha256:model-params-rag-mismatch-001",
			"prompt_digest":           "sha256:prompt-rag-mismatch-001",
		})
		appendEvent(t, bundlePath, event.TypeAIModelOutput, map[string]any{
			"output_digest": "sha256:output-rag-mismatch-001",
			"output_format": "text/plain",
		})
		appendEvent(t, bundlePath, event.TypeAIResponseSent, map[string]any{
			"request_id":    "req-rag-other-001",
			"output_digest": "sha256:output-rag-mismatch-001",
		})

		b := loadBundle(t, bundlePath)
		result := verify.VerifyWithProfile(b, bundlePath, mustProfile(t, profileIDRAGAnswer))
		if result.Profiles[0].Pass {
			t.Fatalf("expected profile failure, got warnings %v", result.Profiles[0].RequiredWarnings)
		}
		if !hasCriticalFailure(result.Profiles[0].CriticalFailures, "relation_violation", "request_to_response: ai.response.sent request_id does not match ai.request.received") {
			t.Fatalf("expected relation failure, got %+v", result.Profiles[0].CriticalFailures)
		}

		report := trust.BuildReport("", bundlePath, profileIDRAGAnswer)
		if report.Status != trust.StatusFail {
			t.Fatalf("expected failing trust report status, got %q", report.Status)
		}
		if !hasTrustCheckDetail(report, "obligation_profile", "relation_violation: request_to_response: ai.response.sent request_id does not match ai.request.received") {
			t.Fatalf("expected trust report relation failure, got %+v", report.Categories)
		}
	})
}

func TestIntegrationProfiles_PolicyDecision(t *testing.T) {
	bundlePath := newTempBundle(t)

	appendEvent(t, bundlePath, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-policy-001",
		"actor_id_hash": "actor-policy-001",
		"purpose_tag":   "policy_decision",
	})
	appendEvent(t, bundlePath, event.TypeAIPolicyDecision, map[string]any{
		"policy_id":             "policy-001",
		"policy_version":        "2026-04",
		"decision":              "allow",
		"decision_reason_codes": []string{"approved"},
		"subject_id_hash":       "subject-policy-001",
		"action_id":             "act-policy-001",
	})

	b := loadBundle(t, bundlePath)
	profile := mustProfile(t, profileIDPolicyDecision)
	result := verify.VerifyWithProfile(b, bundlePath, profile)
	requireSingleProfileResult(t, result)
	if !result.Profiles[0].Pass {
		t.Fatalf("expected profile pass, got failures %+v", result.Profiles[0].CriticalFailures)
	}
	if !containsString(result.Profiles[0].RequiredWarnings, "ai.action.precommit recommended to bind policy to a pending action") {
		t.Fatalf("expected precommit recommendation warning, got %v", result.Profiles[0].RequiredWarnings)
	}
	if result.CAS != nil {
		t.Fatalf("expected CAS to be nil for %q, got %+v", profileIDPolicyDecision, result.CAS)
	}

	report := trust.BuildReport("", bundlePath, profileIDPolicyDecision)
	if report.Status == trust.StatusFail {
		t.Fatalf("expected non-failing trust report status, got %q", report.Status)
	}
	if report.CAS != nil {
		t.Fatalf("expected trust-report CAS to be nil for %q, got %+v", profileIDPolicyDecision, report.CAS)
	}

	t.Run("mismatched_precommit_relation", func(t *testing.T) {
		bundlePath := newTempBundle(t)

		appendEvent(t, bundlePath, event.TypeAIRequestReceived, map[string]any{
			"request_id":    "req-policy-mismatch-001",
			"actor_id_hash": "actor-policy-mismatch-001",
			"purpose_tag":   "policy_decision",
		})
		appendEvent(t, bundlePath, event.TypeAIPolicyDecision, map[string]any{
			"policy_id":             "policy-mismatch-001",
			"policy_version":        "2026-04",
			"decision":              "allow",
			"decision_reason_codes": []string{"approved"},
			"subject_id_hash":       "subject-policy-mismatch-001",
			"action_id":             "act-policy-mismatch-001",
		})
		appendEvent(t, bundlePath, event.TypeAIActionPrecommit, map[string]any{
			"action_id":                "act-policy-other-001",
			"action_type":              "approve_change",
			"action_parameters_digest": "sha256:params-policy-mismatch-001",
		})

		b := loadBundle(t, bundlePath)
		result := verify.VerifyWithProfile(b, bundlePath, mustProfile(t, profileIDPolicyDecision))
		if result.Profiles[0].Pass {
			t.Fatalf("expected profile failure, got warnings %v", result.Profiles[0].RequiredWarnings)
		}
		if !hasCriticalFailure(result.Profiles[0].CriticalFailures, "relation_violation", "policy_binds_action: ai.policy.decision action_id does not match ai.action.precommit") {
			t.Fatalf("expected relation failure, got %+v", result.Profiles[0].CriticalFailures)
		}

		report := trust.BuildReport("", bundlePath, profileIDPolicyDecision)
		if report.Status != trust.StatusFail {
			t.Fatalf("expected failing trust report status, got %q", report.Status)
		}
		if !hasTrustCheckDetail(report, "obligation_profile", "relation_violation: policy_binds_action: ai.policy.decision action_id does not match ai.action.precommit") {
			t.Fatalf("expected trust report relation failure, got %+v", report.Categories)
		}
	})
}

func TestIntegrationProfiles_HumanOverride(t *testing.T) {
	bundlePath := newTempBundle(t)

	appendEvent(t, bundlePath, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-override-001",
		"actor_id_hash": "actor-override-001",
		"purpose_tag":   "human_override",
	})
	appendEvent(t, bundlePath, event.TypeAIHumanApproval, map[string]any{
		"approval_id":          "approval-override-001",
		"approver_id_hash":     "approver-override-001",
		"approval_outcome":     "approved",
		"justification_digest": "sha256:justification-override-001",
		"action_id":            "act-override-001",
	})
	appendEvent(t, bundlePath, event.TypeAIActionPrecommit, map[string]any{
		"action_id":                "act-override-001",
		"action_type":              "override_action",
		"action_parameters_digest": "sha256:params-override-001",
		"target_resource_id":       "svc-prod-001",
		"intended_effect":          "run approved override",
	})
	appendEvent(t, bundlePath, event.TypeAIActionExecuted, map[string]any{
		"action_id":           "act-override-001",
		"execution_outcome":   "success",
		"tool_receipt_digest": "sha256:tool-receipt-override-001",
	})
	appendEvent(t, bundlePath, event.TypeAIActionCommitted, map[string]any{
		"action_id":           "act-override-001",
		"commit_outcome":      "success",
		"sink_receipt_digest": "sha256:sink-receipt-override-001",
	})

	b := loadBundle(t, bundlePath)
	profile := mustProfile(t, profileIDHumanOverride)
	result := verify.VerifyWithProfile(b, bundlePath, profile)
	requireSingleProfileResult(t, result)
	if !result.Profiles[0].Pass {
		t.Fatalf("expected profile pass, got failures %+v", result.Profiles[0].CriticalFailures)
	}
	if result.CAS != nil {
		t.Fatalf("expected CAS to be nil for %q, got %+v", profileIDHumanOverride, result.CAS)
	}

	report := trust.BuildReport("", bundlePath, profileIDHumanOverride)
	if report.Status == trust.StatusFail {
		t.Fatalf("expected non-failing trust report status, got %q", report.Status)
	}
	if report.CAS != nil {
		t.Fatalf("expected trust-report CAS to be nil for %q, got %+v", profileIDHumanOverride, report.CAS)
	}

	t.Run("executed_without_approval", func(t *testing.T) {
		bundlePath := newTempBundle(t)

		appendEvent(t, bundlePath, event.TypeAIRequestReceived, map[string]any{
			"request_id":    "req-override-missing-approval-001",
			"actor_id_hash": "actor-override-missing-approval-001",
			"purpose_tag":   "human_override",
		})
		appendEvent(t, bundlePath, event.TypeAIActionPrecommit, map[string]any{
			"action_id":                "act-override-missing-approval-001",
			"action_type":              "override_action",
			"action_parameters_digest": "sha256:params-override-missing-approval-001",
			"target_resource_id":       "svc-prod-001",
			"intended_effect":          "run approved override",
		})
		appendEvent(t, bundlePath, event.TypeAIActionExecuted, map[string]any{
			"action_id":           "act-override-missing-approval-001",
			"execution_outcome":   "success",
			"tool_receipt_digest": "sha256:tool-receipt-override-missing-approval-001",
		})

		b := loadBundle(t, bundlePath)
		result := verify.VerifyWithProfile(b, bundlePath, mustProfile(t, profileIDHumanOverride))
		if result.Profiles[0].Pass {
			t.Fatalf("expected profile failure, got warnings %v", result.Profiles[0].RequiredWarnings)
		}
		if !hasCriticalFailure(result.Profiles[0].CriticalFailures, "missing_event", "ai.human.approval missing required fields") {
			t.Fatalf("expected missing approval failure, got %+v", result.Profiles[0].CriticalFailures)
		}

		report := trust.BuildReport("", bundlePath, profileIDHumanOverride)
		if report.Status != trust.StatusFail {
			t.Fatalf("expected failing trust report status, got %q", report.Status)
		}
		if !hasTrustCheckDetail(report, "obligation_profile", "missing_event: ai.human.approval missing required fields") {
			t.Fatalf("expected trust report missing approval failure, got %+v", report.Categories)
		}
	})
}

func TestIntegrationProfiles_BackgroundAutomation(t *testing.T) {
	bundlePath := newTempBundle(t)

	appendEvent(t, bundlePath, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-background-001",
		"actor_id_hash": "actor-background-001",
		"purpose_tag":   "background_automation",
	})
	appendEvent(t, bundlePath, event.TypeAIJobScheduled, map[string]any{
		"job_id":       "job-background-001",
		"schedule_id":  "schedule-background-001",
		"job_type":     "nightly_sync",
		"trigger_type": "cron",
	})
	appendEvent(t, bundlePath, event.TypeAIJobStarted, map[string]any{
		"job_id":       "job-background-001",
		"worker_id":    "worker-background-001",
		"start_reason": "scheduled_trigger",
	})
	appendEvent(t, bundlePath, event.TypeAIJobStep, map[string]any{
		"job_id":       "job-background-001",
		"step_id":      "step-background-001",
		"step_name":    "sync_customers",
		"step_outcome": "success",
	})
	appendEvent(t, bundlePath, event.TypeAIJobCompleted, map[string]any{
		"job_id":             "job-background-001",
		"completion_outcome": "success",
		"result_digest":      "sha256:result-background-001",
	})

	b := loadBundle(t, bundlePath)
	profile := mustProfile(t, profileIDBackgroundAutomation)
	result := verify.VerifyWithProfile(b, bundlePath, profile)
	requireSingleProfileResult(t, result)
	if !result.Profiles[0].Pass {
		t.Fatalf("expected profile pass, got failures %+v", result.Profiles[0].CriticalFailures)
	}
	if result.CAS != nil {
		t.Fatalf("expected CAS to be nil for %q, got %+v", profileIDBackgroundAutomation, result.CAS)
	}

	report := trust.BuildReport("", bundlePath, profileIDBackgroundAutomation)
	if report.Status == trust.StatusFail {
		t.Fatalf("expected non-failing trust report status, got %q", report.Status)
	}
	if report.CAS != nil {
		t.Fatalf("expected trust-report CAS to be nil for %q, got %+v", profileIDBackgroundAutomation, report.CAS)
	}

	t.Run("missing_completion", func(t *testing.T) {
		bundlePath := newTempBundle(t)

		appendEvent(t, bundlePath, event.TypeAIRequestReceived, map[string]any{
			"request_id":    "req-background-missing-completion-001",
			"actor_id_hash": "actor-background-missing-completion-001",
			"purpose_tag":   "background_automation",
		})
		appendEvent(t, bundlePath, event.TypeAIJobScheduled, map[string]any{
			"job_id":       "job-background-missing-completion-001",
			"schedule_id":  "schedule-background-missing-completion-001",
			"job_type":     "nightly_sync",
			"trigger_type": "cron",
		})
		appendEvent(t, bundlePath, event.TypeAIJobStarted, map[string]any{
			"job_id":       "job-background-missing-completion-001",
			"worker_id":    "worker-background-missing-completion-001",
			"start_reason": "scheduled_trigger",
		})
		appendEvent(t, bundlePath, event.TypeAIJobStep, map[string]any{
			"job_id":       "job-background-missing-completion-001",
			"step_id":      "step-background-missing-completion-001",
			"step_name":    "sync_customers",
			"step_outcome": "success",
		})

		b := loadBundle(t, bundlePath)
		result := verify.VerifyWithProfile(b, bundlePath, mustProfile(t, profileIDBackgroundAutomation))
		if result.Profiles[0].Pass {
			t.Fatalf("expected profile failure, got warnings %v", result.Profiles[0].RequiredWarnings)
		}
		if !hasCriticalFailure(result.Profiles[0].CriticalFailures, "missing_event", "ai.job.completed missing required fields") {
			t.Fatalf("expected missing completion failure, got %+v", result.Profiles[0].CriticalFailures)
		}

		report := trust.BuildReport("", bundlePath, profileIDBackgroundAutomation)
		if report.Status != trust.StatusFail {
			t.Fatalf("expected failing trust report status, got %q", report.Status)
		}
		if !hasTrustCheckDetail(report, "obligation_profile", "missing_event: ai.job.completed missing required fields") {
			t.Fatalf("expected trust report missing completion failure, got %+v", report.Categories)
		}
	})
}
