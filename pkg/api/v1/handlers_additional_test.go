package apiv1

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

func createRichTestBundle(t *testing.T) (string, *bundle.Bundle) {
	t.Helper()

	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	b := newTestBundle(t)
	appendTestBundleEventAt(t, b, "ai.chain.start", map[string]interface{}{
		"timestamp": "2026-03-12T00:00:00Z",
		"trace_id":  "trace-1",
		"span_id":   "span-1",
		"email":     "auditor@example.com",
	}, "2026-03-12T00:00:00Z")
	appendTestBundleEventAt(t, b, "ai.tool.call", map[string]interface{}{
		"timestamp":      "2026-03-12T00:00:01Z",
		"trace_id":       "trace-1",
		"span_id":        "span-2",
		"parent_span_id": "span-1",
		"user_id":        "usr-1",
		"profile": map[string]interface{}{
			"bio": "hidden",
		},
	}, "2026-03-12T00:00:01Z")
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}
	return bundlePath, b
}

func decodeResponseJSON[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode JSON response: %v body=%s", err, rr.Body.String())
	}
	return out
}

func TestVerificationEndpointStates(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		RevealAuthToken: "test-token",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/verification", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("verification status: got %d want %d", rr.Code, http.StatusOK)
	}
	valid := decodeResponseJSON[VerificationResponse](t, rr)
	if valid.Status != "valid" {
		t.Fatalf("verification status payload: got %q want %q", valid.Status, "valid")
	}
	if valid.ChainLength != 3 {
		t.Fatalf("chain length: got %d want %d", valid.ChainLength, 3)
	}
	if strings.TrimSpace(valid.HeadHash) == "" {
		t.Fatalf("head hash should be present")
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/verification", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method guard status: got %d want %d", rr.Code, http.StatusMethodNotAllowed)
	}

	_, invalidHandler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		VerifyErr:       errors.New("tamper"),
		RevealAuthToken: "test-token",
	})
	req = httptest.NewRequest(http.MethodGet, "/api/v1/verification", nil)
	rr = httptest.NewRecorder()
	invalidHandler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("invalid verification status: got %d want %d", rr.Code, http.StatusOK)
	}
	invalid := decodeResponseJSON[VerificationResponse](t, rr)
	if invalid.Status != "invalid" {
		t.Fatalf("invalid verification payload: got %q want %q", invalid.Status, "invalid")
	}
}

