// SPDX-License-Identifier: MIT
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	atbembed "github.com/pcguest/atb"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/sessionindex"
	verifypkg "github.com/pcguest/atb/internal/verify"
	apiv1 "github.com/pcguest/atb/pkg/api/v1"
	atbauth "github.com/pcguest/atb/pkg/auth"
)

var errViewHelp = errors.New("view help requested")

type viewConfig struct {
	BundlePath   string
	Host         string
	Port         int
	PortSet      bool
	NoOpen       bool
	LogReveals   bool
	ProfilePath  string
	SessionToken string
	SessionPaths []string
	OIDCIssuer   string
	OIDCAudience string
}

const defaultViewHost = "127.0.0.1"

func cmdView() {
	cfg, err := parseViewArgs(os.Args[2:])
	if err != nil {
		if errors.Is(err, errViewHelp) {
			printViewUsage()
			return
		}
		fmt.Fprintf(os.Stderr, "atb view: %v\n", err)
		printViewUsage()
		os.Exit(exitUserError)
	}

	bundlePath, err := resolveBundlePath(cfg.BundlePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb view: resolve bundle path: %v\n", err)
		os.Exit(exitSystemError)
	}

	sessionToken := cfg.SessionToken
	if sessionToken == "" {
		sessionToken, err = generateRevealAuthToken()
		if err != nil {
			fmt.Fprintf(os.Stderr, "atb view: generate session token: %v\n", err)
			os.Exit(exitSystemError)
		}
	}

	handler, eventCount, tamperDetected, openPath, err := buildViewServer(bundlePath, cfg.LogReveals, cfg.ProfilePath, sessionToken, cfg.OIDCIssuer, cfg.OIDCAudience, cfg.SessionPaths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb view: %v\n", err)
		os.Exit(classifyBundleLoadError(err))
	}

	ln, port, err := listenViewPort(cfg.Host, cfg.Port, cfg.PortSet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb view: %v\n", err)
		os.Exit(exitSystemError)
	}
	defer ln.Close()

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))
	url := "http://" + addr + openPath
	if sessionToken != "" {
		url += "#session=" + sessionToken
	}
	if port != cfg.Port {
		fmt.Fprintf(os.Stderr, "atb view: port %d unavailable; using %d\n", cfg.Port, port)
	}
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second, // #nosec G112 -- local ephemeral server; timeout prevents Slowloris
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	if tamperDetected {
		fmt.Printf("atb view: verification failed for %s; serving tamper warning at %s\n", bundlePath, url)
	} else {
		fmt.Printf("✓ Serving %s (%d events) at %s\n", bundlePath, eventCount, url)
	}
	if !cfg.NoOpen {
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(os.Stderr, "atb view: could not auto-open browser: %v\n", err)
		}
	}

	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "atb view: server error: %v\n", err)
		os.Exit(exitSystemError)
	}

	fmt.Println("atb view: stopped")
}

