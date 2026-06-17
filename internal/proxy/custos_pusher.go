package proxy

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pcguest/atb/internal/custos"
)

// CustosPusherInterface pushes a completed bundle to Custos. Custos ingests
// whole bundles, not individual events.
type CustosPusherInterface interface {
	PushBundle(ctx context.Context, bundleBytes []byte) (*custos.Receipt, error)
}

// CustosPusher implements CustosPusherInterface using the internal/custos HTTP client.
type CustosPusher struct {
	client *custos.HTTPClient
	logger *slog.Logger
}

// NewCustosPusher creates a new CustosPusher.
func NewCustosPusher(endpoint, token string, logger *slog.Logger) (*CustosPusher, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("custos endpoint cannot be empty")
	}
	return &CustosPusher{
		client: custos.NewHTTPClient(endpoint, token),
		logger: logger,
	}, nil
}

// PushBundle sends a completed bundle to the configured Custos endpoint and
// returns the signed custody receipt.
func (cp *CustosPusher) PushBundle(ctx context.Context, bundleBytes []byte) (*custos.Receipt, error) {
	if cp.client == nil {
		return nil, fmt.Errorf("CustosPusher not initialized with a client")
	}
	if cp.logger != nil {
		cp.logger.Debug("pushing bundle to Custos", "endpoint", cp.client.Endpoint())
	}
	return cp.client.SendBundle(ctx, bundleBytes)
}
