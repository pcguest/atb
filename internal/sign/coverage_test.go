// SPDX-License-Identifier: MIT
package sign

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/event"
)

func TestParseAndVerifyBundleSignatureErrors(t *testing.T) {
	for _, data := range []any{
		"not-object",
		map[string]any{},
		map[string]any{"bundle_hash": "hash"},
		map[string]any{"bundle_hash": "hash", "signature": "signature"},
	} {
		if _, err := ParseBundleSignature(data); err == nil {
			t.Fatalf("ParseBundleSignature(%#v) succeeded", data)
		}
	}
	parsed, err := ParseBundleSignature(map[string]any{
		"bundle_hash": " hash ", "signature": " signature ", "public_key": " key ",
		"algorithm": " ED25519 ", "key_id": 3,
	})
	if err != nil || parsed.PublicKey != "key" || parsed.Algorithm != "ed25519" || parsed.KeyID != "" {
		t.Fatalf("parsed=%+v err=%v", parsed, err)
	}

	expected := sha256.Sum256([]byte("bundle"))
	base := BundleSignature{BundleHash: hex.EncodeToString(expected[:]), Algorithm: "ed25519"}
	tests := []struct {
		name      string
		signature BundleSignature
		hash      []byte
		want      string
	}{
		{name: "expected hash length", signature: base, hash: []byte("short"), want: "expected bundle hash"},
		{name: "hash hex", signature: BundleSignature{BundleHash: "zz"}, hash: expected[:], want: "decode bundle_hash"},
		{name: "hash length", signature: BundleSignature{BundleHash: "aa"}, hash: expected[:], want: "invalid SHA-256 length"},
		{name: "hash mismatch", signature: BundleSignature{BundleHash: strings.Repeat("00", 32)}, hash: expected[:], want: "bundle_hash mismatch"},
		{name: "signature base64", signature: BundleSignature{BundleHash: base.BundleHash, Signature: "!!!", PublicKey: "AA=="}, hash: expected[:], want: "decode signature"},
		{name: "public key base64", signature: BundleSignature{BundleHash: base.BundleHash, Signature: "AA==", PublicKey: "!!!"}, hash: expected[:], want: "decode pubkey"},
		{name: "signature length", signature: BundleSignature{BundleHash: base.BundleHash, Signature: "AA==", PublicKey: base64.StdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))}, hash: expected[:], want: "signature length"},
		{name: "public key length", signature: BundleSignature{BundleHash: base.BundleHash, Signature: base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)), PublicKey: "AA=="}, hash: expected[:], want: "public key length"},
		{name: "unknown algorithm", signature: BundleSignature{BundleHash: base.BundleHash, Signature: "AA==", PublicKey: "AA==", Algorithm: "quantum"}, hash: expected[:], want: "unsupported algorithm"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := VerifyBundleSignature(tc.signature, tc.hash)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v want=%q", err, tc.want)
			}
		})
	}
}

func TestParseECDSAP256PublicKeyRejectsWrongShapes(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rsaDER, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseECDSAP256PublicKey(rsaDER); err == nil || !strings.Contains(err.Error(), "not ECDSA") {
		t.Fatalf("RSA key error=%v", err)
	}
	p384Key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	p384DER, err := x509.MarshalPKIXPublicKey(&p384Key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseECDSAP256PublicKey(p384DER); err == nil || !strings.Contains(err.Error(), "not P-256") {
		t.Fatalf("P-384 key error=%v", err)
	}
	if _, err := parseECDSAP256PublicKey([]byte("short")); err == nil {
		t.Fatal("short key accepted")
	}
	invalidPoint := make([]byte, 65)
	invalidPoint[0] = 0x04
	if _, err := parseECDSAP256PublicKey(invalidPoint); err == nil {
		t.Fatal("invalid point accepted")
	}
}

func TestPolicySignatureValidationErrors(t *testing.T) {
	if _, err := SignPolicyDecision(nil, make(ed25519.PrivateKey, ed25519.PrivateKeySize)); err == nil {
		t.Fatal("nil policy fields signed")
	}
	if _, err := SignPolicyDecision(map[string]any{}, []byte("short")); err == nil {
		t.Fatal("short private key accepted")
	}
	if _, err := SignPolicyDoc(map[string]any{}, "hash", []byte("short")); err == nil {
		t.Fatal("short policy-doc key accepted")
	}
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SignPolicyDoc(map[string]any{"decision": "allow"}, "zz", privateKey); err == nil {
		t.Fatal("invalid document hash signed")
	}

	verifyCases := []map[string]any{
		{event.FieldPolicySignature: "AA=="},
		{event.FieldPolicySignature: "!!!", event.FieldPolicySignerPubKey: "AA=="},
		{event.FieldPolicySignature: "AA==", event.FieldPolicySignerPubKey: "AA=="},
		{
			event.FieldPolicySignature:    base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)),
			event.FieldPolicySignerPubKey: "!!!",
		},
		{
			event.FieldPolicySignature:    base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)),
			event.FieldPolicySignerPubKey: "AA==",
		},
	}
	for i, fields := range verifyCases {
		if err := VerifyPolicyDecision(fields); err == nil || !errors.Is(err, ErrSignatureInvalid) {
			t.Fatalf("VerifyPolicyDecision case %d error=%v", i, err)
		}
	}

	docCases := []map[string]any{
		{event.FieldPolicyDocSignature: "AA=="},
		{event.FieldPolicyDocSignature: "AA==", event.FieldPolicySignerPubKey: "AA=="},
		{
			event.FieldPolicyDocSignature: "AA==", event.FieldPolicySignerPubKey: "AA==",
			event.FieldPolicyDocHash: "zz",
		},
		{
			event.FieldPolicyDocSignature: "!!!", event.FieldPolicySignerPubKey: "AA==",
			event.FieldPolicyDocHash: strings.Repeat("00", 32),
		},
		{
			event.FieldPolicyDocSignature: "AA==", event.FieldPolicySignerPubKey: "AA==",
			event.FieldPolicyDocHash: strings.Repeat("00", 32),
		},
		{
			event.FieldPolicyDocSignature: base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)),
			event.FieldPolicySignerPubKey: "!!!", event.FieldPolicyDocHash: strings.Repeat("00", 32),
		},
	}
	for i, fields := range docCases {
		if err := VerifyPolicyDocSignature(fields); err == nil {
			t.Fatalf("VerifyPolicyDocSignature case %d succeeded", i)
		}
	}
}