// buildViewServer prepares the HTTP handler for atb view.
// It returns the handler, total event count (for the startup message), a tamper flag, the
// suggested open path, and any error.
// When the embedded review UI is absent (e.g. a go install build), a minimal install-guidance
// page is served at / instead; it exposes no bundle data.
func buildViewServer(bundlePath string, logReveals bool, profilePath string, sessionToken string, oidcIssuer, oidcAudience string, sessionPathSets ...[]string) (http.Handler, int, bool, string, error) {
	_ = logReveals // Privacy reveal auditing is always on; flag retained for CLI compatibility.
	var sessionPaths []string
	if len(sessionPathSets) > 0 {
		sessionPaths = sessionPathSets[0]
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		return nil, 0, false, "/", fmt.Errorf("load bundle %s: %w", bundlePath, err)
	}
	verifyErr := b.Verify()
	revealAuthToken, err := generateRevealAuthToken()
	if err != nil {
		return nil, 0, false, "/", fmt.Errorf("generate privacy reveal auth token: %w", err)
	}
	if revealAuthToken == "" {
		fmt.Fprintln(os.Stderr, "atb view: warning: privacy reveal auth is not configured; reveal operations are disabled until configured")
	}

	var profileReport *apiv1.ProfileReportSummary
	var verifierReport *verifypkg.VerifierReport
	if profilePath != "" && verifyErr == nil {
		profileReport, verifierReport = buildStartupProfileReports(b, bundlePath, profilePath)
	}

	var sessions []sessionindex.SessionEntry
	var sessionsByActor map[string][]sessionindex.SessionEntry
	var sessionIndexErr error
	if len(sessionPaths) > 0 {
		indexPaths := sessionIndexPaths(bundlePath, sessionPaths)
		sessions, sessionIndexErr = sessionindex.BuildIndex(context.Background(), indexPaths)
		sessionsByActor = sessionindex.GroupByActor(sessions)
		if sessionIndexErr != nil {
			fmt.Fprintf(os.Stderr, "atb view: session index built with load errors: %v\n", sessionIndexErr)
		}
		fmt.Fprintf(os.Stderr, "Session index built: %d sessions from %s\n", len(sessions), describeSessionSource(sessionPaths))
	}

	var jwtValidator *atbauth.JWTValidator
	if oidcIssuer != "" && oidcAudience != "" {
		jwtValidator, err = atbauth.NewJWTValidator(context.Background(), oidcIssuer, oidcAudience)
		if err != nil {
			return nil, 0, false, "/", fmt.Errorf("failed to create OIDC JWT validator: %w", err)
		}
		fmt.Fprintf(os.Stderr, "atb view: OIDC/JWT authentication enabled for API routes\n")
	}

	eventCount := len(b.Records)
	mux := http.NewServeMux()
	api := apiv1.NewAPIServer(apiv1.APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		VerifyErr:       verifyErr,
		SessionToken:    sessionToken,
		RevealAuthToken: revealAuthToken,
		ProfileReport:   profileReport,
		VerifierReport:  verifierReport,
		ProfilePath:     profilePath,
		Sessions:        sessions,
		SessionsByActor: sessionsByActor,
		SessionIndexErr: sessionIndexErr,
		SessionIndexed:  len(sessionPaths) > 0,
		JWTValidator:    jwtValidator, // Pass the JWT validator
	})
	api.Register(mux)

	if verifyErr != nil {
		mux.Handle("/", apiv1.NewTamperHandler(bundlePath, verifyErr))
		return withSecurityHeaders(mux, revealAuthToken), eventCount, true, "/", nil
	}

	if dashboardFS, ok := embeddedDashboardFS(); ok {
		// Serve the embedded Next.js dashboard at /view/ and redirect / to it.
		// The session token is in the URL fragment so it reaches the browser JS
		// without being forwarded in HTTP requests.
		static := http.FileServer(dashboardFS)
		mux.Handle("/_next/", static)
		mux.Handle("/view/", static)
		mux.HandleFunc("/view", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/view/", http.StatusTemporaryRedirect)
		})
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/view/", http.StatusTemporaryRedirect)
		})
		return withSecurityHeaders(mux, revealAuthToken), eventCount, false, "/view/", nil
	}

	// Degraded path: the embedded review UI is not available (go install builds).
	// Serve a minimal guidance page that exposes no bundle data.
	mux.Handle("/", newInstallFallbackHandler())
	return withSecurityHeaders(mux, revealAuthToken), eventCount, false, "/", nil
}

func resolveBundlePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return bundle.DefaultPath(), nil
	}
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return filepath.Join(path, bundle.BundleFile), nil
		}
		return path, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	if strings.HasSuffix(path, string(os.PathSeparator)) {
		return filepath.Join(path, bundle.BundleFile), nil
	}
	return path, nil
}

func generateRevealAuthToken() (string, error) {
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(token[:]), nil
}

func candidateViewPorts(port int, explicit bool) []int {
	if explicit || port == 0 {
		return []int{port}
	}
	ports := []int{port}
	for next := port + 1; next <= port+2 && next <= 65535; next++ {
		ports = append(ports, next)
	}
	return ports
}

func listenViewPort(host string, port int, explicit bool) (net.Listener, int, error) {
	var lastErr error
	for _, candidate := range candidateViewPorts(port, explicit) {
		addr := net.JoinHostPort(host, strconv.Itoa(candidate))
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			actualPort := candidate
			if tcpAddr, ok := ln.Addr().(*net.TCPAddr); ok {
				actualPort = tcpAddr.Port
			}
			return ln, actualPort, nil
		}
		lastErr = err
		if explicit || !isAddrInUseError(err) {
			return nil, 0, fmt.Errorf("listen on %s: %w", addr, err)
		}
	}
	return nil, 0, fmt.Errorf("listen on %s:%d: %w", host, port, lastErr)
}

