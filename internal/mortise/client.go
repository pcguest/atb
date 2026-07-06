package mortise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const receiptVersion = "custos.receipt.v1"

// Receipt is the custody receipt returned by the Mortise ingest endpoint.
// Fields mirror custos.receipt.v1. The transparency-log fields are present
// only when the Mortise deployment runs an inclusion log.
type Receipt struct {
	ReceiptVersion string `json:"receipt_version"`
	ReceiptID      string `json:"receipt_id"`
	BundleHash     string `json:"bundle_hash"`
	ContentHash    string `json:"content_hash"`
	ProfileID      string `json:"profile_id,omitempty"`
	SubmittedAt    string `json:"submitted_at,omitempty"`
	LeafIndex      uint64 `json:"leaf_index,omitempty"`
	Checkpoint     string `json:"checkpoint,omitempty"`
	// Raw preserves the complete signed response, including attestation and
	// transparency fields that this lightweight ATB client does not interpret.
	Raw json.RawMessage `json:"-"`
}

// HTTPClient submits ATB bundles to a Mortise ingest endpoint.
type HTTPClient struct {
	endpoint string
	token    string
	client   *http.Client
}

// NewHTTPClient creates a client for a validated Mortise base URL and bearer
// token. Credentials in URLs are rejected so they cannot leak through logs.
func NewHTTPClient(endpoint, token string) (*HTTPClient, error) {
	endpoint = strings.TrimSpace(endpoint)
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("invalid Mortise endpoint")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("mortise endpoint must not contain credentials, query, or fragment")
	}
	return &HTTPClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		token:    token,
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

// Endpoint returns the configured Mortise endpoint URL.
func (c *HTTPClient) Endpoint() string {
	return c.endpoint
}

// SendBundle posts a full ATB bundle to the Mortise ingest endpoint. Mortise
// verifies the bundle before persisting it and returns a signed custody
// receipt. Mortise ingests whole bundles, not individual events.
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
		return nil, fmt.Errorf("send bundle to Mortise: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil {
		return nil, fmt.Errorf("read Mortise response: %w", err)
	}
	if len(body) > 1<<20 {
		return nil, fmt.Errorf("mortise response exceeds 1 MiB")
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mortise ingest returned %d %s: %s", resp.StatusCode, resp.Status, bytes.TrimSpace(body))
	}

	var receipt Receipt
	if err := json.Unmarshal(body, &receipt); err != nil {
		return nil, fmt.Errorf("decode Mortise receipt: %w", err)
	}
	if receipt.ReceiptVersion != receiptVersion {
		return nil, fmt.Errorf("unsupported Mortise receipt version %q", receipt.ReceiptVersion)
	}
	if strings.TrimSpace(receipt.ReceiptID) == "" || strings.TrimSpace(receipt.BundleHash) == "" {
		return nil, fmt.Errorf("mortise receipt is missing receipt_id or bundle_hash")
	}
	receipt.Raw = append(json.RawMessage(nil), body...)
	return &receipt, nil
}
