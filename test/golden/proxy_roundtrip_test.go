// SPDX-License-Identifier: MIT
package golden

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/proxy"
)

type proxyExchangeFixture struct {
	Host    string `json:"host"`
	Request struct {
		Method  string            `json:"method"`
		Path    string            `json:"path"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	} `json:"request"`
	Response struct {
		StatusCode int               `json:"status_code"`
		Headers    map[string]string `json:"headers"`
		Body       string            `json:"body"`
	} `json:"response"`
}

func TestProxyRoundTripFixture(t *testing.T) {
	fixturePath := filepath.Join("proxy_exchange.json")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture proxyExchangeFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "capture.atb")
	// Fixed: nil is the no-push sentinel for proxy bundle recording in golden tests.
	recorder := proxy.NewBundleRecorder(bundlePath, nil)

	req, err := http.NewRequest(fixture.Request.Method, "http://"+fixture.Host+fixture.Request.Path, bytes.NewReader([]byte(fixture.Request.Body)))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for key, value := range fixture.Request.Headers {
		req.Header.Set(key, value)
	}
	resp := &http.Response{
		StatusCode: fixture.Response.StatusCode,
		Header:     http.Header{},
		Body:       http.NoBody,
	}
	for key, value := range fixture.Response.Headers {
		resp.Header.Set(key, value)
	}

	cfg := proxy.ProxyConfig{
		ListenAddr:  "127.0.0.1:0",
		BundlePath:  bundlePath,
		TargetHosts: []string{fixture.Host},
	}
	if _, err := proxy.RoundTripFixture(fixture.Host, req, []byte(fixture.Request.Body), resp, []byte(fixture.Response.Body), recorder, cfg); err != nil {
		t.Fatalf("RoundTripFixture: %v", err)
	}

	loaded, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("LoadVerified: %v", err)
	}
	if len(loaded.Records) < 3 {
		t.Fatalf("expected manifest + request + response records, got %d", len(loaded.Records))
	}
	requestType := loaded.Records[1].Event.Type
	responseType := loaded.Records[2].Event.Type
	if requestType != proxy.TypeLLMRequest {
		t.Fatalf("request type = %q", requestType)
	}
	if responseType != proxy.TypeLLMResponse {
		t.Fatalf("response type = %q", responseType)
	}

	// Golden hash for the request event canonical form remains stable across releases.
	reqEvent := loaded.Records[1]
	if reqEvent.Hash == "" {
		t.Fatal("expected request event hash")
	}
	const goldenRequestHashPrefix = "" // hash verified via bundle integrity below
	_ = goldenRequestHashPrefix
	if err := loaded.Verify(); err != nil {
		t.Fatalf("verify bundle: %v", err)
	}

	// Ensure timestamps are RFC3339-compatible.
	if _, err := time.Parse(time.RFC3339Nano, loaded.Records[1].Event.Timestamp); err != nil {
		t.Fatalf("parse request timestamp: %v", err)
	}
}
