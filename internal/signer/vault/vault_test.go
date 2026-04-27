// SPDX-License-Identifier: MIT

//go:build vault

package vault

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	vaultapi "github.com/hashicorp/vault/api"
)

type fakeVaultLogical struct {
	publicKeyPEM string
	signature    []byte
	keyType      string
}

func (f fakeVaultLogical) WriteWithContext(context.Context, string, map[string]any) (*vaultapi.Secret, error) {
	return &vaultapi.Secret{Data: map[string]any{
		"signature": "vault:v1:" + base64.StdEncoding.EncodeToString(f.signature),
	}}, nil
}

func (f fakeVaultLogical) ReadWithContext(context.Context, string) (*vaultapi.Secret, error) {
	data := map[string]any{
		"latest_version": float64(1),
		"keys": map[string]any{
			"1": map[string]any{
				"public_key": f.publicKeyPEM,
			},
		},
	}
	if f.keyType != "" {
		data["type"] = f.keyType
	}
	return &vaultapi.Secret{Data: data}, nil
}

func TestVaultSignerSign(t *testing.T) {
	s := &VaultSigner{
		logical: fakeVaultLogical{
			publicKeyPEM: testP256PublicKeyPEM(t),
			signature:    []byte("signature"),
			keyType:      "ecdsa-p256",
		},
		keyID: "atb-signing-key",
	}

	signature, pubKey, keyID, backend, algorithm, err := s.Sign(context.Background(), make([]byte, 32))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if string(signature) != "signature" {
		t.Fatalf("signature = %q, want signature", string(signature))
	}
	if len(pubKey) != 65 {
		t.Fatalf("pubKey length = %d, want 65", len(pubKey))
	}
	if keyID != s.keyID || backend != "vault" || algorithm != "ecdsa-p256" {
		t.Fatalf("unexpected provenance: keyID=%q backend=%q algorithm=%q", keyID, backend, algorithm)
	}
}

// vaultMockHandler emits a minimal Vault Transit response for read-key and
// sign endpoints. keyType is echoed in the read-key response and toggles the
// public_key encoding.
func vaultMockHandler(t *testing.T, keyType, publicKeyField string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/transit/keys/"):
			body := map[string]any{
				"data": map[string]any{
					"type":           keyType,
					"latest_version": 1,
					"keys": map[string]any{
						"1": map[string]any{"public_key": publicKeyField},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(body)
		case (r.Method == http.MethodPost || r.Method == http.MethodPut) && strings.HasPrefix(r.URL.Path, "/v1/transit/sign/"):
			body := map[string]any{
				"data": map[string]any{
					"signature": "vault:v1:" + base64.StdEncoding.EncodeToString([]byte("sig")),
				},
			}
			_ = json.NewEncoder(w).Encode(body)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	})
}

func newVaultClient(t *testing.T, addr string) *vaultapi.Client {
	t.Helper()
	cfg := vaultapi.DefaultConfig()
	cfg.Address = addr
	c, err := vaultapi.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.SetToken("test-token")
	return c
}

func TestVaultSignerSignEd25519FromHTTPMock(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519 GenerateKey: %v", err)
	}
	pubB64 := base64.StdEncoding.EncodeToString(priv.Public().(ed25519.PublicKey))
	srv := httptest.NewServer(vaultMockHandler(t, "ed25519", pubB64))
	t.Cleanup(srv.Close)

	client := newVaultClient(t, srv.URL)
	s := &VaultSigner{client: client, logical: client.Logical(), keyID: "test-key"}

	_, pubKey, _, backend, algorithm, err := s.Sign(context.Background(), make([]byte, 32))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if algorithm != "ed25519" {
		t.Fatalf("algorithm = %q, want ed25519", algorithm)
	}
	if backend != "vault" {
		t.Fatalf("backend = %q, want vault", backend)
	}
	if len(pubKey) != 32 {
		t.Fatalf("pubKey length = %d, want 32", len(pubKey))
	}
}

func TestVaultSignerSignECDSAFromHTTPMock(t *testing.T) {
	srv := httptest.NewServer(vaultMockHandler(t, "ecdsa-p256", testP256PublicKeyPEM(t)))
	t.Cleanup(srv.Close)

	client := newVaultClient(t, srv.URL)
	s := &VaultSigner{client: client, logical: client.Logical(), keyID: "test-key"}

	_, _, _, _, algorithm, err := s.Sign(context.Background(), make([]byte, 32))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if algorithm != "ecdsa-p256" {
		t.Fatalf("algorithm = %q, want ecdsa-p256", algorithm)
	}
}

func TestVaultSignerSignUnsupportedKeyType(t *testing.T) {
	srv := httptest.NewServer(vaultMockHandler(t, "rsa-2048", "ignored"))
	t.Cleanup(srv.Close)

	client := newVaultClient(t, srv.URL)
	s := &VaultSigner{client: client, logical: client.Logical(), keyID: "test-key"}

	_, _, _, _, _, err := s.Sign(context.Background(), make([]byte, 32))
	if err == nil {
		t.Fatalf("Sign: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported key type") {
		t.Fatalf("error = %v, want contains 'unsupported key type'", err)
	}
}

func TestVaultSignerKeyTypeMissingFallsBackToECDSA(t *testing.T) {
	// Read-key response with no "type" field — older Vault API. Should warn
	// and fall back to ecdsa-p256.
	srv := httptest.NewServer(vaultMockHandler(t, "", testP256PublicKeyPEM(t)))
	t.Cleanup(srv.Close)

	client := newVaultClient(t, srv.URL)
	var warn bytes.Buffer
	s := &VaultSigner{client: client, logical: client.Logical(), keyID: "test-key", stderr: &warn}

	_, _, _, _, algorithm, err := s.Sign(context.Background(), make([]byte, 32))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if algorithm != "ecdsa-p256" {
		t.Fatalf("algorithm = %q, want ecdsa-p256", algorithm)
	}
	if !strings.Contains(warn.String(), "falling back to ecdsa-p256") {
		t.Fatalf("warning = %q, want contains 'falling back to ecdsa-p256'", warn.String())
	}
}

func testP256PublicKeyPEM(t *testing.T) string {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}
