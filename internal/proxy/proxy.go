// SPDX-License-Identifier: MIT
// Package proxy implements the local HTTPS capture proxy for ATB.
package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pcguest/atb/internal/identity"
)

// Runner controls proxy lifecycle.
type Runner interface {
	Start(ctx context.Context) error
	Stop() error
}

var _ Runner = (*Proxy)(nil)

// Proxy is a local HTTPS capture proxy that records LLM API traffic into a bundle.
type Proxy struct {
	cfg          ProxyConfig
	handler      Handler
	logger       *slog.Logger
	ca           *LocalCA
	recorder     *BundleRecorder
	sessions     *SessionManager
	custosPusher *CustosPusher

	mu         sync.Mutex
	started    bool
	cancel     context.CancelFunc
	httpServer *http.Server
	listenWG   sync.WaitGroup
}

// NewProxy returns a proxy using the stub handler when handler is nil.
func NewProxy(cfg ProxyConfig, handler Handler, logger *slog.Logger) (*Proxy, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if cfg.Identity == nil {
		cfg.Identity = identity.DefaultChain()
	}
	if handler == nil {
		handler = StubHandler{Logger: logger}
	}

	var cp *CustosPusher
	if cfg.CustosEndpoint != "" {
		var err error
		cp, err = NewCustosPusher(cfg.CustosEndpoint, cfg.CustosToken, logger)
		if err != nil {
			return nil, fmt.Errorf("custos push endpoint: %w", err)
		}
	}

	recorder := NewBundleRecorder(cfg.BundlePath, cp)
	sessions := NewSessionManager(recorder.sessionCloseCallback)

	return &Proxy{
		cfg:          cfg,
		handler:      handler,
		logger:       logger,
		recorder:     recorder,
		sessions:     sessions,
		custosPusher: cp,
	}, nil
}

// Start prepares the CA, prints first-run install instructions, and listens for traffic.
func (p *Proxy) Start(ctx context.Context) error {
	if p == nil {
		return errors.New("proxy: nil receiver")
	}
	if err := p.cfg.Validate(); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started {
		return errors.New("proxy: already started")
	}

	ca, err := LoadOrCreateLocalCA()
	if err != nil {
		return err
	}
	p.ca = ca
	if ca.Created() {
		PrintInstallInstructions(p.loggerWriter(), ca.CertPath)
	}

	runCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.started = true

	// Record the capture-coverage attestation first, so the bundle states what
	// this recorder can and cannot see before any traffic is captured.
	if p.recorder != nil {
		if err := p.recorder.AppendEvent(CaptureScopeEvent(p.cfg)); err != nil && p.logger != nil {
			p.logger.Warn("proxy record capture scope failed", "error", err)
		}
	}

	p.listenWG.Add(1)
	go func() {
		defer p.listenWG.Done()
		if err := p.ListenAndServe(runCtx); err != nil && p.logger != nil {
			p.logger.Error("proxy listener stopped", "error", err)
		}
	}()

	logger := p.logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Info("proxy listening",
		"listen_addr", p.cfg.ListenAddr,
		"bundle_path", p.cfg.BundlePath,
		"targets", p.cfg.TargetHosts,
		"custos_endpoint", p.cfg.CustosEndpoint, // Log Custos endpoint
	)
	return nil
}

func (p *Proxy) loggerWriter() io.Writer {
	if p.logger == nil {
		return io.Discard
	}
	return &logWriter{logger: p.logger}
}

type logWriter struct {
	logger *slog.Logger
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.logger.Info(strings.TrimSpace(string(p)))
	return len(p), nil
}

// Stop shuts down a started proxy instance.
func (p *Proxy) Stop() error {
	if p == nil {
		return errors.New("proxy: nil receiver")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.started {
		return nil
	}
	if p.cancel != nil {
		p.cancel()
	}
	p.waitForListener(10 * time.Second)
	if p.sessions != nil {
		p.sessions.CloseAll()
	}
	p.started = false
	if p.logger != nil {
		p.logger.Info("proxy stopped", "listen_addr", p.cfg.ListenAddr)
	}
	return nil
}

func (p *Proxy) waitForListener(timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		p.listenWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		if p.httpServer != nil {
			_ = p.httpServer.Close()
		}
		p.listenWG.Wait()
	}
}

// Handler returns the configured capture handler.
func (p *Proxy) Handler() Handler {
	if p == nil {
		return StubHandler{}
	}
	return p.handler
}

// Config returns a copy of the proxy configuration.
func (p *Proxy) Config() ProxyConfig {
	if p == nil {
		return ProxyConfig{}
	}
	return p.cfg
}

func (p *Proxy) applyIdentity(rec *RequestRecord) {
	if rec == nil {
		return
	}
	id := p.cfg.ResolveIdentity(rec.APIKey)
	rec.DisplayName = id.DisplayName
	rec.Email = id.Email
}

// HandleRequest resolves identity and delegates to the configured handler.
func (p *Proxy) HandleRequest(ctx context.Context, rec RequestRecord) error {
	if p == nil {
		return errors.New("proxy: nil receiver")
	}
	p.applyIdentity(&rec)
	return p.handler.HandleRequest(ctx, rec)
}

// HandleResponse resolves identity and delegates to the configured handler.
func (p *Proxy) HandleResponse(ctx context.Context, rec ResponseRecord, apiKey string) error {
	if p == nil {
		return errors.New("proxy: nil receiver")
	}
	id := p.cfg.ResolveIdentity(apiKey)
	rec.DisplayName = id.DisplayName
	rec.Email = id.Email
	return p.handler.HandleResponse(ctx, rec)
}

// DefaultTargetHosts maps shorthand provider names to upstream hostnames.
func DefaultTargetHosts(names ...string) []string {
	aliases := map[string]string{
		"openai":    "api.openai.com",
		"anthropic": "api.anthropic.com",
	}
	out := make([]string, 0, len(names))
	for _, name := range names {
		key := strings.TrimSpace(strings.ToLower(name))
		if key == "" {
			continue
		}
		if host, ok := aliases[key]; ok {
			out = append(out, host)
			continue
		}
		out = append(out, key)
	}
	return out
}
