// Command generate_tenon_pilot writes tenon-pilot.atb in this directory.
//
// The bundle is a deterministic synthetic pilot session for Tenon onboarding:
// one privileged action is approved and completes, while a separate privileged
// tool call happens before approval and fails. It is safe to run locally and
// requires no provider credentials.
//
// Usage: go run ./examples/bundles/tenon-pilot/
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

const (
	pilotSessionID        = "sess-tenon-pilot-0001"
	pilotActorID          = "agent-support-bot"
	pilotApproverHash     = "sha256:supervisor-0007"
	pilotApprovedActionID = "act-store-credit-0001"
	pilotAnomalyActionID  = "act-delete-records-0001"
	pilotModel            = "gpt-4o-mini"
)

func digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	outDir := filepath.Join(root, "examples", "bundles", "tenon-pilot")
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		fatal(err)
	}

	b, err := buildTenonPilotBundle()
	if err != nil {
		fatal(err)
	}

	bundlePath := filepath.Join(outDir, "tenon-pilot.atb")
	if err := b.Save(bundlePath); err != nil {
		fatal(err)
	}
	fmt.Printf("✓ wrote %s (%d records)\n", bundlePath, len(b.Records))
}

func buildTenonPilotBundle() (*bundle.Bundle, error) {
	b, err := bundle.New()
	if err != nil {
		return nil, err
	}

	base := time.Date(2026, 7, 11, 9, 0, 0, 0, time.UTC)
	step := func(i int) string { return base.Add(time.Duration(i) * time.Second).Format(time.RFC3339) }
	add := func(eventType string, data map[string]any, ts string) {
		if err != nil {
			return
		}
		err = b.AppendWithOptions(eventType, data, &bundle.AppendOptions{Timestamp: ts})
	}

	add(event.TypeCaptureScope, map[string]any{
		"targets":          []any{"api.openai.com"},
		"capture_mode":     "digest",
		"redacted_headers": []any{"authorization", "x-api-key"},
		"out_of_scope":     "Synthetic fixture; no live provider traffic, shell commands, or destructive tools are executed.",
		"recorded_at":      step(0),
	}, step(0))

	add(event.TypeAIRequestReceived, map[string]any{
		"session_id":     pilotSessionID,
		"request_id":     "req-tenon-pilot-0001",
		"actor_id_hash":  "sha256:" + digest(pilotActorID),
		"purpose_tag":    "privileged_tool_action",
		"source_system":  "tenon-pilot-fixture",
		"workflow_label": "support-credit-review",
	}, step(1))
	add(event.TypeAIActionPrecommit, map[string]any{
		"session_id":                  pilotSessionID,
		"action_id":                   pilotApprovedActionID,
		"action_type":                 "issue_store_credit",
		"action_parameters_digest":    "sha256:" + digest(`{"account":"acct-0001","amount_usd":25}`),
		"target_resource_id":          "billing-account-0001",
		"intended_effect":             "issue_usd_25_store_credit",
		"synthetic_fixture":           true,
		"requires_human_approval":     true,
		"destructive_or_irreversible": false,
	}, step(2))
	add(event.TypeAIPolicyDecision, map[string]any{
		"session_id":             pilotSessionID,
		"policy_id":              "pol-store-credit",
		"policy_version":         "2026-07",
		"decision":               "allow",
		"decision_reason_codes":  []any{"below_refund_limit", "supervisor_approval_required"},
		"subject_id_hash":        "sha256:" + digest(pilotActorID),
		"action_id":              pilotApprovedActionID,
		"policy_evaluation_mode": "fixture",
	}, step(3))
	requestBody := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"Review acct-0001 and resolve the support ticket."}]}`
	responseBody := `{"tool_calls":[{"id":"call-delete","name":"delete_user_records"},{"id":"call-credit","name":"issue_store_credit"}]}`
	add(event.TypeLLMRequest, map[string]any{
		"session_id":  pilotSessionID,
		"host":        "api.openai.com",
		"method":      "POST",
		"path":        "/v1/chat/completions",
		"provider":    "openai",
		"model":       pilotModel,
		"body_sha256": "sha256:" + digest(requestBody),
		"body_bytes":  len(requestBody),
	}, step(5))
	add(event.TypeLLMResponse, map[string]any{
		"session_id":  pilotSessionID,
		"host":        "api.openai.com",
		"method":      "POST",
		"path":        "/v1/chat/completions",
		"status_code": 200,
		"provider":    "openai",
		"model":       pilotModel,
		"body_sha256": "sha256:" + digest(responseBody),
		"body_bytes":  len(responseBody),
		"usage":       map[string]int{"prompt_tokens": 31, "completion_tokens": 19, "total_tokens": 50},
	}, step(6))

	// The first tool call deliberately has no preceding atb.human.approval in
	// this session. The incident index must flag it as tool_without_approval.
	add(event.TypeToolCall, map[string]any{
		"session_id":         pilotSessionID,
		"tool_name":          "delete_user_records",
		"actor_id":           pilotActorID,
		"tool_input_digest":  "sha256:" + digest(`{"account":"acct-0001","scope":"all"}`),
		"privileged":         true,
		"synthetic_fixture":  true,
		"approved_action_id": pilotAnomalyActionID,
		"approval_expected":  true,
		"approval_recorded":  false,
	}, step(7))
	add(event.TypeAIActionError, map[string]any{
		"session_id":           pilotSessionID,
		"action_id":            pilotAnomalyActionID,
		"error_class":          "failed",
		"error_detail_digest":  "sha256:" + digest("fixture guard refused destructive delete"),
		"tool_name":            "delete_user_records",
		"synthetic_fixture":    true,
		"operator_consequence": "no records deleted",
	}, step(8))

	add(event.TypeHumanApproval, map[string]any{
		"session_id":         pilotSessionID,
		"approved_action_id": pilotApprovedActionID,
		"actor_id":           "supervisor-0007",
		"approver_id":        "supervisor-0007",
		"note":               "Approve goodwill store credit only; destructive delete remains disallowed.",
		"identity_evidence": map[string]any{
			"identity_provider": "https://login.example.test",
			"subject":           "supervisor-0007",
			"auth_context":      "mfa",
			"assertion_type":    "jwt",
			"assertion_digest":  "sha256:" + digest("fixture-supervisor-jwt"),
		},
	}, step(9))
	add(event.TypeToolCall, map[string]any{
		"session_id":         pilotSessionID,
		"tool_name":          "issue_store_credit",
		"actor_id":           pilotActorID,
		"tool_input_digest":  "sha256:" + digest(`{"account":"acct-0001","amount_usd":25}`),
		"tool_output_digest": "sha256:" + digest(`{"receipt":"credit-0001"}`),
		"approved_action_id": pilotApprovedActionID,
		"privileged":         true,
		"synthetic_fixture":  true,
	}, step(10))
	add(event.TypeAIActionExecuted, map[string]any{
		"session_id":          pilotSessionID,
		"action_id":           pilotApprovedActionID,
		"execution_outcome":   "success",
		"tool_receipt_digest": "sha256:" + digest(`{"receipt":"credit-0001"}`),
		"tool_name":           "issue_store_credit",
		"synthetic_fixture":   true,
	}, step(11))
	add(event.TypeAIHumanApproval, map[string]any{
		"session_id":            pilotSessionID,
		"approval_id":           "appr-store-credit-0001",
		"approver_id_hash":      pilotApproverHash,
		"approval_outcome":      "approved",
		"justification_digest":  "sha256:" + digest("customer eligible for goodwill credit"),
		"action_id":             pilotApprovedActionID,
		"approval_channel":      "fixture-review",
		"identity_provider":     "https://login.example.test",
		"identity_subject_hint": "supervisor-0007",
	}, step(12))
	add(event.TypeAIActionCommitted, map[string]any{
		"session_id":          pilotSessionID,
		"action_id":           pilotApprovedActionID,
		"commit_outcome":      "committed",
		"sink_receipt_digest": "sha256:" + digest(`{"ledger":"credit-0001"}`),
		"sink":                "billing-ledger-fixture",
		"synthetic_fixture":   true,
	}, step(13))
	add(event.TypeExchangeComplete, map[string]any{
		"session_id":       pilotSessionID,
		"exchange_id":      "ex-tenon-pilot-0001",
		"request_event_id": "req-tenon-pilot-0001",
		"actor_id":         pilotActorID,
		"completed_at":     step(14),
		"tool_calls_count": 2,
		"model":            pilotModel,
		"input_tokens":     31,
		"output_tokens":    19,
	}, step(14))
	add(event.TypeSessionClose, map[string]any{
		"session_id":     pilotSessionID,
		"actor_id":       pilotActorID,
		"model":          pilotModel,
		"exchange_count": 1,
		"total_tokens":   50,
		"closed_at":      step(15),
	}, step(15))

	if err != nil {
		return nil, err
	}
	return b, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