func isAddrInUseError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.EADDRINUSE) {
		return true
	}
	var errno syscall.Errno
	if errors.As(err, &errno) && errno == syscall.Errno(10048) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "only one usage of each socket address")
}

func openBrowser(url string) error {
	if err := validateBrowserURL(url); err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url) // #nosec G204 -- validateBrowserURL restricts the argument to a loopback HTTP(S) URL.
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url) // #nosec G204 -- validateBrowserURL restricts the argument to a loopback HTTP(S) URL.
	default:
		cmd = exec.Command("xdg-open", url) // #nosec G204 -- validateBrowserURL restricts the argument to a loopback HTTP(S) URL.
	}
	return cmd.Start()
}

func validateBrowserURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("invalid browser URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("invalid browser URL scheme")
	}
	if parsed.User != nil {
		return fmt.Errorf("browser URL must not contain credentials")
	}
	host := parsed.Hostname()
	if !strings.EqualFold(host, "localhost") {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return fmt.Errorf("browser URL must use a loopback host")
		}
	}
	return nil
}

func buildStartupProfileReports(b *bundle.Bundle, bundlePath string, profilePath string) (*apiv1.ProfileReportSummary, *verifypkg.VerifierReport) {
	selection, _, err := resolveVerifySelection(profilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb view: profile evaluation skipped: %v\n", err)
		return nil, nil
	}
	report, err := verifypkg.EvaluateBundle(verifypkg.EvaluateConfig{
		BundlePath:    bundlePath,
		Records:       b.Records,
		Profiles:      selection.profiles,
		AllApplicable: selection.allApplicable,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb view: profile evaluation skipped: %v\n", err)
		return nil, nil
	}
	summary := startupProfileSummary(*report)
	verifierReport := verifypkg.ReportFromVerifyWithBundle(*report, b)
	return &summary, &verifierReport
}

func startupProfileSummary(report verifypkg.Report) apiv1.ProfileReportSummary {
	summary := apiv1.ProfileReportSummary{
		ChainValid:       report.Integrity.ChainValid,
		AnchorStatus:     report.Anchoring.Status,
		CriticalFailures: []apiv1.FailureDTO{},
		Warnings:         []string{},
	}
	if report.CAS != nil {
		summary.CASScore = report.CAS.Overall
		summary.CASGrade = report.CAS.Grade
		summary.CorroborationBonus = report.CAS.CorroborationBonus
		summary.EffectiveScore = report.CAS.EffectiveScore
		summary.SubScores = report.CAS.SubScores
	}
	if len(report.Profiles) > 0 {
		profile := report.Profiles[0]
		summary.ProfileID = profile.ProfileID
		summary.ProfileVersion = profile.Version
		summary.Pass = report.Integrity.ChainValid && profile.Pass
		summary.Warnings = append(summary.Warnings, profile.RequiredWarnings...)
		for _, failure := range profile.CriticalFailures {
			summary.CriticalFailures = append(summary.CriticalFailures, apiv1.FailureDTO{Kind: failure.Kind, Detail: failure.Detail})
		}
	}
	summary.Exclusions = append(summary.Exclusions, report.Exclusions...)
	summary.ResidualRiskLevel = report.ResidualRisk.Level
	for _, gap := range report.ProvabilityGaps {
		summary.ProvabilityGaps = append(summary.ProvabilityGaps, apiv1.ProvabilityGapDTO{
			Gap:        gap.Gap,
			Layer:      gap.Layer,
			Mitigation: gap.Mitigation,
			ClosedWhen: gap.ClosedWhen,
		})
	}
	return summary
}

func withSecurityHeaders(next http.Handler, revealAuthToken string) http.Handler {
	csp := strings.Join([]string{
		"default-src 'self'",
		"base-uri 'none'",
		"frame-ancestors 'none'",
		"object-src 'none'",
		"script-src 'self' 'unsafe-inline'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: blob:",
		"font-src 'self' data:",
		"connect-src 'self'",
		"worker-src 'self' blob:",
		"form-action 'none'",
	}, "; ")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		if revealAuthToken != "" {
			http.SetCookie(w, &http.Cookie{ // #nosec G124 -- Secure tracks r.TLS so the loopback HTTP viewer remains usable; HttpOnly and SameSite are strict.
				Name:     "atb_reveal_token",
				Value:    revealAuthToken,
				Path:     "/",
				HttpOnly: true,
				Secure:   r.TLS != nil,
				SameSite: http.SameSiteStrictMode,
			})
		}
		next.ServeHTTP(w, r)
	})
}

