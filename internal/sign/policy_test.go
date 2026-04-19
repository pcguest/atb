package sign

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/pcguest/atb/internal/event"
)

func TestSignVerify_RoundTrip(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate keypair: %v", err)
	}

	fields := map[string]any{
		"policy_id":             "policy-1",
		"policy_version":        "2026-04",
		"decision":              "allow",
		"decision_reason_codes": []string{"ticket_present"},
		"subject_id_hash":       "subject-1",
		"action_id":             "act-1",
	}

	signature, err := SignPolicyDecision(fields, privateKey)
	if err != nil {
		t.Fatalf("sign policy decision: %v", err)
	}

	fields[event.FieldPolicySignature] = signature
	fields[event.FieldPolicySignerPubKey] = base64.StdEncoding.EncodeToString(publicKey)

	if err := VerifyPolicyDecision(fields); err != nil {
		t.Fatalf("verify policy decision: %v", err)
	}
}

func TestVerify_AbsentSignature(t *testing.T) {
	fields := map[string]any{
		"policy_id":      "policy-1",
		"policy_version": "2026-04",
	}

	err := VerifyPolicyDecision(fields)
	if err == nil {
		t.Fatalf("expected absent signature error")
	}
	if !errors.Is(err, ErrSignatureAbsent) {
		t.Fatalf("expected ErrSignatureAbsent, got %v", err)
	}
}

func TestSignVerifyPolicyDoc_RoundTrip(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate keypair: %v", err)
	}

	docContents := []byte("This is the policy document text.")
	sum := sha256.Sum256(docContents)
	docHash := hex.EncodeToString(sum[:])

	fields := map[string]any{
		"policy_id":             "policy-1",
		"policy_version":        "2026-04",
		"decision":              "allow",
		"decision_reason_codes": []string{"ticket_present"},
		"subject_id_hash":       "subject-1",
		"action_id":             "act-1",
		event.FieldPolicyDocHash: docHash,
	}

	sig, err := SignPolicyDoc(fields, docHash, privateKey)
	if err != nil {
		t.Fatalf("SignPolicyDoc: %v", err)
	}
	fields[event.FieldPolicyDocSignature] = sig
	fields[event.FieldPolicySignerPubKey] = base64.StdEncoding.EncodeToString(publicKey)

	if err := VerifyPolicyDocSignature(fields); err != nil {
		t.Fatalf("VerifyPolicyDocSignature: %v", err)
	}
}

func TestVerifyPolicyDocSignature_AbsentSignature(t *testing.T) {
	sum := sha256.Sum256([]byte("doc"))
	fields := map[string]any{
		event.FieldPolicyDocHash:    hex.EncodeToString(sum[:]),
		event.FieldPolicySignerPubKey: "somepubkey",
	}
	err := VerifyPolicyDocSignature(fields)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrSignatureAbsent) {
		t.Fatalf("expected ErrSignatureAbsent, got %v", err)
	}
}

func TestVerifyPolicyDocSignature_TamperedDoc(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate keypair: %v", err)
	}

	realDoc := []byte("original policy text")
	sum := sha256.Sum256(realDoc)
	docHash := hex.EncodeToString(sum[:])

	fields := map[string]any{
		"policy_id":            "policy-1",
		"decision":             "allow",
		event.FieldPolicyDocHash: docHash,
	}
	sig, err := SignPolicyDoc(fields, docHash, privateKey)
	if err != nil {
		t.Fatalf("SignPolicyDoc: %v", err)
	}
	fields[event.FieldPolicyDocSignature] = sig
	fields[event.FieldPolicySignerPubKey] = base64.StdEncoding.EncodeToString(publicKey)

	// Replace the doc hash as if the doc was replaced.
	tamperedDoc := []byte("tampered policy text")
	tamperedSum := sha256.Sum256(tamperedDoc)
	fields[event.FieldPolicyDocHash] = hex.EncodeToString(tamperedSum[:])

	if err := VerifyPolicyDocSignature(fields); !errors.Is(err, ErrSignatureInvalid) {
		t.Fatalf("expected ErrSignatureInvalid for tampered doc, got %v", err)
	}
}

func TestVerifyPolicyDocSignature_TamperedEvent(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate keypair: %v", err)
	}

	sum := sha256.Sum256([]byte("doc"))
	docHash := hex.EncodeToString(sum[:])

	fields := map[string]any{
		"policy_id":            "policy-1",
		"decision":             "allow",
		event.FieldPolicyDocHash: docHash,
	}
	sig, err := SignPolicyDoc(fields, docHash, privateKey)
	if err != nil {
		t.Fatalf("SignPolicyDoc: %v", err)
	}
	fields[event.FieldPolicyDocSignature] = sig
	fields[event.FieldPolicySignerPubKey] = base64.StdEncoding.EncodeToString(publicKey)

	// Tamper with the event decision.
	fields["decision"] = "deny"

	if err := VerifyPolicyDocSignature(fields); !errors.Is(err, ErrSignatureInvalid) {
		t.Fatalf("expected ErrSignatureInvalid for tampered event, got %v", err)
	}
}

func TestVerify_TamperedPayload(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate keypair: %v", err)
	}

	fields := map[string]any{
		"policy_id":             "policy-1",
		"policy_version":        "2026-04",
		"decision":              "allow",
		"decision_reason_codes": []string{"ticket_present"},
		"subject_id_hash":       "subject-1",
		"action_id":             "act-1",
	}

	signature, err := SignPolicyDecision(fields, privateKey)
	if err != nil {
		t.Fatalf("sign policy decision: %v", err)
	}

	fields[event.FieldPolicySignature] = signature
	fields[event.FieldPolicySignerPubKey] = base64.StdEncoding.EncodeToString(publicKey)
	fields["decision"] = "deny"

	err = VerifyPolicyDecision(fields)
	if err == nil {
		t.Fatalf("expected invalid signature error")
	}
	if !errors.Is(err, ErrSignatureInvalid) {
		t.Fatalf("expected ErrSignatureInvalid, got %v", err)
	}
}
