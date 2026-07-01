// Command generate_demo_workflow writes demo-workflow.atb in this directory.
// Unsigned output; generate.sh signs and snapshots.
//
// Usage: go run ./examples/bundles/demo-workflow/
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/verify"
)

const (
	actionRefund = "act-refund-1042"
	actionCredit = "act-credit-1042" // #nosec G101 -- stable demo action identifier, not a credential.
	requestID    = "req-support-1042"
	actorHash    = "sha256:agent-support-01"
	approverHash = "sha256:supervisor-07"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	outDir := filepath.Join(root, "examples", "bundles", "demo-workflow")
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		fatal(err)
	}

	b, err := buildDemoBundle()
	if err != nil {
		fatal(err)
	}

	bundlePath := filepath.Join(outDir, "demo-workflow.atb")
	if err := b.Save(bundlePath); err != nil {
		fatal(err)
	}

	assertProfile(bundlePath, "atb.profile.policy_decision", true)
	assertProfile(bundlePath, "atb.profile.human_override", true)

	fmt.Printf("✓ demo-workflow.atb (%d user events)\n", countUserEvents(bundlePath))
	fmt.Printf("✓ wrote %s\n", bundlePath)
}

// buildDemoBundle constructs the deterministic, unsigned support-escalation
// demo bundle. It is shared by main (which then signs + snapshots via
// generate.sh) and by generate_test.go, so the demo's verify behaviour is
// covered by `go test ./...` without depending on the gitignored .atb artefact.
func buildDemoBundle() (*bundle.Bundle, error) {
	b, err := bundle.New()
	if err != nil {
		return nil, err
	}

	base := time.Date(2026, 5, 24, 14, 0, 0, 0, time.UTC)
	step := func(i int) string {
		return base.Add(time.Duration(i) * time.Second).Format(time.RFC3339)
	}
	add := func(eventType string, data map[string]any, ts string) {
		if err != nil {
			return
		}
		err = b.AppendWithOptions(eventType, data, &bundle.AppendOptions{Timestamp: ts})
	}

	// purpose_tag declares the bundle's obligation profile (workflow class) so
	// `atb verify` with no --profile auto-selects policy_decision. Using a
	// narrative label here ("support_escalation") matched no workflow class and
	// silently fell through to the privileged_tool_action heuristic, which
	// reported FAIL for this (correctly policy-denied + human-approved) workflow.
	add(event.TypeAIRequestReceived, map[string]any{
		"request_id": requestID, "actor_id_hash": actorHash, "purpose_tag": "policy_decision",
	}, step(0))
	add(event.TypeAIChainRun, map[string]any{
		"name": "support_triage", "phase": "start", "request_id": requestID,
	}, step(1))
	add(event.TypeRAGIndex, map[string]any{
		"index_hash": "sha256:kb-support-v3", "node_count": 128, "source_uri": "kb://support/playbooks",
	}, step(2))
	add(event.TypeRAGRetrieval, map[string]any{
		"node_id": "node-refund-policy", "page_start": 12, "page_end": 14, "query_digest": "sha256:q-refund-eligibility",
	}, step(3))
	add(event.TypeAIRetrievalExecuted, map[string]any{
		"retrieval_id": "ret-1042", "source": "support_kb", "hit_count": 3,
	}, step(4))
	add(event.TypeAIModelInvoked, map[string]any{
		"model_provider": "openai", "model_id": "gpt-4o-mini",
		"model_parameters_digest": "sha256:params-triage", "prompt_digest": "sha256:prompt-triage",
	}, step(5))
	add(event.TypeAIModelOutput, map[string]any{
		"output_digest": "sha256:analysis-refund-denied", "output_format": "text/plain",
	}, step(6))
	add(event.TypeAIToolExec, map[string]any{
		"request_id": requestID, "tool_name": "crm.lookup_customer",
		"context": map[string]any{"tool_name": "crm.lookup_customer", "customer_id": "cust-1042"},
	}, step(7))
	add(event.TypeAIChainRun, map[string]any{
		"name": "support_triage", "phase": "end", "request_id": requestID, "outcome": "analysis_complete",
	}, step(8))
	add(event.TypeAIActionPrecommit, map[string]any{
		"action_id": actionRefund, "action_type": "issue_refund", "action_parameters_digest": "sha256:params-refund-250",
		"target_resource_id": "billing-account-1042", "intended_effect": "refund_usd_250",
	}, step(9))
	add(event.TypeAIPolicyDecision, map[string]any{
		"policy_id": "pol-refund-tier", "policy_version": "2026-04", "decision": "deny",
		"decision_reason_codes": []any{"amount_exceeds_auto_limit", "requires_supervisor"},
		"subject_id_hash":       actorHash, "action_id": actionRefund,
	}, step(10))
	add(event.TypeAIHumanApproval, map[string]any{
		"approval_id": "appr-credit-1042", "approver_id_hash": approverHash, "approval_outcome": "approved",
		"justification_digest": "sha256:just-store-credit-policy", "action_id": actionCredit,
	}, step(11))
	add(event.TypeAIActionPrecommit, map[string]any{
		"action_id": actionCredit, "action_type": "issue_store_credit", "action_parameters_digest": "sha256:params-credit-250",
		"target_resource_id": "billing-account-1042", "intended_effect": "store_credit_usd_250",
	}, step(12))
	add(event.TypeAIActionExecuted, map[string]any{
		"action_id": actionCredit, "execution_outcome": "success", "tool_receipt_digest": "sha256:receipt-billing-credit",
	}, step(13))
	add(event.TypeAIActionCommitted, map[string]any{
		"action_id": actionCredit, "commit_outcome": "committed", "sink_receipt_digest": "sha256:sink-ledger-credit",
	}, step(14))
	add(event.TypeCorroborationExternal, map[string]any{
		"source": "crm-gateway", "reference_id": "crm-ticket-1042", "digest": "sha256:crm-receipt-body",
		"retrieved_at": "2026-05-24T14:30:00Z", "adapter": "http-gateway",
	}, step(15))
	add(event.TypeAIToolExec, map[string]any{
		"request_id": requestID, "tool_name": "notify.customer",
		"context": map[string]any{"tool_name": "notify.customer", "channel": "email"},
	}, step(16))
	add(event.TypeAIResponseSent, map[string]any{
		"request_id": requestID, "output_digest": "sha256:response-credit-offered", "output_format": "text/plain",
	}, step(17))
	add(event.TypeAIChainRun, map[string]any{
		"name": "support_resolution", "phase": "end", "request_id": requestID,
	}, step(18))

	if err != nil {
		return nil, err
	}
	return b, nil
}

func assertProfile(path, profileID string, wantPass bool) {
	b, err := bundle.Load(path)
	if err != nil {
		fatal(err)
	}
	profile := verify.ProfileByID(profileID)
	if profile == nil {
		fatal(fmt.Errorf("unknown profile %q", profileID))
	}
	result := verify.VerifyWithProfile(b, path, profile)
	if len(result.Profiles) != 1 {
		fatal(fmt.Errorf("%s: expected 1 profile result", profileID))
	}
	got := result.Profiles[0].Pass && result.Integrity.ChainValid
	if got != wantPass {
		fatal(fmt.Errorf("%s: expected pass=%v got pass=%v failures=%+v", profileID, wantPass, got, result.Profiles[0].CriticalFailures))
	}
}

func countUserEvents(path string) int {
	b, err := bundle.Load(path)
	if err != nil {
		fatal(err)
	}
	n := 0
	for _, rec := range b.Records {
		t := rec.Event.Type
		if t != event.TypeBundleManifest && t != event.TypeBundleSignature && t != event.TypeSnapshot {
			n++
		}
	}
	return n
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
