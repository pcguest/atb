// SPDX-License-Identifier: MIT
package sessionindex

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

func TestBuildIndexInfersPrivilegedToolProfile(t *testing.T) {
	path := writeSessionBundle(t, []testEvent{
		{
			eventType: "atb.human.approval",
			timestamp: "2026-05-28T10:00:00Z",
			data: map[string]any{
				"session_id": "session-tool",
				"actor": map[string]any{
					"display_name": "Paddy Guest",
					"email":        "paddy@example.com",
				},
			},
		},
		{
			eventType: "atb.tool.call",
			timestamp: "2026-05-28T10:01:00Z",
			data: map[string]any{
				"session_id": "session-tool",
			},
		},
		{
			eventType: "atb.session.close",
			timestamp: "2026-05-28T10:05:00Z",
			data: map[string]any{
				"session_id":      "session-tool",
				"exchange_count":  3,
				"closed_at":       "2026-05-28T10:05:00Z",
				"completion_note": "complete",
			},
		},
	})

	entries, err := BuildIndex(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries length = %d, want 1", len(entries))
	}

	entry := entries[0]
	if entry.SessionID != "session-tool" {
		t.Fatalf("SessionID = %q, want session-tool", entry.SessionID)
	}
	if entry.Actor.DisplayName != "Paddy Guest" {
		t.Fatalf("Actor.DisplayName = %q", entry.Actor.DisplayName)
	}
	if entry.Actor.Email != "paddy@example.com" {
		t.Fatalf("Actor.Email = %q", entry.Actor.Email)
	}
	if entry.InferredProfile != "atb.profile.privileged_tool_action" {
		t.Fatalf("InferredProfile = %q", entry.InferredProfile)
	}
	if entry.ExchangeCount != 3 {
		t.Fatalf("ExchangeCount = %d, want 3", entry.ExchangeCount)
	}
	if entry.ClosedAt == nil || entry.ClosedAt.Format(time.RFC3339) != "2026-05-28T10:05:00Z" {
		t.Fatalf("ClosedAt = %v", entry.ClosedAt)
	}
	if len(entry.AnomalyFlags) != 0 {
		t.Fatalf("AnomalyFlags = %v, want none", entry.AnomalyFlags)
	}
}

func TestBuildIndexFlagsUnresolvedIdentity(t *testing.T) {
	actorID := "api-key:abcd"
	path := writeSessionBundle(t, []testEvent{
		{
			eventType: "atb.data.export",
			timestamp: "2026-05-28T11:00:00Z",
			actorID:   &actorID,
			data: map[string]any{
				"session_id": "session-export",
				"actor": map[string]any{
					"display_name": actorID,
				},
			},
		},
	})

	entries, err := BuildIndex(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries length = %d, want 1", len(entries))
	}

	entry := entries[0]
	if entry.InferredProfile != "atb.profile.data_export" {
		t.Fatalf("InferredProfile = %q", entry.InferredProfile)
	}
	assertFlags(t, entry.AnomalyFlags, []string{"session_not_closed", "unresolved_identity"})
}

func TestBuildIndexFlagsSessionNotClosed(t *testing.T) {
	path := writeSessionBundle(t, []testEvent{
		{
			eventType: "ai.retrieval.executed",
			timestamp: "2026-05-28T12:00:00Z",
			data: map[string]any{
				"session_id": "session-rag",
				"actor": map[string]any{
					"display_name": "Reviewer",
				},
			},
		},
	})

	entries, err := BuildIndex(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries length = %d, want 1", len(entries))
	}
	if entries[0].InferredProfile != "atb.profile.rag_answer" {
		t.Fatalf("InferredProfile = %q", entries[0].InferredProfile)
	}
	assertFlags(t, entries[0].AnomalyFlags, []string{"session_not_closed"})
}

