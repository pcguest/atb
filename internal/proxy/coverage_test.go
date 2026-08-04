// SPDX-License-Identifier: MIT
package proxy

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/mortise"
)

func TestLocalCALifecycleAndInstructions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ca, err := LoadOrCreateLocalCA()
	if err != nil {
		t.Fatalf("create CA: %v", err)
	}
	if !ca.Created() {
		t.Fatal("new CA was not marked created")
	}
	if _, err := os.Stat(ca.CertPath); err != nil {
		t.Fatalf("stat CA certificate: %v", err)
	}
	if _, err := os.Stat(ca.KeyPath); err != nil {
		t.Fatalf("stat CA key: %v", err)
	}
	certPEM, keyPEM, err := ca.LeafCertificate("127.0.0.1:443")
	if err != nil || len(certPEM) == 0 || len(keyPEM) == 0 {
		t.Fatalf("leaf certificate = %d/%d bytes, %v", len(certPEM), len(keyPEM), err)
	}

	loaded, err := LoadOrCreateLocalCA()
	if err != nil {
		t.Fatalf("load CA: %v", err)
	}
	if loaded.Created() {
		t.Fatal("loaded CA was marked newly created")
	}
	if (*LocalCA)(nil).Created() {
		t.Fatal("nil CA was marked created")
	}
	if _, _, err := (*LocalCA)(nil).LeafCertificate("example.com"); err == nil {
		t.Fatal("nil CA generated a leaf certificate")
	}
	if _, _, err := (&LocalCA{}).LeafCertificate("example.com"); err == nil {
		t.Fatal("unloaded CA generated a leaf certificate")
	}

	var instructions bytes.Buffer
	PrintInstallInstructions(&instructions, ca.CertPath)
	for _, want := range []string{"SSL_CERT_FILE", "CURL_CA_BUNDLE", "NODE_EXTRA_CA_CERTS", "Do not install"} {
		if !strings.Contains(instructions.String(), want) {
			t.Fatalf("instructions missing %q: %s", want, instructions.String())
		}
	}
}

func TestLocalCALoadRejectsPermissiveKeyMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows os.FileMode does not expose key-file ACL permissions")
	}
	t.Setenv("HOME", t.TempDir())
	ca, err := LoadOrCreateLocalCA()
	if err != nil {
		t.Fatalf("create CA: %v", err)
	}
	if err := os.Chmod(ca.KeyPath, 0o644); err != nil {
		t.Fatalf("chmod CA key: %v", err)
	}
	if _, err := LoadOrCreateLocalCA(); err == nil || !strings.Contains(err.Error(), "require 0600") {
		t.Fatalf("permissive CA key mode error = %v", err)
	}
}

func TestLocalCALoadRejectsMalformedMaterial(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key")
	if err := os.WriteFile(certPath, []byte("not pem"), 0o600); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("not pem"), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	if err := (&LocalCA{CertPath: certPath, KeyPath: keyPath}).load(); err == nil {
		t.Fatal("malformed certificate unexpectedly loaded")
	}
	if err := os.WriteFile(certPath, []byte("-----BEGIN CERTIFICATE-----\nAA==\n-----END CERTIFICATE-----\n"), 0o600); err != nil {
		t.Fatalf("rewrite cert: %v", err)
	}
	if err := (&LocalCA{CertPath: certPath, KeyPath: keyPath}).load(); err == nil {
		t.Fatal("invalid DER certificate unexpectedly loaded")
	}
}

type errorReadCloser struct {
	err error
}

func (r errorReadCloser) Read([]byte) (int, error) { return 0, r.err }
func (r errorReadCloser) Close() error             { return nil }

