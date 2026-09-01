// SPDX-License-Identifier: MIT
package apiv1

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/event"
)

func TestInvestigationEndpointsExposeHumanFirstReadModels(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	appendTestBundleEvent(t, b, event.TypeAIContextUnit, map[string]any{
		"unit_id": "context-1",
		"kind":    "retrieved_passage",
		"digest":  "sha256:context-1",
		"source":  "policy-handbook",
	})
	appendTestBundleEvent(t, b, event.TypeAIContextOperation, map[string]any{
		"operation_id":    "assembly-1",
		"operation":       "assemble",
		"input_unit_ids":  []any{"context-1"},
		"output_unit_ids": []any{"context-1"},
		"input_digest":    "sha256:context-1",
		"output_digest":   "sha256:context-1",
	})
	appendTestBundleEvent(t, b, event.TypeAIRetrievalExecuted, map[string]any{
		"retrieval_id":  "retrieval-1",
		"request_id":    "request-1",
		"result_digest": "sha256:result-1",
	})

	_, handler := buildTestAPIServer(t, APIConfig{BundlePath: bundlePath, Bundle: b})

	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/investigation/overview", `"integrity_status":"VERIFIED"`},
		{"/api/v1/investigation/findings", `"findings"`},
		{"/api/v1/investigation/timeline", `"label":"Context unit recorded"`},
		{"/api/v1/investigation/context", `"name":"retrieval.performed"`},
		{"/api/v1/investigation/relationships", `"relationships"`},
		{"/api/v1/investigation/trust", `"proof_statement":"ATB proves the integrity and order of the records presented in a bundle."`},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), test.want) {
				t.Fatalf("body missing %q: %s", test.want, rr.Body.String())
			}
		})
	}
}

func TestInvestigationTamperBoundary(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath: bundlePath,
		Bundle:     b,
		VerifyErr:  errors.New("chain verification failed"),
	})

	for _, path := range []string{
		"/api/v1/investigation/overview",
		"/api/v1/investigation/trust",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status: got %d want %d body=%s", path, rr.Code, http.StatusOK, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), `"integrity_valid":false`) {
			t.Fatalf("%s must expose failed integrity: %s", path, rr.Body.String())
		}
	}

	for _, path := range []string{
		"/api/v1/investigation/findings",
		"/api/v1/investigation/timeline",
		"/api/v1/investigation/context",
		"/api/v1/investigation/relationships",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("%s status: got %d want %d body=%s", path, rr.Code, http.StatusForbidden, rr.Body.String())
		}
	}
}

func TestInvestigationReportUsesCoreIncidentRenderer(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	appendTestBundleEvent(t, b, "ai.tool.call", map[string]any{
		"session_id": "session-report",
		"tool_name":  "search",
	})
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save report fixture: %v", err)
	}
	_, handler := buildTestAPIServer(t, APIConfig{BundlePath: bundlePath, Bundle: b})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/investigation/report?format=markdown&session_id=session-report", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("report status = %d body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Disposition"); got != `attachment; filename="atb-incident-report.md"` {
		t.Fatalf("Content-Disposition = %q", got)
	}
	if !strings.Contains(rr.Body.String(), "# Incident report — session `session-report`") {
		t.Fatalf("report did not use incident renderer: %s", rr.Body.String())
	}
}
