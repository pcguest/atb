package verify

import (
	"crypto/sha256"
	"fmt"
	"os"

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

	expectedHash, err := hashBundleSnapshotBeforeRecord(bundlePath, signatureIndex)
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

func hashBundleSnapshotBeforeRecord(bundlePath string, targetRecordIndex int) ([]byte, error) {
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

		if lineEnd > lineStart {
			if recordIndex == targetRecordIndex {
				sum := sha256.Sum256(raw[:lineStart])
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
