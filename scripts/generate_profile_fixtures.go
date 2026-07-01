// Command generate_profile_fixtures writes passing and failing .atb fixtures
// for all six built-in obligation profiles under examples/bundles/profiles/.
//
// Usage: go run ./scripts/generate_profile_fixtures.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/verify"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	outDir := filepath.Join(root, "examples", "bundles", "profiles")
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		fatal(err)
	}

	cases := []struct {
		name    string
		profile string
		pass    bool
		build   func(*bundle.Bundle)
	}{
		{
			name: "privileged_tool_action", profile: "atb.profile.privileged_tool_action", pass: true,
			build: func(b *bundle.Bundle) {
				appendEv(b, event.TypeAIRequestReceived, map[string]any{
					"request_id": "req-pta-pass", "actor_id_hash": "sha256:actor-pta", "purpose_tag": "privileged_tool_action",
				})
				appendEv(b, event.TypeAIActionPrecommit, map[string]any{
					"action_id": "act-pta-pass", "action_type": "deploy_change", "action_parameters_digest": "sha256:params-pta",
					"target_resource_id": "svc-prod", "intended_effect": "deploy approved build",
				})
				appendEv(b, event.TypeAIPolicyDecision, map[string]any{
					"policy_id": "pol-pta", "policy_version": "2026-04", "decision": "allow",
					"decision_reason_codes": []any{"approved"}, "subject_id_hash": "sha256:subject-pta", "action_id": "act-pta-pass",
				})
				appendEv(b, event.TypeAIActionExecuted, map[string]any{
					"action_id": "act-pta-pass", "execution_outcome": "success", "tool_receipt_digest": "sha256:receipt-pta",
				})
				appendEv(b, event.TypeAIHumanApproval, map[string]any{
					"approval_id": "appr-pta-pass", "approver_id_hash": "sha256:approver-pta", "approval_outcome": "approved",
					"justification_digest": "sha256:just-pta", "action_id": "act-pta-pass",
				})
				appendEv(b, event.TypeAIActionCommitted, map[string]any{
					"action_id": "act-pta-pass", "commit_outcome": "success", "sink_receipt_digest": "sha256:sink-pta",
				})
			},
		},
		{
			name: "privileged_tool_action", profile: "atb.profile.privileged_tool_action", pass: false,
			build: func(b *bundle.Bundle) {
				appendEv(b, event.TypeAIRequestReceived, map[string]any{
					"request_id": "req-pta-fail", "actor_id_hash": "sha256:actor-pta", "purpose_tag": "privileged_tool_action",
				})
				appendEv(b, event.TypeAIPolicyDecision, map[string]any{
					"policy_id": "pol-pta", "policy_version": "2026-04", "decision": "allow",
					"decision_reason_codes": []any{"approved"}, "subject_id_hash": "sha256:subject-pta", "action_id": "act-pta-fail",
				})
				appendEv(b, event.TypeAIActionExecuted, map[string]any{
					"action_id": "act-pta-fail", "execution_outcome": "success", "tool_receipt_digest": "sha256:receipt-pta",
				})
			},
		},
		{
			name: "rag_answer", profile: "atb.profile.rag_answer", pass: true,
			build: func(b *bundle.Bundle) {
				appendEv(b, event.TypeAIRequestReceived, map[string]any{
					"request_id": "req-rag-pass", "actor_id_hash": "sha256:actor-rag", "purpose_tag": "rag_answer",
				})
				appendEv(b, event.TypeAIModelInvoked, map[string]any{
					"model_provider": "openai", "model_id": "gpt-4o",
					"model_parameters_digest": "sha256:params-rag", "prompt_digest": "sha256:prompt-rag",
				})
				appendEv(b, event.TypeAIModelOutput, map[string]any{
					"output_digest": "sha256:output-rag", "output_format": "text/plain",
				})
				appendEv(b, event.TypeAIResponseSent, map[string]any{
					"request_id": "req-rag-pass", "output_digest": "sha256:output-rag",
				})
			},
		},
		{
			name: "rag_answer", profile: "atb.profile.rag_answer", pass: false,
			build: func(b *bundle.Bundle) {
				appendEv(b, event.TypeAIRequestReceived, map[string]any{
					"request_id": "req-rag-fail", "actor_id_hash": "sha256:actor-rag", "purpose_tag": "rag_answer",
				})
				appendEv(b, event.TypeAIModelOutput, map[string]any{
					"output_digest": "sha256:output-rag", "output_format": "text/plain",
				})
			},
		},
		{
			name: "data_export", profile: "atb.profile.data_export", pass: true,
			build: func(b *bundle.Bundle) {
				appendEv(b, event.TypeAIRequestReceived, map[string]any{
					"request_id": "req-export-pass", "actor_id_hash": "sha256:actor-export", "purpose_tag": "data_export",
				})
				appendEv(b, event.TypeAIPolicyDecision, map[string]any{
					"policy_id": "pol-export", "policy_version": "2026-04", "decision": "allow",
					"decision_reason_codes": []any{"export_allowed"}, "subject_id_hash": "sha256:subject-export", "action_id": "act-export-pass",
				})
				appendEv(b, event.TypeDataExportPrecommit, map[string]any{
					"action_id": "act-export-pass", "action_type": "export_data", "action_parameters_digest": "sha256:params-export",
					"target_resource_id": "dataset-1", "intended_effect": "export approved dataset",
				})
				appendEv(b, event.TypeDataExportExecuted, map[string]any{
					"action_id": "act-export-pass", "execution_outcome": "success", "tool_receipt_digest": "sha256:receipt-export",
				})
				appendEv(b, event.TypeAIHumanApproval, map[string]any{
					"approval_id": "appr-export-pass", "approver_id_hash": "sha256:approver-export", "approval_outcome": "approved",
					"justification_digest": "sha256:just-export", "action_id": "act-export-pass",
				})
			},
		},
		{
			name: "data_export", profile: "atb.profile.data_export", pass: false,
			build: func(b *bundle.Bundle) {
				appendEv(b, event.TypeAIRequestReceived, map[string]any{
					"request_id": "req-export-fail", "actor_id_hash": "sha256:actor-export", "purpose_tag": "data_export",
				})
				appendEv(b, event.TypeAIPolicyDecision, map[string]any{
					"policy_id": "pol-export", "policy_version": "2026-04", "decision": "allow",
					"decision_reason_codes": []any{"export_allowed"}, "subject_id_hash": "sha256:subject-export", "action_id": "act-export-fail",
				})
				appendEv(b, event.TypeDataExportExecuted, map[string]any{
					"action_id": "act-export-fail", "execution_outcome": "success", "tool_receipt_digest": "sha256:receipt-export",
				})
			},
		},
		{
			name: "policy_decision", profile: "atb.profile.policy_decision", pass: true,
			build: func(b *bundle.Bundle) {
				appendEv(b, event.TypeAIRequestReceived, map[string]any{
					"request_id": "req-policy-pass", "actor_id_hash": "sha256:actor-policy", "purpose_tag": "policy_decision",
				})
				appendEv(b, event.TypeAIActionPrecommit, map[string]any{
					"action_id": "act-policy-pass", "action_type": "approve_change", "action_parameters_digest": "sha256:params-policy",
				})
				appendEv(b, event.TypeAIPolicyDecision, map[string]any{
					"policy_id": "pol-policy", "policy_version": "2026-04", "decision": "allow",
					"decision_reason_codes": []any{"approved"}, "subject_id_hash": "sha256:subject-policy", "action_id": "act-policy-pass",
				})
			},
		},
		{
			name: "policy_decision", profile: "atb.profile.policy_decision", pass: false,
			build: func(b *bundle.Bundle) {
				appendEv(b, event.TypeAIRequestReceived, map[string]any{
					"request_id": "req-policy-fail", "actor_id_hash": "sha256:actor-policy", "purpose_tag": "policy_decision",
				})
				appendEv(b, event.TypeAIActionPrecommit, map[string]any{
					"action_id": "act-policy-fail", "action_type": "approve_change", "action_parameters_digest": "sha256:params-policy",
				})
			},
		},
		{
			name: "human_override", profile: "atb.profile.human_override", pass: true,
			build: func(b *bundle.Bundle) {
				appendEv(b, event.TypeAIRequestReceived, map[string]any{
					"request_id": "req-override-pass", "actor_id_hash": "sha256:actor-override", "purpose_tag": "human_override",
				})
				appendEv(b, event.TypeAIHumanApproval, map[string]any{
					"approval_id": "appr-override-pass", "approver_id_hash": "sha256:approver-override", "approval_outcome": "approved",
					"justification_digest": "sha256:just-override", "action_id": "act-override-pass",
				})
				appendEv(b, event.TypeAIActionPrecommit, map[string]any{
					"action_id": "act-override-pass", "action_type": "override_action", "action_parameters_digest": "sha256:params-override",
					"target_resource_id": "svc-1", "intended_effect": "run approved override",
				})
				appendEv(b, event.TypeAIActionExecuted, map[string]any{
					"action_id": "act-override-pass", "execution_outcome": "success", "tool_receipt_digest": "sha256:receipt-override",
				})
			},
		},
		{
			name: "human_override", profile: "atb.profile.human_override", pass: false,
			build: func(b *bundle.Bundle) {
				appendEv(b, event.TypeAIRequestReceived, map[string]any{
					"request_id": "req-override-fail", "actor_id_hash": "sha256:actor-override", "purpose_tag": "human_override",
				})
				appendEv(b, event.TypeAIActionPrecommit, map[string]any{
					"action_id": "act-override-fail", "action_type": "override_action", "action_parameters_digest": "sha256:params-override",
					"target_resource_id": "svc-1", "intended_effect": "run override",
				})
				appendEv(b, event.TypeAIActionExecuted, map[string]any{
					"action_id": "act-override-fail", "execution_outcome": "success", "tool_receipt_digest": "sha256:receipt-override",
				})
			},
		},
		{
			name: "background_automation", profile: "atb.profile.background_automation", pass: true,
			build: func(b *bundle.Bundle) {
				appendEv(b, event.TypeAIJobScheduled, map[string]any{
					"job_id": "job-bg-pass", "job_type": "nightly_sync", "trigger_source": "cron", "scheduled_by_id_hash": "sha256:scheduler-bg",
				})
				appendEv(b, event.TypeAIJobStarted, map[string]any{
					"job_id": "job-bg-pass", "worker_id_hash": "sha256:worker-bg", "started_at": "2026-05-24T12:00:00Z",
				})
				appendEv(b, event.TypeAIJobCompleted, map[string]any{
					"job_id": "job-bg-pass", "outcome": "success", "completion_reason": "completed",
				})
			},
		},
		{
			name: "background_automation", profile: "atb.profile.background_automation", pass: false,
			build: func(b *bundle.Bundle) {
				appendEv(b, event.TypeAIJobScheduled, map[string]any{
					"job_id": "job-bg-fail", "job_type": "nightly_sync", "trigger_source": "cron", "scheduled_by_id_hash": "sha256:scheduler-bg",
				})
				appendEv(b, event.TypeAIJobStarted, map[string]any{
					"job_id": "job-bg-fail", "worker_id_hash": "sha256:worker-bg", "started_at": "2026-05-24T12:00:00Z",
				})
			},
		},
	}

	for _, tc := range cases {
		b, err := bundle.New()
		if err != nil {
			fatal(err)
		}
		tc.build(b)
		suffix := "pass"
		if !tc.pass {
			suffix = "fail"
		}
		path := filepath.Join(outDir, fmt.Sprintf("%s-%s.atb", tc.name, suffix))
		if err := b.Save(path); err != nil {
			fatal(err)
		}
		profile := verify.ProfileByID(tc.profile)
		if profile == nil {
			fatal(fmt.Errorf("unknown profile %q", tc.profile))
		}
		loaded, err := bundle.Load(path)
		if err != nil {
			fatal(err)
		}
		result := verify.VerifyWithProfile(loaded, path, profile)
		if len(result.Profiles) != 1 {
			fatal(fmt.Errorf("%s: expected 1 profile result", path))
		}
		gotPass := result.Profiles[0].Pass && result.Integrity.ChainValid
		if gotPass != tc.pass {
			fatal(fmt.Errorf("%s: expected pass=%v got pass=%v failures=%+v", path, tc.pass, gotPass, result.Profiles[0].CriticalFailures))
		}
		fmt.Printf("✓ %s (%s pass=%v)\n", filepath.Base(path), tc.profile, tc.pass)
	}
	fmt.Printf("Wrote %d fixtures to %s\n", len(cases), outDir)
}

func appendEv(b *bundle.Bundle, eventType string, data map[string]any) {
	if err := b.Append(eventType, data); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
