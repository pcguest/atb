// SPDX-License-Identifier: MIT
package otel

import (
	"context"
	"errors"

	"github.com/pcguest/atb/internal/event"
)

// InboundTransport receives OpenTelemetry trace data from an external source
// (collector, gateway, or queue). Phase 9 scaffold only: no OTLP decoding yet.
//
// InboundTransport is intentionally separate from Receiver.Receive: transports
// ingest raw batches; Receiver translates spans into ATB events via Translator.
type InboundTransport interface {
	// Receive ingests one trace batch. Implementations will decode OTLP in a later phase.
	Receive(ctx context.Context, trace OTelTrace) error
}

// ReceiverResult holds span-to-event translation output from Receiver.Receive.
type ReceiverResult struct {
	Events       []*event.Event
	SkippedCount int
	Errors       []error // non-ErrNotImplemented translation errors only
}

// Receiver binds an optional InboundTransport to a Translator for span-to-event conversion.
type Receiver struct {
	Transport  InboundTransport
	Translator Translator
}

// Receive translates each span in trace through Translator.
// ErrNotImplemented from Translate increments SkippedCount and continues.
// Any other translation error aborts and is returned immediately.
func (r *Receiver) Receive(ctx context.Context, trace OTelTrace) (ReceiverResult, error) {
	if r.Translator == nil {
		return ReceiverResult{}, ErrNotImplemented
	}
	if r.Transport != nil {
		if err := r.Transport.Receive(ctx, trace); err != nil && !errors.Is(err, ErrNotImplemented) {
			return ReceiverResult{}, err
		}
	}
	var result ReceiverResult
	for _, span := range trace.Spans {
		ev, err := r.Translator.Translate(span)
		if errors.Is(err, ErrNotImplemented) {
			result.SkippedCount++
			continue
		}
		if err != nil {
			return result, err
		}
		if ev != nil {
			result.Events = append(result.Events, ev)
		}
	}
	return result, nil
}

// StubTransport is a no-op InboundTransport for tests and wiring checks.
type StubTransport struct{}

// Receive implements InboundTransport.
func (StubTransport) Receive(ctx context.Context, trace OTelTrace) error {
	_ = ctx
	_ = trace
	return ErrNotImplemented
}
