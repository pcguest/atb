// SPDX-License-Identifier: MIT
// Package corroboration provides adapters that retrieve external evidence and
// return records ready to append as atb.corroboration.external events.
package corroboration

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// MaxRawEvidenceBytes is the maximum payload size stored in RawEvidence.
	// Payloads larger than this are omitted and Truncated is set to true.
	MaxRawEvidenceBytes = 4096
)

// Record holds corroboration fields ready to append as an
// atb.corroboration.external event.
type Record struct {
	Source      string
	ReferenceID string
	Digest      string
	RetrievedAt time.Time
	Adapter     string
	RawEvidence []byte
	Truncated   bool
}

// EventData returns the record as a map suitable for use as event data
// in an atb.corroboration.external event.
func (r Record) EventData() map[string]any {
	m := map[string]any{
		"source":       r.Source,
		"reference_id": r.ReferenceID,
		"digest":       r.Digest,
		"retrieved_at": r.RetrievedAt.UTC().Format(time.RFC3339),
	}
	if r.Adapter != "" {
		m["adapter"] = r.Adapter
	}
	if len(r.RawEvidence) > 0 {
		m["raw_evidence"] = base64.StdEncoding.EncodeToString(r.RawEvidence)
	}
	if r.Truncated {
		m["truncated"] = true
	}
	return m
}

// Adapter retrieves external corroboration evidence for a given reference ID.
type Adapter interface {
	Fetch(ctx context.Context, referenceID string) (Record, error)
}

// HTTPGatewayAdapter fetches corroboration evidence from an HTTP endpoint.
// The endpoint must return a JSON response body; the adapter records the
// SHA-256 digest of the raw response body.
type HTTPGatewayAdapter struct {
	URL     string
	Timeout time.Duration
}

// Fetch retrieves a JSON receipt from the configured URL, computes the
// SHA-256 digest of the response body, and returns a Record. The
// raw response body is stored in RawEvidence up to MaxRawEvidenceBytes;
// larger payloads set Truncated to true and omit the raw body.
func (a *HTTPGatewayAdapter) Fetch(ctx context.Context, referenceID string) (Record, error) {
	timeout := a.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.URL, nil)
	if err != nil {
		return Record{}, fmt.Errorf("corroboration: build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Record{}, fmt.Errorf("corroboration: fetch %s: %w", a.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Record{}, fmt.Errorf("corroboration: fetch %s: unexpected status %d", a.URL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Record{}, fmt.Errorf("corroboration: read body: %w", err)
	}

	if !json.Valid(body) {
		return Record{}, fmt.Errorf("corroboration: response body is not valid JSON")
	}

	sum := sha256.Sum256(body)
	digest := hex.EncodeToString(sum[:])

	rec := Record{
		Source:      "http-gateway",
		ReferenceID: referenceID,
		Digest:      digest,
		RetrievedAt: time.Now().UTC(),
		Adapter:     "http-gateway",
	}

	if len(body) <= MaxRawEvidenceBytes {
		rec.RawEvidence = body
	} else {
		rec.Truncated = true
	}

	return rec, nil
}