func TestBundleMetaEventsAndGraphEndpoints(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		RevealAuthToken: "test-token",
	})

	metaReq := httptest.NewRequest(http.MethodGet, "/api/v1/bundle/meta", nil)
	metaRR := httptest.NewRecorder()
	handler.ServeHTTP(metaRR, metaReq)
	if metaRR.Code != http.StatusOK {
		t.Fatalf("meta status: got %d want %d body=%s", metaRR.Code, http.StatusOK, metaRR.Body.String())
	}
	meta := decodeResponseJSON[BundleMetaResponse](t, metaRR)
	if meta.EventCount != 3 {
		t.Fatalf("meta event_count: got %d want %d", meta.EventCount, 3)
	}
	if meta.TypeCounts[bundle.ManifestEventType] != 1 || meta.TypeCounts["ai.chain.start"] != 1 || meta.TypeCounts["ai.tool.call"] != 1 {
		t.Fatalf("unexpected type counts: %+v", meta.TypeCounts)
	}
	if meta.FirstTimestamp != "2026-03-12T00:00:00Z" || meta.LastTimestamp != "2026-03-12T00:00:01Z" {
		t.Fatalf("unexpected timestamps: first=%q last=%q", meta.FirstTimestamp, meta.LastTimestamp)
	}

	eventsReq := httptest.NewRequest(http.MethodGet, "/api/v1/bundle/events?offset=1&limit=1", nil)
	eventsRR := httptest.NewRecorder()
	handler.ServeHTTP(eventsRR, eventsReq)
	if eventsRR.Code != http.StatusOK {
		t.Fatalf("events status: got %d want %d body=%s", eventsRR.Code, http.StatusOK, eventsRR.Body.String())
	}
	events := decodeResponseJSON[BundleEventsResponse](t, eventsRR)
	if events.Total != 3 || len(events.Events) != 1 {
		t.Fatalf("unexpected events paging: total=%d len=%d", events.Total, len(events.Events))
	}
	if events.Events[0].Timestamp != "2026-03-12T00:00:00Z" {
		t.Fatalf("unexpected event timestamp: %q", events.Events[0].Timestamp)
	}
	firstData, ok := events.Events[0].Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected event data map, got %T", events.Events[0].Data)
	}
	if firstData["email"] != "[REDACTED]" {
		t.Fatalf("expected email redaction, got %v", firstData["email"])
	}

	eventsReq = httptest.NewRequest(http.MethodGet, "/api/v1/bundle/events?offset=-1", nil)
	eventsRR = httptest.NewRecorder()
	handler.ServeHTTP(eventsRR, eventsReq)
	if eventsRR.Code != http.StatusBadRequest {
		t.Fatalf("events invalid pagination status: got %d want %d", eventsRR.Code, http.StatusBadRequest)
	}

	graphReq := httptest.NewRequest(http.MethodGet, "/api/v1/bundle/graph", nil)
	graphRR := httptest.NewRecorder()
	handler.ServeHTTP(graphRR, graphReq)
	if graphRR.Code != http.StatusOK {
		t.Fatalf("graph status: got %d want %d body=%s", graphRR.Code, http.StatusOK, graphRR.Body.String())
	}
	graph := decodeResponseJSON[BundleGraphResponse](t, graphRR)
	if len(graph.Nodes) != 3 {
		t.Fatalf("expected 3 graph nodes, got %d", len(graph.Nodes))
	}
	foundSequence := false
	foundParent := false
	for _, edge := range graph.Edges {
		if edge.Label == "sequence" {
			foundSequence = true
		}
		if edge.Label == "parent" {
			foundParent = true
		}
	}
	if !foundSequence || !foundParent {
		t.Fatalf("expected both sequence and parent edges, got %+v", graph.Edges)
	}
}

func TestEndpointMethodGuardsAndVerificationBlock(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	_, invalidHandler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		VerifyErr:       errors.New("tamper"),
		RevealAuthToken: "test-token",
	})

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   string
		status int
	}{
		{name: "meta method guard", method: http.MethodPost, path: "/api/v1/bundle/meta", status: http.StatusMethodNotAllowed},
		{name: "events method guard", method: http.MethodPost, path: "/api/v1/bundle/events", status: http.StatusMethodNotAllowed},
		{name: "graph method guard", method: http.MethodPost, path: "/api/v1/bundle/graph", status: http.StatusMethodNotAllowed},
		{name: "meta verify block", method: http.MethodGet, path: "/api/v1/bundle/meta", status: http.StatusForbidden},
		{name: "events verify block", method: http.MethodGet, path: "/api/v1/bundle/events", status: http.StatusForbidden},
		{name: "graph verify block", method: http.MethodGet, path: "/api/v1/bundle/graph", status: http.StatusForbidden},
		{name: "reveal verify block", method: http.MethodPost, path: "/api/v1/privacy/reveal", body: `{"seq":1,"field_path":"email"}`, status: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.path == "/api/v1/privacy/reveal" {
				req.Header.Set(revealAuthHeader, "test-token")
			}
			rr := httptest.NewRecorder()
			invalidHandler.ServeHTTP(rr, req)
			if rr.Code != tc.status {
				t.Fatalf("status: got %d want %d body=%s", rr.Code, tc.status, rr.Body.String())
			}
		})
	}
}

