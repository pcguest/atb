// SPDX-License-Identifier: MIT

package signer

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Contract for all remote signing backends:
//
//  1. Private key material MUST NOT be transmitted to or from the backend.
//     The signing operation happens inside the KMS / signing service; ATB
//     only receives the signature and the public verification key.
//
//  2. The digest passed to the backend MUST be the same pre-image as
//     LocalSigner — the 32-byte SHA-256 of the pre-signature bundle NDJSON.
//     If a backend wraps the digest in an envelope (e.g., GCP CryptoKey
//     digest types), the envelope bytes MUST be recorded in the signature
//     record so verifiers can reconstruct the pre-image.
//
//  3. Verification of a remotely-signed bundle MUST NOT require a live
//     backend call. The pubKey returned by Sign MUST be the raw
//     verification key material (or a certificate from which it can be
//     derived) embedded in the bundle record.
//
//  4. If the backend is unreachable and fallback-to-local is configured at
//     the CLI layer, the CLI is responsible for labelling the resulting
//     signature as "local:fallback:<intended-backend>" — the Signer
//     interface itself does not implement fallback.

// HTTPConfig configures an HTTPRemoteSigner.
//
// Endpoint is the full URL of the signing service.
//
// APIKey, when non-empty, is sent in the X-ATB-Signer-Key header. Bearer
// tokens or other schemes can be plumbed in by a future option; X-ATB-
// Signer-Key is the documented default and keeps the wire format simple
// for first-party signing services.
//
// Backend is the short name recorded in the signature record's `backend`
// field. If empty, "https-http" is used. Cloud-specific backends (e.g.
// AWS KMS or GCP KMS that happen to be fronted by a small proxy) should
// set this to the cloud name.
//
// KeyID, when non-empty, is forwarded to the signing service in the JSON
// request body and is also recorded in the signature record's `key_id`
// field. The signing service MAY override the request value in its
// response (some KMS-fronted services resolve a logical alias to a
// concrete versioned key).
type HTTPConfig struct {
	Endpoint string
	APIKey   string
	Backend  string
	KeyID    string

	// HTTPClient is optional. When nil a default http.Client with a 30s
	// timeout is used. Tests inject httptest.Server-backed clients here.
	HTTPClient *http.Client
}

// HTTPRemoteSigner is a generic HTTP-based remote signer. It implements
// the Signer interface against a small JSON request/response protocol
// described on Sign. Native AWS KMS, GCP KMS, and Vault implementations live
// in sibling packages; other services can use this protocol through a narrow
// signing proxy.
type HTTPRemoteSigner struct {
	cfg    HTTPConfig
	client *http.Client
}

// NewHTTPRemoteSigner returns an HTTPRemoteSigner configured by cfg.
//
// Endpoint is required. Backend defaults to "https-http". HTTPClient
// defaults to an http.Client with a 30s timeout.
func NewHTTPRemoteSigner(cfg HTTPConfig) (*HTTPRemoteSigner, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, fmt.Errorf("signer: HTTPRemoteSigner: Endpoint is required")
	}
	if cfg.Backend == "" {
		cfg.Backend = "https-http"
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &HTTPRemoteSigner{cfg: cfg, client: client}, nil
}

// httpSignRequest is the JSON body posted to the signing endpoint.
type httpSignRequest struct {
	DigestHex string `json:"digest_hex"`
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"key_id,omitempty"`
}

// httpSignResponse is the JSON body expected back from the signing
// endpoint. Fields not understood by older readers are ignored.
type httpSignResponse struct {
	SignatureBase64 string `json:"signature_base64"`
	PubKeyBase64    string `json:"pubkey_base64"`
	KeyID           string `json:"key_id"`
	Backend         string `json:"backend"`
}

// Sign POSTs the digest to the configured endpoint and returns the
// signature and verification material.
//
// SECURITY NOTE: Sign MUST NOT log the digest or the response body if
// they may contain sensitive information. The default error path includes
// a short prefix of the response body for debugging non-2xx responses;
// callers that need stricter handling should wrap and re-classify the
// returned error.
//
// Pre-image: identical to LocalSigner. The bundle layer passes the
// 32-byte SHA-256 digest of the pre-signature bundle NDJSON.
func (s *HTTPRemoteSigner) Sign(ctx context.Context, digest []byte) (sig, pubKey []byte, keyID, backend, algorithm string, err error) {
	body := httpSignRequest{
		DigestHex: hex.EncodeToString(digest),
		Algorithm: "ed25519",
		KeyID:     s.cfg.KeyID,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, nil, "", "", "", fmt.Errorf("signer: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.Endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, nil, "", "", "", fmt.Errorf("signer: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if s.cfg.APIKey != "" {
		// X-ATB-Signer-Key is the documented header for first-party
		// signing services. Other schemes can be added as additional
		// HTTPConfig fields if needed.
		req.Header.Set("X-ATB-Signer-Key", s.cfg.APIKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, nil, "", "", "", fmt.Errorf("signer: http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if readErr != nil {
		return nil, nil, "", "", "", fmt.Errorf("signer: read response: %w", readErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		preview := string(respBody)
		if len(preview) > 256 {
			preview = preview[:256] + "…"
		}
		return nil, nil, "", "", "", fmt.Errorf("signer: remote returned status %d: %s", resp.StatusCode, preview)
	}

	var parsed httpSignResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, nil, "", "", "", fmt.Errorf("signer: parse response: %w", err)
	}

	sigBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(parsed.SignatureBase64))
	if err != nil {
		return nil, nil, "", "", "", fmt.Errorf("signer: decode signature_base64: %w", err)
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return nil, nil, "", "", "", fmt.Errorf("signer: signature length = %d, want %d (Ed25519)", len(sigBytes), ed25519.SignatureSize)
	}
	pubBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(parsed.PubKeyBase64))
	if err != nil {
		return nil, nil, "", "", "", fmt.Errorf("signer: decode pubkey_base64: %w", err)
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		return nil, nil, "", "", "", fmt.Errorf("signer: pubkey length = %d, want %d (Ed25519)", len(pubBytes), ed25519.PublicKeySize)
	}

	chosenBackend := parsed.Backend
	if chosenBackend == "" {
		chosenBackend = s.cfg.Backend
	}
	chosenKeyID := parsed.KeyID
	if chosenKeyID == "" {
		chosenKeyID = s.cfg.KeyID
	}
	return sigBytes, pubBytes, chosenKeyID, chosenBackend, "ed25519", nil
}

// Compile-time interface satisfaction check.
var _ Signer = (*HTTPRemoteSigner)(nil)
