// SPDX-License-Identifier: MIT
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"
)

// HealthResponse is returned by GET /healthz.
type HealthResponse struct {
	Status string `json:"status"`
}

// InfoResponse is returned by GET /v1/info.
type InfoResponse struct {
	Version string            `json:"version"`
	Build   map[string]string `json:"build,omitempty"`
	Config  InfoConfigSummary `json:"config"`
}

// InfoConfigSummary exposes non-secret agent configuration.
type InfoConfigSummary struct {
	ListenAddr string `json:"listen_addr"`
	DataDir    string `json:"data_dir"`
}

// Server serves the minimal ATB Agent HTTP surface.
type Server struct {
	cfg            Config
	logger         *slog.Logger
	bundleManager  BundleManager
	workspaceIndex *WorkspaceIndex
	mux            *http.ServeMux
}

// Runtime owns the Agent HTTP server and internal bundle lifecycle manager.
type Runtime struct {
	cfg           Config
	logger        *slog.Logger
	bundleManager BundleManager
	server        *Server
}

// NewRuntime constructs an Agent runtime with a disk-backed bundle manager.
func NewRuntime(cfg Config, logger *slog.Logger) (*Runtime, error) {
	bundleManager := NewBundleFileManager(cfg.DataDir)
	server, err := NewServer(cfg, logger, bundleManager)
	if err != nil {
		return nil, err
	}
	return &Runtime{
		cfg:           cfg,
		logger:        logger,
		bundleManager: bundleManager,
		server:        server,
	}, nil
}

// Shutdown releases runtime resources.
func (r *Runtime) Shutdown(ctx context.Context) error {
	if r.bundleManager == nil {
		return nil
	}
	return r.bundleManager.Shutdown(ctx)
}

// NewServer constructs an agent HTTP server for the given configuration.
// When bundleManager is nil a BundleFileManager rooted at cfg.DataDir is used.
func NewServer(cfg Config, logger *slog.Logger, bundleManager BundleManager) (*Server, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if bundleManager == nil {
		bundleManager = NewBundleFileManager(cfg.DataDir)
	}
	s := &Server{
		cfg:            cfg,
		logger:         logger,
		bundleManager:  bundleManager,
		workspaceIndex: NewWorkspaceIndex(cfg.DataDir),
		mux:            http.NewServeMux(),
	}
	s.registerRoutes()
	return s, nil
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /v1/info", s.handleInfo)
	s.mux.HandleFunc("POST /v1/session/open", s.handleSessionOpen)
	s.mux.HandleFunc("POST /v1/session/{id}/event", s.handleSessionEvent)
	s.mux.HandleFunc("POST /v1/session/{id}/close", s.handleSessionClose)
	s.mux.HandleFunc("GET /v1/workspace/bundles", s.handleWorkspaceBundles)
}

// Handler returns the root HTTP handler for the agent server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

func (s *Server) handleInfo(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, InfoResponse{
		Version: s.cfg.Version,
		Build:   readBuildInfo(),
		Config: InfoConfigSummary{
			ListenAddr: s.cfg.ListenAddr,
			DataDir:    s.cfg.DataDir,
		},
	})
}

func readBuildInfo() map[string]string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil
	}
	out := map[string]string{
		"go_version": info.GoVersion,
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision", "vcs.time", "vcs.modified":
			out[setting.Key] = setting.Value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// Run starts the agent HTTP server and blocks until ctx is cancelled.
func Run(ctx context.Context, cfg Config, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	if err := PrepareWorkspace(cfg.DataDir, logger); err != nil {
		return err
	}

	rt, err := NewRuntime(cfg, logger)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()

	httpServer := &http.Server{
		Handler:           rt.server.Handler(),
		ReadHeaderTimeout: 5 * time.Second, // #nosec G112 -- local loopback agent; timeout prevents Slowloris
	}

	logger.Info("ATB agent starting", "listen_addr", cfg.ListenAddr, "data_dir", cfg.DataDir)

	errCh := make(chan error, 1)
	go func() {
		serveErr := httpServer.Serve(ln)
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Info("ATB agent received shutdown signal")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		if err := rt.Shutdown(shutdownCtx); err != nil {
			return err
		}
		logger.Info("ATB agent stopped cleanly")
		return nil
	case serveErr := <-errCh:
		if serveErr != nil {
			return serveErr
		}
		shutdownErr := rt.Shutdown(context.Background())
		if shutdownErr != nil {
			return shutdownErr
		}
		return nil
	}
}
