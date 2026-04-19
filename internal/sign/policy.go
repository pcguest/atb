package sign

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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

// SignPolicyDoc produces a compound Ed25519 signature over:
//
//	SHA-256(canonical policy-decision payload) || SHA-256(docHash bytes)
//
// The docHash argument is the hex-encoded SHA-256 of the policy document
// file contents (as already embedded in the event fields).
func SignPolicyDoc(fields map[string]any, docHash string, privKey ed25519.PrivateKey) (string, error) {
	if len(privKey) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("sign policy doc: invalid Ed25519 private key")
	}
	payload, err := canonicalPolicyDecisionPayload(fields)
	if err != nil {
		return "", fmt.Errorf("sign policy doc: %w", err)
	}
	payloadHash := sha256.Sum256(payload)
	docHashBytes, err := hex.DecodeString(docHash)
	if err != nil {
		return "", fmt.Errorf("sign policy doc: decode doc hash: %w", err)
	}
	msg := append(payloadHash[:], docHashBytes...)
	sig := ed25519.Sign(privKey, msg)
	return base64.StdEncoding.EncodeToString(sig), nil
}

// VerifyPolicyDocSignature verifies the compound policy-doc signature stored
// in a policy-decision event. It returns nil on success.
func VerifyPolicyDocSignature(fields map[string]any) error {
	sigText := fieldString(fields, event.FieldPolicyDocSignature)
	if sigText == "" {
		return &policySignatureError{kind: ErrSignatureAbsent, detail: "policy_doc_signature absent"}
	}
	pubKeyText := fieldString(fields, event.FieldPolicySignerPubKey)
	if pubKeyText == "" {
		return &policySignatureError{kind: ErrSignatureInvalid, detail: "policy_signer_pubkey absent"}
	}
	docHashText := fieldString(fields, event.FieldPolicyDocHash)
	if docHashText == "" {
		return &policySignatureError{kind: ErrSignatureInvalid, detail: "policy_doc_hash absent"}
	}

	// Reconstruct the message: SHA-256(canonical payload) || SHA-256(doc hash bytes).
	fieldsForPayload := make(map[string]any, len(fields))
	for k, v := range fields {
		if k == event.FieldPolicyDocSignature {
			continue
		}
		fieldsForPayload[k] = v
	}
	payload, err := canonicalPolicyDecisionPayload(fieldsForPayload)
	if err != nil {
		return &policySignatureError{kind: ErrSignatureInvalid, detail: fmt.Sprintf("canonicalize payload: %v", err)}
	}
	payloadHash := sha256.Sum256(payload)
	docHashBytes, err := hex.DecodeString(docHashText)
	if err != nil {
		return &policySignatureError{kind: ErrSignatureInvalid, detail: fmt.Sprintf("decode policy_doc_hash: %v", err)}
	}
	msg := append(payloadHash[:], docHashBytes...)

	sig, err := base64.StdEncoding.DecodeString(sigText)
	if err != nil {
		return &policySignatureError{kind: ErrSignatureInvalid, detail: fmt.Sprintf("decode policy_doc_signature: %v", err)}
	}
	if len(sig) != ed25519.SignatureSize {
		return &policySignatureError{kind: ErrSignatureInvalid, detail: "policy_doc_signature: invalid length"}
	}

	pubKey, err := base64.StdEncoding.DecodeString(pubKeyText)
	if err != nil {
		return &policySignatureError{kind: ErrSignatureInvalid, detail: fmt.Sprintf("decode policy_signer_pubkey: %v", err)}
	}
	if len(pubKey) != ed25519.PublicKeySize {
		return &policySignatureError{kind: ErrSignatureInvalid, detail: "policy_signer_pubkey: invalid length"}
	}

	if !ed25519.Verify(ed25519.PublicKey(pubKey), msg, sig) {
		return &policySignatureError{kind: ErrSignatureInvalid, detail: "policy_doc_signature mismatch"}
	}
	return nil
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