func TestHTTPBodyThreadAndHeaderHelpers(t *testing.T) {
	if body, err := ReadRequestBody(nil); err != nil || body != nil {
		t.Fatalf("nil request body = %q, %v", body, err)
	}
	req := httptest.NewRequest(http.MethodPost, "https://api.openai.com/v1", strings.NewReader(`{"thread_id":"thread-1","model":"gpt-4"}`))
	req.RemoteAddr = "127.0.0.1:1234"
	body, err := ReadRequestBody(req)
	if err != nil {
		t.Fatalf("ReadRequestBody: %v", err)
	}
	replayed, err := io.ReadAll(req.Body)
	if err != nil || !bytes.Equal(body, replayed) {
		t.Fatalf("restored body = %q, %v", replayed, err)
	}
	if got := ExtractThreadKey("api.openai.com", req, body); got != "openai-thread:thread-1" {
		t.Fatalf("body thread key = %q", got)
	}
	req.Header.Set("X-Session-Id", "session-1")
	if got := ExtractThreadKey("api.openai.com", req, body); got != "session:session-1" {
		t.Fatalf("session header key = %q", got)
	}
	req.Header.Set("Anthropic-Session-Id", "anthropic-1")
	if got := ExtractThreadKey("api.anthropic.com", req, body); got != "anthropic:anthropic-1" {
		t.Fatalf("Anthropic thread key = %q", got)
	}
	if got := ExtractThreadKey("", nil, nil); got != "" {
		t.Fatalf("nil thread key = %q", got)
	}

	pathReq := httptest.NewRequest(http.MethodPost, "https://api.openai.com/v1/threads/path-thread/messages", nil)
	if got := ExtractThreadKey("api.openai.com", pathReq, nil); got != "openai-thread:path-thread" {
		t.Fatalf("path thread key = %q", got)
	}
	connReq := httptest.NewRequest(http.MethodPost, "https://example.com/v1", nil)
	connReq.RemoteAddr = "10.0.0.1:42"
	if got := ExtractThreadKey("Example.COM:443", connReq, []byte("invalid")); got != "conn:example.com:10.0.0.1:42" {
		t.Fatalf("connection thread key = %q", got)
	}

	badReq := httptest.NewRequest(http.MethodPost, "https://example.com", nil)
	badReq.Body = errorReadCloser{err: errors.New("read failed")}
	if _, err := ReadRequestBody(badReq); err == nil {
		t.Fatal("request body read error was ignored")
	}

	oversize := httptest.NewRequest(http.MethodPost, "https://example.com", strings.NewReader(strings.Repeat("x", 64)))
	if _, err := ReadRequestBodyLimited(oversize, 16); !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("oversize body error = %v, want ErrBodyTooLarge", err)
	} else {
		var limitErr *BodyTooLargeError
		if !errors.As(err, &limitErr) || limitErr.Observed != 64 {
			t.Fatalf("oversize body detail = %#v, want observed 64", limitErr)
		}
	}
	cfg := ProxyConfig{}
	if cfg.EffectiveMaxBodyBytes() != DefaultMaxBodyBytes {
		t.Fatalf("default max body = %d, want %d", cfg.EffectiveMaxBodyBytes(), DefaultMaxBodyBytes)
	}
	cfg.MaxBodyBytes = 1024
	if cfg.EffectiveMaxBodyBytes() != 1024 {
		t.Fatalf("custom max body = %d, want 1024", cfg.EffectiveMaxBodyBytes())
	}
	if err := (&ProxyConfig{ListenAddr: "127.0.0.1:1", BundlePath: "b.atb", TargetHosts: []string{"api.openai.com"}, MaxBodyBytes: -1}).Validate(); err == nil {
		t.Fatal("negative MaxBodyBytes should fail validation")
	}
	if err := (&ProxyConfig{ListenAddr: "127.0.0.1:1", BundlePath: "b.atb", TargetHosts: []string{"api.openai.com"}, MaxBodyBytes: MaxBodyBytesLimit + 1}).Validate(); err == nil {
		t.Fatal("MaxBodyBytes above the hard limit should fail validation")
	}

	if err := DrainAndClose(context.Background(), nil, nil); err != nil {
		t.Fatalf("DrainAndClose nil body: %v", err)
	}
	called := false
	if err := DrainAndClose(context.Background(), io.NopCloser(strings.NewReader("body")), func() error {
		called = true
		return nil
	}); err != nil || !called {
		t.Fatalf("DrainAndClose callback called=%v err=%v", called, err)
	}
	drainErr := errors.New("drain failed")
	if err := DrainAndClose(context.Background(), errorReadCloser{err: drainErr}, nil); !errors.Is(err, drainErr) {
		t.Fatalf("drain error = %v", err)
	}

	headers := http.Header{"Authorization": {"Bearer token"}}
	if got := ExtractAPIKey(headers); got != "token" {
		t.Fatalf("bearer key = %q", got)
	}
	headers.Set("X-Api-Key", "provider-key")
	if got := ExtractAPIKey(headers); got != "provider-key" {
		t.Fatalf("provider key = %q", got)
	}
	if got := ExtractAPIKey(nil); got != "" {
		t.Fatalf("nil header key = %q", got)
	}
	if !IsTargetHost("sub.API.OPENAI.COM:443", []string{"api.openai.com"}) {
		t.Fatal("subdomain target was not matched")
	}
	if IsTargetHost("attacker.example", []string{"api.openai.com"}) {
		t.Fatal("unlisted host was matched")
	}
}

func TestReadBodyLimitedBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		limit   int64
		wantErr bool
	}{
		{name: "below", body: "123", limit: 4},
		{name: "exact", body: "1234", limit: 4},
		{name: "above", body: "12345", limit: 4, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadBodyLimited(strings.NewReader(tt.body), tt.limit)
			if tt.wantErr {
				if !errors.Is(err, ErrBodyTooLarge) {
					t.Fatalf("error = %v, want ErrBodyTooLarge", err)
				}
				return
			}
			if err != nil || string(got) != tt.body {
				t.Fatalf("body = %q error = %v", got, err)
			}
		})
	}
}

func TestHTTPOversizeRequestRecordsRejection(t *testing.T) {
	bundlePath := filepath.Join(t.TempDir(), "bundle.atb")
	p, err := NewProxy(ProxyConfig{
		ListenAddr:   "127.0.0.1:0",
		BundlePath:   bundlePath,
		TargetHosts:  []string{"api.openai.com"},
		MaxBodyBytes: 4,
	}, nil, nil)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://api.openai.com/v1/messages", strings.NewReader("12345"))
	rr := httptest.NewRecorder()
	(&forwarder{proxy: p}).ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusRequestEntityTooLarge)
	}
	b, err := loadBundleForAppend(bundlePath)
	if err != nil {
		t.Fatalf("load rejection bundle: %v", err)
	}
	found := false
	for _, rec := range b.Records {
		if rec.Event.Type == TypeCaptureRejected {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("capture rejection event was not recorded")
	}
}

func TestHTTPRequestUnknownLengthBoundaries(t *testing.T) {
	for _, tt := range []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "below", body: "123", wantStatus: http.StatusOK},
		{name: "exact", body: "1234", wantStatus: http.StatusOK},
		{name: "above", body: "12345", wantStatus: http.StatusRequestEntityTooLarge},
	} {
		t.Run(tt.name, func(t *testing.T) {
			upstream := newLoopbackHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read upstream request: %v", err)
					return
				}
				_, _ = w.Write(body)
			}))
			t.Cleanup(upstream.Close)
			bundlePath := filepath.Join(t.TempDir(), "bundle.atb")
			p, err := NewProxy(ProxyConfig{
				ListenAddr:   "127.0.0.1:0",
				BundlePath:   bundlePath,
				TargetHosts:  []string{"127.0.0.1"},
				MaxBodyBytes: 4,
			}, nil, nil)
			if err != nil {
				t.Fatalf("NewProxy: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, upstream.URL+"/chunked", strings.NewReader(tt.body))
			req.ContentLength = -1
			req.TransferEncoding = []string{"chunked"}
			rr := httptest.NewRecorder()
			(&forwarder{proxy: p}).ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if tt.wantStatus == http.StatusOK && rr.Body.String() != tt.body {
				t.Fatalf("forwarded body = %q, want %q", rr.Body.String(), tt.body)
			}
			if tt.wantStatus == http.StatusRequestEntityTooLarge {
				assertCaptureRejection(t, bundlePath, "request", "body_too_large")
			}
		})
	}
}

