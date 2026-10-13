// SPDX-License-Identifier: MIT
package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const captureLogRedactionSecret = "sk-test-secret-abc123"

// leakingHandler is a custom capture Handler whose errors deliberately embed
// sensitive record content (the API key on the request path, the body on the
// response path). It models a handler that surfaces request data in its error,
// which the capture-failure log must never print verbatim.
type leakingHandler struct {
	failRequest  bool
	failResponse bool
}

func (h *leakingHandler) HandleRequest(_ context.Context, rec RequestRecord) error {
	if !h.failRequest {
		return nil
	}
	return fmt.Errorf("rejected request with key %s", rec.APIKey)
}

func (h *leakingHandler) HandleResponse(_ context.Context, rec ResponseRecord) error {
	if !h.failResponse {
		return nil
	}
	return fmt.Errorf("rejected response with body %s", string(rec.Body))
}

// TestCaptureRequestFailureLogDoesNotLeakHandlerError drives the real
// handleHTTP request path with a custom Handler that returns an error embedding
// the request API key, and asserts the capture-failure warning is still emitted
// (capture-gap honesty) but the key never appears in the log output.
func TestCaptureRequestFailureLogDoesNotLeakHandlerError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer upstream.Close()

	host := strings.TrimPrefix(upstream.URL, "http://")
	var logs bytes.Buffer
	cfg := ProxyConfig{
		ListenAddr:  "127.0.0.1:0",
		BundlePath:  filepath.Join(t.TempDir(), "bundle.atb"),
		TargetHosts: []string{"127.0.0.1"},
	}
	p, err := NewProxy(cfg, &leakingHandler{failRequest: true}, slog.New(slog.NewTextHandler(&logs, nil)))
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://"+host+"/v1/messages", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+captureLogRedactionSecret)
	f := &forwarder{proxy: p}
	rr := httptest.NewRecorder()
	f.handleHTTP(rr, req)

	out := logs.String()
	if !strings.Contains(out, "proxy capture request failed") {
		t.Fatalf("capture-gap warning missing from log:\n%s", out)
	}
	if strings.Contains(out, captureLogRedactionSecret) {
		t.Fatalf("API key leaked into capture-failure log:\n%s", out)
	}
	if strings.Contains(out, "rejected request with key") {
		t.Fatalf("handler error text leaked into capture-failure log:\n%s", out)
	}
}

// TestCaptureResponseFailureBoundsHandlerError verifies the response capture
// path wraps a leaking custom-Handler error so its log representation is a
// fixed stage name, while the error still propagates and the underlying handler
// error remains reachable via errors.Unwrap.
func TestCaptureResponseFailureBoundsHandlerError(t *testing.T) {
	cfg := ProxyConfig{
		ListenAddr:  "127.0.0.1:0",
		BundlePath:  filepath.Join(t.TempDir(), "bundle.atb"),
		TargetHosts: []string{"api.anthropic.com"},
	}
	p, err := NewProxy(cfg, &leakingHandler{failResponse: true}, nil)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, "http://api.anthropic.com/v1/messages", strings.NewReader("req"))
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: http.NoBody}
	err = (&forwarder{proxy: p}).captureResponse("api.anthropic.com", req, resp, []byte(captureLogRedactionSecret), exchangeCorrelation{})
	if err == nil {
		t.Fatal("expected the handler rejection to propagate")
	}

	var he *handlerCaptureError
	if !errors.As(err, &he) {
		t.Fatalf("error = %T (%v), want *handlerCaptureError", err, err)
	}
	if got := err.Error(); got != "proxy: capture response handler failed" {
		t.Fatalf("Error() = %q, want the bounded stage message", got)
	}
	if strings.Contains(err.Error(), captureLogRedactionSecret) {
		t.Fatalf("response body leaked into bounded Error(): %q", err.Error())
	}
	underlying := errors.Unwrap(err)
	if underlying == nil || !strings.Contains(underlying.Error(), captureLogRedactionSecret) {
		t.Fatalf("underlying handler error not preserved: %v", underlying)
	}
}

type typedCaptureFailure struct{ detail string }

func (e *typedCaptureFailure) Error() string { return e.detail }

type propagationHandler struct {
	requestErr, responseErr error
}

func (h propagationHandler) HandleRequest(context.Context, RequestRecord) error { return h.requestErr }
func (h propagationHandler) HandleResponse(context.Context, ResponseRecord) error {
	return h.responseErr
}

