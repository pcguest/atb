// SPDX-License-Identifier: MIT
// Package otel maps supported OpenTelemetry trace input into ATB events.
//
// OTLP/JSON ExportTraceServiceRequest payloads can be decoded and translated
// into canonical ATB events (internal/event) before append to a bundle. Binary
// protobuf and gRPC collector transports are outside this package's scope. The
// mapper recognises a bounded attribute set and retains additional attributes
// under digest-first privacy rules.
package otel