func TestHTTPResponseBodyBoundariesAndEncoding(t *testing.T) {
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	if _, err := zw.Write([]byte(strings.Repeat("a", 1024))); err != nil {
		t.Fatalf("compress response: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close gzip response: %v", err)
	}

	for _, tt := range []struct {
		name         string
		body         []byte
		limit        int64
		chunked      bool
		encoding     string
		wantStatus   int
		wantRejected bool
	}{
		{name: "below", body: []byte("123"), limit: 4, wantStatus: http.StatusOK},
		{name: "exact", body: []byte("1234"), limit: 4, wantStatus: http.StatusOK},
		{name: "above", body: []byte("12345"), limit: 4, wantStatus: http.StatusBadGateway, wantRejected: true},
		{name: "chunked exact", body: []byte("1234"), limit: 4, chunked: true, wantStatus: http.StatusOK},
		{
			name:       "compressed representation",
			body:       compressed.Bytes(),
			limit:      int64(compressed.Len()),
			encoding:   "gzip",
			wantStatus: http.StatusOK,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			upstream := newLoopbackHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tt.encoding != "" {
					w.Header().Set("Content-Encoding", tt.encoding)
				}
				if tt.chunked {
					w.WriteHeader(http.StatusOK)
					if flusher, ok := w.(http.Flusher); ok {
						flusher.Flush()
					}
				}
				_, _ = w.Write(tt.body)
			}))
			t.Cleanup(upstream.Close)
			bundlePath := filepath.Join(t.TempDir(), "bundle.atb")
			p, err := NewProxy(ProxyConfig{
				ListenAddr:   "127.0.0.1:0",
				BundlePath:   bundlePath,
				TargetHosts:  []string{"127.0.0.1"},
				MaxBodyBytes: tt.limit,
			}, nil, nil)
			if err != nil {
				t.Fatalf("NewProxy: %v", err)
			}
			req := httptest.NewRequest(http.MethodGet, upstream.URL+"/response", nil)
			if tt.encoding != "" {
				// An explicit Accept-Encoding makes the transport preserve the
				// compressed representation. The proxy bounds the bytes it stores
				// and forwards; it never expands compressed content in memory.
				req.Header.Set("Accept-Encoding", tt.encoding)
			}
			rr := httptest.NewRecorder()
			(&forwarder{proxy: p}).ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if !tt.wantRejected && !bytes.Equal(rr.Body.Bytes(), tt.body) {
				t.Fatalf("forwarded response length = %d, want %d", rr.Body.Len(), len(tt.body))
			}
			if tt.wantRejected {
				assertCaptureRejection(t, bundlePath, "response", "body_too_large")
			}
		})
	}
}

func TestBodyBudgetRejectsConcurrentMaximumReservations(t *testing.T) {
	var startedOnce sync.Once
	started := make(chan struct{})
	release := make(chan struct{})
	upstream := newLoopbackHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		startedOnce.Do(func() { close(started) })
		<-release
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(upstream.Close)
	bundlePath := filepath.Join(t.TempDir(), "bundle.atb")
	p, err := NewProxy(ProxyConfig{
		ListenAddr:   "127.0.0.1:0",
		BundlePath:   bundlePath,
		TargetHosts:  []string{"127.0.0.1"},
		MaxBodyBytes: MaxBodyBytesLimit,
	}, nil, nil)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	f := &forwarder{proxy: p}
	firstDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		rr := httptest.NewRecorder()
		f.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, upstream.URL+"/first", strings.NewReader("one")))
		firstDone <- rr
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("first request did not reach upstream")
	}

	second := httptest.NewRecorder()
	f.ServeHTTP(second, httptest.NewRequest(http.MethodPost, upstream.URL+"/second", strings.NewReader("two")))
	if second.Code != http.StatusServiceUnavailable {
		t.Fatalf("concurrent status = %d, want %d", second.Code, http.StatusServiceUnavailable)
	}
	assertCaptureRejection(t, bundlePath, "request", "memory_budget_exhausted")
	close(release)
	select {
	case first := <-firstDone:
		if first.Code != http.StatusOK {
			t.Fatalf("first status = %d, want %d", first.Code, http.StatusOK)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("first request did not finish")
	}
}

