// SPDX-License-Identifier: MIT
// Package event defines the canonical ATB event taxonomy.
// Event type strings follow the reverse-DNS dotted-namespace convention.
// Event type constants and generated registry metadata are generated from
// schemas/event.v1.json.
package event

//go:generate go run ../../tools/eventgen/main.go -schema ../../schemas/event.v1.json

// Canonical event type constants (TypeBundleManifest, TypeToolCall, ...) are
// generated from schemas/event.v1.json into types_generated.go. Do not declare
// them by hand here; add the type to the schema and run `go generate ./...`.

// Policy decision events.
const (
	FieldPolicySignature    = "policy_signature"
	FieldPolicySignerPubKey = "policy_signer_pubkey"
	FieldPolicyDocHash      = "policy_doc_hash"
	FieldPolicyDocSignature = "policy_doc_signature"
)

// EventInfo describes a registered event type with its metadata.
type EventInfo struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	// Profile lists the obligation profile IDs where this event is primary.
	// Multiple profiles are comma-separated. Empty means no specific profile.
	Profile string `json:"profile"`
	// Criticality is one of: "critical", "required", "informational", "".
	Criticality string `json:"criticality"`
}

// Registry is the legacy ordered list of all canonical event types.
//
// Deprecated: schemas/event.v1.json event_types is the source of truth. Use
// RegistryGenerated for generated metadata. Registry remains synchronized for
// callers that have not yet migrated to the generated registry.
var Registry = []EventInfo{
	{TypeBundleManifest, "Bundle manifest (seq 0, always first in a new bundle)", "atb.profile.privileged_tool_action,atb.profile.rag_answer,atb.profile.data_export,atb.profile.policy_decision,atb.profile.human_override,atb.profile.background_automation", "critical"},
	{TypeBundleAnchor, "RFC 3161 TSA timestamp anchor", "all", "required"},
	{TypeBundleSignature, "Ed25519 bundle signature", "all", "required"},
	{TypeSnapshot, "Bundle snapshot marker", "", "informational"},
	{TypeAIRequestReceived, "AI request received at app boundary", "atb.profile.rag_answer,atb.profile.privileged_tool_action,atb.profile.data_export,atb.profile.policy_decision,atb.profile.human_override", "critical"},
	{TypeAIResponseSent, "AI response sent from app boundary", "atb.profile.rag_answer", "required"},
	{TypeAILLMCall, "Canonical LLM lifecycle event for integrations", "", "informational"},
	{TypeAIToolExec, "Canonical tool execution lifecycle event for integrations", "", "informational"},
	{TypeAIChainRun, "Canonical chain or step lifecycle event for integrations", "", "informational"},
	{TypeAIPolicyDecision, "Policy engine decision (allow/deny)", "atb.profile.privileged_tool_action,atb.profile.rag_answer,atb.profile.data_export,atb.profile.policy_decision", "critical"},
	{TypeAIRetrievalExecuted, "Retrieval step executed (RAG)", "atb.profile.rag_answer", "required"},
	{TypeAIModelInvoked, "LLM invocation sent", "atb.profile.rag_answer", "critical"},
	{TypeAIModelOutput, "LLM output received", "atb.profile.rag_answer", "critical"},
	{TypeAIActionPrecommit, "Pre-commit record for a gated privileged action", "atb.profile.privileged_tool_action,atb.profile.data_export,atb.profile.policy_decision,atb.profile.human_override", "critical"},
	{TypeAIActionExecuted, "Privileged action executed through gate", "atb.profile.privileged_tool_action,atb.profile.data_export,atb.profile.human_override", "critical"},
	{TypeAIActionCommitted, "Privileged action committed to sink", "atb.profile.privileged_tool_action,atb.profile.data_export,atb.profile.human_override", "critical"},
	{TypeAIActionError, "Privileged action attempted but did not succeed (failed, blocked, timed out, or denied at the sink)", "", "required"},
	{TypeAIHumanApproval, "Human approval of an action or override", "atb.profile.privileged_tool_action,atb.profile.data_export,atb.profile.human_override", "required"},
	{TypeAIJobScheduled, "Background job scheduled", "atb.profile.background_automation", "critical"},
	{TypeAIJobStarted, "Background job started by worker", "atb.profile.background_automation", "critical"},
	{TypeAIJobStep, "Individual step within a background job", "atb.profile.background_automation", "required"},
	{TypeAIJobCompleted, "Background job completed", "atb.profile.background_automation", "critical"},
	{TypeDataExportPrecommit, "Pre-commit record for a data export", "atb.profile.data_export", "critical"},
	{TypeDataExportExecuted, "Data export executed to sink", "atb.profile.data_export", "critical"},
	{TypeDataExportError, "Data export attempted but did not succeed (failed, blocked, timed out, or denied at the sink)", "", "required"},
	{TypeDataRetentionPolicySet, "Deployment retention policy configured", "", "required"},
	{TypeDataRetentionPolicyChanged, "Deployment retention policy changed", "", "required"},
	{TypeDataRetentionEnforced, "Retention-relevant local operation completed or remote retention request accepted", "", "required"},
	{TypeDevSession, "Developer session marker (tooling use)", "", "informational"},
	{TypeCorroborationExternal, "External corroboration record (adapter-retrieved evidence)", "", "informational"},
	{TypeRAGIndex, "PageIndex document tree build record (index_hash, node_count)", "atb.profile.rag_answer", "required"},
	{TypeRAGRetrieval, "PageIndex reasoning-based retrieval result (node_id, page_start/end)", "atb.profile.rag_answer", "required"},
	{TypeToolCall, "Tool invocation recorded for session oversight", "", "required"},
	{TypeDataExport, "Data export outside session boundary", "", "required"},
	{TypeHumanOverride, "Human operator overrode an AI-recommended action", "", "required"},
	{TypeHumanApproval, "Human operator approved a pending action", "", "required"},
	{TypeCaptureScope, "Capture-coverage attestation written by atb intercept at startup: what the recorder can and cannot see", "", "required"},
	{TypeCaptureRejected, "Capture rejection or incomplete exchange (proxy-internal)", "", "required"},
	{TypeLLMRequest, "Captured upstream LLM API request (proxy-internal)", "", "informational"},
	{TypeLLMResponse, "Captured upstream LLM API response (proxy-internal)", "", "informational"},
	{TypeSessionClose, "Capture session closed (proxy-internal lifecycle marker)", "", "informational"},
	{TypeExchangeComplete, "Request/response exchange completed within a capture session (proxy-internal)", "", "informational"},
}
