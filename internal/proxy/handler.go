// SPDX-License-Identifier: MIT
package proxy

import (
	"context"
	"log/slog"
)

// Handler processes captured proxy records. Item 3 adds real forwarding; the
// scaffold handler logs records without forwarding upstream traffic.
type Handler interface {
	HandleRequest(ctx context.Context, rec RequestRecord) error
	HandleResponse(ctx context.Context, rec ResponseRecord) error
}

var _ Handler = (*StubHandler)(nil)

// StubHandler logs captured records and does not forward requests yet.
type StubHandler struct {
	Logger *slog.Logger
}

// HandleRequest logs the captured request metadata.
func (h StubHandler) HandleRequest(ctx context.Context, rec RequestRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	logger := h.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Info("proxy capture request",
		"session_id", rec.SessionID,
		"host", rec.Host,
		"method", rec.Method,
		"path", rec.Path,
		"provider", rec.Provider,
		"model", rec.Model,
		"actor", rec.DisplayName,
	)
	return nil
}

// HandleResponse logs the captured response metadata.
func (h StubHandler) HandleResponse(ctx context.Context, rec ResponseRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	logger := h.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Info("proxy capture response",
		"session_id", rec.SessionID,
		"host", rec.Host,
		"status_code", rec.StatusCode,
		"provider", rec.Provider,
		"model", rec.Model,
		"actor", rec.DisplayName,
	)
	return nil
}
