// SPDX-License-Identifier: MIT
package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/identity"
)

// forwarder performs HTTPS MITM forwarding and capture for configured targets.
type forwarder struct {
	proxy *Proxy
}

func (f *forwarder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		f.handleConnect(w, r)
		return
	}
	f.handleHTTP(w, r)
}

func (f *forwarder) handleConnect(w http.ResponseWriter, r *http.Request) {
	if !IsTargetHost(r.Host, f.proxy.cfg.TargetHosts) {
		http.Error(w, "connect host not allowed", http.StatusForbidden)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "connect hijack unsupported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hj.Hijack()
	if err != nil {
		return
	}

	if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		_ = clientConn.Close()
		return
	}

	certPEM, keyPEM, err := f.proxy.ca.LeafCertificate(r.Host)
	if err != nil {
		_ = clientConn.Close()
		return
	}
	leaf, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		_ = clientConn.Close()
		return
	}

	tlsConn := tls.Server(clientConn, &tls.Config{
		Certificates: []tls.Certificate{leaf},
		NextProtos:   []string{"http/1.1"},
	})
	if err := tlsConn.Handshake(); err != nil {
		_ = tlsConn.Close()
		return
	}

	upstreamConn, err := tls.Dial("tcp", r.Host, &tls.Config{
		ServerName: stripPort(r.Host),
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		_ = tlsConn.Close()
		return
	}

	reader := bufio.NewReader(tlsConn)
	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			break
		}
		maxBody := f.proxy.cfg.EffectiveMaxBodyBytes()
		releaseBudget, ok := f.acquireBodyBudget(maxBody)
		if !ok {
			f.recordCaptureRejection(r.Host, req, "request", "memory_budget_exhausted", maxBody, 0)
			_ = writeTunnelError(tlsConn, req, http.StatusServiceUnavailable, "proxy capture capacity exceeded")
			break
		}
		body, err := ReadRequestBodyLimited(req, maxBody)
		if err != nil {
			reason, observed := classifyBodyReadError(err, maxBody)
			f.recordCaptureRejection(r.Host, req, "request", reason, maxBody, observed)
			status := http.StatusBadRequest
			if errors.Is(err, ErrBodyTooLarge) {
				status = http.StatusRequestEntityTooLarge
			}
			_ = writeTunnelError(tlsConn, req, status, http.StatusText(status))
			releaseBudget()
			break
		}
		if err := f.captureRequest(r.Host, req, body); err != nil && f.proxy.logger != nil {
			f.proxy.logger.Warn("proxy capture request failed", "error", err)
		}

		if err := req.Write(upstreamConn); err != nil {
			releaseBudget()
			break
		}
		resp, err := http.ReadResponse(bufio.NewReader(upstreamConn), req)
		if err != nil {
			releaseBudget()
			break
		}
		respBody, err := ReadBodyLimited(resp.Body, maxBody)
		_ = resp.Body.Close()
		if err != nil {
			reason, observed := classifyBodyReadError(err, maxBody)
			f.recordCaptureRejection(r.Host, req, "response", reason, maxBody, observed)
			_ = writeTunnelError(tlsConn, req, http.StatusBadGateway, "upstream response could not be captured")
			releaseBudget()
			break
		}
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		if err := f.captureResponse(r.Host, req, resp, respBody); err != nil && f.proxy.logger != nil {
			f.proxy.logger.Warn("proxy capture response failed", "error", err)
		}
		if err := resp.Write(tlsConn); err != nil {
			releaseBudget()
			break
		}
		releaseBudget()
	}
	_ = tlsConn.Close()
	_ = upstreamConn.Close()
}

