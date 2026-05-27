// SPDX-License-Identifier: MIT
// Package ingestion defines the Custos boundary for receiving tool events and
// producing ATB event DTOs.
package ingestion

import atbv1 "github.com/pcguest/atb/pkg/api/v1"

// ToolEvent is the normalised input observed from an authorised AI tool.
type ToolEvent struct {
	ToolName  string
	Vendor    string
	Kind      string
	Endpoint  string
	ActorID   string
	OrgID     string
	TeamID    string
	Payload   []byte
	Timestamp string
}

// Ingestor receives authorised tool events and maps them into ATB event DTOs.
type Ingestor interface {
	// TODO: define validation, persistence, and bundle append semantics.
	Receives(toolEvent ToolEvent) (*atbv1.EventRecordDTO, error)
}