func TestPrivacyRevealAuthTable(t *testing.T) {
	bundlePath, b := createTestBundle(t)
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:       bundlePath,
		Bundle:           b,
		RevealAuthToken:  "test-token",
		RevealRateLimit:  100,
		RevealRateWindow: time.Minute,
	})

	payload := `{"seq":1,"field_path":"email","reason":"qa"}`
	tests := []struct {
		name       string
		addAuth    func(*http.Request)
		wantStatus int
	}{
		{
			name: "valid x header token",
			addAuth: func(r *http.Request) {
				r.Header.Set(revealAuthHeader, "test-token")
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "valid bearer token",
			addAuth: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer test-token")
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "valid cookie token",
			addAuth: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: revealAuthCookie, Value: "test-token"})
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid token",
			addAuth: func(r *http.Request) {
				r.Header.Set(revealAuthHeader, "wrong-token")
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing token",
			addAuth:    func(*http.Request) {},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			tc.addAuth(req)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status: got %d want %d body=%s", rr.Code, tc.wantStatus, rr.Body.String())
			}
		})
	}
}

func TestPrivacyRevealFailurePaths(t *testing.T) {
	bundlePath, b := createTestBundle(t)
	srv, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:       bundlePath,
		Bundle:           b,
		RevealAuthToken:  "test-token",
		RevealRateLimit:  100,
		RevealRateWindow: time.Hour,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/privacy/reveal", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method guard status: got %d want %d", rr.Code, http.StatusMethodNotAllowed)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", strings.NewReader("{"))
	req.Header.Set(revealAuthHeader, "test-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid json status: got %d want %d", rr.Code, http.StatusBadRequest)
	}

	for _, tc := range []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "bad seq", body: `{"seq":0,"field_path":"email"}`, wantStatus: http.StatusBadRequest},
		{name: "missing field path", body: `{"seq":1,"field_path":" "}`, wantStatus: http.StatusBadRequest},
		{name: "sequence not found", body: `{"seq":9,"field_path":"email"}`, wantStatus: http.StatusNotFound},
		{name: "field not found", body: `{"seq":1,"field_path":"not.real"}`, wantStatus: http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", strings.NewReader(tc.body))
			req.Header.Set(revealAuthHeader, "test-token")
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status: got %d want %d body=%s", rr.Code, tc.wantStatus, rr.Body.String())
			}
		})
	}

	_, rateLimitHandler := buildTestAPIServer(t, APIConfig{
		BundlePath:       bundlePath,
		Bundle:           b,
		RevealAuthToken:  "test-token",
		RevealRateLimit:  1,
		RevealRateWindow: time.Hour,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", strings.NewReader(`{"seq":1,"field_path":"email"}`))
	req.Header.Set(revealAuthHeader, "test-token")
	rr = httptest.NewRecorder()
	rateLimitHandler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("first reveal status: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", strings.NewReader(`{"seq":1,"field_path":"email"}`))
	req.Header.Set(revealAuthHeader, "test-token")
	rr = httptest.NewRecorder()
	rateLimitHandler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("rate-limit status: got %d want %d body=%s", rr.Code, http.StatusTooManyRequests, rr.Body.String())
	}

	srv.bundlePath = t.TempDir()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", strings.NewReader(`{"seq":1,"field_path":"email"}`))
	req.Header.Set(revealAuthHeader, "test-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("audit write failure status: got %d want %d body=%s", rr.Code, http.StatusInternalServerError, rr.Body.String())
	}
}

func TestPrivacyRevealRequiresConfiguredToken(t *testing.T) {
	bundlePath, b := createTestBundle(t)
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath: bundlePath,
		Bundle:     b,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", strings.NewReader(`{"seq":1,"field_path":"email"}`))
	req.Header.Set(revealAuthHeader, "any")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d body=%s", rr.Code, http.StatusInternalServerError, rr.Body.String())
	}
}

func TestAllowPrivacyRevealWindowReset(t *testing.T) {
	bundlePath, b := createTestBundle(t)
	srv, _ := buildTestAPIServer(t, APIConfig{
		BundlePath:       bundlePath,
		Bundle:           b,
		RevealAuthToken:  "test-token",
		RevealRateLimit:  1,
		RevealRateWindow: 10 * time.Millisecond,
	})

	makeReq := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		req.Header.Set(revealAuthHeader, "test-token")
		return req
	}

	if !srv.allowPrivacyReveal(makeReq()) {
		t.Fatalf("first request should be allowed")
	}
	if srv.allowPrivacyReveal(makeReq()) {
		t.Fatalf("second request in same window should be denied")
	}
	time.Sleep(15 * time.Millisecond)
	if !srv.allowPrivacyReveal(makeReq()) {
		t.Fatalf("request after window reset should be allowed")
	}
}

func TestRevealHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", nil)
	req.Header.Set(revealAuthHeader, "token-1")
	req.Header.Set(revealLegacyHeader, "token-legacy")
	req.Header.Set("Authorization", "Bearer token-2")
	req.AddCookie(&http.Cookie{Name: revealAuthCookie, Value: "token-3"})
	tokens := authTokensForRequest(req)
	if len(tokens) < 4 {
		t.Fatalf("expected auth tokens from header, legacy, bearer and cookie, got %d", len(tokens))
	}
	if got := primaryRateLimitToken(req); got != "token-1" {
		t.Fatalf("unexpected primary rate limit token: %q", got)
	}
	if got := normalizeRevealFieldPath("data.email"); got != "email" {
		t.Fatalf("normalize data prefix mismatch: %q", got)
	}
	if got := normalizeRevealFieldPath("email"); got != "email" {
		t.Fatalf("normalize plain path mismatch: %q", got)
	}

	ipAddr, user := resolveRemoteIdentity("127.0.0.1:8080")
	if ipAddr != "127.0.0.1" || user != "localhost" {
		t.Fatalf("unexpected localhost identity: ip=%q user=%q", ipAddr, user)
	}
	ipAddr, user = resolveRemoteIdentity("10.0.0.2")
	if ipAddr != "10.0.0.2" || user != "10.0.0.2" {
		t.Fatalf("unexpected remote identity: ip=%q user=%q", ipAddr, user)
	}
}

func TestPaginationAndExtractionHelpers(t *testing.T) {
	tests := []struct {
		name       string
		offsetRaw  string
		limitRaw   string
		wantOffset int
		wantLimit  int
		wantErr    bool
	}{
		{name: "defaults", wantOffset: 0, wantLimit: defaultEventsLimit},
		{name: "custom", offsetRaw: "2", limitRaw: "50", wantOffset: 2, wantLimit: 50},
		{name: "limit capped", limitRaw: "5000", wantOffset: 0, wantLimit: maxEventsLimit},
		{name: "bad offset", offsetRaw: "-1", wantErr: true},
		{name: "bad limit", limitRaw: "0", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			offset, limit, err := parsePagination(tc.offsetRaw, tc.limitRaw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if offset != tc.wantOffset || limit != tc.wantLimit {
				t.Fatalf("unexpected pagination: got offset=%d limit=%d", offset, limit)
			}
		})
	}

	if graphNodeType("ai.llm.prompt") != "llm" {
		t.Fatalf("expected llm node type")
	}
	if graphNodeType("ai.tool.call") != "tool" {
		t.Fatalf("expected tool node type")
	}
	if graphNodeType("ai.chain.start") != "chain" {
		t.Fatalf("expected chain node type")
	}
	if graphNodeType("other") != "event" {
		t.Fatalf("expected default event node type")
	}

	traceID, spanID, parent := extractSpanContext(map[string]interface{}{
		"trace_id":       "t1",
		"span_id":        "s1",
		"parent_span_id": "p1",
	})
	if traceID != "t1" || spanID != "s1" || parent != "p1" {
		t.Fatalf("unexpected span context: %q %q %q", traceID, spanID, parent)
	}
	traceID, spanID, parent = extractSpanContext("not-a-map")
	if traceID != "" || spanID != "" || parent != "" {
		t.Fatalf("non-map span context should be empty")
	}

	if got := extractTimestamp(map[string]interface{}{"timestamp": "ts-1"}); got != "ts-1" {
		t.Fatalf("timestamp extract mismatch: %q", got)
	}
	if got := extractTimestamp(map[string]interface{}{"timing": map[string]interface{}{"started_at": "ts-2"}}); got != "ts-2" {
		t.Fatalf("timing timestamp extract mismatch: %q", got)
	}
	if got := extractTimestamp("bad"); got != "" {
		t.Fatalf("non-map timestamp should be empty, got %q", got)
	}

	if got := stringAtKey(map[string]interface{}{"k": " value "}, "k"); got != "value" {
		t.Fatalf("stringAtKey trim mismatch: %q", got)
	}
	if got := stringAtKey(map[string]interface{}{"k": 1}, "k"); got != "" {
		t.Fatalf("stringAtKey non-string should be empty, got %q", got)
	}

	root := map[string]interface{}{
		"profile": map[string]interface{}{
			"emails": []interface{}{"a@example.com", "b@example.com"},
		},
	}
	value, ok := resolveField(root, "profile.emails.1")
	if !ok || value != "b@example.com" {
		t.Fatalf("resolveField nested index failed: value=%v ok=%v", value, ok)
	}
	if _, ok := resolveField(root, "profile..emails"); ok {
		t.Fatalf("resolveField should fail on empty path segment")
	}
	if _, ok := resolveField(root, "profile.emails.bad"); ok {
		t.Fatalf("resolveField should fail on non-numeric slice index")
	}
}

