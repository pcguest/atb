package main

import (
	"context"
	"errors"
	"fmt"
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

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/viewer"
)

var errViewHelp = errors.New("view help requested")

type viewConfig struct {
	BundlePath string
	Port       int
	PortSet    bool
	NoOpen     bool
	LogReveals bool
}

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

	handler, page, tamperDetected, openPath, err := buildViewServer(bundlePath, cfg.LogReveals)
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb view: %v\n", err)
		os.Exit(classifyBundleLoadError(err))
	}

	ln, port, err := listenViewPort(cfg.Port, cfg.PortSet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb view: %v\n", err)
		os.Exit(exitSystemError)
	}
	defer ln.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
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

func buildViewServer(bundlePath string, logReveals bool) (http.Handler, viewer.PageData, bool, string, error) {
	b, err := bundle.Load(bundlePath)
	if err != nil {
		return nil, viewer.PageData{}, false, "/", fmt.Errorf("load bundle %s: %w", bundlePath, err)
	}
	verifyErr := b.Verify()

	page := viewer.BuildPageData(b, bundlePath)
	mux := http.NewServeMux()
	api := viewer.NewAPIServer(viewer.APIConfig{
		BundlePath: bundlePath,
		Bundle:     b,
		VerifyErr:  verifyErr,
		LogReveals: logReveals,
	})
	api.Register(mux)
	openPath := "/"
	if dashboardDir, ok := findDashboardOutDir(); ok {
		openPath = "/view/"
		static := http.FileServer(http.Dir(dashboardDir))
		mux.Handle("/_next/", static)
		mux.Handle("/view/", static)
		mux.HandleFunc("/view", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/view/", http.StatusTemporaryRedirect)
		})
	}

	if verifyErr != nil {
		mux.Handle("/", viewer.NewTamperHandler(bundlePath, verifyErr))
		return mux, page, true, openPath, nil
	}

	mux.Handle("/", viewer.NewHandler(page))
	return mux, page, false, openPath, nil
}

func printViewUsage() {
	fmt.Println("Usage: atb view [bundle_path] [--bundle path/to/file.atb] [--port 8080] [--no-open] [--log-reveals]")
}

func parseViewArgs(args []string) (viewConfig, error) {
	cfg := viewConfig{Port: 8080}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errViewHelp
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

func listenViewPort(startPort int, explicit bool) (net.Listener, int, error) {
	for _, port := range candidateViewPorts(startPort, explicit) {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
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

func findDashboardOutDir() (string, bool) {
	candidates := []string{
		filepath.Join("web", "out"),
		filepath.Join("..", "web", "out"),
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			continue
		}
		indexPath := filepath.Join(candidate, "view", "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			return candidate, true
		}
	}
	return "", false
}
