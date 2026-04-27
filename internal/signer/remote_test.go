// SPDX-License-Identifier: MIT
package signer

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newRemoteTestKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return pub, priv
}

func TestNewHTTPRemoteSigner_RequiresEndpoint(t *testing.T) {
	if _, err := NewHTTPRemoteSigner(HTTPConfig{}); err == nil {
		t.Fatalf("expected error for empty endpoint")
	}
}

func TestNewHTTPRemoteSigner_DefaultsBackend(t *testing.T) {
	s, err := NewHTTPRemoteSigner(HTTPConfig{Endpoint: "https://example.invalid/sign"})
	if err != nil {
		t.Fatalf("NewHTTPRemoteSigner: %v", err)
	}
	if s.cfg.Backend != "https-http" {
		t.Fatalf("default backend = %q, want %q", s.cfg.Backend, "https-http")
	}
}

func TestHTTPRemoteSigner_HappyPath(t *testing.T) {
	pub, priv := newRemoteTestKey(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		if got := r.Header.Get("X-ATB-Signer-Key"); got != "secret-token" {
			t.Errorf("X-ATB-Signer-Key = %q, want secret-token", got)
		}

		var req httpSignRequest
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		if req.Algorithm != "ed25519" {
			t.Errorf("algorithm = %q, want ed25519", req.Algorithm)
		}
		if req.KeyID != "kms-alias-prod" {
			t.Errorf("key_id = %q, want kms-alias-prod", req.KeyID)
		}
		digest, err := hex.DecodeString(req.DigestHex)
		if err != nil {
			t.Fatalf("decode digest_hex: %v", err)
		}
		if len(digest) != 32 {
			t.Errorf("digest length = %d, want 32", len(digest))
		}

		// Sign the digest with our test key.
		sig := ed25519.Sign(priv, digest)
		resp := httpSignResponse{
			SignatureBase64: base64.StdEncoding.EncodeToString(sig),
			PubKeyBase64:    base64.StdEncoding.EncodeToString(pub),
			KeyID:           "kms-alias-prod-v3",
			Backend:         "test-kms",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s, err := NewHTTPRemoteSigner(HTTPConfig{
		Endpoint:   srv.URL,
		APIKey:     "secret-token",
		KeyID:      "kms-alias-prod",
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("NewHTTPRemoteSigner: %v", err)
	}

	digest := make([]byte, 32)
	for i := range digest {
		digest[i] = byte(i)
	}

	gotSig, gotPub, gotKeyID, gotBackend, gotAlgorithm, err := s.Sign(context.Background(), digest)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if !ed25519.Verify(ed25519.PublicKey(gotPub), digest, gotSig) {
		t.Fatalf("returned signature does not verify against returned pubkey")
	}
	if gotKeyID != "kms-alias-prod-v3" {
		t.Errorf("keyID = %q, want kms-alias-prod-v3 (response should override config)", gotKeyID)
	}
	if gotBackend != "test-kms" {
		t.Errorf("backend = %q, want test-kms", gotBackend)
	}
	if gotAlgorithm != "ed25519" {
		t.Errorf("algorithm = %q, want ed25519", gotAlgorithm)
	}
}

func TestHTTPRemoteSigner_BackendDefaultUsedWhenResponseEmpty(t *testing.T) {
	pub, priv := newRemoteTestKey(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req httpSignRequest
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		digest, _ := hex.DecodeString(req.DigestHex)
		sig := ed25519.Sign(priv, digest)
		// Response omits backend; signer should fall back to cfg.Backend.
		_ = json.NewEncoder(w).Encode(httpSignResponse{
			SignatureBase64: base64.StdEncoding.EncodeToString(sig),
			PubKeyBase64:    base64.StdEncoding.EncodeToString(pub),
		})
	}))
	defer srv.Close()

	s, err := NewHTTPRemoteSigner(HTTPConfig{Endpoint: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewHTTPRemoteSigner: %v", err)
	}
	digest := make([]byte, 32)
	_, _, _, backend, _, err := s.Sign(context.Background(), digest)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if backend != "https-http" {
		t.Fatalf("backend = %q, want https-http (config default)", backend)
	}
}

func TestHTTPRemoteSigner_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden: bad token"))
	}))
	defer srv.Close()

	s, err := NewHTTPRemoteSigner(HTTPConfig{Endpoint: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewHTTPRemoteSigner: %v", err)
	}
	_, _, _, _, _, err = s.Sign(context.Background(), make([]byte, 32))
	if err == nil {
		t.Fatalf("expected error from non-2xx status")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error %q should include status 403", err)
	}
	if !strings.Contains(err.Error(), "forbidden") {
		t.Errorf("error %q should include response body prefix", err)
	}
}

func TestHTTPRemoteSigner_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	s, err := NewHTTPRemoteSigner(HTTPConfig{Endpoint: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewHTTPRemoteSigner: %v", err)
	}
	_, _, _, _, _, err = s.Sign(context.Background(), make([]byte, 32))
	if err == nil || !strings.Contains(err.Error(), "parse response") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestHTTPRemoteSigner_WrongSignatureLength(t *testing.T) {
	pub, _ := newRemoteTestKey(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(httpSignResponse{
			SignatureBase64: base64.StdEncoding.EncodeToString([]byte("too short")),
			PubKeyBase64:    base64.StdEncoding.EncodeToString(pub),
		})
	}))
	defer srv.Close()

	s, err := NewHTTPRemoteSigner(HTTPConfig{Endpoint: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewHTTPRemoteSigner: %v", err)
	}
	_, _, _, _, _, err = s.Sign(context.Background(), make([]byte, 32))
	if err == nil || !strings.Contains(err.Error(), "signature length") {
		t.Fatalf("expected signature length error, got %v", err)
	}
}

func TestHTTPRemoteSigner_WrongPubKeyLength(t *testing.T) {
	_, priv := newRemoteTestKey(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req httpSignRequest
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		digest, _ := hex.DecodeString(req.DigestHex)
		sig := ed25519.Sign(priv, digest)
		_ = json.NewEncoder(w).Encode(httpSignResponse{
			SignatureBase64: base64.StdEncoding.EncodeToString(sig),
			PubKeyBase64:    base64.StdEncoding.EncodeToString([]byte("not 32 bytes")),
		})
	}))
	defer srv.Close()

	s, err := NewHTTPRemoteSigner(HTTPConfig{Endpoint: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewHTTPRemoteSigner: %v", err)
	}
	_, _, _, _, _, err = s.Sign(context.Background(), make([]byte, 32))
	if err == nil || !strings.Contains(err.Error(), "pubkey length") {
		t.Fatalf("expected pubkey length error, got %v", err)
	}
}
