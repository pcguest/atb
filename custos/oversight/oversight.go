// SPDX-License-Identifier: MIT
// Package oversight defines human review boundaries for Custos.
package oversight

// NB: ATB bundles are automatically signed at capture time by
// internal/signer.NewLocalSigner. Human review here is post-facto
// flagging of already-signed and ingested bundles only — it is not
// a signing gate.
// ReviewRequest describes an event or bundle that requires human review.
type ReviewRequest struct {
	BundleID      string
	EventID       string
	Summary       string
	RequiresHuman bool
}

// Reviewer queues and resolves human review requests.
type Reviewer interface {
	// TODO: define queue storage, assignment, and SLA semantics.
	Queue(req ReviewRequest) error
	// TODO: define approval authority and audit record semantics.
	Approve(reviewID string) error
	// TODO: define rejection reason requirements and escalation semantics.
	Reject(reviewID, reason string) error
}