func TestBuildIndexDoesNotInferBackgroundAutomationForProxyTraffic(t *testing.T) {
	path := writeSessionBundle(t, []testEvent{
		{
			eventType: "atb.llm.request",
			timestamp: "2026-05-28T12:00:00Z",
			data: map[string]any{
				"session_id": "session-proxy",
				"actor": map[string]any{
					"display_name": "Reviewer",
				},
			},
		},
		{
			eventType: "atb.llm.response",
			timestamp: "2026-05-28T12:00:01Z",
			data: map[string]any{
				"session_id": "session-proxy",
			},
		},
		{
			eventType: "atb.session.close",
			timestamp: "2026-05-28T12:05:00Z",
			data:      map[string]any{"session_id": "session-proxy"},
		},
	})

	entries, err := BuildIndex(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries length = %d, want 1", len(entries))
	}
	if entries[0].InferredProfile != "" {
		t.Fatalf("InferredProfile = %q, want empty for proxy-only traffic", entries[0].InferredProfile)
	}
	if entries[0].CASGrade != "" {
		t.Fatalf("CASGrade = %q, want empty when no profile inferred", entries[0].CASGrade)
	}
}

func TestBuildIndexInfersPolicyDecisionProfile(t *testing.T) {
	path := writeSessionBundle(t, []testEvent{
		{
			eventType: "ai.policy.decision",
			timestamp: "2026-05-28T13:00:00Z",
			data: map[string]any{
				"session_id": "session-policy",
				"actor": map[string]any{
					"display_name": "Reviewer",
				},
			},
		},
		{
			eventType: "atb.session.close",
			timestamp: "2026-05-28T13:05:00Z",
			data:      map[string]any{"session_id": "session-policy"},
		},
	})

	entries, err := BuildIndex(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries length = %d, want 1", len(entries))
	}
	if entries[0].InferredProfile != "atb.profile.policy_decision" {
		t.Fatalf("InferredProfile = %q, want atb.profile.policy_decision", entries[0].InferredProfile)
	}
}

func TestToolWithoutApprovalOrdering(t *testing.T) {
	approval := func(ts string) testEvent {
		return testEvent{
			eventType: "atb.human.approval",
			timestamp: ts,
			data:      map[string]any{"session_id": "s", "approved_action_id": "a1"},
		}
	}
	toolCall := func(ts string) testEvent {
		return testEvent{
			eventType: "atb.tool.call",
			timestamp: ts,
			data:      map[string]any{"session_id": "s", "tool_name": "search"},
		}
	}
	closeEvt := func(ts string) testEvent {
		return testEvent{
			eventType: "atb.session.close",
			timestamp: ts,
			data:      map[string]any{"session_id": "s"},
		}
	}

	cases := []struct {
		name        string
		events      []testEvent
		wantFlagged bool
	}{
		{
			name:        "approval before tool clears flag",
			events:      []testEvent{approval("2026-05-28T10:00:00Z"), toolCall("2026-05-28T10:01:00Z"), closeEvt("2026-05-28T10:02:00Z")},
			wantFlagged: false,
		},
		{
			name:        "tool before approval is flagged",
			events:      []testEvent{toolCall("2026-05-28T10:00:00Z"), approval("2026-05-28T10:01:00Z"), closeEvt("2026-05-28T10:02:00Z")},
			wantFlagged: true,
		},
		{
			name:        "tool with no approval is flagged",
			events:      []testEvent{toolCall("2026-05-28T10:00:00Z"), closeEvt("2026-05-28T10:01:00Z")},
			wantFlagged: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeSessionBundle(t, tc.events)
			entries, err := BuildIndex(context.Background(), []string{path})
			if err != nil {
				t.Fatalf("BuildIndex: %v", err)
			}
			if len(entries) != 1 {
				t.Fatalf("entries length = %d, want 1", len(entries))
			}
			gotFlagged := containsFlag(entries[0].AnomalyFlags, "tool_without_approval")
			if gotFlagged != tc.wantFlagged {
				t.Fatalf("tool_without_approval = %v, want %v (flags=%v)", gotFlagged, tc.wantFlagged, entries[0].AnomalyFlags)
			}
		})
	}
}

