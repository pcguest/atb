// SPDX-License-Identifier: MIT
package apiv1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/pcguest/atb/internal/event"
)

func TestComputeSchemaStatus(t *testing.T) {
	b := newTestBundle(t)
	// Complete tool.call (both required fields present).
	appendTestBundleEvent(t, b, event.TypeToolCall, map[string]any{
		"session_id": "sess-1",
		"tool_name":  "search",
	})
	// Incomplete tool.call (missing required tool_name).
	appendTestBundleEvent(t, b, event.TypeToolCall, map[string]any{
		"session_id": "sess-1",
	})
	// Undeclared type: not in the schema registry.
	appendTestBundleEvent(t, b, "custom.unknown.type", map[string]any{
		"foo": "bar",
	})

	srv := NewAPIServer(APIConfig{Bundle: b})
	status := srv.computeSchemaStatus()

	if status.DeclaredTypes != len(event.RegistryGenerated) {
		t.Errorf("DeclaredTypes = %d, want %d", status.DeclaredTypes, len(event.RegistryGenerated))
	}
	if status.SchemaSource != "schemas/event.v1.json" {
		t.Errorf("SchemaSource = %q", status.SchemaSource)
	}

	byType := make(map[string]EventTypeStatusDTO, len(status.Types))
	for _, ts := range status.Types {
		byType[ts.Type] = ts
	}

	toolCall, ok := byType[event.TypeToolCall]
	if !ok {
		t.Fatalf("tool.call missing from status types")
	}
	if !toolCall.Declared {
		t.Errorf("tool.call should be declared")
	}
	if toolCall.Observed != 2 {
		t.Errorf("tool.call Observed = %d, want 2", toolCall.Observed)
	}
	if toolCall.Incomplete != 1 {
		t.Errorf("tool.call Incomplete = %d, want 1", toolCall.Incomplete)
	}
	if !slices.Contains(toolCall.MissingFields, "tool_name") {
		t.Errorf("tool.call MissingFields = %v, want to contain tool_name", toolCall.MissingFields)
	}

	unknown, ok := byType["custom.unknown.type"]
	if !ok {
		t.Fatalf("undeclared type missing from status types")
	}
	if unknown.Declared {
		t.Errorf("custom.unknown.type should not be declared")
	}
	if unknown.Observed != 1 {
		t.Errorf("custom.unknown.type Observed = %d, want 1", unknown.Observed)
	}
	if !slices.Contains(status.UndeclaredTypes, "custom.unknown.type") {
		t.Errorf("UndeclaredTypes = %v, want to contain custom.unknown.type", status.UndeclaredTypes)
	}

	if status.IncompleteEvents < 1 {
		t.Errorf("IncompleteEvents = %d, want >= 1", status.IncompleteEvents)
	}
	// manifest + 3 appended events.
	if status.TotalEvents != 4 {
		t.Errorf("TotalEvents = %d, want 4", status.TotalEvents)
	}
}

func TestSchemaStatusEndpointAuthAndShape(t *testing.T) {
	b := newTestBundle(t)
	appendTestBundleEvent(t, b, event.TypeToolCall, map[string]any{
		"session_id": "sess-1",
		"tool_name":  "search",
	})
	_, handler := buildTestAPIServer(t, APIConfig{
		Bundle:       b,
		SessionToken: "test-token",
	})

	// Without the session token the endpoint is gated like the other read routes.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema/status", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("without token: got %d want %d", rr.Code, http.StatusUnauthorized)
	}

	// With the token it returns the contract status payload.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/schema/status", nil)
	req.Header.Set(sessionAuthHeader, "test-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("with token: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var got SchemaStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.DeclaredTypes != len(event.RegistryGenerated) {
		t.Errorf("DeclaredTypes = %d, want %d", got.DeclaredTypes, len(event.RegistryGenerated))
	}
	if got.TotalEvents != 2 {
		t.Errorf("TotalEvents = %d, want 2 (manifest + tool.call)", got.TotalEvents)
	}

	// Non-GET is rejected.
	req = httptest.NewRequest(http.MethodPost, "/api/v1/schema/status", nil)
	req.Header.Set(sessionAuthHeader, "test-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST: got %d want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
