// SPDX-License-Identifier: MIT
package mcp

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBuildOperationEventDigestsSensitiveMetadataAndBindsTrace(t *testing.T) {
	t.Parallel()
	ttl := 5000
	event, err := BuildOperationEvent(OperationInput{
		ProtocolVersion:      ProtocolVersion,
		Method:               "tools/call",
		Name:                 "search",
		Status:               "success",
		Request:              map[string]any{"query": "controls"},
		Result:               map[string]any{"count": 1},
		ClientMetadata:       map[string]any{"name": "agent"},
		ClientCapabilities:   map[string]any{"extensions": map[string]any{}},
		AuthorizationContext: map[string]any{"authorization": "Bearer secret-token"},
		Traceparent:          "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		Tracestate:           "vendor=sensitive",
		Baggage:              "tenant=sensitive",
		TTLMS:                &ttl,
		CacheScope:           "private",
		Timestamp:            time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("BuildOperationEvent() error = %v", err)
	}
	if event.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" || event.SpanID != "00f067aa0ba902b7" {
		t.Fatalf("trace binding = %s/%s", event.TraceID, event.SpanID)
	}
	encoded, err := json.Marshal(event.Data)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"secret-token", "vendor=sensitive", "tenant=sensitive"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("event data retained sensitive value %q: %s", secret, encoded)
		}
	}
}

func TestBuildOperationEventRejectsUnsupportedStatus(t *testing.T) {
	t.Parallel()
	_, err := BuildOperationEvent(OperationInput{ProtocolVersion: ProtocolVersion, Method: "tools/call", Status: "unknown", Request: map[string]any{}})
	if err == nil {
		t.Fatal("expected unsupported status error")
	}
}
