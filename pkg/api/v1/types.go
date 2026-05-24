// SPDX-License-Identifier: MIT
package apiv1

// VerificationResponse reports top-level integrity status for a bundle view session.
type VerificationResponse struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	BundlePath  string `json:"bundle_path"`
	ChainLength int    `json:"chain_length"`
	HeadHash    string `json:"head_hash,omitempty"`
}

// BundleMetaResponse contains aggregate bundle metadata used by dashboard summary widgets.
type BundleMetaResponse struct {
	BundlePath      string         `json:"bundle_path"`
	EventCount      int            `json:"event_count"`
	TypeCounts      map[string]int `json:"type_counts"`
	FirstTimestamp  string         `json:"first_timestamp,omitempty"`
	LastTimestamp   string         `json:"last_timestamp,omitempty"`
	GenesisHash     string         `json:"genesis_hash"`
	VerifiedAt      string         `json:"verified_at"`
	Verified        bool           `json:"verified"`
	VerificationMsg string         `json:"verification_message"`
}

// EventRecordDTO is a sanitized record representation for paginated event API responses.
type EventRecordDTO struct {
	Sequence     int         `json:"seq"`
	Type         string      `json:"type"`
	Hash         string      `json:"hash"`
	PrevHash     string      `json:"prev_hash"`
	Timestamp    string      `json:"timestamp,omitempty"`
	TraceID      string      `json:"trace_id,omitempty"`
	SpanID       string      `json:"span_id,omitempty"`
	ParentSpanID string      `json:"parent_span_id,omitempty"`
	Data         interface{} `json:"data"`
}

// BundleEventsResponse contains a paginated slice of event records.
type BundleEventsResponse struct {
	Offset int              `json:"offset"`
	Limit  int              `json:"limit"`
	Total  int              `json:"total"`
	Events []EventRecordDTO `json:"events"`
}

// GraphNodeDTO models a single node in trace/span visualizations.
type GraphNodeDTO struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

// GraphEdgeDTO models directed links in trace/span visualizations.
type GraphEdgeDTO struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
}

// BundleGraphResponse returns graph nodes and edges for React Flow rendering.
type BundleGraphResponse struct {
	Nodes []GraphNodeDTO `json:"nodes"`
	Edges []GraphEdgeDTO `json:"edges"`
}

// PrivacyRevealRequest asks the API to reveal one masked field.
type PrivacyRevealRequest struct {
	Sequence  int    `json:"seq"`
	FieldPath string `json:"field_path"`
	Reason    string `json:"reason,omitempty"`
}

// PrivacyRevealResponse returns a single revealed field value.
type PrivacyRevealResponse struct {
	Sequence  int         `json:"seq"`
	FieldPath string      `json:"field_path"`
	Value     interface{} `json:"value"`
}

// APIError is a common error envelope used by JSON API endpoints.
type APIError struct {
	Error      string `json:"error"`
	RetryAfter int    `json:"retry_after,omitempty"`
}

// FailureDTO is a sanitized obligation-failure entry for API responses.
type FailureDTO struct {
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}

// ProvabilityGapDTO describes a conditional limitation and closure criteria.
type ProvabilityGapDTO struct {
	Gap        string `json:"gap"`
	Layer      string `json:"layer"`
	Mitigation string `json:"mitigation"`
	ClosedWhen string `json:"closed_when"`
}

// ProfileReportSummary contains the profile evaluation and CAS outcome for the viewer.
// GET /api/v1/bundle/profile returns 204 No Content when no verify report has been computed.
// POST /api/v1/bundle/verify runs (or re-runs) verify and returns a fresh summary.
type ProfileReportSummary struct {
	ProfileID          string              `json:"profile_id"`
	Pass               bool                `json:"pass"`
	ChainValid         bool                `json:"chain_valid"`
	AnchorStatus       string              `json:"anchor_status"`
	CASScore           float64             `json:"cas_score,omitempty"`
	CASGrade           string              `json:"cas_grade,omitempty"`
	SubScores          map[string]float64  `json:"sub_scores,omitempty"`
	CriticalFailures   []FailureDTO        `json:"critical_failures"`
	Warnings           []string            `json:"warnings"`
	CorroborationBonus float64             `json:"corroboration_bonus,omitempty"`
	EffectiveScore     float64             `json:"effective_score,omitempty"`
	ProvabilityGaps    []ProvabilityGapDTO `json:"provability_gaps,omitempty"`
}
