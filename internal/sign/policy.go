package sign

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/pcguest/atb/internal/canonicalize"
	"github.com/pcguest/atb/internal/event"
)

var (
	ErrSignatureAbsent  = errors.New("policy signature absent")
	ErrSignatureInvalid = errors.New("policy signature invalid")
)

type policySignatureError struct {
	kind   error
	detail string
}

func (e *policySignatureError) Error() string {
	return e.detail
}

func (e *policySignatureError) Unwrap() error {
	return e.kind
}

func SignPolicyDecision(fields map[string]any, privKey ed25519.PrivateKey) (string, error) {
	if len(privKey) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("sign policy decision: invalid Ed25519 private key")
	}

	payload, err := canonicalPolicyDecisionPayload(fields)
	if err != nil {
		return "", fmt.Errorf("sign policy decision: %w", err)
	}

	signature := ed25519.Sign(privKey, payload)
	return base64.StdEncoding.EncodeToString(signature), nil
}

func VerifyPolicyDecision(fields map[string]any) error {
	signatureText := fieldString(fields, event.FieldPolicySignature)
	if signatureText == "" {
		return &policySignatureError{
			kind:   ErrSignatureAbsent,
			detail: "policy_signature absent",
		}
	}

	publicKeyText := fieldString(fields, event.FieldPolicySignerPubKey)
	if publicKeyText == "" {
		return &policySignatureError{
			kind:   ErrSignatureInvalid,
			detail: "policy_signer_pubkey absent",
		}
	}

	payload, err := canonicalPolicyDecisionPayload(fields)
	if err != nil {
		return &policySignatureError{
			kind:   ErrSignatureInvalid,
			detail: fmt.Sprintf("canonicalize payload: %v", err),
		}
	}

	signature, err := base64.StdEncoding.DecodeString(signatureText)
	if err != nil {
		return &policySignatureError{
			kind:   ErrSignatureInvalid,
			detail: fmt.Sprintf("decode policy_signature: %v", err),
		}
	}
	if len(signature) != ed25519.SignatureSize {
		return &policySignatureError{
			kind:   ErrSignatureInvalid,
			detail: "decode policy_signature: invalid Ed25519 signature length",
		}
	}

	publicKey, err := base64.StdEncoding.DecodeString(publicKeyText)
	if err != nil {
		return &policySignatureError{
			kind:   ErrSignatureInvalid,
			detail: fmt.Sprintf("decode policy_signer_pubkey: %v", err),
		}
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return &policySignatureError{
			kind:   ErrSignatureInvalid,
			detail: "decode policy_signer_pubkey: invalid Ed25519 public key length",
		}
	}

	if !ed25519.Verify(ed25519.PublicKey(publicKey), payload, signature) {
		return &policySignatureError{
			kind:   ErrSignatureInvalid,
			detail: "signature mismatch",
		}
	}

	return nil
}

func canonicalPolicyDecisionPayload(fields map[string]any) ([]byte, error) {
	if fields == nil {
		return nil, fmt.Errorf("policy decision fields are nil")
	}

	payload := make(map[string]any, len(fields))
	for key, value := range fields {
		if key == event.FieldPolicySignature || key == event.FieldPolicySignerPubKey {
			continue
		}
		payload[key] = value
	}

	canonical, err := canonicalize.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return canonical, nil
}

func fieldString(fields map[string]any, key string) string {
	if fields == nil {
		return ""
	}
	value, ok := fields[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}
