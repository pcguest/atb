// SPDX-License-Identifier: MIT
package verify

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	signpkg "github.com/pcguest/atb/internal/sign"
)

type BundleSignatureResult struct {
	Present    bool   `json:"present"`
	Verified   bool   `json:"verified"`
	BundleHash string `json:"bundle_hash,omitempty"`
	Error      string `json:"error,omitempty"`
}

// SignatureProvenance is one entry in Report.Signatures. It carries the
// optional provenance fields recorded by recent (post-Signer-interface)
// signers, plus the cryptographic verdict for that signature record.
//
// Backend is "local" when the on-disk record carries no explicit backend
// (legacy/local default); otherwise it is the recorded backend string,
// e.g. "https-http" or "local:fallback:https-http".
//
// Valid is true iff the signature verifies cryptographically against the
// bundle prefix that existed immediately before the signature record.
type SignatureProvenance struct {
	Sequence   int    `json:"sequence"`
	Backend    string `json:"backend"`
	KeyID      string `json:"key_id"`
	SignedAt   string `json:"signed_at"`
	PubKey     string `json:"pubkey"`
	BundleHash string `json:"bundle_hash,omitempty"`
	Valid      bool   `json:"valid"`
	Error      string `json:"error,omitempty"`
}

func inspectBundleSignature(b *bundle.Bundle, bundlePath string) *BundleSignatureResult {
	if b == nil {
		return nil
	}

	signatureIndex := latestBundleSignatureIndex(b.Records)
	if signatureIndex < 0 {
		return nil
	}

	result := &BundleSignatureResult{Present: true}
	signature, err := signpkg.ParseBundleSignature(b.Records[signatureIndex].Event.Data)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.BundleHash = signature.BundleHash

	expectedHash, err := hashBundleSnapshotBeforeRecord(bundlePath, signatureIndex, signature.BundleHash)
	if err != nil {
		result.Error = fmt.Sprintf("hash bundle snapshot: %v", err)
		return result
	}

	if err := signpkg.VerifyBundleSignature(signature, expectedHash); err != nil {
		result.Error = err.Error()
		return result
	}

	result.Verified = true
	return result
}

func latestBundleSignatureIndex(records []bundle.Record) int {
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].Event.Type == event.TypeBundleSignature {
			return i
		}
	}
	return -1
}

// inspectAllBundleSignatures walks every atb.bundle.signature record in the
// bundle (in order) and produces a per-signature provenance entry,
// including the cryptographic verdict for that signature against the
// bundle prefix immediately before it.
func inspectAllBundleSignatures(b *bundle.Bundle, bundlePath string) []SignatureProvenance {
	if b == nil {
		return nil
	}
	var out []SignatureProvenance
	for i, record := range b.Records {
		if record.Event.Type != event.TypeBundleSignature {
			continue
		}
		out = append(out, buildSignatureProvenance(record, i, bundlePath))
	}
	return out
}

func buildSignatureProvenance(record bundle.Record, index int, bundlePath string) SignatureProvenance {
	prov := SignatureProvenance{
		Sequence: record.Event.Sequence,
		Backend:  "local", // default label when the record omits an explicit backend
	}

	fields, ok := record.Event.Data.(map[string]any)
	if !ok {
		prov.Error = "signature data is not an object"
		return prov
	}

	if v := stringField(fields, "backend"); v != "" {
		prov.Backend = v
	}
	prov.KeyID = stringField(fields, "key_id")
	prov.SignedAt = stringField(fields, "signed_at")
	prov.PubKey = stringField(fields, "pubkey")
	if prov.PubKey == "" {
		prov.PubKey = stringField(fields, "public_key") // legacy alias
	}

	signature, err := signpkg.ParseBundleSignature(record.Event.Data)
	if err != nil {
		prov.Error = err.Error()
		return prov
	}
	prov.BundleHash = signature.BundleHash

	expectedHash, err := hashBundleSnapshotBeforeRecord(bundlePath, index, signature.BundleHash)
	if err != nil {
		prov.Error = "hash bundle snapshot: " + err.Error()
		return prov
	}
	if err := signpkg.VerifyBundleSignature(signature, expectedHash); err != nil {
		prov.Error = err.Error()
		return prov
	}
	prov.Valid = true
	return prov
}

func stringField(fields map[string]any, key string) string {
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
	return text
}

func hashBundleSnapshotBeforeRecord(bundlePath string, targetRecordIndex int, signedHash string) ([]byte, error) {
	if targetRecordIndex < 0 {
		return nil, fmt.Errorf("record index must be non-negative")
	}

	raw, err := os.ReadFile(bundlePath) // #nosec G304 -- bundle path is provided by the caller and already resolved by the CLI
	if err != nil {
		return nil, err
	}

	recordIndex := 0
	lineStart := 0
	for lineStart < len(raw) {
		lineEnd := lineStart
		for lineEnd < len(raw) && raw[lineEnd] != '\n' {
			lineEnd++
		}

		// The loader skips blank lines, including the CRLF form that scanner
		// line splitting reduces to an empty record. Count physical records the
		// same way so a blank CRLF line cannot consume a signature index.
		blank := lineEnd == lineStart || (lineEnd == lineStart+1 && raw[lineStart] == '\r')
		if !blank {
			if recordIndex == targetRecordIndex {
				sum := sha256.Sum256(raw[:lineStart])
				// Sign hashes the original input bytes, then inserts one LF only
				// if that input was unterminated. Recover that original prefix
				// only when the recorded digest matches exactly; do not trim
				// whitespace, CRs or an already terminated/blank line. Signature
				// verification against this digest is still mandatory below.
				if lineStart > 1 && raw[lineStart-1] == '\n' && raw[lineStart-2] != '\n' {
					original := sha256.Sum256(raw[:lineStart-1])
					if strings.EqualFold(strings.TrimSpace(signedHash), hex.EncodeToString(original[:])) {
						return original[:], nil
					}
				}
				return sum[:], nil
			}
			recordIndex++
		}

		if lineEnd == len(raw) {
			break
		}
		lineStart = lineEnd + 1
	}

	return nil, fmt.Errorf("record index %d out of range", targetRecordIndex)
}