func assertCaptureRejection(t *testing.T, bundlePath, direction, reason string) {
	t.Helper()
	b, err := loadBundleForAppend(bundlePath)
	if err != nil {
		t.Fatalf("load rejection bundle: %v", err)
	}
	for _, rec := range b.Records {
		if rec.Event.Type != TypeCaptureRejected {
			continue
		}
		data, ok := rec.Event.Data.(map[string]any)
		if !ok {
			continue
		}
		if data["direction"] == direction && data["reason"] == reason {
			if _, present := data["body"]; present {
				t.Fatal("capture rejection evidence contains a body")
			}
			if _, present := data["headers"]; present {
				t.Fatal("capture rejection evidence contains headers")
			}
			return
		}
	}
	t.Fatalf("no %s %s capture rejection in bundle", direction, reason)
}

func newLoopbackHTTPServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("loopback listener unavailable: %v", err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.Start()
	return server
}

func TestHTTPRequestAndForwarderBoundaryHelpers(t *testing.T) {
	raw := "GET /v1 HTTP/1.1\r\nHost: api.openai.com\r\n\r\n"
	req, err := ReadHTTPRequest(bufio.NewReader(strings.NewReader(raw)))
	if err != nil || req.Host != "api.openai.com" {
		t.Fatalf("ReadHTTPRequest = %+v, %v", req, err)
	}
	if got := urlForHost("api.openai.com"); got.Scheme != "http" || got.Host != "api.openai.com" {
		t.Fatalf("urlForHost = %s", got)
	}

	p := &Proxy{cfg: ProxyConfig{TargetHosts: []string{"allowed.example"}}}
	f := &forwarder{proxy: p}
	for _, method := range []string{http.MethodConnect, http.MethodGet} {
		req := httptest.NewRequest(method, "http://blocked.example/path", nil)
		req.Host = "blocked.example"
		rr := httptest.NewRecorder()
		f.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("%s forbidden status = %d", method, rr.Code)
		}
	}

	allowed := httptest.NewRequest(http.MethodConnect, "http://allowed.example", nil)
	allowed.Host = "allowed.example"
	rr := httptest.NewRecorder()
	f.ServeHTTP(rr, allowed)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("CONNECT without hijacker status = %d", rr.Code)
	}
}

