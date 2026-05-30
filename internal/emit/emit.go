// SPDX-License-Identifier: MIT

// Package emit provides the canonical Go surface for emitting the four
// oversight event types declared in docs/spec-ai-traces.md.
package emit

import (
	"fmt"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/event"
)

// Emitter writes typed ATB events to a bundle via an append function.
// It is the canonical Go surface for emitting the four oversight event types.
type Emitter struct {
	sessionID string
	appendFn  func(event.Event) (string, error)
}

// NewEmitter constructs an Emitter. sessionID is required. appendFn is the
// caller-supplied function that appends a raw event to the destination bundle
// and returns the record hash.
func NewEmitter(sessionID string, appendFn func(event.Event) (string, error)) (*Emitter, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("atb: NewEmitter requires a non-empty sessionID")
	}
	if appendFn == nil {
		return nil, fmt.Errorf("atb: NewEmitter requires an append function")
	}
	return &Emitter{
		sessionID: strings.TrimSpace(sessionID),
		appendFn:  appendFn,
	}, nil
}

// ToolCallOptions carries fields for an atb.tool.call event.
type ToolCallOptions struct {
	ToolName     string // required
	ActorID      string
	SessionID    string // override; uses Emitter.sessionID when empty
	InputDigest  string // SHA-256 hex of the canonicalised tool input
	OutputDigest string // SHA-256 hex of the canonicalised tool output
}

// ToolCall emits an atb.tool.call event. opts.ToolName is required.
func (e *Emitter) ToolCall(opts ToolCallOptions) error {
	if strings.TrimSpace(opts.ToolName) == "" {
		return fmt.Errorf("atb: ToolCall requires ToolName")
	}
	data := map[string]any{
		"session_id": e.sessionIDFor(opts.SessionID),
		"tool_name":  strings.TrimSpace(opts.ToolName),
	}
	if v := strings.TrimSpace(opts.ActorID); v != "" {
		data["actor_id"] = v
	}
	if v := strings.TrimSpace(opts.InputDigest); v != "" {
		data["tool_input_digest"] = v
	}
	if v := strings.TrimSpace(opts.OutputDigest); v != "" {
		data["tool_output_digest"] = v
	}
	return e.append(event.TypeToolCall, data)
}

// DataExportOptions carries fields for an atb.data.export event.
type DataExportOptions struct {
	ExportTarget   string // required — the data type / destination
	ActorID        string
	RecordCount    int
	Classification string
}

// DataExport emits an atb.data.export event. opts.ExportTarget is required.
func (e *Emitter) DataExport(opts DataExportOptions) error {
	if strings.TrimSpace(opts.ExportTarget) == "" {
		return fmt.Errorf("atb: DataExport requires ExportTarget")
	}
	data := map[string]any{
		"session_id":    e.sessionID,
		"export_target": strings.TrimSpace(opts.ExportTarget),
	}
	if v := strings.TrimSpace(opts.ActorID); v != "" {
		data["actor_id"] = v
	}
	if opts.RecordCount > 0 {
		data["record_count"] = opts.RecordCount
	}
	if v := strings.TrimSpace(opts.Classification); v != "" {
		data["classification"] = v
	}
	return e.append(event.TypeDataExport, data)
}

// HumanOverrideOptions carries fields for an atb.human.override event.
type HumanOverrideOptions struct {
	OverrideReason     string // required
	ActorID            string
	OverriddenActionID string
}

// HumanOverride emits an atb.human.override event. opts.OverrideReason is required.
func (e *Emitter) HumanOverride(opts HumanOverrideOptions) error {
	if strings.TrimSpace(opts.OverrideReason) == "" {
		return fmt.Errorf("atb: HumanOverride requires OverrideReason")
	}
	data := map[string]any{
		"session_id":      e.sessionID,
		"override_reason": strings.TrimSpace(opts.OverrideReason),
	}
	if v := strings.TrimSpace(opts.ActorID); v != "" {
		data["actor_id"] = v
	}
	if v := strings.TrimSpace(opts.OverriddenActionID); v != "" {
		data["overridden_action_id"] = v
	}
	return e.append(event.TypeHumanOverride, data)
}

// HumanApprovalOptions carries fields for an atb.human.approval event.
type HumanApprovalOptions struct {
	ApprovedActionID string // required
	ActorID          string
	ApproverID       string
	Note             string
}

// HumanApproval emits an atb.human.approval event. opts.ApprovedActionID is required.
func (e *Emitter) HumanApproval(opts HumanApprovalOptions) error {
	if strings.TrimSpace(opts.ApprovedActionID) == "" {
		return fmt.Errorf("atb: HumanApproval requires ApprovedActionID")
	}
	data := map[string]any{
		"session_id":         e.sessionID,
		"approved_action_id": strings.TrimSpace(opts.ApprovedActionID),
	}
	if v := strings.TrimSpace(opts.ActorID); v != "" {
		data["actor_id"] = v
	}
	if v := strings.TrimSpace(opts.ApproverID); v != "" {
		data["approver_id"] = v
	}
	if v := strings.TrimSpace(opts.Note); v != "" {
		data["note"] = v
	}
	return e.append(event.TypeHumanApproval, data)
}

func (e *Emitter) sessionIDFor(override string) string {
	if v := strings.TrimSpace(override); v != "" {
		return v
	}
	return e.sessionID
}

func (e *Emitter) append(eventType string, data map[string]any) error {
	_, err := e.appendFn(event.Event{
		Type:      eventType,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Data:      data,
	})
	return err
}