func TestCaptureHandlerErrorsPreserveIsAndAs(t *testing.T) {
	for _, tc := range []struct {
		name  string
		stage string
		typed *typedCaptureFailure
	}{
		{name: "request", stage: "request", typed: &typedCaptureFailure{detail: "typed request failure"}},
		{name: "response", stage: "response", typed: &typedCaptureFailure{detail: "typed response failure"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := ProxyConfig{ListenAddr: "127.0.0.1:0", BundlePath: filepath.Join(t.TempDir(), "bundle.atb"), TargetHosts: []string{"api.openai.com"}}
			h := propagationHandler{}
			if tc.stage == "request" {
				h.requestErr = fmt.Errorf("wrapped: %w", tc.typed)
			} else {
				h.responseErr = fmt.Errorf("wrapped: %w", tc.typed)
			}
			p, err := NewProxy(cfg, h, nil)
			if err != nil {
				t.Fatalf("NewProxy: %v", err)
			}
			f := &forwarder{proxy: p}
			var got error
			if tc.stage == "request" {
				req := httptest.NewRequest(http.MethodPost, "https://api.openai.com/v1/messages", strings.NewReader("body"))
				got = func() error { _, e := f.captureRequest("api.openai.com", req, []byte("body")); return e }()
			} else {
				req := httptest.NewRequest(http.MethodPost, "https://api.openai.com/v1/messages", nil)
				got = f.captureResponse("api.openai.com", req, &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: http.NoBody}, []byte("response"), exchangeCorrelation{})
			}
			if got == nil || !errors.Is(got, tc.typed) {
				t.Fatalf("error = %v, errors.Is typed = false", got)
			}
			var typed *typedCaptureFailure
			if !errors.As(got, &typed) || typed != tc.typed {
				t.Fatalf("error = %v, errors.As did not preserve typed error", got)
			}
			if !strings.Contains(got.Error(), "capture "+tc.stage+" handler failed") {
				t.Fatalf("bounded error = %v", got)
			}
		})
	}
}

func TestHTTPResponseErrorHandlerRedactsHandlerError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Upstream-Canary", "header-canary")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("body-canary"))
	}))
	defer upstream.Close()
	var logs bytes.Buffer
	cfg := ProxyConfig{ListenAddr: "127.0.0.1:0", BundlePath: filepath.Join(t.TempDir(), "bundle.atb"), TargetHosts: []string{"127.0.0.1"}}
	p, err := NewProxy(cfg, &leakingHandler{failResponse: true}, slog.New(slog.NewTextHandler(&logs, nil)))
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, upstream.URL+"/v1/messages", strings.NewReader("request-body-canary"))
	req.Header.Set("Authorization", "Bearer "+captureLogRedactionSecret)
	rr := httptest.NewRecorder()
	(&forwarder{proxy: p}).handleHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadGateway)
	}
	out := logs.String()
	if !strings.Contains(out, "proxy upstream response failed") {
		t.Fatalf("response failure warning missing: %s", out)
	}
	for _, want := range []string{"body-canary", "header-canary", captureLogRedactionSecret, "rejected response with body"} {
		if strings.Contains(out, want) {
			t.Fatalf("%q leaked into response error log: %s", want, out)
		}
	}
}

func TestCONNECTRequestFailureLogRedactsHandlerError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var logs bytes.Buffer
	p, err := NewProxy(ProxyConfig{ListenAddr: "127.0.0.1:0", BundlePath: filepath.Join(t.TempDir(), "bundle.atb"), TargetHosts: []string{"localhost"}}, &leakingHandler{failRequest: true}, slog.New(slog.NewTextHandler(&logs, nil)))
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	p.ca, err = LoadOrCreateLocalCA()
	if err != nil {
		t.Fatalf("LoadOrCreateLocalCA: %v", err)
	}
	upstreamProxy, upstreamServer := net.Pipe()
	defer upstreamProxy.Close()
	defer upstreamServer.Close()
	if err := setPipeDeadline(upstreamProxy, upstreamServer); err != nil {
		t.Fatalf("set upstream pipe deadline: %v", err)
	}
	upstreamDone := make(chan struct{})
	go func() {
		defer close(upstreamDone)
		_, _ = http.ReadRequest(bufio.NewReader(upstreamServer))
		_ = upstreamServer.Close()
	}()
	serverConn, clientConn := net.Pipe()
	if err := setPipeDeadline(serverConn, clientConn); err != nil {
		t.Fatalf("set client pipe deadline: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		req := httptest.NewRequest(http.MethodConnect, "http://localhost:1", nil)
		req.Host = "localhost:1"
		(&forwarder{proxy: p, dialTLS: func(*net.Dialer, string, string, *tls.Config) (net.Conn, error) { return upstreamProxy, nil }}).ServeHTTP(&hijackResponseWriter{conn: serverConn, header: make(http.Header)}, req)
	}()
	defer serverConn.Close()
	defer clientConn.Close()
	reader := bufio.NewReader(clientConn)
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			t.Fatalf("read CONNECT response: %v", readErr)
		}
		if line == "\r\n" {
			break
		}
	}
	roots := x509.NewCertPool()
	certPEM, err := os.ReadFile(p.ca.CertPath)
	if err != nil {
		t.Fatalf("read generated CA certificate: %v", err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("generated CA certificate is not PEM encoded")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse generated CA certificate: %v", err)
	}
	roots.AddCert(cert)
	clientTLS := tls.Client(clientConn, &tls.Config{RootCAs: roots, ServerName: "localhost"})
	if err := clientTLS.Handshake(); err != nil {
		t.Fatalf("TLS handshake: %v", err)
	}
	if _, err := fmt.Fprintf(clientTLS, "POST /v1/messages HTTP/1.1\r\nHost: localhost:1\r\nAuthorization: Bearer %s\r\nContent-Length: 0\r\n\r\n", captureLogRedactionSecret); err != nil {
		t.Fatalf("write inner request: %v", err)
	}
	_ = clientTLS.Close()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("CONNECT did not finish")
	}
	<-upstreamDone
	out := logs.String()
	if !strings.Contains(out, "proxy capture request failed") {
		t.Fatalf("warning missing: %s", out)
	}
	for _, want := range []string{captureLogRedactionSecret, "rejected request with key"} {
		if strings.Contains(out, want) {
			t.Fatalf("%q leaked: %s", want, out)
		}
	}
}

