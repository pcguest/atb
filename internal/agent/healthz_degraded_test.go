// SPDX-License-Identifier: MIT
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type degradedManager struct{ err error }

func (d degradedManager) OpenSession(context.Context, OpenParams) (SessionID, error) {
	return "", d.err
}

func (d degradedManager) AppendEvent(context.Context, SessionID, PendingEvent) error {
	return d.err
}

func (d degradedManager) CloseSession(context.Context, SessionID, CloseSessionOpts) (BundleMetadata, error) {
	return BundleMetadata{}, d.err
}

func (d degradedManager) Shutdown(context.Context) error { return nil }

func (d degradedManager) Health() error { return d.err }

func TestHealthzReportsDegradedWhenCaptureFailing(t *testing.T) {
	cfg, err := LoadConfigFromEnv("1.11.0-test", func(string) string { return "" })
	if err != nil {
		t.Fatalf("LoadConfigFromEnv: %v", err)
	}
	cfg.DataDir = t.TempDir()

	srv, err := NewServer(cfg, nil, degradedManager{err: errors.New("disk full")})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	var body HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Status != "degraded" || !body.Degraded {
		t.Fatalf("health body = %+v, want degraded", body)
	}
	if body.Error == "" {
		t.Fatalf("degraded health must include the error")
	}
}
