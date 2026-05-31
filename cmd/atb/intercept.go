// SPDX-License-Identifier: MIT
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/pcguest/atb/internal/proxy"
)

const interceptLongDescription = `Start a local HTTPS capture proxy scaffold.

Usage:
  atb intercept [--port 8080] --bundle <path> [--target openai,anthropic] [--identity-map key=name]... [--custos <endpoint>]
`

func cmdIntercept() {
	os.Exit(runInterceptCommand(os.Args[2:], os.Stdout, os.Stderr))
}

func runInterceptCommand(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseInterceptArgs(args)
	if err != nil {
		if errors.Is(err, errInterceptHelp) {
			printInterceptHelp(stdout)
			return exitSuccess
		}
		fmt.Fprintf(stderr, "atb intercept: %v\n", err)
		printInterceptHelp(stderr)
		return exitUserError
	}

	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	p, err := proxy.NewProxy(cfg, nil, logger)
	if err != nil {
		fmt.Fprintf(stderr, "atb intercept: %v\n", err)
		return exitUserError
	}

	port := extractPort(cfg.ListenAddr)
	printInterceptEnvHints(stdout, port)

	if cfg.CustosEndpoint != "" {
		fmt.Fprintf(stdout, "Auto-push to Custos: %s\n", cfg.CustosEndpoint)
	} else {
		fmt.Fprintln(stdout, "Auto-push disabled (--custos not set)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := p.Start(ctx); err != nil {
		fmt.Fprintf(stderr, "atb intercept: %v\n", err)
		return exitSystemError
	}
	<-ctx.Done()
	if err := p.Stop(); err != nil {
		fmt.Fprintf(stderr, "atb intercept: %v\n", err)
		return exitSystemError
	}
	return exitSuccess
}

var errInterceptHelp = errors.New("intercept help requested")

func parseInterceptArgs(args []string) (proxy.ProxyConfig, error) {
	port := 8080
	bundlePath := ""
	targets := []string{"openai", "anthropic"}
	identityMap := map[string]string{}
	custosEndpoint := "" // New field
	captureBodies := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h", arg == "--help", arg == "help":
			return proxy.ProxyConfig{}, errInterceptHelp
		case arg == "--port":
			if i+1 >= len(args) {
				return proxy.ProxyConfig{}, fmt.Errorf("missing value for --port")
			}
			parsed, err := strconv.Atoi(strings.TrimSpace(args[i+1]))
			if err != nil || parsed <= 0 || parsed > 65535 {
				return proxy.ProxyConfig{}, fmt.Errorf("invalid --port value %q", args[i+1])
			}
			port = parsed
			i++
		case strings.HasPrefix(arg, "--port="):
			value := strings.TrimPrefix(arg, "--port=")
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || parsed <= 0 || parsed > 65535 {
				return proxy.ProxyConfig{}, fmt.Errorf("invalid --port value %q", value)
			}
			port = parsed
		case arg == "--bundle":
			if i+1 >= len(args) {
				return proxy.ProxyConfig{}, fmt.Errorf("missing value for --bundle")
			}
			bundlePath = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(arg, "--bundle="):
			bundlePath = strings.TrimSpace(strings.TrimPrefix(arg, "--bundle="))
		case arg == "--target":
			if i+1 >= len(args) {
				return proxy.ProxyConfig{}, fmt.Errorf("missing value for --target")
			}
			targets = splitCSV(args[i+1])
			i++
		case strings.HasPrefix(arg, "--target="):
			targets = splitCSV(strings.TrimPrefix(arg, "--target="))
		case arg == "--identity-map":
			if i+1 >= len(args) {
				return proxy.ProxyConfig{}, fmt.Errorf("missing value for --identity-map")
			}
			key, name, err := parseIdentityPair(args[i+1])
			if err != nil {
				return proxy.ProxyConfig{}, err
			}
			identityMap[key] = name
			i++
		case strings.HasPrefix(arg, "--identity-map="):
			key, name, err := parseIdentityPair(strings.TrimPrefix(arg, "--identity-map="))
			if err != nil {
				return proxy.ProxyConfig{}, err
			}
			identityMap[key] = name
		case arg == "--custos": // New case for --custos flag
			if i+1 >= len(args) {
				return proxy.ProxyConfig{}, fmt.Errorf("missing value for --custos")
			}
			custosEndpoint = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(arg, "--custos="): // New case for --custos=value
			custosEndpoint = strings.TrimSpace(strings.TrimPrefix(arg, "--custos="))
		case arg == "--capture-bodies":
			captureBodies = true
		default:
			return proxy.ProxyConfig{}, fmt.Errorf("unknown argument %q", arg)
		}
	}

	if bundlePath == "" {
		return proxy.ProxyConfig{}, fmt.Errorf("--bundle is required")
	}

	cfg := proxy.ProxyConfig{
		ListenAddr:     fmt.Sprintf("127.0.0.1:%d", port),
		BundlePath:     bundlePath,
		TargetHosts:    proxy.DefaultTargetHosts(targets...),
		IdentityMap:    identityMap,
		CustosEndpoint: custosEndpoint,
		CaptureBodies:  captureBodies,
	}
	return cfg, cfg.Validate()
}

func parseIdentityPair(raw string) (string, string, error) {
	key, name, ok := strings.Cut(raw, "=")
	key = strings.TrimSpace(key)
	name = strings.TrimSpace(name)
	if !ok || key == "" || name == "" {
		return "", "", fmt.Errorf("invalid --identity-map %q (expected key=name)", raw)
	}
	return key, name, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func extractPort(listenAddr string) int {
	_, portStr, ok := strings.Cut(listenAddr, ":")
	if !ok {
		return 8080
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 8080
	}
	return port
}

func printInterceptEnvHints(w io.Writer, port int) {
	fmt.Fprintf(w, "export OPENAI_BASE_URL=http://localhost:%d/openai\n", port)
	fmt.Fprintf(w, "export ANTHROPIC_BASE_URL=http://localhost:%d/anthropic\n", port)
}

func printInterceptHelp(w io.Writer) {
	fmt.Fprint(w, interceptLongDescription)
	fmt.Fprint(w, `
Flags:
  --port <n>                 Loopback listen port (default 8080)
  --bundle <path>            Target ATB bundle path (required)
  --target <names>           Comma-separated provider shorthand or hostnames (default openai,anthropic)
  --identity-map key=name    Map API keys to display names (repeatable)
  --custos <endpoint>        Custos ingest endpoint for auto-push on session close
  --capture-bodies           Record raw request/response bodies (default: digest only)

By default only a SHA-256 digest and byte length of each request/response body
are recorded, so the bundle never persists prompts, completions, or PII.
Pass --capture-bodies to retain raw bodies where that tradeoff is acceptable.
`)
}
