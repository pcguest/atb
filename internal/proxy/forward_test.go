// SPDX-License-Identifier: MIT
package proxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/proxy"
)

func TestLocalCARoundTrip(t *testing.T) {
	t.Parallel()

	ca, err := proxy.LoadOrCreateLocalCA()
	if err != nil {
		t.Fatalf("LoadOrCreateLocalCA: %v", err)
	}
	certPEM, keyPEM, err := ca.LeafCertificate("api.openai.com")
	if err != nil {
		t.Fatalf("LeafCertificate: %v", err)
	}
	if len(certPEM) == 0 || len(keyPEM) == 0 {
		t.Fatal("expected leaf cert material")
	}
}

func TestBundleRecorderAppend(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "bundle.atb")
	// Fixed: nil preserves no-push behaviour in recorder append tests.
	rec := proxy.NewBundleRecorder(path, nil)
	ev, err := proxy.RequestRecord{
		SessionID: "sess-1",
		Host:      "api.openai.com",
		Method:    "POST",
		Path:      "/v1/chat/completions",
	}.ToEvent()
	if err != nil {
		t.Fatalf("ToEvent: %v", err)
	}
	if err := rec.AppendEvent(ev); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}
}

func TestHTTPForwardCapture(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"model":"gpt-4.1-mini","usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`)
	}))
	defer upstream.Close()

	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	cfg := proxy.ProxyConfig{
		ListenAddr:  "127.0.0.1:0",
		BundlePath:  bundlePath,
		TargetHosts: []string{"127.0.0.1"},
	}
	// Fixed: nil preserves no-push behaviour in forwarding capture tests.
	recorder := proxy.NewBundleRecorder(bundlePath, nil)
	reqBody := `{"model":"gpt-4.1-mini"}`
	req, err := http.NewRequest(http.MethodPost, upstream.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upstream request: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	host := strings.TrimPrefix(strings.TrimPrefix(upstream.URL, "http://"), "https://")
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	if _, err := proxy.RoundTripFixture(host, req, []byte(reqBody), resp, body, recorder, cfg); err != nil {
		t.Fatalf("RoundTripFixture: %v", err)
	}
	loaded, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("LoadVerified: %v", err)
	}
	if len(loaded.Records) < 3 {
		t.Fatalf("records = %d", len(loaded.Records))
	}
}