func embeddedDashboardFS() (http.FileSystem, bool) {
	dashboard, err := fs.Sub(atbembed.WebOutFS, "web/out")
	if err != nil {
		return nil, false
	}
	if _, err := fs.Stat(dashboard, "view/index.html"); err != nil {
		return nil, false
	}
	return http.FS(dashboard), true
}

// newInstallFallbackHandler returns a minimal HTML page for builds that do not include
// the embedded review UI (e.g. go install from the module proxy). It exposes no bundle data.
func newInstallFallbackHandler() http.Handler {
	body := `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>ATB Viewer</title>
  <style>
    body { margin: 0; font-family: ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #0a0a0f; color: #e2e8f0; }
    main { max-width: 680px; margin: 80px auto; padding: 24px; }
    h1 { font-size: 1.25rem; font-weight: 600; margin: 0 0 12px; }
    p { color: #94a3b8; line-height: 1.6; margin: 0 0 8px; }
    code { background: #1e293b; color: #e2e8f0; padding: 2px 6px; border-radius: 4px; font-family: ui-monospace, monospace; font-size: 0.875em; }
  </style>
</head>
<body>
  <main>
    <h1>ATB Viewer</h1>
    <p>The local review UI is not available in this build.</p>
    <p>The CLI interface works normally — run <code>atb verify</code> to check bundle integrity.</p>
    <p>To use the visual interface, build from source:</p>
    <p><code>cd web &amp;&amp; npm ci &amp;&amp; npm run build &amp;&amp; cd .. &amp;&amp; go build -o atb ./cmd/atb</code></p>
  </main>
</body>
</html>`
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})
}

func printViewUsage() {
	fmt.Println("Usage: atb view [bundle_path] [--bundle path/to/file.atb] [--host 127.0.0.1] [--port 8080] [--no-open] [--log-reveals] [--profile <id-or-path>] [--session-token <hex-token>] [--sessions <glob-or-dir>] [--oidc-issuer <url>] [--oidc-audience <aud>]")
}

