package custos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Receipt is the custody receipt returned by the Custos ingest endpoint.
// Fields mirror custos.receipt.v1. The transparency-log fields are present
// only when the Custos deployment runs an inclusion log.
type Receipt struct {
	ReceiptVersion string `json:"receipt_version"`
	ReceiptID      string `json:"receipt_id"`
	BundleHash     string `json:"bundle_hash"`
	ContentHash    string `json:"content_hash"`
	ProfileID      string `json:"profile_id,omitempty"`
	SubmittedAt    string `json:"submitted_at,omitempty"`
	LeafIndex      uint64 `json:"leaf_index,omitempty"`
	Checkpoint     string `json:"checkpoint,omitempty"`
}

// HTTPClient submits ATB bundles to a Custos ingest endpoint.
type HTTPClient struct {
	endpoint string
	token    string
	client   *http.Client
}

// NewHTTPClient creates a client for the given Custos endpoint and bearer token.
func NewHTTPClient(endpoint, token string) *HTTPClient {
	return &HTTPClient{
		endpoint: endpoint,
		token:    token,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Endpoint returns the configured Custos endpoint URL.
func (c *HTTPClient) Endpoint() string {
	return c.endpoint
}

// SendBundle posts a full ATB bundle to the Custos ingest endpoint. Custos
// verifies the bundle before persisting it and returns a signed custody
// receipt. Custos ingests whole bundles, not individual events.
func (c *HTTPClient) SendBundle(ctx context.Context, bundleBytes []byte) (*Receipt, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/ingest", bytes.NewReader(bundleBytes))
	if err != nil {
		return nil, fmt.Errorf("create ingest request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send bundle to Custos: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("custos ingest returned %d %s: %s", resp.StatusCode, resp.Status, bytes.TrimSpace(body))
	}

	var receipt Receipt
	if err := json.Unmarshal(body, &receipt); err != nil {
		return nil, fmt.Errorf("decode custos receipt: %w", err)
	}
	return &receipt, nil
}
