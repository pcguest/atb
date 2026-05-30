// SPDX-License-Identifier: MIT
// Package corroborate defines shared types for external evidence adapters.
package corroborate

import (
	"errors"
	"time"
)

// ErrEventTypeNotSupported indicates an adapter cannot map the supplied event type.
var ErrEventTypeNotSupported = errors.New(
	// Added: The shared sentinel lets all adapters expose one errors.Is-compatible value.
	"event type not supported by this corroborator",
)

// Actor identifies the recorded event actor for corroboration queries.
type Actor struct {
	// Email is the actor email address when the bundle captured one.
	Email string
	// DisplayName is the human-readable actor label when email is absent.
	DisplayName string
}

// Event is the adapter-facing subset of an ATB event used for corroboration.
type Event struct {
	// Type is the canonical ATB event type.
	Type string
	// OccurredAt is the event timestamp used to narrow external evidence lookups.
	OccurredAt time.Time
	// Metadata contains adapter-specific lookup hints copied from the recorded event payload.
	Metadata map[string]string
	// Actor carries caller-asserted actor attribution for external evidence lookups.
	Actor Actor
}

// CorrobResult reports whether an external source matched a bundle event.
type CorrobResult struct {
	// Matched is false while adapters only construct query URLs.
	Matched bool
	// Source identifies the external corroboration source.
	Source string
	// Timestamp records the event time associated with the corroboration query.
	Timestamp time.Time
	// Note describes the corroboration status without certifying the external result.
	Note string
	// QueryURL is the fully-constructed URL for manual or
	// automated verification against the external source.
	// ATB constructs this URL but does not make the external
	// request and does not certify the result.
	QueryURL string
}

// Corroborator verifies an ATB event against an external evidence source.
type Corroborator interface {
	// Verify maps an ATB event to an external corroboration result.
	Verify(event *Event) (CorrobResult, error)
}
