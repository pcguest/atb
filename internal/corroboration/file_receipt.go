// SPDX-License-Identifier: MIT
package corroboration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// FileReceiptAdapter reads a local JSON receipt file and records its digest.
type FileReceiptAdapter struct {
	Path string
}

// Fetch reads the configured file, validates JSON, and returns a Record.
func (a *FileReceiptAdapter) Fetch(_ context.Context, referenceID string) (Record, error) {
	if strings.TrimSpace(a.Path) == "" {
		return Record{}, fmt.Errorf("corroboration: file path is required")
	}

	body, err := os.ReadFile(a.Path)
	if err != nil {
		return Record{}, fmt.Errorf("corroboration: read file %s: %w", a.Path, err)
	}
	if !json.Valid(body) {
		return Record{}, fmt.Errorf("corroboration: file %s is not valid JSON", a.Path)
	}

	sum := sha256.Sum256(body)
	digest := hex.EncodeToString(sum[:])

	rec := Record{
		Source:      "file-receipt",
		ReferenceID: referenceID,
		Digest:      digest,
		RetrievedAt: time.Now().UTC(),
		Adapter:     "file-receipt",
	}
	if len(body) <= MaxRawEvidenceBytes {
		rec.RawEvidence = body
	} else {
		rec.Truncated = true
	}
	return rec, nil
}
