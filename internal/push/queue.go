// SPDX-License-Identifier: MIT
package push

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// QueueEnvelope is the signed JSON payload sent to a queue gateway.
type QueueEnvelope struct {
	BundleID      string `json:"bundle_id"`
	Digest        string `json:"digest"`
	SealTimestamp string `json:"seal_timestamp"`
	ProfileID     string `json:"profile_id"`
	ATBVersion    string `json:"atb_version"`
}

// QueuePusher publishes signed push envelopes to an HTTP endpoint.
type QueuePusher struct {
	EndpointURL string
	HMACKey     []byte
	ATBVersion  string
	HTTPClient  *http.Client
}

// MarshalEnvelope returns the JSON envelope for meta.
func (p QueuePusher) MarshalEnvelope(meta PushMeta) ([]byte, error) {
	envelope := QueueEnvelope{
		BundleID:      meta.BundleID,
		Digest:        meta.Digest,
		SealTimestamp: meta.SealTimestamp.UTC().Format(time.RFC3339),
		ProfileID:     meta.ProfileID,
		ATBVersion:    p.ATBVersion,
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("queue push: marshal envelope: %w", err)
	}
	return body, nil
}

// SignatureHex returns the hex HMAC-SHA256 for body.
func (p QueuePusher) SignatureHex(body []byte) string {
	mac := hmac.New(sha256.New, p.HMACKey)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Push publishes the signed envelope to the configured endpoint.
func (p QueuePusher) Push(ctx context.Context, bundle []byte, meta PushMeta) error {
	_ = bundle

	if strings.TrimSpace(p.EndpointURL) == "" {
		return fmt.Errorf("queue push: endpoint URL is required")
	}
	if len(p.HMACKey) == 0 {
		return fmt.Errorf("queue push: HMAC key is required")
	}
	if strings.TrimSpace(p.ATBVersion) == "" {
		return fmt.Errorf("queue push: ATB version is required")
	}

	body, err := p.MarshalEnvelope(meta)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.EndpointURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("queue push: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ATB-Signature", p.SignatureHex(body))

	hc := p.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}

	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("queue push: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("queue push: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}
