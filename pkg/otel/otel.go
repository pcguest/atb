// SPDX-License-Identifier: MIT
// Package otel maps supported OpenTelemetry trace input into ATB events.
//
// OTLP/JSON ExportTraceServiceRequest payloads can be decoded and translated
// into canonical ATB events (internal/event) before append to a bundle. Binary
// protobuf and gRPC collector transports, and broader GenAI semantic-convention
// compatibility, are not implemented here.
package otel
