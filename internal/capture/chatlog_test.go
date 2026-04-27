// SPDX-License-Identifier: MIT
package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

func TestParseGenericJSONL(t *testing.T) {
	raw := strings.Join([]string{
		`{"role":"system","content":"You answer with policy citations.","timestamp":"2026-04-24T09:00:00Z","session_id":"sess-1"}`,
		`{"role":"user","content":"Can I carry annual leave into next year?","timestamp":"2026-04-24T09:00:10Z","session_id":"sess-1","request_id":"req-1"}`,
		`{"role":"tool","content":"{\"answer\":\"up to five days\"}","timestamp":"2026-04-24T09:00:11Z","session_id":"sess-1","tool_name":"hr.policy.lookup","tool_args":{"query":"annual leave carry over"}}`,
		`{"role":"assistant","content":"Yes. Up to five days can be carried over.","timestamp":"2026-04-24T09:00:12Z","session_id":"sess-1","model":"gpt-4o-mini"}`,
	}, "\n")

	messages, err := ParseGenericJSONL(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseGenericJSONL() error = %v", err)
	}
	if len(messages) != 4 {
		t.Fatalf("len(messages) = %d, want 4", len(messages))
	}
	if messages[2].ToolName != "hr.policy.lookup" {
		t.Fatalf("tool name = %q, want %q", messages[2].ToolName, "hr.policy.lookup")
	}
	if messages[3].Model != "gpt-4o-mini" {
		t.Fatalf("model = %q, want %q", messages[3].Model, "gpt-4o-mini")
	}
}

func TestImportFixtureMapping(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTypes []string
	}{
		{
			name:      "empty array fixture",
			input:     filepath.Join("testdata", "import_empty.jsonl"),
			wantTypes: nil,
		},
		{
			name:      "single user fixture",
			input:     filepath.Join("testdata", "import_single_user.jsonl"),
			wantTypes: []string{event.TypeAIRequestReceived},
		},
		{
			name:      "empty object skipped",
			input:     "{}\n",
			wantTypes: nil,
		},
		{
			name:      "unknown role skipped",
			input:     `{"role":"unknown","content":"x"}` + "\n",
			wantTypes: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := tc.input
			if strings.HasPrefix(tc.input, "testdata") {
				data, err := os.ReadFile(tc.input)
				if err != nil {
					t.Fatalf("read fixture: %v", err)
				}
				raw = string(data)
			}

			messages, err := ParseGenericJSONL(strings.NewReader(raw))
			if err != nil {
				t.Fatalf("ParseGenericJSONL() error = %v", err)
			}
			result, err := mapMessagesWithImportSkip(messages)
			if err != nil {
				t.Fatalf("mapMessagesWithImportSkip() error = %v", err)
			}
			if len(result.Events) != len(tc.wantTypes) {
				t.Fatalf("len(result.Events) = %d, want %d", len(result.Events), len(tc.wantTypes))
			}
			for i, want := range tc.wantTypes {
				if result.Events[i].Type != want {
					t.Fatalf("event[%d].Type = %q, want %q", i, result.Events[i].Type, want)
				}
			}
		})
	}
}

func mapMessagesWithImportSkip(messages []ChatMessage) (MappingResult, error) {
	result, err := MapMessagesToEvents(messages)
	if errors.Is(err, ErrUnknownTurn) {
		var unknown []int
		messages, unknown = FilterKnownTurns(messages)
		result, err = MapMessagesToEvents(messages)
		if err == nil {
			result.SkippedRecords += len(unknown)
		}
	}
	return result, err
}

func TestParseGenericJSONLRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "missing content", raw: `{"role":"user","timestamp":"2026-04-24T09:00:00Z"}`},
		{name: "invalid timestamp", raw: `{"role":"user","content":"hello","timestamp":"24-04-2026"}`},
		{name: "tool missing name", raw: `{"role":"tool","content":"{}","timestamp":"2026-04-24T09:00:00Z"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseGenericJSONL(strings.NewReader(tc.raw)); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestParseGenericJSONLAllowsMissingRoleForImportPolicy(t *testing.T) {
	raw := `{"content":"hello","timestamp":"2026-04-24T09:00:00Z"}`

	messages, err := ParseGenericJSONL(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseGenericJSONL() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(messages))
	}
	if messages[0].Role != "" {
		t.Fatalf("Role = %q, want empty", messages[0].Role)
	}
}

func TestMapMessagesToEventsRejectsUnknownTurn(t *testing.T) {
	messages := []ChatMessage{
		{Line: 1, Role: "user", Content: "hello", Timestamp: "2026-04-24T09:00:00Z"},
		{Line: 2, Role: "critic", Content: "unknown", Timestamp: "2026-04-24T09:00:01Z"},
	}

	_, err := MapMessagesToEvents(messages)
	if !errors.Is(err, ErrUnknownTurn) {
		t.Fatalf("MapMessagesToEvents() error = %v, want ErrUnknownTurn", err)
	}
	if !strings.Contains(err.Error(), "capture: turn 1") {
		t.Fatalf("error = %q, want turn context", err.Error())
	}
}

func TestMapMessagesToEventsSystemOnlyProducesNoEvents(t *testing.T) {
	messages := []ChatMessage{
		{Line: 1, Role: "system", Content: "Use the handbook.", Timestamp: "2026-04-24T09:00:00Z"},
		{Line: 2, Role: "system", Content: "Prefer concise answers.", Timestamp: "2026-04-24T09:00:01Z"},
	}

	result, err := MapMessagesToEvents(messages)
	if err != nil {
		t.Fatalf("MapMessagesToEvents() error = %v", err)
	}
	if len(result.Events) != 0 {
		t.Fatalf("len(result.Events) = %d, want 0", len(result.Events))
	}
	if result.SkippedRecords != 2 {
		t.Fatalf("SkippedRecords = %d, want 2", result.SkippedRecords)
	}
}

func TestMapMessagesToEventsToolTurnIncludesCompletePayload(t *testing.T) {
	messages := []ChatMessage{
		{Line: 1, Role: "user", Content: "look up policy", Timestamp: "2026-04-24T09:00:00Z", RequestID: "req-1"},
		{
			Line:       2,
			Role:       "tool",
			Content:    `{"policy":"up to five days"}`,
			Timestamp:  "2026-04-24T09:00:01Z",
			ToolName:   "hr.policy.lookup",
			ToolCallID: "call-123",
			ToolArgs:   []byte(`{"query":"annual leave","limit":1}`),
		},
	}

	result, err := MapMessagesToEvents(messages)
	if err != nil {
		t.Fatalf("MapMessagesToEvents() error = %v", err)
	}
	if len(result.Events) != 2 {
		t.Fatalf("len(result.Events) = %d, want 2", len(result.Events))
	}
	tool := result.Events[1]
	if tool.Type != event.TypeAIToolExec {
		t.Fatalf("tool.Type = %q, want %q", tool.Type, event.TypeAIToolExec)
	}
	if tool.Data["tool_name"] != "hr.policy.lookup" {
		t.Fatalf("tool_name = %v", tool.Data["tool_name"])
	}
	if tool.Data["tool_call_id"] != "call-123" {
		t.Fatalf("tool_call_id = %v", tool.Data["tool_call_id"])
	}
	args, ok := tool.Data["tool_args"].(map[string]any)
	if !ok {
		t.Fatalf("tool_args = %#v, want map", tool.Data["tool_args"])
	}
	if args["query"] != "annual leave" || args["limit"] != float64(1) {
		t.Fatalf("tool_args = %#v", args)
	}
	if tool.Data["tool_output"] != `{"policy":"up to five days"}` {
		t.Fatalf("tool_output = %v", tool.Data["tool_output"])
	}
}

func TestMapMessagesToEvents(t *testing.T) {
	messages := []ChatMessage{
		{Line: 1, Role: "system", Content: "Use the employee handbook.", Timestamp: "2026-04-24T09:00:00Z", SessionID: "sess-1"},
		{Line: 2, Role: "user", Content: "Can I carry annual leave into next year?", Timestamp: "2026-04-24T09:00:10Z", SessionID: "sess-1", RequestID: "req-1", ActorIDHash: "sha256:user-1", PurposeTag: "rag_answer"},
		{Line: 3, Role: "tool", Content: `{"answer":"up to five days"}`, Timestamp: "2026-04-24T09:00:11Z", SessionID: "sess-1", ToolName: "hr.policy.lookup", ToolArgs: []byte(`{"query":"annual leave carry over"}`)},
		{Line: 4, Role: "assistant", Content: "Yes. Up to five days can be carried over.", Timestamp: "2026-04-24T09:00:12Z", SessionID: "sess-1", Model: "gpt-4o-mini"},
	}

	result, err := MapMessagesToEvents(messages)
	if err != nil {
		t.Fatalf("MapMessagesToEvents() error = %v", err)
	}
	if result.SkippedRecords != 1 {
		t.Fatalf("SkippedRecords = %d, want 1", result.SkippedRecords)
	}
	if result.SuggestedProfileID != suggestedProfileRAG {
		t.Fatalf("SuggestedProfileID = %q, want %q", result.SuggestedProfileID, suggestedProfileRAG)
	}

	wantTypes := []string{
		event.TypeAIRequestReceived,
		event.TypeAIToolExec,
		event.TypeAIModelInvoked,
		event.TypeAIModelOutput,
		event.TypeAIResponseSent,
	}
	if len(result.Events) != len(wantTypes) {
		t.Fatalf("len(result.Events) = %d, want %d", len(result.Events), len(wantTypes))
	}
	for i, want := range wantTypes {
		if got := result.Events[i].Type; got != want {
			t.Fatalf("event[%d].Type = %q, want %q", i, got, want)
		}
	}

	invoked := result.Events[2].Data
	if invoked["model_provider"] != "openai" {
		t.Fatalf("model_provider = %v, want %q", invoked["model_provider"], "openai")
	}
	if invoked["prompt_digest"] == "" {
		t.Fatalf("expected prompt_digest to be populated")
	}
	response := result.Events[len(result.Events)-1].Data
	if response["request_id"] != "req-1" {
		t.Fatalf("response request_id = %v, want %q", response["request_id"], "req-1")
	}
}

func TestAppendEventsToBundleAndVerifyRAGProfile(t *testing.T) {
	raw := strings.Join([]string{
		`{"role":"user","content":"Summarise the policy for annual leave carry-over.","timestamp":"2026-04-24T09:00:10Z","session_id":"sess-1","request_id":"req-1","actor_id_hash":"sha256:user-1","purpose_tag":"rag_answer"}`,
		`{"role":"assistant","content":"I will check the handbook.","timestamp":"2026-04-24T09:00:11Z","session_id":"sess-1","model":"gpt-4o-mini"}`,
		`{"role":"tool","content":"{\"policy\":\"up to five days\"}","timestamp":"2026-04-24T09:00:12Z","session_id":"sess-1","tool_name":"hr.policy.lookup","tool_args":{"query":"annual leave carry-over"}}`,
		`{"role":"assistant","content":"You can carry over up to five days with manager approval.","timestamp":"2026-04-24T09:00:13Z","session_id":"sess-1","model":"gpt-4o-mini"}`,
	}, "\n")

	messages, err := ParseGenericJSONL(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseGenericJSONL() error = %v", err)
	}
	mapped, err := MapMessagesToEvents(messages)
	if err != nil {
		t.Fatalf("MapMessagesToEvents() error = %v", err)
	}

	bundlePath := t.TempDir() + "/bundle.atb"
	summary, err := AppendEventsToBundle(bundlePath, mapped.Events)
	if err != nil {
		t.Fatalf("AppendEventsToBundle() error = %v", err)
	}
	if summary.EventsWritten != len(mapped.Events) {
		t.Fatalf("EventsWritten = %d, want %d", summary.EventsWritten, len(mapped.Events))
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("bundle.Load() error = %v", err)
	}
	if len(b.Records) != len(mapped.Events)+1 {
		t.Fatalf("len(b.Records) = %d, want %d", len(b.Records), len(mapped.Events)+1)
	}

	report, err := verifypkg.EvaluateBundle(verifypkg.EvaluateConfig{
		BundlePath: bundlePath,
		Records:    b.Records,
		Profiles:   []verifypkg.Profile{verifypkg.ProfileByID("atb.profile.rag_answer")},
	})
	if err != nil {
		t.Fatalf("EvaluateBundle() error = %v", err)
	}
	if !report.Profiles[0].Pass {
		t.Fatalf("expected rag profile to pass, got %+v", report.Profiles[0])
	}
	if report.CAS == nil || report.CAS.Overall <= 0 {
		t.Fatalf("expected CAS result, got %+v", report.CAS)
	}
}

func TestAppendEventsToBundleShowsUnderstandableFailureWhenIncomplete(t *testing.T) {
	raw := strings.Join([]string{
		`{"role":"user","content":"Summarise the policy.","timestamp":"2026-04-24T09:00:10Z","session_id":"sess-1","request_id":"req-1","actor_id_hash":"sha256:user-1","purpose_tag":"rag_answer"}`,
		`{"role":"assistant","content":"Here is the answer.","timestamp":"2026-04-24T09:00:11Z","session_id":"sess-1"}`,
	}, "\n")

	messages, err := ParseGenericJSONL(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseGenericJSONL() error = %v", err)
	}
	mapped, err := MapMessagesToEvents(messages)
	if err != nil {
		t.Fatalf("MapMessagesToEvents() error = %v", err)
	}

	bundlePath := t.TempDir() + "/bundle.atb"
	if _, err := AppendEventsToBundle(bundlePath, mapped.Events); err != nil {
		t.Fatalf("AppendEventsToBundle() error = %v", err)
	}
	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("bundle.Load() error = %v", err)
	}

	report, err := verifypkg.EvaluateBundle(verifypkg.EvaluateConfig{
		BundlePath: bundlePath,
		Records:    b.Records,
		Profiles:   []verifypkg.Profile{verifypkg.ProfileByID("atb.profile.rag_answer")},
	})
	if err != nil {
		t.Fatalf("EvaluateBundle() error = %v", err)
	}
	if report.Profiles[0].Pass {
		t.Fatalf("expected rag profile to fail for incomplete evidence")
	}
	if len(report.Profiles[0].CriticalFailures) == 0 {
		t.Fatalf("expected critical failures, got %+v", report.Profiles[0])
	}
	if !strings.Contains(report.Profiles[0].CriticalFailures[0].Detail, event.TypeAIModelInvoked) {
		t.Fatalf("unexpected failure detail: %+v", report.Profiles[0].CriticalFailures)
	}
}
