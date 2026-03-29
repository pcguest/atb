package verify

import (
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

func TestPrivilegedToolAction_AllCriticalPresent(t *testing.T) {
	profile := &PrivilegedToolActionProfile{}
	result := profile.Evaluate(newPrivilegedToolActionBundle(t).Records)
	if !result.Pass {
		t.Fatalf("expected pass, got failures %+v", result.CriticalFailures)
	}
	if len(result.CriticalFailures) != 0 {
		t.Fatalf("expected no critical failures, got %d", len(result.CriticalFailures))
	}
}

func TestPrivilegedToolAction_MissingPrecommit(t *testing.T) {
	b := bundle.New()
	appendVerifyRecord(t, b, "ai.request.received", map[string]any{
		"request_id":    "req-1",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "approve-change",
	}, "2026-03-27T12:00:00Z")
	appendVerifyRecord(t, b, "ai.policy.decision", map[string]any{
		"policy_id":             "pol-1",
		"policy_version":        "2026-03",
		"decision":              "allow",
		"decision_reason_codes": []any{"ticket_present"},
		"subject_id_hash":       "subject-hash",
		"action_id":             "act-1",
	}, "2026-03-27T12:02:00Z")
	appendVerifyRecord(t, b, "ai.action.executed", map[string]any{
		"action_id":           "act-1",
		"execution_outcome":   "success",
		"tool_receipt_digest": "tool-digest",
	}, "2026-03-27T12:05:00Z")
	appendVerifyRecord(t, b, "ai.action.committed", map[string]any{
		"action_id":           "act-1",
		"commit_outcome":      "success",
		"sink_receipt_digest": "sink-digest",
	}, "2026-03-27T12:06:00Z")

	result := (&PrivilegedToolActionProfile{}).Evaluate(b.Records)
	if result.Pass {
		t.Fatalf("expected profile failure")
	}
	if !hasFailure(result.CriticalFailures, "missing_event", "ai.action.precommit") {
		t.Fatalf("expected missing precommit failure, got %+v", result.CriticalFailures)
	}
}

func TestPrivilegedToolAction_RelationViolation(t *testing.T) {
	b := newPrivilegedToolActionBundle(t)
	committedData := dataMap(b.Records[len(b.Records)-2].Event.Data)
	committedData["action_id"] = "act-2"

	result := (&PrivilegedToolActionProfile{}).Evaluate(b.Records)
	if !hasFailure(result.CriticalFailures, "relation_violation", "commit_requires_precommit") {
		t.Fatalf("expected relation violation, got %+v", result.CriticalFailures)
	}
}

func TestRAGAnswer_AllCriticalPresent(t *testing.T) {
	result := (&RAGAnswerProfile{}).Evaluate(newRAGAnswerBundle(t).Records)
	if !result.Pass {
		t.Fatalf("expected pass, got failures %+v", result.CriticalFailures)
	}
}

func TestRAGAnswer_MissingModelInvoked(t *testing.T) {
	b := bundle.New()
	appendVerifyRecord(t, b, "ai.request.received", map[string]any{
		"request_id":    "req-1",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "answer-query",
	}, "2026-03-27T12:00:00Z")
	appendVerifyRecord(t, b, "ai.model.output", map[string]any{
		"output_digest": "out-digest",
		"output_format": "text/plain",
	}, "2026-03-27T12:02:00Z")

	result := (&RAGAnswerProfile{}).Evaluate(b.Records)
	if result.Pass {
		t.Fatalf("expected failure when ai.model.invoked is missing")
	}
	if !hasFailure(result.CriticalFailures, "missing_event", "ai.model.invoked") {
		t.Fatalf("expected missing model invocation failure, got %+v", result.CriticalFailures)
	}
}

func TestDataMap_NonMapData(t *testing.T) {
	if got := dataMap("not-a-map"); got != nil {
		t.Fatalf("expected nil map, got %#v", got)
	}
}

func newRAGAnswerBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b := bundle.New()
	appendVerifyRecord(t, b, "ai.request.received", map[string]any{
		"request_id":    "req-1",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "answer-query",
	}, "2026-03-27T12:00:00Z")
	appendVerifyRecord(t, b, "ai.policy.decision", map[string]any{
		"policy_id":             "pol-1",
		"policy_version":        "2026-03",
		"decision":              "allow",
		"decision_reason_codes": []any{"public-corpus"},
	}, "2026-03-27T12:01:00Z")
	appendVerifyRecord(t, b, "ai.retrieval.executed", map[string]any{
		"retrieval_query_hash":     "query-hash",
		"retrieval_corpus_id":      "corpus-1",
		"retrieval_corpus_version": "v7",
		"top_k":                    5,
		"result_set_digest":        "results-digest",
	}, "2026-03-27T12:01:30Z")
	appendVerifyRecord(t, b, "ai.model.invoked", map[string]any{
		"model_provider":          "provider",
		"model_id":                "model-1",
		"model_parameters_digest": "params-digest",
		"prompt_digest":           "prompt-digest",
	}, "2026-03-27T12:02:00Z")
	appendVerifyRecord(t, b, "ai.model.output", map[string]any{
		"output_digest": "out-digest",
		"output_format": "text/plain",
	}, "2026-03-27T12:02:10Z")
	appendVerifyRecord(t, b, "ai.response.sent", map[string]any{
		"request_id":    "req-1",
		"output_digest": "out-digest",
	}, "2026-03-27T12:02:20Z")
	return b
}

func hasFailure(failures []CriticalFailure, kind string, contains string) bool {
	for _, failure := range failures {
		if failure.Kind == kind && strings.Contains(failure.Detail, contains) {
			return true
		}
	}
	return false
}