func parseViewArgs(args []string) (viewConfig, error) {
	cfg := viewConfig{Host: defaultViewHost, Port: 8080}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errViewHelp
		case arg == "--host":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --host")
			}
			i++
			host := strings.TrimSpace(args[i])
			if host == "" {
				return cfg, fmt.Errorf("--host cannot be empty")
			}
			cfg.Host = host
		case strings.HasPrefix(arg, "--host="):
			host := strings.TrimSpace(strings.TrimPrefix(arg, "--host="))
			if host == "" {
				return cfg, fmt.Errorf("--host cannot be empty")
			}
			cfg.Host = host
		case arg == "--port":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --port")
			}
			i++
			p, err := strconv.Atoi(args[i])
			if err != nil {
				return cfg, fmt.Errorf("invalid --port value %q", args[i])
			}
			cfg.Port = p
			cfg.PortSet = true
		case strings.HasPrefix(arg, "--port="):
			raw := strings.TrimPrefix(arg, "--port=")
			p, err := strconv.Atoi(raw)
			if err != nil {
				return cfg, fmt.Errorf("invalid --port value %q", raw)
			}
			cfg.Port = p
			cfg.PortSet = true
		case arg == "--bundle":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --bundle")
			}
			i++
			if cfg.BundlePath != "" {
				return cfg, fmt.Errorf("bundle path already set")
			}
			cfg.BundlePath = args[i]
		case strings.HasPrefix(arg, "--bundle="):
			if cfg.BundlePath != "" {
				return cfg, fmt.Errorf("bundle path already set")
			}
			cfg.BundlePath = strings.TrimPrefix(arg, "--bundle=")
		case arg == "--no-open":
			cfg.NoOpen = true
		case arg == "--log-reveals":
			cfg.LogReveals = true
		case arg == "--profile":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --profile")
			}
			i++
			cfg.ProfilePath = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--profile="):
			cfg.ProfilePath = strings.TrimSpace(strings.TrimPrefix(arg, "--profile="))
		case arg == "--session-token":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --session-token")
			}
			i++
			cfg.SessionToken = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--session-token="):
			cfg.SessionToken = strings.TrimSpace(strings.TrimPrefix(arg, "--session-token="))
		case arg == "--sessions":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --sessions")
			}
			i++
			pattern := args[i]
			paths, err := resolveSessionPaths(pattern)
			if err != nil {
				return cfg, fmt.Errorf("failed to resolve session paths for %q: %w", pattern, err)
			}
			cfg.SessionPaths = append(cfg.SessionPaths, paths...)
		case strings.HasPrefix(arg, "--sessions="):
			pattern := strings.TrimPrefix(arg, "--sessions=")
			paths, err := resolveSessionPaths(pattern)
			if err != nil {
				return cfg, fmt.Errorf("failed to resolve session paths for %q: %w", pattern, err)
			}
			cfg.SessionPaths = append(cfg.SessionPaths, paths...)
		case arg == "--oidc-issuer":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --oidc-issuer")
			}
			i++
			cfg.OIDCIssuer = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--oidc-issuer="):
			cfg.OIDCIssuer = strings.TrimSpace(strings.TrimPrefix(arg, "--oidc-issuer="))
		case arg == "--oidc-audience":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --oidc-audience")
			}
			i++
			cfg.OIDCAudience = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--oidc-audience="):
			cfg.OIDCAudience = strings.TrimSpace(strings.TrimPrefix(arg, "--oidc-audience="))
		case strings.HasPrefix(arg, "-"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			if cfg.BundlePath != "" {
				return cfg, fmt.Errorf("expected at most one bundle path")
			}
			cfg.BundlePath = arg
		}
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return cfg, fmt.Errorf("--port must be between 1 and 65535")
	}
	if !isLoopbackViewHost(cfg.Host) {
		return cfg, fmt.Errorf("--host must be localhost or a loopback IP address")
	}
	if (cfg.OIDCIssuer == "") != (cfg.OIDCAudience == "") {
		return cfg, fmt.Errorf("--oidc-issuer and --oidc-audience must be set together")
	}
	return cfg, nil
}

func isLoopbackViewHost(host string) bool {
	if strings.EqualFold(strings.TrimSpace(host), "localhost") {
		return true
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	return ip != nil && ip.IsLoopback()
}

func resolveSessionPaths(pattern string) ([]string, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, fmt.Errorf("session path cannot be empty")
	}
	info, err := os.Stat(pattern) // #nosec G703 -- pattern is operator-supplied --sessions glob or directory from CLI
	if err == nil && info.IsDir() {
		var paths []string
		err := filepath.WalkDir(pattern, func(path string, d fs.DirEntry, err error) error { // #nosec G703 -- walks only under operator-supplied session directory
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(d.Name(), ".atb") {
				paths = append(paths, path)
			}
			return nil
		})
		sort.Strings(paths)
		return paths, err
	}

	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func sessionIndexPaths(bundlePath string, sessionPaths []string) []string {
	seen := make(map[string]struct{}, len(sessionPaths)+1)
	paths := make([]string, 0, len(sessionPaths)+1)
	for _, path := range append([]string{bundlePath}, sessionPaths...) {
		key, err := filepath.Abs(path)
		if err != nil {
			key = path
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		paths = append(paths, path)
	}
	return paths
}

func describeSessionSource(sessionPaths []string) string {
	if len(sessionPaths) == 0 {
		return ""
	}
	if len(sessionPaths) == 1 {
		info, err := os.Stat(sessionPaths[0])
		if err == nil && info.IsDir() {
			return sessionPaths[0]
		}
		return filepath.Dir(sessionPaths[0])
	}
	return fmt.Sprintf("%d bundle paths", len(sessionPaths))
}
