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
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	atbembed "github.com/pcguest/atb"
	"github.com/pcguest/atb/internal/bundle"
	apiv1 "github.com/pcguest/atb/pkg/api/v1"
)

var errViewHelp = errors.New("view help requested")

type viewConfig struct {
	BundlePath     string
	Host           string
	Port           int
	PortSet        bool
	NoOpen         bool
	LogReveals     bool
	UIExperimental bool
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

	handler, page, tamperDetected, openPath, err := buildViewServer(bundlePath, cfg.LogReveals, cfg.UIExperimental)
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
		fmt.Printf("✓ Serving %s (%d events) at %s\n", bundlePath, len(page.Events), url)
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

func buildViewServer(bundlePath string, logReveals bool, uiExperimental bool) (http.Handler, legacyPageData, bool, string, error) {
	_ = logReveals // Privacy reveal auditing is always on; flag retained for CLI compatibility.

	b, err := bundle.Load(bundlePath)
	if err != nil {
		return nil, legacyPageData{}, false, "/", fmt.Errorf("load bundle %s: %w", bundlePath, err)
	}
	verifyErr := b.Verify()
	revealAuthToken, err := generateRevealAuthToken()
	if err != nil {
		return nil, legacyPageData{}, false, "/", fmt.Errorf("generate privacy reveal auth token: %w", err)
	}

	page := buildLegacyPageData(b, bundlePath)
	mux := http.NewServeMux()
	api := apiv1.NewAPIServer(apiv1.APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		VerifyErr:       verifyErr,
		RevealAuthToken: revealAuthToken,
	})
	api.Register(mux)
	if verifyErr != nil {
		mux.Handle("/", apiv1.NewTamperHandler(bundlePath, verifyErr))
		return withSecurityHeaders(mux, revealAuthToken), page, true, "/", nil
	}

	openPath := "/"
	if uiExperimental {
		// The experimental dashboard and JSON API share one listener.
		// Keep this listener loopback-only or same-origin. There is no
		// separate cross-origin browser policy for exposing it remotely.
		if dashboardFS, ok := embeddedDashboardFS(); ok {
			openPath = "/view/"
			static := http.FileServer(dashboardFS)
			mux.Handle("/_next/", static)
			mux.Handle("/view/", static)
			mux.HandleFunc("/view", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/view/", http.StatusTemporaryRedirect)
			})
		}
	}

	mux.Handle("/", newLegacyViewHandler(page))
	return withSecurityHeaders(mux, revealAuthToken), page, false, openPath, nil
}

func printViewUsage() {
	fmt.Println("Usage: atb view [bundle_path] [--bundle path/to/file.atb] [--host 127.0.0.1] [--port 8080] [--no-open] [--log-reveals] [--ui-experimental]")
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
		case arg == "--ui-experimental":
			cfg.UIExperimental = true
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
	return cfg, nil
}

func resolveBundlePath(raw string) (string, error) {
	if raw == "" {
		return bundle.DefaultPath(), nil
	}

	info, err := os.Stat(raw)
	if err == nil {
		if info.IsDir() {
			return filepath.Join(raw, bundle.BundleFile), nil
		}
		return raw, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	if strings.HasSuffix(raw, string(os.PathSeparator)) {
		return filepath.Join(raw, bundle.BundleFile), nil
	}
	return raw, nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url) // #nosec G204 -- url is internally constructed http://localhost link, not user input
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url) // #nosec G204 -- url is internally constructed http://localhost link, not user input
	default:
		cmd = exec.Command("xdg-open", url) // #nosec G204 -- url is internally constructed http://localhost link, not user input
	}
	return cmd.Start()
}

func listenViewPort(host string, startPort int, explicit bool) (net.Listener, int, error) {
	if strings.TrimSpace(host) == "" {
		host = defaultViewHost
	}
	for _, port := range candidateViewPorts(startPort, explicit) {
		addr := net.JoinHostPort(host, strconv.Itoa(port))
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			return ln, port, nil
		}
		if !isAddrInUseError(err) {
			return nil, 0, fmt.Errorf("listen %s: %w", addr, err)
		}
	}
	if explicit {
		return nil, 0, fmt.Errorf("port %d is already in use", startPort)
	}
	return nil, 0, fmt.Errorf("ports %d-%d are already in use", startPort, startPort+2)
}

func candidateViewPorts(startPort int, explicit bool) []int {
	if explicit {
		return []int{startPort}
	}

	ports := make([]int, 0, 3)
	for i := 0; i < 3; i++ {
		p := startPort + i
		if p > 65535 {
			break
		}
		ports = append(ports, p)
	}
	if len(ports) == 0 {
		return []int{startPort}
	}
	return ports
}

func isAddrInUseError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.EADDRINUSE) {
		return true
	}

	var errno syscall.Errno
	if errors.As(err, &errno) {
		// 10048 is WSAEADDRINUSE on Windows.
		if errno == syscall.Errno(10048) {
			return true
		}
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "only one usage of each socket address")
}

func embeddedDashboardFS() (http.FileSystem, bool) {
	sub, err := fs.Sub(atbembed.WebOutFS, "web/out")
	if err != nil {
		return nil, false
	}
	f, err := sub.Open(filepath.ToSlash(filepath.Join("view", "index.html")))
	if err != nil {
		return nil, false
	}
	_ = f.Close()
	return http.FS(sub), true
}

func generateRevealAuthToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
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
			http.SetCookie(w, &http.Cookie{
				Name:     "atb_reveal_token",
				Value:    revealAuthToken,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
		}

		next.ServeHTTP(w, r)
	})
}