func (f *forwarder) handleHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Host
	if host == "" {
		host = r.Host
	}
	if !IsTargetHost(host, f.proxy.cfg.TargetHosts) {
		http.Error(w, "host not allowed", http.StatusForbidden)
		return
	}

	maxBody := f.proxy.cfg.EffectiveMaxBodyBytes()
	releaseBudget, ok := f.acquireBodyBudget(maxBody)
	if !ok {
		f.recordCaptureRejection(host, r, "request", "memory_budget_exhausted", maxBody, 0)
		http.Error(w, "proxy capture capacity exceeded", http.StatusServiceUnavailable)
		return
	}
	defer releaseBudget()
	body, err := ReadRequestBodyLimited(r, maxBody)
	if err != nil {
		reason, observed := classifyBodyReadError(err, maxBody)
		f.recordCaptureRejection(host, r, "request", reason, maxBody, observed)
		status := http.StatusBadRequest
		message := "request body could not be read"
		if errors.Is(err, ErrBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
			message = "request body too large"
		}
		http.Error(w, message, status)
		return
	}
	if err := f.captureRequest(host, r, body); err != nil && f.proxy.logger != nil {
		f.proxy.logger.Warn("proxy capture request failed", "error", err)
	}

	targetURL := urlForHost(host)
	proxy := httputil.NewSingleHostReverseProxy(targetURL) // #nosec G704 -- host validated by IsTargetHost allowlist before proxy construction
	proxy.ModifyResponse = func(resp *http.Response) error {
		respBody, err := ReadBodyLimited(resp.Body, maxBody)
		if err != nil {
			_ = resp.Body.Close()
			reason, observed := classifyBodyReadError(err, maxBody)
			f.recordCaptureRejection(host, r, "response", reason, maxBody, observed)
			return err
		}
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		return f.captureResponse(host, r, resp, respBody)
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, _ *http.Request, err error) {
		if f.proxy.logger != nil {
			f.proxy.logger.Warn("proxy upstream response failed", "error", err, "host", stripPort(host))
		}
		http.Error(rw, "upstream response could not be captured", http.StatusBadGateway)
	}
	proxy.ServeHTTP(w, r)
}

func (f *forwarder) acquireBodyBudget(maxBody int64) (func(), bool) {
	if f == nil || f.proxy == nil || f.proxy.bodyBudget == nil {
		return func() {}, true
	}
	reservation := maxBody * 2
	if !f.proxy.bodyBudget.TryAcquire(reservation) {
		return nil, false
	}
	return func() { f.proxy.bodyBudget.Release(reservation) }, true
}

func classifyBodyReadError(err error, limit int64) (string, int64) {
	var tooLarge *BodyTooLargeError
	if errors.As(err, &tooLarge) {
		return "body_too_large", tooLarge.Observed
	}
	return "body_read_error", 0
}

func (f *forwarder) recordCaptureRejection(host string, req *http.Request, direction, reason string, limit, observed int64) {
	if f == nil || f.proxy == nil || f.proxy.recorder == nil || f.proxy.sessions == nil || req == nil {
		return
	}
	sess := f.proxy.sessions.Resolve(ExtractThreadKey(host, req, nil))
	ev := CaptureRejectedEvent(sess.ID, host, req.Method, req.URL.Path, direction, reason, limit, observed)
	if err := f.proxy.recorder.AppendEvent(ev); err != nil && f.proxy.logger != nil {
		f.proxy.logger.Warn("proxy record capture rejection failed", "error", err, "host", stripPort(host))
	}
}

func writeTunnelError(conn io.Writer, req *http.Request, status int, message string) error {
	body := message + "\n"
	resp := &http.Response{
		StatusCode:    status,
		Status:        http.StatusText(status),
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: int64(len(body)),
		Header:        make(http.Header),
		Body:          io.NopCloser(bytes.NewBufferString(body)),
		Request:       req,
	}
	resp.Header.Set("Content-Type", "text/plain; charset=utf-8")
	resp.Header.Set("Connection", "close")
	return resp.Write(conn)
}

