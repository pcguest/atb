// SPDX-License-Identifier: MIT
package apiv1

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
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
		{"/api/v1/investigation/trust", `"proof_statement":"ATB proves the integrity and order of records presented in a bundle."`},
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

func TestInvestigationCoverageSurvivesValidIntegrityClone(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath: bundlePath,
		Bundle:     b,
		ProfileReport: &ProfileReportSummary{
			ProfileID:      "atb.profile.privileged_tool_action",
			CoverageScore:  0.76,
			CoverageGrade:  "Moderate coverage",
			IntegrityValid: true,
		},
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
		if !strings.Contains(rr.Body.String(), `"coverage_score":0.76`) {
			t.Fatalf("%s must keep assessed coverage_score: %s", path, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), `"coverage_grade":"Moderate coverage"`) {
			t.Fatalf("%s must keep coverage_grade: %s", path, rr.Body.String())
		}
	}
}

func TestInvestigationTamperBoundary(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	appendTestBundleEvent(t, b, event.TypeCorroborationExternal, map[string]any{"source": "untrusted"})
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath: bundlePath,
		Bundle:     b,
		VerifyErr:  errors.New("chain verification failed"),
		ProfileReport: &ProfileReportSummary{
			ProfileID:      "atb.profile.rag_answer",
			CoverageScore:  0.9,
			CoverageGrade:  "High coverage",
			IntegrityValid: false,
		},
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
		if strings.Contains(rr.Body.String(), `"coverage_score"`) {
			t.Fatalf("%s must omit coverage_score when integrity failed: %s", path, rr.Body.String())
		}
		if strings.Contains(rr.Body.String(), `"external_corroboration":true`) {
			t.Fatalf("invalid evidence asserted corroboration: %s", rr.Body.String())
		}
		if path == "/api/v1/investigation/overview" && strings.Contains(rr.Body.String(), `"finding_count":`) {
			if !strings.Contains(rr.Body.String(), `"finding_count":0`) {
				t.Fatalf("overview must not invent findings on invalid integrity: %s", rr.Body.String())
			}
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

func TestInvestigationReportRejectsTamperAfterServerLoad(t *testing.T) {
	path, b, session := investigationReportXSSFixture(t)
	_, handler := buildTestAPIServer(t, APIConfig{BundlePath: path, Bundle: b})
	b.Records[len(b.Records)-1].Event.Data = map[string]any{"session_id": session, "tool_name": "tampered sentinel"}
	if err := b.Save(path); err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"json", "markdown"} {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/investigation/report?format="+format+"&session_id="+session, nil))
		if rr.Code != http.StatusForbidden || strings.Contains(rr.Body.String(), "tampered sentinel") {
			t.Fatalf("untrusted report: status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func TestInvestigationFindingsDoNotCrossBundles(t *testing.T) {
	path, b, session := investigationReportXSSFixture(t)
	other, err := bundle.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := other.Append(event.TypeToolCall, map[string]any{"session_id": session, "tool_name": "other"}); err != nil {
		t.Fatal(err)
	}
	if err := other.Save(filepath.Join(filepath.Dir(path), "other.atb")); err != nil {
		t.Fatal(err)
	}
	_, handler := buildTestAPIServer(t, APIConfig{BundlePath: path, Bundle: b})
	request := func(suffix string) string {
		t.Helper()
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/investigation/findings"+suffix, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		return rr.Body.String()
	}
	if got, want := request("?bundle_dir="+url.QueryEscape(filepath.Dir(path))), request(""); got != want {
		t.Fatalf("directory changed bundle findings: got=%s want=%s", got, want)
	}
}

func TestInvestigationReportJSONEncodesWithoutRawHTML(t *testing.T) {
	bundlePath, b, sessionID := investigationReportXSSFixture(t)
	_, handler := buildTestAPIServer(t, APIConfig{BundlePath: bundlePath, Bundle: b})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/investigation/report?format=json&session_id="+sessionID, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("report status = %d body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
	body := rr.Body.String()
	if strings.Contains(body, "<script>") {
		t.Fatalf("JSON body contained raw HTML: %s", body)
	}
	var decoded map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("body is not encoded JSON: %v", err)
	}
}

func TestInvestigationReportMarkdownEscapesHTML(t *testing.T) {
	bundlePath, b, sessionID := investigationReportXSSFixture(t)
	_, handler := buildTestAPIServer(t, APIConfig{BundlePath: bundlePath, Bundle: b})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/investigation/report?format=markdown&session_id="+sessionID, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("report status = %d body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/markdown") {
		t.Fatalf("Content-Type = %q", got)
	}
	body := rr.Body.String()
	if strings.Contains(body, "<script>") {
		t.Fatalf("markdown body contained raw HTML: %s", body)
	}
	if !strings.Contains(body, "\\<script\\>") {
		t.Fatalf("markdown body did not neutralize injected field: %s", body)
	}
}

func investigationReportXSSFixture(t *testing.T) (string, *bundle.Bundle, string) {
	t.Helper()
	const sessionID = "session-xss"
	bundlePath, b := createRichTestBundle(t)
	appendTestBundleEvent(t, b, event.TypeToolCall, map[string]any{
		"session_id": sessionID,
		"tool_name":  "<script>alert(1)</script>",
	})
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save xss fixture: %v", err)
	}
	return bundlePath, b, sessionID
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
	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
	if !strings.Contains(rr.Body.String(), "# Incident report — session `session-report`") {
		t.Fatalf("report did not use incident renderer: %s", rr.Body.String())
	}
}

func TestEventFamilyIsDisplayOverlay(t *testing.T) {
	cases := map[string]string{
		event.TypeAILLMCall:           "llm",
		event.TypeLLMRequest:          "llm",
		event.TypeAIModelInvoked:      "llm",
		event.TypeToolCall:            "tool",
		event.TypeAIActionExecuted:    "action",
		event.TypeMcpOperation:        "action",
		event.TypeAIContextUnit:       "context",
		event.TypeAIRetrievalExecuted: "context",
		event.TypeRAGRetrieval:        "context",
		event.TypeAIHumanApproval:     "human",
		event.TypeHumanApproval:       "human",
		event.TypeDataExportExecuted:  "export",
		event.TypeDevSession:          "other",
	}
	for eventType, want := range cases {
		if got := eventFamily(eventType); got != want {
			t.Fatalf("eventFamily(%q)=%q want %q", eventType, got, want)
		}
	}
}
