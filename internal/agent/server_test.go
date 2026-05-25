// SPDX-License-Identifier: MIT
package agent

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthzHandler(t *testing.T) {
	srv := mustTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	var body HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("status field = %q, want ok", body.Status)
	}
}

func TestInfoHandler(t *testing.T) {
	srv := mustTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/info", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body InfoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Version != "1.11.0-test" {
		t.Fatalf("version = %q, want %q", body.Version, "1.11.0-test")
	}
	if body.Config.ListenAddr != defaultListenAddr {
		t.Fatalf("config.listen_addr = %q, want %q", body.Config.ListenAddr, defaultListenAddr)
	}
	if body.Config.DataDir == "" {
		t.Fatal("expected non-empty config.data_dir")
	}
}

func TestInfoHandlerViaHTTP(t *testing.T) {
	srv := mustTestServer(t)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/v1/info")
	if err != nil {
		t.Fatalf("GET /v1/info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !json.Valid(body) {
		t.Fatalf("response is not valid JSON: %s", body)
	}
}

func mustTestServer(t *testing.T) *Server {
	t.Helper()
	cfg, err := LoadConfigFromEnv("1.11.0-test", func(string) string { return "" })
	if err != nil {
		t.Fatalf("LoadConfigFromEnv: %v", err)
	}
	cfg.DataDir = t.TempDir()
	srv, err := NewServer(cfg, nil, nil)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return srv
}
