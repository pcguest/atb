package apiv1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrivacyRevealAuditAndMasking(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		RevealAuthToken: "test-token",
		RevealRateLimit: 100,
	})

	// 1. Confirm masking is active on normal events endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bundle/events", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	events := decodeResponseJSON[BundleEventsResponse](t, rr)
	foundSensitive := false
	for _, e := range events.Events {
		data, _ := e.Data.(map[string]any)
		if email, ok := data["email"]; ok && email == "[REDACTED]" {
			foundSensitive = true
		}
	}
	if !foundSensitive {
		t.Fatal("expected sensitive fields to be masked in events output")
	}

	// 2. Perform a successful reveal
	revealPayload := `{"seq":1,"field_path":"email","reason":"audit_test"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", strings.NewReader(revealPayload))
	req.Header.Set(revealAuthHeader, "test-token")
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("reveal failed: %d %s", rr.Code, rr.Body.String())
	}
	revealResp := decodeResponseJSON[PrivacyRevealResponse](t, rr)
	if revealResp.Value != "auditor@example.com" {
		t.Fatalf("unexpected revealed value: %v", revealResp.Value)
	}

	// 3. Confirm audit record was appended to the bundle
	if len(b.Records) != 4 {
		t.Fatalf("expected 4 records (3 original + 1 reveal audit), got %d", len(b.Records))
	}
	lastRecord := b.Records[3]
	if lastRecord.Event.Type != "privacy.reveal" {
		t.Fatalf("expected last record to be privacy.reveal, got %s", lastRecord.Event.Type)
	}
	auditData := lastRecord.Event.Data.(map[string]any)
	if auditData["field_path"] != "email" || fmt.Sprintf("%v", auditData["seq"]) != "1" {
		t.Fatalf("unexpected audit data: %+v", auditData)
	}

	// 4. Verify bundle integrity remains valid after audit append
	if err := b.Verify(); err != nil {
		t.Fatalf("bundle integrity broken after audit append: %v", err)
	}
}

func TestPrivacyRevealRateLimitAdversarial(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	// Set a very low limit for the test
	limit := 5
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		RevealAuthToken: "test-token",
		RevealRateLimit: limit,
	})

	for i := 1; i <= limit; i++ {
		payload := `{"seq":1,"field_path":"email"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", strings.NewReader(payload))
		req.Header.Set(revealAuthHeader, "test-token")
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d failed: %d", i, rr.Code)
		}
	}

	// The (limit + 1)-th request should be blocked
	req := httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", strings.NewReader(`{"seq":1,"field_path":"email"}`))
	req.Header.Set(revealAuthHeader, "test-token")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests, got %d", rr.Code)
	}
	
	var apiErr APIError
	if err := json.Unmarshal(rr.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("failed to decode error: %v", err)
	}
	if !strings.Contains(apiErr.Error, "rate limit exceeded") {
		t.Fatalf("unexpected error message: %q", apiErr.Error)
	}
}
