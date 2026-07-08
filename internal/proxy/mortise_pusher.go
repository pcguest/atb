package proxy

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pcguest/atb/internal/mortise"
)

// MortisePusherInterface pushes a completed bundle to Mortise. Mortise ingests
// whole bundles, not individual events.
type MortisePusherInterface interface {
	PushBundle(ctx context.Context, bundleBytes []byte) (*mortise.Receipt, error)
}

// MortisePusher implements MortisePusherInterface using the internal/mortise HTTP client.
type MortisePusher struct {
	client *mortise.HTTPClient
	logger *slog.Logger
}

// NewMortisePusher creates a new MortisePusher.
func NewMortisePusher(endpoint, token string, logger *slog.Logger) (*MortisePusher, error) {
	client, err := mortise.NewHTTPClient(endpoint, token)
	if err != nil {
		return nil, err
	}
	return &MortisePusher{
		client: client,
		logger: logger,
	}, nil
}

// PushBundle sends a completed bundle to the configured Mortise endpoint and
// returns the signed custody receipt.
func (cp *MortisePusher) PushBundle(ctx context.Context, bundleBytes []byte) (*mortise.Receipt, error) {
	if cp.client == nil {
		return nil, fmt.Errorf("MortisePusher not initialized with a client")
	}
	if cp.logger != nil {
		cp.logger.Debug("pushing bundle to Mortise", "endpoint", cp.client.Endpoint())
	}
	return cp.client.SendBundle(ctx, bundleBytes)
}