func TestProxyLifecycleAndNilBoundaries(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := ProxyConfig{
		ListenAddr:  "127.0.0.1:0",
		BundlePath:  filepath.Join(t.TempDir(), "bundle.atb"),
		TargetHosts: []string{"api.openai.com"},
	}
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	p, err := NewProxy(cfg, nil, logger)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	if p.Handler() == nil || p.Config().BundlePath != cfg.BundlePath {
		t.Fatalf("proxy accessors returned invalid values")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := p.Start(ctx); err == nil {
		t.Fatal("second Start unexpectedly succeeded")
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
	if err := (*Proxy)(nil).Start(ctx); err == nil {
		t.Fatal("nil Start unexpectedly succeeded")
	}
	if err := (*Proxy)(nil).Stop(); err == nil {
		t.Fatal("nil Stop unexpectedly succeeded")
	}
	if (*Proxy)(nil).Config().ListenAddr != "" {
		t.Fatal("nil Config was not empty")
	}
	if (*Proxy)(nil).Handler() == nil {
		t.Fatal("nil Handler returned nil")
	}
	if err := (*Proxy)(nil).HandleRequest(ctx, RequestRecord{}); err == nil {
		t.Fatal("nil HandleRequest unexpectedly succeeded")
	}
	if err := (*Proxy)(nil).HandleResponse(ctx, ResponseRecord{}, ""); err == nil {
		t.Fatal("nil HandleResponse unexpectedly succeeded")
	}

	writer := &logWriter{logger: logger}
	if n, err := writer.Write([]byte("message\n")); err != nil || n != len("message\n") {
		t.Fatalf("log writer = %d, %v", n, err)
	}
	p.waitForListener(time.Millisecond)
}

func TestMortisePusherValidation(t *testing.T) {
	if _, err := NewMortisePusher("", "", nil); err == nil {
		t.Fatal("empty endpoint unexpectedly accepted")
	}
	if pusher, err := NewMortisePusher("https://custody.example", "token", nil); err != nil || pusher.client == nil {
		t.Fatalf("valid pusher = %+v, %v", pusher, err)
	}
	if _, err := (&MortisePusher{}).PushBundle(context.Background(), []byte("bundle")); err == nil {
		t.Fatal("uninitialized pusher unexpectedly succeeded")
	}
}

type captureHandler struct {
	request  RequestRecord
	response ResponseRecord
}

func (h *captureHandler) HandleRequest(_ context.Context, rec RequestRecord) error {
	h.request = rec
	return nil
}

func (h *captureHandler) HandleResponse(_ context.Context, rec ResponseRecord) error {
	h.response = rec
	return nil
}

func TestProxyDelegationAndIdleSessionClose(t *testing.T) {
	handler := &captureHandler{}
	p, err := NewProxy(ProxyConfig{
		ListenAddr:  "127.0.0.1:0",
		BundlePath:  filepath.Join(t.TempDir(), "bundle.atb"),
		TargetHosts: []string{"api.openai.com"},
		IdentityMap: map[string]string{"key": "Operator"},
	}, handler, nil)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	if err := p.HandleRequest(context.Background(), RequestRecord{APIKey: "key"}); err != nil {
		t.Fatalf("HandleRequest: %v", err)
	}
	if err := p.HandleResponse(context.Background(), ResponseRecord{}, "key"); err != nil {
		t.Fatalf("HandleResponse: %v", err)
	}
	if handler.request.DisplayName != "Operator" || handler.response.DisplayName != "Operator" {
		t.Fatalf("delegated identities request=%+v response=%+v", handler.request, handler.response)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := (StubHandler{}).HandleRequest(ctx, RequestRecord{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled request error = %v", err)
	}
	if err := (StubHandler{}).HandleResponse(ctx, ResponseRecord{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled response error = %v", err)
	}

	closed := make(chan *Session, 1)
	manager := NewSessionManager(func(sess *Session) error {
		closed <- sess
		return nil
	})
	manager.idleTimeout = 5 * time.Millisecond
	session := manager.Resolve("idle-thread")
	select {
	case got := <-closed:
		if got != session {
			t.Fatalf("idle closed session = %p, want %p", got, session)
		}
	case <-time.After(time.Second):
		t.Fatal("idle session did not close")
	}
}

func TestBundleRecorderAndSessionLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")
	recorder := NewBundleRecorder(path, nil)
	if _, err := (*BundleRecorder)(nil).AppendEventHash(nil); err == nil {
		t.Fatal("nil recorder/event unexpectedly appended")
	}
	if err := recorder.AppendSessionClose(nil); err != nil {
		t.Fatalf("nil session close: %v", err)
	}
	sess := &Session{
		ID:            "session-1",
		ThreadKey:     "thread-1",
		Model:         "model-1",
		ExchangeCount: 2,
		TotalTokens:   5,
		ActorID:       "actor-1",
	}
	if err := recorder.AppendSessionClose(sess); err != nil {
		t.Fatalf("AppendSessionClose: %v", err)
	}
	if err := recorder.sessionCloseCallback(nil); err != nil {
		t.Fatalf("nil session callback: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("session-close bundle: %v", err)
	}

	closed := make(chan *Session, 2)
	manager := NewSessionManager(func(sess *Session) error {
		closed <- sess
		return nil
	})
	resolved := manager.Resolve("thread-2")
	if again := manager.Resolve("thread-2"); again != resolved {
		t.Fatal("Resolve did not reuse the existing thread session")
	}
	resolved.UpdateActorID("")
	resolved.UpdateActorID("actor-2")
	if got := resolved.NextExchangeID(); got != "0001" {
		t.Fatalf("exchange id = %q", got)
	}
	started := time.Now().UTC().Add(-time.Second)
	resolved.noteExchangeStarted(started)
	resolved.setLastRequestEventHash("request-hash")
	manager.NoteExchange(resolved, "model-2", 3, 4)
	record := ExchangeCompleteRecord(resolved, "0001", resolved.lastRequestEventHashLocked(), "model-2", 3, 4, 1, time.Now().UTC())
	if record["actor_id"] != "actor-2" || record["request_event_id"] != "request-hash" {
		t.Fatalf("exchange record = %+v", record)
	}
	manager.CloseAll()
	manager.CloseAll()
	select {
	case got := <-closed:
		if got != resolved {
			t.Fatalf("closed session = %p, want %p", got, resolved)
		}
	case <-time.After(time.Second):
		t.Fatal("session close callback not called")
	}
	if resolved.markClosed() {
		t.Fatal("already closed session closed twice")
	}
	manager.NoteExchange(resolved, "", 0, 0)
	if got := (*SessionManager)(nil).Resolve(""); got == nil || got.ID == "" {
		t.Fatalf("nil manager Resolve = %+v", got)
	}
	(*SessionManager)(nil).NoteExchange(nil, "", 0, 0)
	(*SessionManager)(nil).CloseAll()
	if got := SessionCloseRecord(nil); len(got) != 0 {
		t.Fatalf("nil close record = %+v", got)
	}
	if got := ExchangeCompleteRecord(nil, "", "", "", 0, 0, 0, time.Time{}); len(got) != 0 {
		t.Fatalf("nil exchange record = %+v", got)
	}
}

type recordingMortisePusher struct {
	bundles chan []byte
	err     error
}

func (p recordingMortisePusher) PushBundle(_ context.Context, bundleBytes []byte) (*mortise.Receipt, error) {
	p.bundles <- append([]byte(nil), bundleBytes...)
	if p.err != nil {
		return nil, p.err
	}
	return &mortise.Receipt{ReceiptID: "receipt-1", BundleHash: "hash-1"}, nil
}

func TestBundleRecorderPushesImmutableSessionSnapshot(t *testing.T) {
	bundles := make(chan []byte, 1)
	path := filepath.Join(t.TempDir(), "bundle.atb")
	recorder := NewBundleRecorder(path, recordingMortisePusher{bundles: bundles})
	session := &Session{ID: "session-push", ThreadKey: "thread-push", ActorID: "actor"}
	if err := recorder.sessionCloseCallback(session); err != nil {
		t.Fatalf("sessionCloseCallback: %v", err)
	}
	select {
	case snapshot := <-bundles:
		if len(snapshot) == 0 {
			t.Fatal("pushed snapshot is empty")
		}
		onDisk, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read on-disk bundle: %v", err)
		}
		if !bytes.Equal(snapshot, onDisk) {
			t.Fatal("pushed bytes differ from the locked on-disk session-close snapshot")
		}
	case <-time.After(time.Second):
		t.Fatal("Mortise push was not invoked")
	}

	failedBundles := make(chan []byte, 1)
	failedRecorder := NewBundleRecorder(
		filepath.Join(t.TempDir(), "failed.atb"),
		recordingMortisePusher{bundles: failedBundles, err: errors.New("push failed")},
	)
	if err := failedRecorder.sessionCloseCallback(&Session{ID: "session-failed"}); err != nil {
		t.Fatalf("failed-push callback append: %v", err)
	}
	select {
	case <-failedBundles:
	case <-time.After(time.Second):
		t.Fatal("failing Mortise pusher was not invoked")
	}
}
