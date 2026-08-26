// SPDX-License-Identifier: MIT
package apiv1

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/hash"
	atbauth "github.com/pcguest/atb/pkg/auth"
)

const testAPISessionToken = "test-api-session-token"

func buildTestAPIServer(t *testing.T, cfg APIConfig) (*APIServer, http.Handler) {
	t.Helper()

	useDefaultSession := cfg.SessionToken == "" && cfg.JWTValidator == nil
	if useDefaultSession {
		cfg.SessionToken = testAPISessionToken
	}
	srv := NewAPIServer(cfg)
	mux := http.NewServeMux()
	srv.Register(mux)
	if useDefaultSession {
		return srv, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clone := r.Clone(r.Context())
			clone.Header.Set(sessionAuthHeader, testAPISessionToken)
			mux.ServeHTTP(w, clone)
		})
	}
	return srv, mux
}

func TestAPIServerFailsClosedWithoutAuthentication(t *testing.T) {
	srv := NewAPIServer(APIConfig{Bundle: newTestBundle(t)})
	mux := http.NewServeMux()
	srv.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/verification", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status without authentication: got %d want %d body=%s", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
}

func TestAPIServerJWTAuthentication(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	key, err := jwk.FromRaw(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("create JWK: %v", err)
	}
	if err := key.Set(jwk.KeyIDKey, "api-test-kid"); err != nil {
		t.Fatalf("set JWK key ID: %v", err)
	}
	set := jwk.NewSet()
	if err := set.AddKey(key); err != nil {
		t.Fatalf("add JWK: %v", err)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot bind local JWT test issuer: %v", err)
	}
	issuer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/jwks.json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(set); err != nil {
			t.Errorf("encode JWKS: %v", err)
		}
	}))
	issuer.Listener = listener
	issuer.Start()
	t.Cleanup(issuer.Close)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	validator, err := atbauth.NewJWTValidator(ctx, issuer.URL+"/", "api-test-audience")
	if err != nil {
		t.Fatalf("create JWT validator: %v", err)
	}
	signToken := func(t *testing.T, tokenIssuer, audience string, expiry time.Time, role atbauth.Role) string {
		t.Helper()
		claims := jwt.MapClaims{
			"iss":  tokenIssuer,
			"aud":  audience,
			"exp":  expiry.Unix(),
			"iat":  time.Now().Add(-time.Minute).Unix(),
			"role": string(role),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = "api-test-kid"
		signed, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatalf("sign JWT: %v", err)
		}
		return signed
	}
	validViewer := signToken(t, issuer.URL+"/", "api-test-audience", time.Now().Add(time.Hour), atbauth.RoleViewer)
	validOperator := signToken(t, issuer.URL+"/", "api-test-audience", time.Now().Add(time.Hour), atbauth.RoleOperator)

	_, handler := buildTestAPIServer(t, APIConfig{
		Bundle:       newTestBundle(t),
		JWTValidator: validator,
	})
	for _, tc := range []struct {
		name       string
		method     string
		path       string
		authHeader string
		wantStatus int
	}{
		{name: "valid viewer JWT", method: http.MethodGet, path: "/api/v1/verification", authHeader: "Bearer " + validViewer, wantStatus: http.StatusOK},
		{name: "wrong audience", method: http.MethodGet, path: "/api/v1/verification", authHeader: "Bearer " + signToken(t, issuer.URL+"/", "other-audience", time.Now().Add(time.Hour), atbauth.RoleViewer), wantStatus: http.StatusUnauthorized},
		{name: "wrong issuer", method: http.MethodGet, path: "/api/v1/verification", authHeader: "Bearer " + signToken(t, issuer.URL+"/other", "api-test-audience", time.Now().Add(time.Hour), atbauth.RoleViewer), wantStatus: http.StatusUnauthorized},
		{name: "expired", method: http.MethodGet, path: "/api/v1/verification", authHeader: "Bearer " + signToken(t, issuer.URL+"/", "api-test-audience", time.Now().Add(-time.Minute), atbauth.RoleViewer), wantStatus: http.StatusUnauthorized},
		{name: "invalid JWT", method: http.MethodGet, path: "/api/v1/verification", authHeader: "Bearer invalid", wantStatus: http.StatusUnauthorized},
		{name: "missing JWT", method: http.MethodGet, path: "/api/v1/verification", wantStatus: http.StatusUnauthorized},
		{name: "viewer cannot verify", method: http.MethodPost, path: "/api/v1/bundle/verify", authHeader: "Bearer " + validViewer, wantStatus: http.StatusForbidden},
		{name: "operator can verify", method: http.MethodPost, path: "/api/v1/bundle/verify", authHeader: "Bearer " + validOperator, wantStatus: http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rr.Code, tc.wantStatus, rr.Body.String())
			}
		})
	}
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
	if firstData["field_path"] != "email" || firstData["reason"] != "first" {
		t.Fatalf("unexpected first audit event data: %v", firstData)
	}
	secondData, ok := secondAudit.Event.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected second audit event data map, got %T", secondAudit.Event.Data)
	}
	if secondData["field_path"] != "user_id" || secondData["reason"] != "second" {
		t.Fatalf("unexpected second audit event data: %v", secondData)
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
