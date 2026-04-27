// SPDX-License-Identifier: MIT
// Package event defines the canonical ATB event taxonomy.
// Event type strings follow the reverse-DNS dotted-namespace convention.
// These constants are the authoritative source of truth across the Go
// runtime, CLI, and obligation profiles.
package event

// Bundle lifecycle events.
const (
	TypeBundleManifest  = "atb.bundle.manifest"
	TypeBundleAnchor    = "atb.bundle.anchor"
	TypeBundleSignature = "atb.bundle.signature"
	// TypeSnapshot data may include optional Article 14 human oversight fields:
	// actor_id, actor_role, and oversight_note.
	TypeSnapshot     = "atb.snapshot"
	TypeBundlePushed = "atb.bundle.pushed"
)

// AI request and response events.
const (
	TypeAIRequestReceived = "ai.request.received"
	TypeAIResponseSent    = "ai.response.sent"
	TypeAILLMCall         = "ai.llm.call"
	TypeAIToolExec        = "ai.tool.exec"
	TypeAIChainRun        = "ai.chain.run"
)

// Policy decision events.
const (
	TypeAIPolicyDecision    = "ai.policy.decision"
	FieldPolicySignature    = "policy_signature"
	FieldPolicySignerPubKey = "policy_signer_pubkey"
	FieldPolicyDocHash      = "policy_doc_hash"
	FieldPolicyDocSignature = "policy_doc_signature"
)

// RAG events.
const (
	TypeAIRetrievalExecuted = "ai.retrieval.executed"
	TypeAIModelInvoked      = "ai.model.invoked"
	TypeAIModelOutput       = "ai.model.output"

	// PageIndex-specific RAG events (Mode B in-process integration).
	// See sdk/python/atb/pageindex.py and THIRD_PARTY_NOTICES.
	TypeRAGIndex     = "atb.event.rag_index"
	TypeRAGRetrieval = "atb.event.rag_retrieval"
)

// Privileged action / ACP-gating events.
const (
	TypeAIActionPrecommit = "ai.action.precommit"
	TypeAIActionExecuted  = "ai.action.executed"
	TypeAIActionCommitted = "ai.action.committed"
)

// Human oversight events.
const (
	TypeAIHumanApproval     = "ai.human.approval"
	TypeAIOverrideRequested = "ai.override.requested"
)

// Background automation events.
const (
	TypeAIJobScheduled = "ai.job.scheduled"
	TypeAIJobStarted   = "ai.job.started"
	TypeAIJobStep      = "ai.job.step"
	TypeAIJobCompleted = "ai.job.completed"
)

// Data export events.
const (
	TypeDataExportPrecommit = "data.export.precommit"
	TypeDataExportExecuted  = "data.export.executed"
)

// Developer / session events used in tests and tooling.
const (
	TypeDevSession    = "dev.session"
	TypeSnapshotBuild = "snapshot.build"
)

// Corroboration events record evidence retrieved from external systems.
const (
	TypeCorroborationExternal = "atb.corroboration.external"
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

// Registry is the ordered list of all canonical event types.
// It is the source of truth used by `atb events`.
var Registry = []EventInfo{
	{TypeBundleManifest, "Bundle manifest (seq 0, always first in a new bundle)", "atb.profile.privileged_tool_action,atb.profile.rag_answer,atb.profile.data_export,atb.profile.policy_decision,atb.profile.human_override,atb.profile.background_automation", "critical"},
	{TypeBundleAnchor, "RFC 3161 TSA timestamp anchor", "all", "required"},
	{TypeBundleSignature, "Ed25519 bundle signature", "all", "required"},
	{TypeSnapshot, "Bundle snapshot marker", "", "informational"},
	{TypeBundlePushed, "Bundle pushed to remote WORM target (atb push)", "", "informational"},
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
	{TypeAIHumanApproval, "Human approval of an action or override", "atb.profile.privileged_tool_action,atb.profile.data_export,atb.profile.human_override", "required"},
	{TypeAIOverrideRequested, "Human override requested", "", "critical"},
	{TypeAIJobScheduled, "Background job scheduled", "atb.profile.background_automation", "critical"},
	{TypeAIJobStarted, "Background job started by worker", "atb.profile.background_automation", "critical"},
	{TypeAIJobStep, "Individual step within a background job", "atb.profile.background_automation", "critical"},
	{TypeAIJobCompleted, "Background job completed", "atb.profile.background_automation", "critical"},
	{TypeDataExportPrecommit, "Pre-commit record for a data export", "atb.profile.data_export", "critical"},
	{TypeDataExportExecuted, "Data export executed to sink", "atb.profile.data_export", "critical"},
	{TypeDevSession, "Developer session marker (tooling use)", "", "informational"},
	{TypeSnapshotBuild, "Build snapshot (tooling use)", "", "informational"},
	{TypeCorroborationExternal, "External corroboration record (adapter-retrieved evidence)", "", "informational"},
	{TypeRAGIndex, "PageIndex document tree build record (index_hash, node_count)", "atb.profile.rag_answer", "required"},
	{TypeRAGRetrieval, "PageIndex reasoning-based retrieval result (node_id, page_start/end)", "atb.profile.rag_answer", "required"},
}
