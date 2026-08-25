// SPDX-License-Identifier: MIT

// Package emit provides the canonical Go surface for emitting the four
// oversight event types declared in docs/specification/events.md.
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

// IdentityEvidence carries digest-only caller-provided reviewer identity
// context. ATB preserves it but does not verify the asserted identity.
type IdentityEvidence struct {
	IdentityProvider  string
	Subject           string
	AuthContext       string
	AssertionType     string
	AssertionDigest   string
	RawEvidenceDigest string
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
	IdentityEvidence   *IdentityEvidence
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
	if evidence, err := identityEvidenceData(opts.IdentityEvidence); err != nil {
		return err
	} else if evidence != nil {
		data["identity_evidence"] = evidence
	}
	return e.append(event.TypeHumanOverride, data)
}

// HumanApprovalOptions carries fields for an atb.human.approval event.
type HumanApprovalOptions struct {
	ApprovedActionID string // required
	ActorID          string
	ApproverID       string
	Note             string
	IdentityEvidence *IdentityEvidence
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
	if evidence, err := identityEvidenceData(opts.IdentityEvidence); err != nil {
		return err
	} else if evidence != nil {
		data["identity_evidence"] = evidence
	}
	return e.append(event.TypeHumanApproval, data)
}

// ActionErrorOptions carries fields for an ai.action.error event.
type ActionErrorOptions struct {
	ActionID     string // required
	ErrorClass   string // required: failed|blocked|timeout|exception|denied_at_sink
	ActorID      string
	DetailDigest string // SHA-256 hex of the failure detail
	ActionType   string
	SessionID    string // override; uses Emitter.sessionID when empty
}

// ActionError emits an ai.action.error event recording a privileged action that
// was attempted but did not succeed. ActionID and ErrorClass are required.
func (e *Emitter) ActionError(opts ActionErrorOptions) error {
	if strings.TrimSpace(opts.ActionID) == "" {
		return fmt.Errorf("atb: ActionError requires ActionID")
	}
	if strings.TrimSpace(opts.ErrorClass) == "" {
		return fmt.Errorf("atb: ActionError requires ErrorClass")
	}
	data := map[string]any{
		"action_id":   strings.TrimSpace(opts.ActionID),
		"error_class": strings.TrimSpace(opts.ErrorClass),
	}
	if v := strings.TrimSpace(e.sessionIDFor(opts.SessionID)); v != "" {
		data["session_id"] = v
	}
	if v := strings.TrimSpace(opts.ActorID); v != "" {
		data["actor_id"] = v
	}
	if v := strings.TrimSpace(opts.DetailDigest); v != "" {
		data["error_detail_digest"] = v
	}
	if v := strings.TrimSpace(opts.ActionType); v != "" {
		data["action_type"] = v
	}
	return e.append(event.TypeAIActionError, data)
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

func identityEvidenceData(evidence *IdentityEvidence) (map[string]any, error) {
	if evidence == nil {
		return nil, nil
	}
	provider := strings.TrimSpace(evidence.IdentityProvider)
	subject := strings.TrimSpace(evidence.Subject)
	assertionType := strings.TrimSpace(evidence.AssertionType)
	assertionDigest := strings.TrimSpace(evidence.AssertionDigest)
	if provider == "" || subject == "" || assertionType == "" || assertionDigest == "" {
		return nil, fmt.Errorf("atb: identity evidence requires provider, subject, assertion type, and assertion digest")
	}
	switch assertionType {
	case "jwt", "x509", "saml", "opaque":
	default:
		// event.v1 constrains identity_evidence.assertion_type; reject early
		// so schema-validating readers never see an out-of-enum value.
		return nil, fmt.Errorf("atb: identity evidence assertion_type must be one of jwt, x509, saml, opaque")
	}
	data := map[string]any{
		"identity_provider": provider,
		"subject":           subject,
		"assertion_type":    assertionType,
		"assertion_digest":  assertionDigest,
	}
	if value := strings.TrimSpace(evidence.AuthContext); value != "" {
		data["auth_context"] = value
	}
	if value := strings.TrimSpace(evidence.RawEvidenceDigest); value != "" {
		data["raw_evidence_digest"] = value
	}
	return data, nil
}
