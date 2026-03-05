package main

import (
	"context"
	"errors"
	"fmt"
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

	handler, page, err := buildViewHandler(bundlePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb view: %v\n", err)
		os.Exit(classifyBundleLoadError(err))
	}

	addr := fmt.Sprintf("127.0.0.1:%d", cfg.Port)
	url := "http://" + addr
	srv := &http.Server{Addr: addr, Handler: handler}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	fmt.Printf("✓ Serving %s (%d events) at %s\n", bundlePath, len(page.Events), url)
	if err := openBrowser(url); err != nil {
		fmt.Fprintf(os.Stderr, "atb view: could not auto-open browser: %v\n", err)
	}

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "atb view: server error: %v\n", err)
		os.Exit(exitSystemError)
	}

	fmt.Println("atb view: stopped")
}

func buildViewHandler(bundlePath string) (http.Handler, viewer.PageData, error) {
	b, err := bundle.Load(bundlePath)
	if err != nil {
		return nil, viewer.PageData{}, fmt.Errorf("load bundle %s: %w", bundlePath, err)
	}
	page := viewer.BuildPageData(b, bundlePath)
	return viewer.NewHandler(page), page, nil
}

func printViewUsage() {
	fmt.Println("Usage: atb view [bundle_path] [--port 8080]")
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
		case strings.HasPrefix(arg, "--port="):
			raw := strings.TrimPrefix(arg, "--port=")
			p, err := strconv.Atoi(raw)
			if err != nil {
				return cfg, fmt.Errorf("invalid --port value %q", raw)
			}
			cfg.Port = p
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
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