func TestAnomalyFlagsScopedPerSession(t *testing.T) {
	path := writeSessionBundle(t, []testEvent{
		// Session A: tool call with no approval -> flagged.
		{
			eventType: "atb.tool.call",
			timestamp: "2026-05-28T10:00:00Z",
			data:      map[string]any{"session_id": "sess-a", "tool_name": "search"},
		},
		{
			eventType: "atb.session.close",
			timestamp: "2026-05-28T10:01:00Z",
			data:      map[string]any{"session_id": "sess-a"},
		},
		// Session B: approval before tool call -> not flagged.
		{
			eventType: "atb.human.approval",
			timestamp: "2026-05-28T10:02:00Z",
			data:      map[string]any{"session_id": "sess-b", "approved_action_id": "a1"},
		},
		{
			eventType: "atb.tool.call",
			timestamp: "2026-05-28T10:03:00Z",
			data:      map[string]any{"session_id": "sess-b", "tool_name": "search"},
		},
		{
			eventType: "atb.session.close",
			timestamp: "2026-05-28T10:04:00Z",
			data:      map[string]any{"session_id": "sess-b"},
		},
	})

	entries, err := BuildIndex(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	byID := map[string]SessionEntry{}
	for _, entry := range entries {
		byID[entry.SessionID] = entry
	}
	if !containsFlag(byID["sess-a"].AnomalyFlags, "tool_without_approval") {
		t.Fatalf("sess-a should be flagged: %v", byID["sess-a"].AnomalyFlags)
	}
	if containsFlag(byID["sess-b"].AnomalyFlags, "tool_without_approval") {
		t.Fatalf("sess-b should NOT be flagged: %v", byID["sess-b"].AnomalyFlags)
	}
}

func containsFlag(flags []string, want string) bool {
	for _, f := range flags {
		if f == want {
			return true
		}
	}
	return false
}

func TestBuildIndexReturnsLoadErrorEntry(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.atb")

	entries, err := BuildIndex(context.Background(), []string{missing})
	if err == nil {
		t.Fatalf("expected load error")
	}
	var loadErrors LoadErrors
	if !errors.As(err, &loadErrors) {
		t.Fatalf("error type = %T, want LoadErrors", err)
	}
	if len(loadErrors) != 1 {
		t.Fatalf("load error count = %d, want 1", len(loadErrors))
	}
	if len(entries) != 1 {
		t.Fatalf("entries length = %d, want 1", len(entries))
	}
	assertFlags(t, entries[0].AnomalyFlags, []string{"load_error"})
}

func TestGroupByActorUsesDisplayName(t *testing.T) {
	start := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)
	entries := []SessionEntry{
		{SessionID: "older", Actor: ActorSummary{DisplayName: "User A"}, StartedAt: start.Add(-time.Hour)},
		{SessionID: "api", Actor: ActorSummary{DisplayName: "api-key:1234"}, StartedAt: start},
		{SessionID: "newer", Actor: ActorSummary{DisplayName: "User A"}, StartedAt: start.Add(time.Hour)},
	}

	grouped := GroupByActor(entries)
	if len(grouped["User A"]) != 2 {
		t.Fatalf("User A group length = %d, want 2", len(grouped["User A"]))
	}
	if grouped["User A"][0].SessionID != "newer" {
		t.Fatalf("User A first session = %q, want newer", grouped["User A"][0].SessionID)
	}
	if len(grouped["api-key:1234"]) != 1 {
		t.Fatalf("api-key group length = %d, want 1", len(grouped["api-key:1234"]))
	}
}

type testEvent struct {
	eventType string
	timestamp string
	data      map[string]any
	actorID   *string
}

func TestBuildIndexSkipsBundleLevelEvents(t *testing.T) {
	path := writeSessionBundle(t, []testEvent{
		{
			eventType: "atb.tool.call",
			timestamp: "2026-05-28T10:01:00Z",
			data:      map[string]any{"session_id": "session-real", "tool_name": "deploy"},
		},
		{
			eventType: "atb.session.close",
			timestamp: "2026-05-28T10:05:00Z",
			data:      map[string]any{"session_id": "session-real", "actor_id": "agent-x"},
		},
		// Bundle-level events carry no session_id. Before the fix these seeded a
		// spurious path-derived pseudo-session (visible once a bundle is signed).
		{
			eventType: "atb.snapshot",
			timestamp: "2026-05-28T10:06:00Z",
			data:      map[string]any{"name": "snap"},
		},
		{
			eventType: "atb.bundle.signature",
			timestamp: "2026-05-28T10:07:00Z",
			data:      map[string]any{"bundle_hash": "abc", "signature": "sig", "pubkey": "pk"},
		},
	})

	entries, err := BuildIndex(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries length = %d, want 1 (bundle-level events must not create sessions)", len(entries))
	}
	if entries[0].SessionID != "session-real" {
		t.Errorf("session id = %q, want session-real", entries[0].SessionID)
	}
}

