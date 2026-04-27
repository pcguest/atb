// SPDX-License-Identifier: MIT

package signer

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
)

func newTestKey(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}
	return priv
}

func TestLocalSigner_SignProducesVerifiableSignature(t *testing.T) {
	priv := newTestKey(t)
	s := NewLocalSigner(priv)

	digest := []byte("0123456789abcdef0123456789abcdef") // 32 bytes
	sig, pub, keyID, backend, algorithm, err := s.Sign(context.Background(), digest)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if len(sig) != ed25519.SignatureSize {
		t.Fatalf("signature length = %d, want %d", len(sig), ed25519.SignatureSize)
	}
	if len(pub) != ed25519.PublicKeySize {
		t.Fatalf("pubkey length = %d, want %d", len(pub), ed25519.PublicKeySize)
	}
	if !ed25519.Verify(ed25519.PublicKey(pub), digest, sig) {
		t.Fatalf("ed25519.Verify failed for LocalSigner output")
	}
	if keyID != "" {
		t.Errorf("keyID = %q, want empty", keyID)
	}
	if backend != "" {
		// Empty is the legacy/implicit-local sentinel that keeps newly
		// signed bundles byte-identical to pre-Signer-interface bundles.
		t.Errorf("backend = %q, want empty (legacy local default)", backend)
	}
	if algorithm != "ed25519" {
		t.Errorf("algorithm = %q, want ed25519", algorithm)
	}
}

func TestLocalSigner_SignIsDeterministic(t *testing.T) {
	priv := newTestKey(t)
	s := NewLocalSigner(priv)

	digest := []byte("0123456789abcdef0123456789abcdef")
	sig1, _, _, _, _, err := s.Sign(context.Background(), digest)
	if err != nil {
		t.Fatalf("Sign #1: %v", err)
	}
	sig2, _, _, _, _, err := s.Sign(context.Background(), digest)
	if err != nil {
		t.Fatalf("Sign #2: %v", err)
	}
	if string(sig1) != string(sig2) {
		t.Fatalf("Ed25519 signatures should be deterministic for the same key/message")
	}
}

func TestLocalSigner_SignReturnsDefensiveCopyOfPubKey(t *testing.T) {
	priv := newTestKey(t)
	s := NewLocalSigner(priv)

	digest := []byte("0123456789abcdef0123456789abcdef")
	_, pub1, _, _, _, err := s.Sign(context.Background(), digest)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	// Mutate the returned slice — it must not affect a subsequent Sign call.
	for i := range pub1 {
		pub1[i] = 0xff
	}
	_, pub2, _, _, _, err := s.Sign(context.Background(), digest)
	if err != nil {
		t.Fatalf("Sign #2: %v", err)
	}
	for _, b := range pub2 {
		if b == 0xff {
			continue
		}
		// As soon as we see a non-0xff byte we're confident the second
		// pubkey was not aliased to the first.
		return
	}
	t.Fatalf("second pubkey appears to alias the first (all bytes 0xff)")
}

func TestLocalSigner_RejectsInvalidKey(t *testing.T) {
	cases := []struct {
		name string
		key  ed25519.PrivateKey
	}{
		{"nil key", nil},
		{"empty key", ed25519.PrivateKey{}},
		{"truncated key", ed25519.PrivateKey(make([]byte, ed25519.PrivateKeySize-1))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewLocalSigner(tc.key)
			_, _, _, _, _, err := s.Sign(context.Background(), []byte("digest"))
			if !errors.Is(err, ErrInvalidKey) {
				t.Fatalf("err = %v, want ErrInvalidKey", err)
			}
		})
	}
}

func TestLocalSigner_SatisfiesSignerInterface(t *testing.T) {
	var _ Signer = (*LocalSigner)(nil)
}