func (f *forwarder) captureRequest(host string, req *http.Request, body []byte) error {
	threadKey := ExtractThreadKey(host, req, body)
	sess := f.proxy.sessions.Resolve(threadKey)
	apiKey := ExtractAPIKey(req.Header)
	rec := RequestRecord{
		SessionID:   sess.ID,
		Host:        stripPort(host),
		Method:      req.Method,
		Path:        req.URL.Path,
		Provider:    ProviderFromHost(host),
		Model:       ParseModelFromBody(body),
		APIKey:      apiKey,
		Headers:     ScanHeaders(req.Header),
		Body:        body,
		RecordedAt:  time.Now().UTC(),
		CaptureBody: f.proxy.cfg.CaptureBodies,
	}
	id := f.proxy.cfg.ResolveIdentity(apiKey)
	actorID := id.DisplayName
	if actorID == "" && apiKey != "" {
		actorID = identity.FallbackDisplayName(apiKey)
	}
	if actorID != "" {
		sess.UpdateActorID(actorID)
	}
	if err := f.proxy.HandleRequest(context.Background(), rec); err != nil {
		return err
	}
	ev, err := rec.ToEvent()
	if err != nil {
		return err
	}
	sess.noteExchangeStarted(rec.RecordedAt)

	requestHash, err := f.proxy.recorder.AppendEventHash(ev)
	if err != nil {
		return err
	}
	sess.setLastRequestEventHash(requestHash)
	f.recordToolResultErrors(sess.ID, rec.RecordedAt, body)
	return nil
}

func (f *forwarder) captureResponse(host string, req *http.Request, resp *http.Response, body []byte) error {
	threadKey := ExtractThreadKey(host, req, nil)
	sess := f.proxy.sessions.Resolve(threadKey)
	prompt, output, total := ParseUsageFromResponse(body)
	model := ParseModelFromBody(body)
	if model == "" {
		sess.mu.Lock()
		model = sess.Model
		sess.mu.Unlock()
	}
	f.proxy.sessions.NoteExchange(sess, model, prompt, output)
	apiKey := ExtractAPIKey(req.Header)
	id := f.proxy.cfg.ResolveIdentity(apiKey)
	actorID := id.DisplayName
	if actorID == "" && apiKey != "" {
		actorID = identity.FallbackDisplayName(apiKey)
	}
	if actorID != "" {
		sess.UpdateActorID(actorID)
	}

	rec := ResponseRecord{
		SessionID:    sess.ID,
		Host:         stripPort(host),
		Method:       req.Method,
		Path:         req.URL.Path,
		Provider:     ProviderFromHost(host),
		Model:        model,
		StatusCode:   resp.StatusCode,
		Headers:      ScanHeaders(resp.Header),
		Body:         body,
		RecordedAt:   time.Now().UTC(),
		PromptTokens: prompt,
		OutputTokens: output,
		TotalTokens:  total,
		CaptureBody:  f.proxy.cfg.CaptureBodies,
	}
	if err := f.proxy.HandleResponse(context.Background(), rec, apiKey); err != nil {
		return err
	}
	responseEvent, err := rec.ToEvent()
	if err != nil {
		return err
	}
	if err := f.proxy.recorder.AppendEvent(responseEvent); err != nil {
		return err
	}

	completedAt := rec.RecordedAt
	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	}
	exchangeID := sess.NextExchangeID()
	toolCalls := CountToolCallsFromResponse(body)
	exchangeEvent := &event.Event{
		Type:      TypeExchangeComplete,
		Timestamp: completedAt.Format(time.RFC3339Nano),
		Data: ExchangeCompleteRecord(
			sess,
			exchangeID,
			sess.lastRequestEventHashLocked(),
			model,
			prompt,
			output,
			toolCalls,
			completedAt,
		),
	}
	if err := f.proxy.recorder.AppendEvent(exchangeEvent); err != nil {
		return err
	}
	f.recordToolCalls(sess, body, actorID, completedAt)
	return nil
}

