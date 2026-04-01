package sign

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
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
