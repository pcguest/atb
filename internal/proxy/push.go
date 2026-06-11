package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	// defaultCustosPushRetryDelay is the delay before retrying a Custos push.
	defaultCustosPushRetryDelay = 5 * time.Second
)

// CustosPusher handles pushing bundles to a Custos ingest endpoint.
type CustosPusher struct {
	Endpoint string
	// Token is sent as an Authorization: Bearer header when non-empty.
	// Custos ingest endpoints reject unauthenticated pushes outside dev mode.
	Token  string
	Client *http.Client // HTTP client for making requests
}

// Fixed: This local response shape avoids importing Custos internals across module boundaries.
type pushReceipt struct {
	// Fixed: The proxy only needs the receipt identifier for operator feedback.
	ReceiptID string `json:"receipt_id"`
}

// NewCustosPusher creates a new CustosPusher. token may be empty for
// dev-mode (unauthenticated) Custos endpoints.
func NewCustosPusher(endpoint, token string) (*CustosPusher, error) {
	validated, err := validatePushEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	return &CustosPusher{
		Endpoint: validated,
		Token:    strings.TrimSpace(token),
		Client: &http.Client{
			Timeout: 30 * time.Second, // Reasonable timeout for network operations
		},
	}, nil
}

// validatePushEndpoint parses and normalises an operator-configured Custos ingest URL.
func validatePushEndpoint(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("custos endpoint is not configured")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("custos endpoint URL: %w", err)
	}
	switch parsed.Scheme {
	case "http", "https":
	default:
		return "", fmt.Errorf("custos endpoint must use http or https, got %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("custos endpoint must include a host")
	}
	return parsed.String(), nil
}

// Push reads the bundle from bundlePath, POSTs it to the Custos endpoint,
// parses the receipt, and logs the receipt_id. It retries once on network
// faults.
func (p *CustosPusher) Push(ctx context.Context, bundlePath string) error {
	// Fixed: Push now delegates to PushBytes so both callers share one retry implementation.
	bundleBytes, err := os.ReadFile(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to read bundle from %s: %w", bundlePath, err)
	}
	// Fixed: PushBytes receives the immutable byte snapshot used by the shared POST path.
	return p.PushBytes(ctx, bundleBytes)
}

// Fixed: PushBytes lets callers push a verified snapshot without re-reading a changing bundle file.
func (p *CustosPusher) PushBytes(ctx context.Context, bundleBytes []byte) error {
	// Fixed: Endpoint validation is centralised for both file and byte-snapshot pushes.
	if p.Endpoint == "" {
		return fmt.Errorf("custos endpoint is not configured")
	}
	// Fixed: The shared POST helper keeps retry behaviour identical for both public push methods.
	return p.postBundleBytes(ctx, bundleBytes)
}

// Fixed: postBundleBytes contains the single HTTP POST retry path used by Push and PushBytes.
func (p *CustosPusher) postBundleBytes(ctx context.Context, bundleBytes []byte) error {
	// Attempt to push, with one retry on network errors
	for i := 0; i < 2; i++ { // 0 for initial attempt, 1 for retry
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewReader(bundleBytes))
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}
		req.Header.Set("Content-Type", "application/octet-stream") // Or appropriate content type
		if p.Token != "" {
			req.Header.Set("Authorization", "Bearer "+p.Token)
		}

		resp, err := p.Client.Do(req) // #nosec G704 -- endpoint validated in NewCustosPusher to http/https with host
		if err != nil {
			// Network error, retry if it's the first attempt
			if i == 0 {
				fmt.Fprintf(os.Stderr, "Warning: Custos push failed (attempt %d), retrying in %v: %v\n", i+1, defaultCustosPushRetryDelay, err)
				time.Sleep(defaultCustosPushRetryDelay)
				continue
			}
			return fmt.Errorf("custos push failed after retry: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Success
			// Fixed: Decode only the stable receipt_id field instead of depending on Custos internals.
			var rec pushReceipt
			if err := json.NewDecoder(resp.Body).Decode(&rec); err != nil {
				return fmt.Errorf("failed to parse Custos receipt: %w", err)
			}
			fmt.Printf("Custos push successful, receipt_id: %s\n", rec.ReceiptID)
			return nil
		}

		// Handle non-2xx status codes
		bodyBytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			// 4xx errors are typically client-side validation errors, do not retry
			return fmt.Errorf("custos push failed with client error %d: %s", resp.StatusCode, string(bodyBytes))
		}

		// Other server errors (5xx) might be transient, retry if it's the first attempt
		if i == 0 {
			fmt.Fprintf(os.Stderr, "Warning: Custos push failed with server error %d (attempt %d), retrying in %v: %s\n", resp.StatusCode, i+1, defaultCustosPushRetryDelay, string(bodyBytes))
			time.Sleep(defaultCustosPushRetryDelay)
			continue
		}
		return fmt.Errorf("custos push failed with server error %d after retry: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil // Should not be reached
}