// recordToolCalls appends an atb.tool.call accountability event for each
// tool/function call the model requested in the response. Tool arguments are
// digested, never stored raw. Recording failures are logged, not fatal: a
// hiccup recording an accountability detail must not break the proxied flow.
func (f *forwarder) recordToolCalls(sess *Session, body []byte, actorID string, ts time.Time) {
	for _, tc := range ExtractToolCalls(body) {
		if tc.Name == "" {
			continue // atb.tool.call requires tool_name
		}
		data := map[string]any{
			"session_id": sess.ID,
			"tool_name":  tc.Name,
		}
		if d := tc.InputDigest(); d != "" {
			data["tool_input_digest"] = d
		}
		if actorID != "" {
			data["actor_id"] = actorID
		}
		ev := &event.Event{
			Type:      event.TypeToolCall,
			Timestamp: ts.UTC().Format(time.RFC3339Nano),
			Data:      data,
		}
		if err := f.proxy.recorder.AppendEvent(ev); err != nil && f.proxy.logger != nil {
			f.proxy.logger.Warn("proxy record tool call failed", "error", err)
		}
	}
}

// recordToolResultErrors appends an ai.action.error accountability event for
// each failed tool result carried in a request body (Anthropic tool_result
// with is_error=true). Error detail is digested, never stored raw.
func (f *forwarder) recordToolResultErrors(sessionID string, ts time.Time, body []byte) {
	for _, tre := range ExtractToolResultErrors(body) {
		data := map[string]any{
			"session_id":  sessionID,
			"action_id":   tre.ToolUseID,
			"error_class": "failed",
		}
		if d := tre.DetailDigest(); d != "" {
			data["error_detail_digest"] = d
		}
		ev := &event.Event{
			Type:      event.TypeAIActionError,
			Timestamp: ts.UTC().Format(time.RFC3339Nano),
			Data:      data,
		}
		if err := f.proxy.recorder.AppendEvent(ev); err != nil && f.proxy.logger != nil {
			f.proxy.logger.Warn("proxy record tool error failed", "error", err)
		}
	}
}

func urlForHost(host string) *url.URL {
	// Preserve an explicit upstream port. Host allowlisting deliberately ignores
	// ports, but forwarding must not silently redirect a validated :port target
	// to the scheme default.
	return &url.URL{Scheme: "http", Host: strings.TrimSpace(host)}
}

// RoundTripFixture applies a recorded HTTP exchange through capture helpers.
func RoundTripFixture(host string, req *http.Request, reqBody []byte, resp *http.Response, respBody []byte, recorder *BundleRecorder, cfg ProxyConfig) (*Proxy, error) {
	p, err := NewProxy(cfg, LoggingHandler{}, nil)
	if err != nil {
		return nil, err
	}
	p.recorder = recorder
	p.sessions = NewSessionManager(func(sess *Session) error {
		return recorder.AppendSessionClose(sess)
	})
	f := &forwarder{proxy: p}
	if err := f.captureRequest(host, req, reqBody); err != nil {
		return nil, err
	}
	if err := f.captureResponse(host, req, resp, respBody); err != nil {
		return nil, err
	}
	return p, nil
}

// ListenAndServe starts an HTTP proxy listener until the context is cancelled.
func (p *Proxy) ListenAndServe(ctx context.Context) error {
	if p == nil {
		return errors.New("proxy: nil receiver")
	}
	if err := p.cfg.Validate(); err != nil {
		return err
	}
	if p.ca == nil {
		ca, err := LoadOrCreateLocalCA()
		if err != nil {
			return err
		}
		p.ca = ca
	}
	if p.recorder == nil {
		// Fixed: nil preserves no-push behaviour for recorder fallback construction.
		p.recorder = NewBundleRecorder(p.cfg.BundlePath, nil)
	}
	if p.sessions == nil {
		p.sessions = NewSessionManager(func(sess *Session) error {
			return p.recorder.AppendSessionClose(sess)
		})
	}

	server := &http.Server{
		Addr:    p.cfg.ListenAddr,
		Handler: &forwarder{proxy: p},
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
		ReadHeaderTimeout: 5 * time.Second, // Mitigate Slowloris attacks
	}
	p.httpServer = server

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
