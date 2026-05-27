// SPDX-License-Identifier: MIT
// Package ingestion defines the Custos boundary for receiving tool events and
// producing ATB event DTOs.
package ingestion

import (
	"github.com/pcguest/custos/signing"

	atbv1 "github.com/pcguest/atb/pkg/api/v1"
)

// ToolEvent is the normalised input observed from an authorised AI tool.
type ToolEvent struct {
	ToolName string
	Vendor   string
	Kind     string
	Endpoint string
	ActorID  string
	OrgID    string
	TeamID   string
	Payload  []byte
	// SigningResult holds the automatic signing outcome recorded by ATB core
	// before this event arrived at the ingestion boundary.
	SigningResult *signing.BundleSigningResult
	Timestamp     string
}

// Ingestor receives tool events that have already been automatically signed by
// ATB core. SigningResult must be non-nil for authorised tools unless the
// signing key is misconfigured.
type Ingestor interface {
	// TODO: define validation, persistence, and bundle append semantics.
	Receives(toolEvent ToolEvent) (*atbv1.EventRecordDTO, error)
}
