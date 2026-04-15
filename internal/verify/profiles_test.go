package verify

import (
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	profiledsl "github.com/pcguest/atb/internal/profiles"
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

// stubSchemaProfile implements Profile and profileWithSchema for testing.
type stubSchemaProfile struct {
	schema profiledsl.ProfileSchema
}

func (s *stubSchemaProfile) ID() string            { return s.schema.ID }
func (s *stubSchemaProfile) Version() int          { return s.schema.Version }
func (s *stubSchemaProfile) WorkflowClass() string { return s.schema.WorkflowClass }
func (s *stubSchemaProfile) BlindSpots() []string  { return nil }
func (s *stubSchemaProfile) DefaultWeights() map[string]float64 {
	return map[string]float64{"EC": 0, "FC": 0, "RC": 0, "TC": 0, "SC": 0, "XC": 0, "AC": 0, "GC": 0}
}
func (s *stubSchemaProfile) Evaluate(_ []bundle.Record) ProfileResult { return ProfileResult{} }
func (s *stubSchemaProfile) profileSchema() profiledsl.ProfileSchema  { return s.schema }

func dslSchema(supportsCAS bool) profiledsl.ProfileSchema {
	return profiledsl.ProfileSchema{
		ID:            "org.example.stub",
		Version:       1,
		WorkflowClass: "stub",
		SupportsCAS:   supportsCAS,
		Weights: map[string]float64{
			"EC": 0, "FC": 0, "RC": 0, "TC": 0,
			"SC": 0, "XC": 0, "AC": 0, "GC": 0,
		},
		Required: []profiledsl.EventRule{
			{Type: "ai.request.received", Fields: []string{"request_id"}, Severity: "critical"},
			{Type: "ai.action.precommit", Fields: []string{"action_id"}, Severity: "critical"},
		},
	}
}

func makeRecord(eventType string, data map[string]any) bundle.Record {
	b, err := bundle.New()
	if err != nil {
		panic(err)
	}
	if err := b.Append(eventType, data); err != nil {
		panic(err)
	}
	return b.Records[len(b.Records)-1]
}

func TestGenericSchemaSubScores_EventsPresent(t *testing.T) {
	schema := dslSchema(true)
	records := []bundle.Record{
		makeRecord("ai.request.received", map[string]any{"request_id": "req-1"}),
		makeRecord("ai.action.precommit", map[string]any{"action_id": "act-1"}),
	}
	scores := genericSchemaSubScores(schema, records, AnchorAbsent)
	if scores["EC"] != 1.0 {
		t.Errorf("EC = %f, want 1.0 (all required events present)", scores["EC"])
	}
	if scores["FC"] != 1.0 {
		t.Errorf("FC = %f, want 1.0 (all required fields present)", scores["FC"])
	}
	// RC/TC/SC/GC must be zero — not computed generically.
	for _, k := range []string{"RC", "TC", "SC", "GC"} {
		if scores[k] != 0.0 {
			t.Errorf("%s = %f, want 0.0", k, scores[k])
		}
	}
}

func TestGenericSchemaSubScores_EventsMissing(t *testing.T) {
	schema := dslSchema(true)
	scores := genericSchemaSubScores(schema, nil, AnchorAbsent)
	if scores["EC"] != 0.0 {
		t.Errorf("EC = %f, want 0.0 (no events)", scores["EC"])
	}
	if scores["FC"] != 0.0 {
		t.Errorf("FC = %f, want 0.0 (no events)", scores["FC"])
	}
}

func TestGenericSchemaSubScores_AnchorScores(t *testing.T) {
	schema := dslSchema(true)
	records := []bundle.Record{
		makeRecord("ai.request.received", map[string]any{"request_id": "req-1"}),
	}
	cases := []struct {
		anchor AnchorVerifyResult
		wantXC float64
		wantAC float64
	}{
		{AnchorVerified, 1.0, 1.0},
		{AnchorAbsent, 0.0, 0.0},
		{AnchorDigestOnly, 0.5, 0.0},
	}
	for _, tc := range cases {
		scores := genericSchemaSubScores(schema, records, tc.anchor)
		if scores["XC"] != tc.wantXC {
			t.Errorf("anchor=%v XC = %f, want %f", tc.anchor, scores["XC"], tc.wantXC)
		}
		if scores["AC"] != tc.wantAC {
			t.Errorf("anchor=%v AC = %f, want %f", tc.anchor, scores["AC"], tc.wantAC)
		}
	}
}

func TestSubScoresForProfile_DSLProfile(t *testing.T) {
	profile := &stubSchemaProfile{schema: dslSchema(true)}
	records := []bundle.Record{
		makeRecord("ai.request.received", map[string]any{"request_id": "req-1"}),
		makeRecord("ai.action.precommit", map[string]any{"action_id": "act-1"}),
	}
	scores := subScoresForProfile(profile, records, AnchorVerified)
	if scores["EC"] != 1.0 {
		t.Errorf("EC = %f, want 1.0", scores["EC"])
	}
	// XC and AC should reflect AnchorVerified.
	if scores["XC"] != 1.0 {
		t.Errorf("XC = %f, want 1.0", scores["XC"])
	}
	if scores["AC"] != 1.0 {
		t.Errorf("AC = %f, want 1.0", scores["AC"])
	}
}

func TestSubScoresForProfile_Nil(t *testing.T) {
	scores := subScoresForProfile(nil, nil, AnchorAbsent)
	for k, v := range scores {
		if v != 0.0 {
			t.Errorf("nil profile: scores[%s] = %f, want 0.0", k, v)
		}
	}
}

func TestProfileSupportsCAS_DSLWithWeights(t *testing.T) {
	profile := &stubSchemaProfile{schema: dslSchema(true)}
	if !profileSupportsCAS(profile) {
		t.Errorf("expected CAS support for DSL profile with SupportsCAS=true")
	}
}

func TestProfileSupportsCAS_DSLWithoutWeights(t *testing.T) {
	profile := &stubSchemaProfile{schema: dslSchema(false)}
	if profileSupportsCAS(profile) {
		t.Errorf("expected no CAS support for DSL profile with SupportsCAS=false")
	}
}

// TestRAGAnswerSubScores_GCIsFixed asserts that the GC sub-score for the
// rag_answer profile is always 0.3 regardless of bundle contents. This value
// is hardcoded in ragAnswerSubScores to reflect that RAG pipelines do not have
// a full pre-commit gating chain. A refactor accidentally setting it to 0.0
// would otherwise pass all other tests.
func TestRAGAnswerSubScores_GCIsFixed(t *testing.T) {
	scores := ragAnswerSubScores(nil, AnchorAbsent)
	const want = 0.3
	if got := scores["GC"]; got != want {
		t.Fatalf("ragAnswerSubScores GC = %v, want %v", got, want)
	}
}
