// Package event defines the canonical ATB event model shared by hashing and bundles.
package event

// Event represents a single auditable event in an ATB bundle.
type Event struct {
	// Sequence is the 1-based position of this event in the bundle.
	Sequence int `json:"seq"`
	// PrevHash is the hex-encoded SHA-256 hash of the preceding event.
	// For the first event this MUST equal the genesis hash.
	PrevHash string `json:"prev_hash"`
	// Type is the event type identifier (e.g. "dev.session", "decision").
	Type string `json:"type"`
	// Data is the arbitrary payload associated with this event.
	Data interface{} `json:"data"`
	// ActorID identifies who performed the action in multi-tenant scenarios.
	ActorID *string `json:"actor_id,omitempty"`
	// OrgID identifies the organization context.
	OrgID *string `json:"org_id,omitempty"`
	// WorkspaceID identifies the workspace context within an org.
	WorkspaceID *string `json:"workspace_id,omitempty"`
}