func TestCONNECTResponseFailureLogRedactsHandlerError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var logs bytes.Buffer
	p, err := NewProxy(ProxyConfig{ListenAddr: "127.0.0.1:0", BundlePath: filepath.Join(t.TempDir(), "bundle.atb"), TargetHosts: []string{"localhost"}}, &leakingHandler{failResponse: true}, slog.New(slog.NewTextHandler(&logs, nil)))
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	p.ca, err = LoadOrCreateLocalCA()
	if err != nil {
		t.Fatalf("LoadOrCreateLocalCA: %v", err)
	}
	upstreamProxy, upstreamServer := net.Pipe()
	defer upstreamProxy.Close()
	defer upstreamServer.Close()
	if err := setPipeDeadline(upstreamProxy, upstreamServer); err != nil {
		t.Fatalf("set upstream pipe deadline: %v", err)
	}
	upstreamDone := make(chan struct{})
	go func() {
		defer close(upstreamDone)
		req, readErr := http.ReadRequest(bufio.NewReader(upstreamServer))
		if readErr == nil {
			_, _ = fmt.Fprintf(upstreamServer, "HTTP/1.1 200 OK\r\nX-Upstream-Canary: header-canary\r\nContent-Length: 11\r\n\r\nbody-canary")
			_ = req.Body.Close()
		}
		_ = upstreamServer.Close()
	}()
	serverConn, clientConn := net.Pipe()
	if err := setPipeDeadline(serverConn, clientConn); err != nil {
		t.Fatalf("set client pipe deadline: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		req := httptest.NewRequest(http.MethodConnect, "http://localhost:1", nil)
		req.Host = "localhost:1"
		(&forwarder{proxy: p, dialTLS: func(*net.Dialer, string, string, *tls.Config) (net.Conn, error) { return upstreamProxy, nil }}).ServeHTTP(&hijackResponseWriter{conn: serverConn, header: make(http.Header)}, req)
	}()
	defer serverConn.Close()
	defer clientConn.Close()
	reader := bufio.NewReader(clientConn)
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			t.Fatalf("read CONNECT response: %v", readErr)
		}
		if line == "\r\n" {
			break
		}
	}
	roots := x509.NewCertPool()
	certPEM, err := os.ReadFile(p.ca.CertPath)
	if err != nil {
		t.Fatalf("read generated CA certificate: %v", err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("generated CA certificate is not PEM encoded")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse generated CA certificate: %v", err)
	}
	roots.AddCert(cert)
	clientTLS := tls.Client(clientConn, &tls.Config{RootCAs: roots, ServerName: "localhost"})
	if err := clientTLS.Handshake(); err != nil {
		t.Fatalf("TLS handshake: %v", err)
	}
	if _, err := fmt.Fprintf(clientTLS, "POST /v1/messages HTTP/1.1\r\nHost: localhost:1\r\nAuthorization: Bearer %s\r\nContent-Length: 0\r\n\r\n", captureLogRedactionSecret); err != nil {
		t.Fatalf("write inner request: %v", err)
	}
	response, err := http.ReadResponse(bufio.NewReader(clientTLS), nil)
	if err != nil {
		t.Fatalf("read CONNECT tunnel response: %v", err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	_ = clientTLS.Close()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("CONNECT did not finish")
	}
	<-upstreamDone
	out := logs.String()
	if !strings.Contains(out, "proxy capture response failed") {
		t.Fatalf("warning missing: %s", out)
	}
	for _, want := range []string{"body-canary", "header-canary", captureLogRedactionSecret, "rejected response with body"} {
		if strings.Contains(out, want) {
			t.Fatalf("%q leaked: %s", want, out)
		}
	}
}

func setPipeDeadline(conns ...net.Conn) error {
	deadline := time.Now().Add(3 * time.Second)
	for _, conn := range conns {
		if err := conn.SetDeadline(deadline); err != nil {
			return err
		}
	}
	return nil
}
