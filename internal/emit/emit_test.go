// SPDX-License-Identifier: MIT
package emit_test

import (
	"testing"

	"github.com/pcguest/atb/internal/emit"
	"github.com/pcguest/atb/internal/event"
)

func stubAppendFn() (func(event.Event) (string, error), *[]event.Event) {
	captured := []event.Event{}
	stub := func(e event.Event) (string, error) {
		captured = append(captured, e)
		return "stub-hash", nil
	}
	return stub, &captured
}

func dataMap(t *testing.T, ev event.Event) map[string]any {
	t.Helper()
	data, ok := ev.Data.(map[string]any)
	if !ok {
		t.Fatalf("Data type = %T, want map[string]any", ev.Data)
	}
	return data
}

func TestNewEmitter_EmptySessionID(t *testing.T) {
	t.Parallel()

	stub, _ := stubAppendFn()
	if _, err := emit.NewEmitter("", stub); err == nil {
		t.Fatal("expected error for empty sessionID")
	}
	if _, err := emit.NewEmitter("sess-1", stub); err != nil {
		t.Fatalf("NewEmitter: %v", err)
	}
}

func TestToolCall_MissingRequired(t *testing.T) {
	t.Parallel()

	stub, captured := stubAppendFn()
	emitter, err := emit.NewEmitter("sess-1", stub)
	if err != nil {
		t.Fatalf("NewEmitter: %v", err)
	}
	if err := emitter.ToolCall(emit.ToolCallOptions{}); err == nil {
		t.Fatal("expected error for missing ToolName")
	}
	if len(*captured) != 0 {
		t.Fatalf("expected no append on validation failure, got %d events", len(*captured))
	}
}

func TestToolCall_Valid(t *testing.T) {
	t.Parallel()

	stub, captured := stubAppendFn()
	emitter, err := emit.NewEmitter("sess-1", stub)
	if err != nil {
		t.Fatalf("NewEmitter: %v", err)
	}
	if err := emitter.ToolCall(emit.ToolCallOptions{ToolName: "search"}); err != nil {
		t.Fatalf("ToolCall: %v", err)
	}
	if len(*captured) != 1 {
		t.Fatalf("captured = %d, want 1", len(*captured))
	}
	ev := (*captured)[0]
	if ev.Type != event.TypeToolCall {
		t.Fatalf("Type = %q, want %q", ev.Type, event.TypeToolCall)
	}
	data := dataMap(t, ev)
	if data["session_id"] != "sess-1" {
		t.Fatalf("session_id = %v, want sess-1", data["session_id"])
	}
	if data["tool_name"] != "search" {
		t.Fatalf("tool_name = %v, want search", data["tool_name"])
	}
}

func TestDataExport_Valid(t *testing.T) {
	t.Parallel()

	stub, captured := stubAppendFn()
	emitter, err := emit.NewEmitter("sess-1", stub)
	if err != nil {
		t.Fatalf("NewEmitter: %v", err)
	}
	if err := emitter.DataExport(emit.DataExportOptions{ExportTarget: "csv"}); err != nil {
		t.Fatalf("DataExport: %v", err)
	}
	if len(*captured) != 1 {
		t.Fatalf("captured = %d, want 1", len(*captured))
	}
	ev := (*captured)[0]
	if ev.Type != event.TypeDataExport {
		t.Fatalf("Type = %q, want %q", ev.Type, event.TypeDataExport)
	}
	data := dataMap(t, ev)
	if data["export_target"] != "csv" {
		t.Fatalf("export_target = %v, want csv", data["export_target"])
	}
}

func TestHumanOverride_Valid(t *testing.T) {
	t.Parallel()

	stub, captured := stubAppendFn()
	emitter, err := emit.NewEmitter("sess-1", stub)
	if err != nil {
		t.Fatalf("NewEmitter: %v", err)
	}
	if err := emitter.HumanOverride(emit.HumanOverrideOptions{
		OverrideReason: "emergency",
		ActorID:        "alice",
	}); err != nil {
		t.Fatalf("HumanOverride: %v", err)
	}
	if len(*captured) != 1 {
		t.Fatalf("captured = %d, want 1", len(*captured))
	}
	ev := (*captured)[0]
	if ev.Type != event.TypeHumanOverride {
		t.Fatalf("Type = %q, want %q", ev.Type, event.TypeHumanOverride)
	}
	data := dataMap(t, ev)
	if data["actor_id"] != "alice" {
		t.Fatalf("actor_id = %v, want alice", data["actor_id"])
	}
}

func TestHumanApproval_Valid(t *testing.T) {
	t.Parallel()

	stub, captured := stubAppendFn()
	emitter, err := emit.NewEmitter("sess-1", stub)
	if err != nil {
		t.Fatalf("NewEmitter: %v", err)
	}
	if err := emitter.HumanApproval(emit.HumanApprovalOptions{
		ApprovedActionID: "action-42",
	}); err != nil {
		t.Fatalf("HumanApproval: %v", err)
	}
	if len(*captured) != 1 {
		t.Fatalf("captured = %d, want 1", len(*captured))
	}
	ev := (*captured)[0]
	if ev.Type != event.TypeHumanApproval {
		t.Fatalf("Type = %q, want %q", ev.Type, event.TypeHumanApproval)
	}
	data := dataMap(t, ev)
	if data["approved_action_id"] != "action-42" {
		t.Fatalf("approved_action_id = %v, want action-42", data["approved_action_id"])
	}
}

func TestHumanApproval_MissingRequired(t *testing.T) {
	t.Parallel()

	stub, captured := stubAppendFn()
	emitter, err := emit.NewEmitter("sess-1", stub)
	if err != nil {
		t.Fatalf("NewEmitter: %v", err)
	}
	if err := emitter.HumanApproval(emit.HumanApprovalOptions{}); err == nil {
		t.Fatal("expected error for missing ApprovedActionID")
	}
	if len(*captured) != 0 {
		t.Fatalf("expected no append on validation failure, got %d events", len(*captured))
	}
}
