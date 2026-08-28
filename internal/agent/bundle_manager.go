// SPDX-License-Identifier: MIT
package agent

import (
	"context"
	"errors"
	"time"
)

// ErrSessionNotFound indicates that no open session matches the given ID.
var ErrSessionNotFound = errors.New("agent: session not found")

// ErrSessionClosed indicates that the session is no longer accepting events.
var ErrSessionClosed = errors.New("agent: session closed")

// SessionID is an opaque identifier for an agent-managed bundle session.
type SessionID string

// String returns the underlying session identifier.
func (id SessionID) String() string {
	return string(id)
}

// OpenParams describes how the Agent should prepare a new bundle session.
type OpenParams struct {
	ActorID    string
	PurposeTag string
	ProfileID  string
	// BundlePath optionally overrides the on-disk location. When empty the
	// manager assigns a path under the workspace data directory.
	BundlePath string
}

// PendingEvent is a validated event queued during a session before the file
// manager commits hash-chained records on close.
type PendingEvent struct {
	EventType string
	Payload   string
}

// CloseSessionOpts carries optional parameters when closing a session.
type CloseSessionOpts struct {
	SnapshotName string
}

// BundleMetadata summarises a closed bundle session.
type BundleMetadata struct {
	SessionID  SessionID `json:"session_id"`
	Path       string    `json:"path"`
	ProfileID  string    `json:"profile_id,omitempty"`
	HeadHash   string    `json:"head_hash"`
	EventCount int       `json:"event_count"`
	CreatedAt  time.Time `json:"created_at"`
	ClosedAt   time.Time `json:"closed_at"`
}

// BundleManager coordinates bundle session lifecycle inside the Agent.
// Implementations must be safe for concurrent use.
type BundleManager interface {
	OpenSession(ctx context.Context, params OpenParams) (SessionID, error)
	AppendEvent(ctx context.Context, sessionID SessionID, event PendingEvent) error
	CloseSession(ctx context.Context, sessionID SessionID, opts CloseSessionOpts) (BundleMetadata, error)
	// Shutdown releases manager resources. Open sessions may be discarded.
	Shutdown(ctx context.Context) error
}
