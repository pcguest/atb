// SPDX-License-Identifier: MIT
package anchor

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequestAndHashBundleBoundaries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/timestamp-query" {
			t.Errorf("content type=%q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("timestamp-response"))
	}))
	t.Cleanup(server.Close)
	got, err := Request(server.URL, []byte("hash"))
	if err != nil || string(got) != "timestamp-response" {
		t.Fatalf("Request=%q err=%v", got, err)
	}

	statusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(statusServer.Close)
	if _, err := Request(statusServer.URL, []byte("hash")); err == nil || !strings.Contains(err.Error(), "returned 503") {
		t.Fatalf("status error=%v", err)
	}
	if _, err := Request("://bad-url", []byte("hash")); err == nil || !strings.Contains(err.Error(), "invalid TSA URL") {
		t.Fatalf("URL error=%v", err)
	}
	for _, rawURL := range []string{
		"file:///tmp/tsa",
		"https://user:password@tsa.example.test",
		"https://",
	} {
		if _, err := Request(rawURL, []byte("hash")); err == nil || !strings.Contains(err.Error(), "invalid TSA URL") {
			t.Fatalf("Request(%q) error=%v, want invalid TSA URL", rawURL, err)
		}
	}

	path := filepath.Join(t.TempDir(), "bundle.atb")
	body := []byte("bundle")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	hash, err := HashBundle(path)
	want := sha256.Sum256(body)
	if err != nil || string(hash) != string(want[:]) {
		t.Fatalf("HashBundle=%x err=%v", hash, err)
	}
	if _, err := HashBundle(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing bundle hash succeeded")
	}
}

func TestASN1SequenceAndSetValidation(t *testing.T) {
	sequence, err := asn1.Marshal([]int{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	if values, err := parseSequenceElements(sequence); err != nil || len(values) != 2 {
		t.Fatalf("sequence values=%d err=%v", len(values), err)
	}
	set := append([]byte{0x31, sequence[1]}, sequence[2:]...)
	if values, err := parseSetElements(set); err != nil || len(values) != 2 {
		t.Fatalf("set values=%d err=%v", len(values), err)
	}

	for _, data := range [][]byte{
		{0xff},
		append(append([]byte(nil), sequence...), 0x00),
		{0x02, 0x01, 0x01},
		{0x30, 0x02, 0xff, 0xff},
	} {
		if _, err := parseSequenceElements(data); err == nil {
			t.Fatalf("invalid sequence %x succeeded", data)
		}
	}
	for _, data := range [][]byte{
		{0xff},
		append(append([]byte(nil), set...), 0x00),
		sequence,
		{0x31, 0x02, 0xff, 0xff},
	} {
		if _, err := parseSetElements(data); err == nil {
			t.Fatalf("invalid set %x succeeded", data)
		}
	}
}

func TestVerifySignerSignatureAlgorithms(t *testing.T) {
	content := []byte("timestamp-info")
	digest := sha256.Sum256(content)
	base := signerInfo{DigestAlgorithm: algorithmIdentifier{Algorithm: oidDigestAlgorithmSHA256}}

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rsaSignature, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	rsaInfo := base
	rsaInfo.Signature = rsaSignature
	rsaCert := &x509.Certificate{PublicKey: &rsaKey.PublicKey, PublicKeyAlgorithm: x509.RSA}
	if err := verifySignerSignature(rsaInfo, rsaCert, content); err != nil {
		t.Fatalf("RSA signature: %v", err)
	}
	rsaInfo.Signature = []byte("bad")
	if err := verifySignerSignature(rsaInfo, rsaCert, content); err == nil {
		t.Fatal("invalid RSA signature accepted")
	}

	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ecdsaSignature, err := ecdsa.SignASN1(rand.Reader, ecdsaKey, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	ecdsaInfo := base
	ecdsaInfo.Signature = ecdsaSignature
	ecdsaCert := &x509.Certificate{PublicKey: &ecdsaKey.PublicKey, PublicKeyAlgorithm: x509.ECDSA}
	if err := verifySignerSignature(ecdsaInfo, ecdsaCert, content); err != nil {
		t.Fatalf("ECDSA signature: %v", err)
	}
	ecdsaInfo.Signature = []byte("bad")
	if err := verifySignerSignature(ecdsaInfo, ecdsaCert, content); err == nil {
		t.Fatal("invalid ECDSA signature accepted")
	}

	wrongDigest := base
	wrongDigest.DigestAlgorithm.Algorithm = asn1.ObjectIdentifier{1, 2, 3}
	if err := verifySignerSignature(wrongDigest, rsaCert, content); err == nil {
		t.Fatal("unsupported digest accepted")
	}
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	unsupportedCert := &x509.Certificate{PublicKey: publicKey, PublicKeyAlgorithm: x509.Ed25519}
	if err := verifySignerSignature(base, unsupportedCert, content); err == nil {
		t.Fatal("unsupported public key accepted")
	}
}
