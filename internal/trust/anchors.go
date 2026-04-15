package trust

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	anchorpkg "github.com/pcguest/atb/internal/anchor"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

// AnchorVerificationResult describes the outcome of verifying one RFC 3161
// anchor record in a bundle.
type AnchorVerificationResult struct {
	AnchorIndex         int    `json:"anchor_index"`
	BundleDigest        string `json:"bundle_digest"`
	TSAGenTime          string `json:"tsa_gen_time"`
	MessageImprintMatch bool   `json:"message_imprint_match"`
	// TSAVerified is true when the message imprint matches and the TSA
	// signature verifies against the signer certificate.
	TSAVerified bool `json:"tsa_verified"`
	// CertChainVerified is true when the TSA certificate chain was verified
	// against a trusted root pool (system roots or caller-supplied).
	CertChainVerified bool   `json:"cert_chain_verified"`
	Error             string `json:"error,omitempty"`
}

type trustAnchorEventData struct {
	TSAGenTime string `json:"certified_time"`
	BundleHash string `json:"bundle_hash"`
	TSRHash    string `json:"tsr_hash"`
	TSRDER     string `json:"tsr_der"`
}

// VerifyAnchors scans the bundle for anchor records and verifies each one
// using the full RFC 3161 chain path (internal/anchor). Returns one result
// per anchor record and an empty slice when none exist.
func VerifyAnchors(b *bundle.Bundle) []AnchorVerificationResult {
	if b == nil {
		return nil
	}

	results := make([]AnchorVerificationResult, 0, len(b.Records))
	for i, record := range b.Records {
		if record.Event.Type != event.TypeBundleAnchor {
			continue
		}

		result := AnchorVerificationResult{AnchorIndex: i}

		raw, ok := record.Event.Data.(string)
		if !ok {
			result.Error = "anchor event data must be a JSON string"
			results = append(results, result)
			continue
		}

		var data trustAnchorEventData
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			result.Error = fmt.Sprintf("parse anchor event data: %v", err)
			results = append(results, result)
			continue
		}

		result.BundleDigest = data.BundleHash
		result.TSAGenTime = data.TSAGenTime

		tsrBytes, err := decodeTSRBytes(data.TSRDER, data.TSRHash)
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}

		hashBytes, err := hex.DecodeString(strings.TrimSpace(data.BundleHash))
		if err != nil {
			result.Error = fmt.Sprintf("decode bundle hash: %v", err)
			results = append(results, result)
			continue
		}

		verification, err := anchorpkg.VerifyTokenDetailed(tsrBytes, hashBytes, nil)
		result.MessageImprintMatch = verification.MessageImprintVerified
		result.CertChainVerified = verification.CertChainVerified
		// TSAVerified: imprint matches and signer signature verified, regardless
		// of whether the full cert chain was trusted by the system root store.
		result.TSAVerified = verification.MessageImprintVerified && verification.SignatureVerified
		if err != nil {
			result.Error = err.Error()
		}
		results = append(results, result)
	}

	return results
}

func decodeTSRBytes(tsrDER, tsrHash string) ([]byte, error) {
	if value := strings.TrimSpace(tsrDER); value != "" {
		decoded, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return nil, fmt.Errorf("decode embedded TimeStampResponse bytes: %w", err)
		}
		if len(decoded) == 0 {
			return nil, fmt.Errorf("embedded TimeStampResponse bytes are empty")
		}
		return decoded, nil
	}

	value := strings.TrimSpace(tsrHash)
	switch {
	case value == "":
		return nil, fmt.Errorf("anchor event tsr_der field is empty; embedded TimeStampResponse bytes are unavailable")
	case isHexDigest(value):
		return nil, fmt.Errorf("anchor event tsr_hash field contains only the token digest; embedded TimeStampResponse bytes are unavailable")
	default:
		decoded, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return nil, fmt.Errorf("decode embedded TimeStampResponse bytes: %w", err)
		}
		if len(decoded) == 0 {
			return nil, fmt.Errorf("embedded TimeStampResponse bytes are empty")
		}
		return decoded, nil
	}
}

func isHexDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
