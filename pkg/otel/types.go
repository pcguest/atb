// SPDX-License-Identifier: MIT
package otel

import "time"

// OTelSpan is a transport-layer representation of an OpenTelemetry span.
// Phase 9 scaffold only: fields mirror common OTLP span payloads without
// importing the OpenTelemetry SDK.
type OTelSpan struct {
	TraceID       string
	SpanID        string
	ParentSpanID  string
	Name          string
	Kind          string
	StartTime     time.Time
	EndTime       time.Time
	StatusCode    string
	StatusMessage string
	Attributes    map[string]any
	Events        []OTelSpanEvent
}

// OTelSpanEvent is a timed annotation on a span (OTel span events / logs).
type OTelSpanEvent struct {
	Name       string
	Timestamp  time.Time
	Attributes map[string]any
}

// OTelTrace groups spans received in one inbound batch (e.g. from a collector).
type OTelTrace struct {
	TraceID string
	Spans   []OTelSpan
}
