package encrypt

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/canonicalize"
	"github.com/pcguest/atb/internal/hash"
)

type payloadForTest struct {
	HeadHash string          `json:"head_hash"`
	Records  []bundle.Record `json:"records"`
}

func fixedSalt() []byte {
	return []byte{
		0x10, 0x11, 0x12, 0x13,
		0x14, 0x15, 0x16, 0x17,
		0x18, 0x19, 0x1A, 0x1B,
		0x1C, 0x1D, 0x1E, 0x1F,
	}
}

func fixedNonce() []byte {
	return []byte{
		0x20, 0x21, 0x22, 0x23,
		0x24, 0x25, 0x26, 0x27,
		0x28, 0x29, 0x2A, 0x2B,
	}
}

func TestEncryptDecryptRoundTripDeterministic(t *testing.T) {
	plaintext := []byte(`{"head_hash":"abc","records":[]}`)

	first, err := EncryptWithSaltNonce(plaintext, "test123", fixedSalt(), fixedNonce())
	if err != nil {
		t.Fatalf("encrypt first: %v", err)
	}
	second, err := EncryptWithSaltNonce(plaintext, "test123", fixedSalt(), fixedNonce())
	if err != nil {
		t.Fatalf("encrypt second: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("expected deterministic ciphertext for fixed salt/nonce")
	}
	if len(first) != HeaderSize+len(plaintext) {
		t.Fatalf("unexpected encrypted length: got %d want %d", len(first), HeaderSize+len(plaintext))
	}
	if got := string(first[:len(Magic)]); got != Magic {
		t.Fatalf("unexpected magic: got %q want %q", got, Magic)
	}
	if got := first[len(Magic)]; got != Version {
		t.Fatalf("unexpected version: got 0x%02x want 0x%02x", got, Version)
	}

	gotPlaintext, err := Decrypt(first, "test123")
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(gotPlaintext) != string(plaintext) {
		t.Fatalf("plaintext mismatch\ngot:  %s\nwant: %s", gotPlaintext, plaintext)
	}
}

func TestDecryptWrongPasswordFails(t *testing.T) {
	plaintext := []byte(`{"head_hash":"abc","records":[]}`)
	enc, err := EncryptWithSaltNonce(plaintext, "test123", fixedSalt(), fixedNonce())
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	_, err = Decrypt(enc, "wrong-pass")
	if err == nil {
		t.Fatalf("expected decryption error")
	}
	if !errors.Is(err, ErrDecryptFailed) {
		t.Fatalf("expected ErrDecryptFailed, got: %v", err)
	}
}

func TestDecryptTamperedCiphertextFails(t *testing.T) {
	plaintext := []byte(`{"head_hash":"abc","records":[]}`)
	enc, err := EncryptWithSaltNonce(plaintext, "test123", fixedSalt(), fixedNonce())
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	enc[len(enc)-1] ^= 0x01
	_, err = Decrypt(enc, "test123")
	if err == nil {
		t.Fatalf("expected decryption error")
	}
	if !errors.Is(err, ErrDecryptFailed) {
		t.Fatalf("expected ErrDecryptFailed, got: %v", err)
	}
}

func TestDecryptRejectsUnsupportedVersion(t *testing.T) {
	plaintext := []byte(`{"head_hash":"abc","records":[]}`)
	enc, err := EncryptWithSaltNonce(plaintext, "test123", fixedSalt(), fixedNonce())
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	enc[len(Magic)] = 0x02
	_, err = Decrypt(enc, "test123")
	if err == nil {
		t.Fatalf("expected unsupported version error")
	}
	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("expected ErrUnsupportedVersion, got: %v", err)
	}
}

func TestDecryptThenVerify(t *testing.T) {
	b := bundle.New()
	timestamp := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	if err := b.AppendWithOptions("dev.session", map[string]any{"msg": "hello"}, &bundle.AppendOptions{Timestamp: timestamp}); err != nil {
		t.Fatalf("append 1: %v", err)
	}
	if err := b.AppendWithOptions("test.decision", map[string]any{"choice": "ship"}, &bundle.AppendOptions{Timestamp: timestamp}); err != nil {
		t.Fatalf("append 2: %v", err)
	}
	head := hash.GenesisHash
	if len(b.Records) > 0 {
		head = b.Records[len(b.Records)-1].Hash
	}
	payload := payloadForTest{
		HeadHash: head,
		Records:  b.Records,
	}
	plaintext, err := canonicalize.Marshal(payload)
	if err != nil {
		t.Fatalf("canonicalize payload: %v", err)
	}

	enc, err := EncryptWithSaltNonce(plaintext, "test123", fixedSalt(), fixedNonce())
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	dec, err := Decrypt(enc, "test123")
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	var decoded payloadForTest
	if err := json.Unmarshal(dec, &decoded); err != nil {
		t.Fatalf("unmarshal decrypted payload: %v", err)
	}
	events := make([]hash.Event, len(decoded.Records))
	hashes := make([]string, len(decoded.Records))
	for i, r := range decoded.Records {
		events[i] = r.Event
		hashes[i] = r.Hash
	}
	if err := hash.Verify(events, hashes); err != nil {
		t.Fatalf("hash chain verify after decrypt: %v", err)
	}
	recomputedHead := hash.GenesisHash
	if len(decoded.Records) > 0 {
		recomputedHead = decoded.Records[len(decoded.Records)-1].Hash
	}
	if decoded.HeadHash != recomputedHead {
		t.Fatalf("head hash mismatch after decrypt: got %s want %s", decoded.HeadHash, recomputedHead)
	}
}
