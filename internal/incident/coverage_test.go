// SPDX-License-Identifier: MIT
package incident

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/sessionindex"
	"github.com/pcguest/atb/internal/verify"
)

func TestSummariseEventVariants(t *testing.T) {
	tests := []struct {
		eventType string
		data      map[string]any
		want      string
	}{
		{
			eventType: event.TypeAIActionPrecommit,
			data: map[string]any{
				"action_type": "deploy",
				"principal":   map[string]any{"type": "agent", "id_hash": "sha256:agent", "on_behalf_of": "sha256:user"},
			},
			want: "by agent:sha256:agent on_behalf_of sha256:user",
		},
		{eventType: event.TypeAIActionPrecommit, data: map[string]any{"action_type": "deploy"}, want: "action=deploy"},
		{eventType: event.TypeAIActionExecuted, data: map[string]any{"execution_outcome": "success", "effective_scope": "prod"}, want: "scope=prod"},
		{eventType: event.TypeAIActionExecuted, data: map[string]any{"execution_outcome": "success"}, want: "executed outcome=success"},
		{eventType: event.TypeToolCall, data: map[string]any{"tool_name": "shell"}, want: "tool=shell"},
		{eventType: event.TypeAIActionError, data: map[string]any{"action_id": "a1", "error_class": "blocked"}, want: "error_class=blocked"},
		{eventType: event.TypeLLMRequest, data: map[string]any{"method": "POST", "path": "/chat"}, want: "POST /chat"},
		{eventType: event.TypeLLMResponse, data: map[string]any{"method": "POST", "path": "/chat", "status_code": json.Number("201")}, want: "→ 201"},
		{eventType: event.TypeAIPolicyDecision, data: map[string]any{"decision": "deny"}, want: "decision=deny"},
		{eventType: event.TypeAIHumanApproval, data: map[string]any{"approval_outcome": "approved"}, want: "approval=approved"},
		{eventType: event.TypeHumanApproval, data: map[string]any{"approved_action_id": "a1"}, want: "approved=a1"},
		{eventType: event.TypeHumanOverride, data: map[string]any{"overridden_action_id": "a1"}, want: "override=a1"},
		{eventType: event.TypeHumanOverride, data: map[string]any{}, want: "human override"},
		{eventType: event.TypeCaptureScope, data: map[string]any{"capture_mode": "proxy"}, want: "capture=proxy"},
		{eventType: event.TypeDataExportExecuted, data: map[string]any{"execution_outcome": "success"}, want: "export outcome=success"},
		{eventType: event.TypeDataExportExecuted, data: map[string]any{}, want: "export"},
		{eventType: event.TypeDataExportError, data: map[string]any{"action_id": "a1", "error_class": "timeout"}, want: "error_class=timeout"},
		{eventType: event.TypeDataExportError, data: map[string]any{}, want: "export error"},
		{eventType: event.TypeDataExport, data: map[string]any{"export_target": "archive"}, want: "export=archive"},
		{eventType: event.TypeDataExport, data: map[string]any{}, want: "export"},
		{eventType: event.TypeDataRetentionPolicySet, data: map[string]any{"days": int64(30)}, want: "retention=30d"},
		{eventType: event.TypeDataRetentionPolicyChanged, data: map[string]any{}, want: "retention policy"},
		{eventType: event.TypeDataRetentionEnforced, data: map[string]any{"operation": "delete", "outcome": "success"}, want: "delete outcome=success"},
		{eventType: "custom.event", data: map[string]any{}, want: ""},
	}
	for _, tc := range tests {
		got := summarise(event.Event{Type: tc.eventType, Data: tc.data})
		if !strings.Contains(got, tc.want) {
			t.Fatalf("summarise(%s)=%q want substring %q", tc.eventType, got, tc.want)
		}
	}
}

func TestIncidentFormattingHelpers(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  int
		ok    bool
	}{
		{value: 1, want: 1, ok: true},
		{value: int64(2), want: 2, ok: true},
		{value: float64(3), want: 3, ok: true},
		{value: json.Number("4"), want: 4, ok: true},
		{value: json.Number("bad"), ok: false},
		{value: "5", ok: false},
	} {
		got, ok := intField(map[string]any{"value": tc.value}, "value")
		if got != tc.want || ok != tc.ok {
			t.Fatalf("intField(%#v)=%d,%v want=%d,%v", tc.value, got, ok, tc.want, tc.ok)
		}
	}

	if got := principalSummary(map[string]any{"principal": "bad"}); got != "" {
		t.Fatalf("bad principal=%q", got)
	}
	if got := principalSummary(map[string]any{"principal": map[string]any{"type": "agent"}}); got != "" {
		t.Fatalf("incomplete principal=%q", got)
	}

	if got := signatureSummary(nil); got != "none (unsigned bundle)" {
		t.Fatalf("unsigned summary=%q", got)
	}
	sigs := []verify.SignatureProvenance{
		{Valid: true},
		{
			Valid: false, PubKey: strings.Repeat("a", 32), SignedAt: "2026-06-30T00:00:00Z",
			Backend: "ed25519", Error: "signature mismatch",
		},
	}
	got := signatureSummary(sigs)
	for _, want := range []string{"2 signatures", "INVALID", "pubkey", "signed", "backend", "signature mismatch"} {
		if !strings.Contains(got, want) {
			t.Fatalf("signature summary=%q missing %q", got, want)
		}
	}
	if got := signatureSummary([]verify.SignatureProvenance{{Valid: true}}); got != "valid" {
		t.Fatalf("valid summary=%q", got)
	}

	if got := actorLabel(sessionindex.ActorSummary{DisplayName: "Patrick", Email: "p@example.com"}); got != "Patrick <p@example.com>" {
		t.Fatalf("full actor=%q", got)
	}
	if got := actorLabel(sessionindex.ActorSummary{DisplayName: "Patrick"}); got != "Patrick" {
		t.Fatalf("named actor=%q", got)
	}
	if got := actorLabel(sessionindex.ActorSummary{}); got != "—" {
		t.Fatalf("empty actor=%q", got)
	}
}
