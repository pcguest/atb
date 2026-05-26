// SPDX-License-Identifier: MIT
package otel

import (
	"errors"
	"fmt"

	"github.com/pcguest/atb/internal/event"
)

// ErrNotImplemented indicates the Phase 9 translator is scaffold-only.
var ErrNotImplemented = errors.New("otel: translation not implemented")

// Translator maps inbound OpenTelemetry spans into canonical ATB events.
type Translator interface {
	Translate(span OTelSpan) (*event.Event, error)
}

// DefaultTranslator is the scaffold implementation of Translator.
type DefaultTranslator struct {
	// DefaultEventType is the ATB event type used when mapping is not yet specialised.
	DefaultEventType string
}

// Translate maps span to an ATB event. Scaffold: returns ErrNotImplemented.
func Translate(span OTelSpan) (*event.Event, error) {
	return DefaultTranslator{}.Translate(span)
}

// Translate implements Translator.
func (t DefaultTranslator) Translate(span OTelSpan) (*event.Event, error) {
	if span.TraceID == "" && span.SpanID == "" {
		return nil, fmt.Errorf("otel: span missing trace_id and span_id: %w", ErrNotImplemented)
	}
	_ = t.eventType()
	return nil, ErrNotImplemented
}

func (t DefaultTranslator) eventType() string {
	if t.DefaultEventType != "" {
		return t.DefaultEventType
	}
	return "atb.otel.span.mapped"
}