func TestBuildIndexFlagsPolicyDeniedExecuted(t *testing.T) {
	path := writeSessionBundle(t, []testEvent{
		{
			eventType: "ai.policy.decision",
			timestamp: "2026-05-28T10:00:00Z",
			data:      map[string]any{"session_id": "sess-deny", "action_id": "act-1", "decision": "deny"},
		},
		{
			eventType: "ai.action.executed",
			timestamp: "2026-05-28T10:01:00Z",
			data:      map[string]any{"session_id": "sess-deny", "action_id": "act-1", "execution_outcome": "success"},
		},
		{
			eventType: "atb.session.close",
			timestamp: "2026-05-28T10:02:00Z",
			data:      map[string]any{"session_id": "sess-deny", "actor_id": "agent-x"},
		},
	})
	entries, err := BuildIndex(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 session, got %d", len(entries))
	}
	if !contains(entries[0].AnomalyFlags, "policy_denied_executed") {
		t.Errorf("want policy_denied_executed, got %v", entries[0].AnomalyFlags)
	}
}

func TestBuildIndexFlagsActionFailed(t *testing.T) {
	path := writeSessionBundle(t, []testEvent{
		{
			eventType: "ai.action.error",
			timestamp: "2026-05-28T10:01:00Z",
			data:      map[string]any{"session_id": "sess-fail", "action_id": "act-1", "error_class": "exception"},
		},
		{
			eventType: "atb.session.close",
			timestamp: "2026-05-28T10:02:00Z",
			data:      map[string]any{"session_id": "sess-fail", "actor_id": "agent-x"},
		},
	})
	entries, err := BuildIndex(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if len(entries) != 1 || !contains(entries[0].AnomalyFlags, "action_failed") {
		t.Errorf("want action_failed, got %v", entries)
	}
}

// TestBuildIndexNoFalsePositiveDeniedExecuted: a deny for one action and a
// (different) approved action executing must NOT raise policy_denied_executed.
func TestBuildIndexNoFalsePositiveDeniedExecuted(t *testing.T) {
	path := writeSessionBundle(t, []testEvent{
		{
			eventType: "ai.policy.decision",
			timestamp: "2026-05-28T10:00:00Z",
			data:      map[string]any{"session_id": "sess-ok", "action_id": "act-refund", "decision": "deny"},
		},
		{
			eventType: "ai.action.executed",
			timestamp: "2026-05-28T10:01:00Z",
			data:      map[string]any{"session_id": "sess-ok", "action_id": "act-credit", "execution_outcome": "success"},
		},
		{
			eventType: "atb.session.close",
			timestamp: "2026-05-28T10:02:00Z",
			data:      map[string]any{"session_id": "sess-ok", "actor_id": "agent-x"},
		},
	})
	entries, err := BuildIndex(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 session, got %d", len(entries))
	}
	if contains(entries[0].AnomalyFlags, "policy_denied_executed") {
		t.Errorf("false positive: %v", entries[0].AnomalyFlags)
	}
}

func contains(flags []string, want string) bool {
	for _, f := range flags {
		if f == want {
			return true
		}
	}
	return false
}

func writeSessionBundle(t *testing.T, events []testEvent) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "bundle.atb")
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("bundle.New: %v", err)
	}
	for _, event := range events {
		if err := b.AppendWithOptions(event.eventType, event.data, &bundle.AppendOptions{
			ActorID:   event.actorID,
			Timestamp: event.timestamp,
		}); err != nil {
			t.Fatalf("append %s: %v", event.eventType, err)
		}
	}
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}
	return path
}

func assertFlags(t *testing.T, got []string, want []string) {
	t.Helper()

	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("flags = %v, want %v", got, want)
	}
}
