// SPDX-License-Identifier: MIT
package apiv1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/hash"
)

func buildTestAPIServer(t *testing.T, cfg APIConfig) (*APIServer, http.Handler) {
	t.Helper()

	srv := NewAPIServer(cfg)
	mux := http.NewServeMux()
	srv.Register(mux)
	return srv, mux
}

func createTestBundle(t *testing.T) (string, *bundle.Bundle) {
	t.Helper()

	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")

	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "agent.prompt", map[string]interface{}{
		"email":    "auditor@example.com",
		"user_id":  "usr_123",
		"trace_id": "trc_123",
	})
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}
	return bundlePath, b
}

func TestPrivacyRevealRequiresAuthToken(t *testing.T) {
	bundlePath, b := createTestBundle(t)
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		RevealAuthToken: "test-token",
	})

	payload := []byte(`{"seq":1,"field_path":"email","reason":"qa_test"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status without auth token: got %d want %d", rr.Code, http.StatusUnauthorized)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(revealAuthHeader, "test-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status with auth token: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
}

func TestPrivacyRevealRateLimited(t *testing.T) {
	bundlePath, b := createTestBundle(t)
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		RevealAuthToken: "test-token",
	})

	payload := []byte(`{"seq":1,"field_path":"email","reason":"qa_test"}`)
	for i := 0; i < 11; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", bytes.NewReader(payload))
		req.RemoteAddr = "127.0.0.1:4010"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(revealAuthHeader, "test-token")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if i < 10 && rr.Code != http.StatusOK {
			t.Fatalf("request %d expected 200, got %d body=%s", i+1, rr.Code, rr.Body.String())
		}
		if i == 10 && rr.Code != http.StatusTooManyRequests {
			t.Fatalf("request %d expected 429, got %d body=%s", i+1, rr.Code, rr.Body.String())
		}
	}
}

func TestPrivacyRevealRecordsToSidecarNotBundle(t *testing.T) {
	bundlePath, b := createTestBundle(t)
	originalRecords := len(b.Records)
	originalBytes, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle before reveals: %v", err)
	}
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		RevealAuthToken: "test-token",
	})

	payloads := [][]byte{
		[]byte(`{"seq":1,"field_path":"email","reason":"first"}`),
		[]byte(`{"seq":1,"field_path":"user_id","reason":"second"}`),
	}
	for _, payload := range payloads {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(revealAuthHeader, "test-token")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("unexpected reveal status: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}
	}

	// The authoritative bundle must be byte-for-byte unchanged by reveals.
	afterBytes, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle after reveals: %v", err)
	}
	if !bytes.Equal(originalBytes, afterBytes) {
		t.Fatalf("authoritative bundle was mutated by reveal")
	}
	if len(b.Records) != originalRecords {
		t.Fatalf("in-memory bundle grew from %d to %d records after reveal", originalRecords, len(b.Records))
	}

	// The reveals must live in the sidecar, in their own verifiable chain.
	sidecar, err := bundle.LoadVerified(bundlePath + ".reveals")
	if err != nil {
		t.Fatalf("load reveal sidecar: %v", err)
	}
	if len(sidecar.Records) != 3 {
		t.Fatalf("expected sidecar manifest + 2 reveals, got %d records", len(sidecar.Records))
	}

	firstAudit := sidecar.Records[1]
	secondAudit := sidecar.Records[2]
	if firstAudit.Event.Type != "privacy.reveal" || secondAudit.Event.Type != "privacy.reveal" {
		t.Fatalf("expected reveal audit events, got %q and %q", firstAudit.Event.Type, secondAudit.Event.Type)
	}
	if firstAudit.Event.Timestamp == "" || secondAudit.Event.Timestamp == "" {
		t.Fatalf("expected timestamps on reveal audit events")
	}
	if secondAudit.Event.PrevHash != firstAudit.Hash {
		t.Fatalf("expected hash-chain link prev_hash=%q got %q", firstAudit.Hash, secondAudit.Event.PrevHash)
	}

	firstData, ok := firstAudit.Event.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected first audit event data map, got %T", firstAudit.Event.Data)
	}
	if firstData["field_path"] != "email" {
		t.Fatalf("unexpected field_path in first audit event: %v", firstData["field_path"])
	}
}

func TestSensitiveFieldRulesCoverPIIContract(t *testing.T) {
	path, err := findPIIFieldsPath()
	if err != nil {
		t.Fatalf("find pii fields file: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read pii fields file: %v", err)
	}

	var rules piiFieldRulesFile
	if err := json.Unmarshal(raw, &rules); err != nil {
		t.Fatalf("decode pii fields file: %v", err)
	}

	for _, key := range rules.SensitiveFields {
		if !isSensitiveFieldKey(key) {
			t.Fatalf("expected field %q to be marked sensitive", key)
		}
	}
	for _, key := range rules.AllowFields {
		if isSensitiveFieldKey(key) {
			t.Fatalf("expected field %q to be allowed", key)
		}
	}
}

func TestMaskSensitiveUsesConfiguredRules(t *testing.T) {
	payload := map[string]interface{}{
		"email":    "auditor@example.com",
		"user_id":  "usr_123",
		"trace_id": "trc_123",
		"profile": map[string]interface{}{
			"bio": "private",
		},
	}

	masked, ok := maskSensitive(payload).(map[string]interface{})
	if !ok {
		t.Fatalf("masked payload should remain a map")
	}

	if masked["email"] != "[REDACTED]" {
		t.Fatalf("expected email to be redacted, got %v", masked["email"])
	}
	if masked["user_id"] != "[REDACTED]" {
		t.Fatalf("expected user_id to be redacted, got %v", masked["user_id"])
	}
	if masked["trace_id"] != "trc_123" {
		t.Fatalf("expected trace_id to remain visible, got %v", masked["trace_id"])
	}

	profile, ok := masked["profile"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected profile to remain a map")
	}
	if profile["bio"] != "[REDACTED]" {
		t.Fatalf("expected profile.bio to be redacted, got %v", profile["bio"])
	}
}

func TestBundleMetaIncludesIntegrityFields(t *testing.T) {
	bundlePath, b := createTestBundle(t)
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		RevealAuthToken: "test-token",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bundle/meta", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var out BundleMetaResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.GenesisHash != hash.GenesisHash {
		t.Fatalf("genesis_hash: got %q want %q", out.GenesisHash, hash.GenesisHash)
	}
	if out.VerifiedAt == "" {
		t.Fatalf("expected verified_at to be set")
	}
	if _, err := time.Parse(time.RFC3339, out.VerifiedAt); err != nil {
		t.Fatalf("verified_at should be RFC3339, got %q err=%v", out.VerifiedAt, err)
	}
	if out.FirstTimestamp != testBundleTimestamp() {
		t.Fatalf("first_timestamp: got %q want %q", out.FirstTimestamp, testBundleTimestamp())
	}
	if out.LastTimestamp != testBundleTimestamp() {
		t.Fatalf("last_timestamp: got %q want %q", out.LastTimestamp, testBundleTimestamp())
	}
}
