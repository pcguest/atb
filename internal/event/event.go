// SPDX-License-Identifier: MIT
// Package event defines the canonical ATB event model shared by hashing and bundles.
package event

// Event represents a single auditable event in an ATB bundle.
type Event struct {
	// Sequence is the event position in the bundle.
	// New bundles reserve seq 0 for the manifest record; subsequent records use 1-based positions.
	Sequence int `json:"seq"`
	// PrevHash is the hex-encoded SHA-256 hash of the preceding event.
	// For the manifest record, and for legacy manifest-less bundles, the first event MUST equal the genesis hash.
	PrevHash string `json:"prev_hash"`
	// Type is the event type identifier (e.g. "dev.session", "workflow.decision").
	Type string `json:"type"`
	// HashAlgo identifies the algorithm used to compute this event's hash.
	// Defaults to "sha256". Present on all events emitted by this runtime.
	HashAlgo string `json:"hash_algo,omitempty"`
	// Data is the arbitrary payload associated with this event.
	Data interface{} `json:"data"`
	// ActorID is caller-asserted actor attribution context.
	ActorID *string `json:"actor_id,omitempty"`
	// OrgID is caller-asserted organisation attribution context.
	OrgID *string `json:"org_id,omitempty"`
	// WorkspaceID is caller-asserted workspace attribution context.
	WorkspaceID *string `json:"workspace_id,omitempty"`
	// Timestamp is the RFC 3339 UTC time at which the event was created.
	// When present, it is included in the canonical hash.
	Timestamp string `json:"timestamp,omitempty"`
	// TraceID is the W3C trace context trace identifier (32 hex chars).
	TraceID string `json:"trace_id,omitempty"`
	// SpanID is the W3C trace context span identifier (16 hex chars).
	SpanID string `json:"span_id,omitempty"`
	// ParentSpanID is the W3C trace context parent span identifier (16 hex chars).
	ParentSpanID string `json:"parent_span_id,omitempty"`

	// Acquisition provides provenance metadata for how this event was acquired.
	// This field is optional and only populated for events acquired through import/capture.
	Acquisition *AcquisitionInfo `json:"acquisition,omitempty"`
}

// AcquisitionInfo provides provenance metadata for how an event was acquired.
// This distinguishes between evidence ATB witnessed live vs evidence acquired
// retrospectively, and provides traceability to the original source representation.
type AcquisitionInfo struct {
	// Mode indicates how this event was acquired.
	// One of: "live", "retrospective", "replayed", "derived", "manual".
	Mode string `json:"mode,omitempty"`

	// SourceSystem identifies the originating system (e.g., "openai", "anthropic", "langsmith", "otel", "chatlog").
	SourceSystem string `json:"source_system,omitempty"`

	// SourceRecordID is a stable identifier for the source record within the source system.
	// For chatlog imports, this is the line number. For OTel, this is the span ID.
	SourceRecordID string `json:"source_record_id,omitempty"`

	// SourceTimestamp is the original timestamp from the source record.
	SourceTimestamp string `json:"source_timestamp,omitempty"`

	// AcquiredAt is the RFC 3339 timestamp when ATB acquired this record.
	AcquiredAt string `json:"acquired_at,omitempty"`

	// SourceDigest is the SHA-256 digest of the raw source representation BEFORE
	// semantic translation. This enables verification that the imported ATB evidence
	// came from the same acquired source representation.
	SourceDigest string `json:"source_digest,omitempty"`

	// Adapter identifies the importer/translator used (e.g., "atb.chatlog.generic-jsonl",
	// "atb.chatlog.openai-jsonl", "atb.otel.otlp-json").
	Adapter string `json:"adapter,omitempty"`

	// AdapterVersion is the version of the adapter/translator used.
	AdapterVersion string `json:"adapter_version,omitempty"`

	// Checkpoint is the acquisition checkpoint state when this record was acquired.
	// This enables incremental continuation and reconciliation.
	Checkpoint *CheckpointInfo `json:"checkpoint,omitempty"`
}

// CheckpointInfo represents an acquisition checkpoint for incremental continuation.
type CheckpointInfo struct {
	// SourceSystem identifies the source system this checkpoint belongs to.
	SourceSystem string `json:"source_system"`

	// AcquisitionStream identifies the acquisition stream (e.g., file path, OTel endpoint).
	AcquisitionStream string `json:"acquisition_stream"`

	// Position is the position/watermark in the source stream (e.g., byte offset, line number, cursor).
	Position string `json:"position"`

	// ObservedAt is the RFC 3339 timestamp when this checkpoint was observed.
	ObservedAt string `json:"observed_at"`

	// Adapter identifies the adapter used for this checkpoint.
	Adapter string `json:"adapter"`

	// AdapterVersion is the version of the adapter used.
	AdapterVersion string `json:"adapter_version,omitempty"`
}

// SourceIdentity represents a stable identity for a source record.
// Used for deduplication and reconciliation across imports.
type SourceIdentity struct {
	// System identifies the source system (e.g., "openai", "otel", "chatlog").
	System string `json:"system"`

	// RecordID is the stable identifier for the record within the source system.
	RecordID string `json:"record_id"`

	// Derived indicates whether this identity was derived (vs. native provider ID).
	Derived bool `json:"derived,omitempty"`
}
