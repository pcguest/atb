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
	b := newVerifyTestBundle(t)
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
	b := newVerifyTestBundle(t)
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

func TestAnchorSubScoreScaling(t *testing.T) {
	records := newPrivilegedToolActionBundle(t).Records

	cases := []struct {
		name   string
		input  AnchorVerifyResult
		wantXC float64
		wantAC float64
	}{
		{name: "digest_only", input: AnchorDigestOnly, wantXC: 0.5, wantAC: 0.0},
		{name: "present_bad_data", input: AnchorPresentBadData, wantXC: 0.1, wantAC: 0.0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := privilegedToolActionSubScores(records, tc.input)
			if got["XC"] != tc.wantXC {
				t.Fatalf("XC = %.1f, want %.1f", got["XC"], tc.wantXC)
			}
			if got["AC"] != tc.wantAC {
				t.Fatalf("AC = %.1f, want %.1f", got["AC"], tc.wantAC)
			}
		})
	}
}

func newRAGAnswerBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b := newVerifyTestBundle(t)
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

func TestProfileWeightSums(t *testing.T) {
	expected := map[string]struct{}{
		"atb.profile.privileged_tool_action": {},
		"atb.profile.rag_answer":             {},
		"atb.profile.policy_decision":        {},
		"atb.profile.human_override":         {},
		"atb.profile.background_automation":  {},
		"atb.profile.data_export":            {},
	}
	seen := map[string]struct{}{}

	for _, profile := range AllProfiles() {
		profile := profile
		t.Run(profile.ID(), func(t *testing.T) {
			sum := 0.0
			for _, weight := range profile.DefaultWeights() {
				sum += weight
			}
			diff := sum - 1.0
			if diff < 0 {
				diff = -diff
			}
			if diff > 1e-9 {
				t.Fatalf("profile %q weights sum to %.12f", profile.ID(), sum)
			}
			seen[profile.ID()] = struct{}{}
		})
	}

	for id := range expected {
		if _, ok := seen[id]; !ok {
			t.Fatalf("expected profile %q in registry", id)
		}
	}
}

func TestProfileIdentity(t *testing.T) {
	cases := []struct {
		id           string
		wantVersion  int
		wantWorkflow string
	}{
		{"atb.profile.privileged_tool_action", 1, "privileged_tool_action"},
		{"atb.profile.rag_answer", 1, "rag_answer"},
		{"atb.profile.policy_decision", 1, "policy_decision"},
		{"atb.profile.human_override", 1, "human_override"},
		{"atb.profile.background_automation", 1, "background_automation"},
		{"atb.profile.data_export", 1, "data_export"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.id, func(t *testing.T) {
			p := ProfileByID(tc.id)
			if p == nil {
				t.Fatalf("ProfileByID(%q) returned nil", tc.id)
			}
			if got := p.ID(); got != tc.id {
				t.Fatalf("ID() = %q, want %q", got, tc.id)
			}
			if got := p.Version(); got != tc.wantVersion {
				t.Fatalf("Version() = %d, want %d", got, tc.wantVersion)
			}
			if got := p.WorkflowClass(); got != tc.wantWorkflow {
				t.Fatalf("WorkflowClass() = %q, want %q", got, tc.wantWorkflow)
			}
			if len(p.BlindSpots()) == 0 {
				t.Fatalf("BlindSpots() returned no entries for %q", tc.id)
			}
		})
	}
}

func TestProfileEvaluateEmptyBundle(t *testing.T) {
	expected := map[string]struct{}{
		"atb.profile.privileged_tool_action": {},
		"atb.profile.rag_answer":             {},
		"atb.profile.policy_decision":        {},
		"atb.profile.human_override":         {},
		"atb.profile.background_automation":  {},
		"atb.profile.data_export":            {},
	}
	seen := map[string]struct{}{}

	for _, profile := range AllProfiles() {
		profile := profile
		t.Run(profile.ID(), func(t *testing.T) {
			result := profile.Evaluate(nil)
			if len(result.CriticalFailures) == 0 {
				t.Fatalf("expected critical failures for empty bundle")
			}
			for i, failure := range result.CriticalFailures {
				if failure.Kind == "" {
					t.Fatalf("CriticalFailures[%d].Kind is empty", i)
				}
				if failure.Detail == "" {
					t.Fatalf("CriticalFailures[%d].Detail is empty", i)
				}
			}
			for i, warning := range result.RequiredWarnings {
				if warning == "" {
					t.Fatalf("RequiredWarnings[%d] is empty", i)
				}
			}
			for i, note := range result.InformationalNotes {
				if note == "" {
					t.Fatalf("InformationalNotes[%d] is empty", i)
				}
			}
			seen[profile.ID()] = struct{}{}
		})
	}

	for id := range expected {
		if _, ok := seen[id]; !ok {
			t.Fatalf("expected profile %q in registry", id)
		}
	}
}

func TestProfileByIDUnknown(t *testing.T) {
	if got := ProfileByID("atb.profile.nonexistent"); got != nil {
		t.Fatalf("ProfileByID(nonexistent) = %#v, want nil", got)
	}
}