func TestFindRecordTamperHandlerAndPIIRuleLoad(t *testing.T) {
	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "evt.one", map[string]interface{}{"a": 1})
	appendTestBundleEvent(t, b, "evt.two", map[string]interface{}{"a": 2})
	records := b.Records
	if _, ok := findRecordBySequence(records, 2); !ok {
		t.Fatalf("expected sequence to be found")
	}
	if _, ok := findRecordBySequence(records, 3); ok {
		t.Fatalf("unexpected record found for missing sequence")
	}

	tamper := NewTamperHandler("<bundle>", errors.New("<tamper>"))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	tamper.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("tamper handler status: got %d want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "&lt;bundle&gt;") || !strings.Contains(body, "&lt;tamper&gt;") {
		t.Fatalf("tamper handler should escape HTML content, body=%s", body)
	}

	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "pii-fields.json")
	raw := []byte(`{"allow_fields":["trace_id"],"sensitive_fields":["secret_field"],"sensitive_substrings":["token"],"sensitive_suffixes":["_secret"]}`)
	if err := os.WriteFile(configPath, raw, 0o600); err != nil {
		t.Fatalf("write pii config: %v", err)
	}
	t.Setenv(piiFieldsPathEnv, configPath)
	rules := loadPIIMaskRules()
	if _, ok := rules.sensitiveFields["secret_field"]; !ok {
		t.Fatalf("expected custom sensitive field in rules")
	}
	if len(rules.sensitiveSubstrings) != 1 || rules.sensitiveSubstrings[0] != "token" {
		t.Fatalf("unexpected substring rules: %+v", rules.sensitiveSubstrings)
	}
	path, err := findPIIFieldsPath()
	if err != nil {
		t.Fatalf("findPIIFieldsPath env override: %v", err)
	}
	if path != configPath {
		t.Fatalf("findPIIFieldsPath mismatch: got %q want %q", path, configPath)
	}
}

func TestLoadPIIMaskRulesUsesEmbeddedDefaultsOutsideRepo(t *testing.T) {
	tmp := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	t.Setenv(piiFieldsPathEnv, "")

	rules := loadPIIMaskRules()
	if _, ok := rules.sensitiveFields["driver_license"]; !ok {
		t.Fatalf("expected embedded pii rules to include driver_license")
	}
	foundFingerprint := false
	for _, item := range rules.sensitiveSubstrings {
		if item == "fingerprint" {
			foundFingerprint = true
			break
		}
	}
	if !foundFingerprint {
		t.Fatalf("expected embedded pii rules to include fingerprint substring")
	}
}

func TestWriteJSONSetsContentType(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusAccepted, map[string]string{"status": "ok"})

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status: got %d want %d", rr.Code, http.StatusAccepted)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type: got %q want %q", got, "application/json")
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte(`"status":"ok"`)) {
		t.Fatalf("response body missing JSON payload: %s", rr.Body.String())
	}
}
