// SPDX-License-Identifier: MIT
// Package otel defines the Phase 9 inbound transport scaffold for OpenTelemetry.
//
// Spans received from collectors or gateways are translated into canonical ATB
// events (internal/event) before append to a bundle. Full OTLP decode and GenAI
// semconv mapping are deferred to a later implementation phase.
package otel
