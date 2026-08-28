// SPDX-License-Identifier: MIT
package proxy

import (
	"context"
	"log/slog"
)

// Handler observes request and response records after the proxy's forwarding
// path has captured them.
type Handler interface {
	HandleRequest(ctx context.Context, rec RequestRecord) error
	HandleResponse(ctx context.Context, rec ResponseRecord) error
}

var _ Handler = (*LoggingHandler)(nil)

// LoggingHandler is a logging-only observation hook. Forwarding remains the
// responsibility of Proxy.
type LoggingHandler struct {
	Logger *slog.Logger
}

// StubHandler preserves the pre-v1.15.3 name for source compatibility.
// Deprecated: use LoggingHandler.
type StubHandler = LoggingHandler

// HandleRequest logs the captured request metadata.
func (h LoggingHandler) HandleRequest(ctx context.Context, rec RequestRecord) error {
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
func (h LoggingHandler) HandleResponse(ctx context.Context, rec ResponseRecord) error {
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
