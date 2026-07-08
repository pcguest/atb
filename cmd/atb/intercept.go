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

const interceptLongDescription = `Start the local HTTPS capture proxy.

Records AI provider API traffic, tool calls, and failures into a live ATB
bundle. Completeness is bounded by what flows through the proxy.

Clients route traffic via HTTPS_PROXY plus trust of the local capture CA
(path printed on first run); provider base-URL overrides are not supported.

Usage:
  atb intercept [--port 8080] --bundle <path> [--target openai,anthropic] [--identity-map key=name]... [--mortise <endpoint>]
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

	p, err := proxy.NewProxy(cfg, nil, logger) // Pass nil for handler
	if err != nil {
		fmt.Fprintf(stderr, "atb intercept: %v\n", err)
		return exitUserError
	}

	port := extractPort(cfg.ListenAddr)
	printInterceptEnvHints(stdout, port)

	if cfg.MortiseEndpoint != "" {
		if cfg.MortiseToken != "" {
			_, source := mortiseTokenFromEnv()
			fmt.Fprintf(stdout, "Auto-push to Mortise: %s (Bearer auth from %s)\n", cfg.MortiseEndpoint, source)
		} else {
			fmt.Fprintf(stdout, "Auto-push to Mortise: %s (unauthenticated; set ATB_MORTISE_TOKEN for Bearer auth)\n", cfg.MortiseEndpoint)
		}
	} else {
		fmt.Fprintln(stdout, "Auto-push disabled (--mortise not set)")
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
	mortiseEndpoint := ""
	mortiseEndpointFlag := ""
	captureBodies := false
	setMortiseEndpoint := func(flag, value string) error {
		if mortiseEndpointFlag != "" {
			return fmt.Errorf("cannot combine %s with %s", mortiseEndpointFlag, flag)
		}
		mortiseEndpointFlag = flag
		mortiseEndpoint = strings.TrimSpace(value)
		return nil
	}

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
		case arg == "--mortise" || arg == "--custos":
			if i+1 >= len(args) {
				return proxy.ProxyConfig{}, fmt.Errorf("missing value for %s", arg)
			}
			if err := setMortiseEndpoint(arg, args[i+1]); err != nil {
				return proxy.ProxyConfig{}, err
			}
			i++
		case strings.HasPrefix(arg, "--mortise="):
			if err := setMortiseEndpoint("--mortise", strings.TrimPrefix(arg, "--mortise=")); err != nil {
				return proxy.ProxyConfig{}, err
			}
		case strings.HasPrefix(arg, "--custos="):
			if err := setMortiseEndpoint("--custos", strings.TrimPrefix(arg, "--custos=")); err != nil {
				return proxy.ProxyConfig{}, err
			}
		case arg == "--capture-bodies":
			captureBodies = true
		default:
			return proxy.ProxyConfig{}, fmt.Errorf("unknown argument %q", arg)
		}
	}

	if bundlePath == "" {
		return proxy.ProxyConfig{}, fmt.Errorf("--bundle is required")
	}

	mortiseToken, _ := mortiseTokenFromEnv()
	cfg := proxy.ProxyConfig{
		ListenAddr:      fmt.Sprintf("127.0.0.1:%d", port),
		BundlePath:      bundlePath,
		TargetHosts:     proxy.DefaultTargetHosts(targets...),
		IdentityMap:     identityMap,
		MortiseEndpoint: mortiseEndpoint,
		// The token comes from the environment, not a flag, so it never
		// lands in shell history or process listings.
		MortiseToken:  mortiseToken,
		CaptureBodies: captureBodies,
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
	caPath := "$HOME/.atb/ca.crt"
	if certPath, _, err := proxy.DefaultCAPaths(); err == nil {
		caPath = certPath
	}
	fmt.Fprintln(w, "Route provider traffic through the proxy (HTTPS forward proxy):")
	fmt.Fprintf(w, "  export HTTPS_PROXY=http://127.0.0.1:%d\n", port)
	fmt.Fprintf(w, "  export SSL_CERT_FILE=%s        # Python (httpx/requests)\n", caPath)
	fmt.Fprintf(w, "  export CURL_CA_BUNDLE=%s       # curl and compatible clients\n", caPath)
	fmt.Fprintf(w, "  export NODE_EXTRA_CA_CERTS=%s  # Node.js\n", caPath)
	fmt.Fprintln(w, "Provider base-URL path overrides are not supported; only hosts in --target are intercepted.")
	fmt.Fprintln(w, "Clients that ignore HTTPS_PROXY or use certificate pinning will bypass or reject interception; use an SDK wrapper for those calls.")
}

func printInterceptHelp(w io.Writer) {
	fmt.Fprint(w, interceptLongDescription)
	fmt.Fprint(w, `
Flags:
  --port <n>                 Loopback listen port (default 8080)
  --bundle <path>            Target ATB bundle path (required)
  --target <names>           Comma-separated provider shorthand or hostnames (default openai,anthropic)
  --identity-map key=name    Map API keys to display names (repeatable)
  --mortise <endpoint>       Mortise ingest endpoint for auto-push on session close
                             (set ATB_MORTISE_TOKEN to authenticate with a Bearer token)
  --custos <endpoint>        Deprecated compatibility alias for --mortise
  --capture-bodies           Record raw request/response bodies (default: digest only)

By default only a SHA-256 digest and byte length of each request/response body
are recorded, so the bundle never persists prompts, completions, or PII.
Pass --capture-bodies to retain raw bodies where that tradeoff is acceptable.
The generated CA is local to the current user. Configure trust only for the
captured process where possible; do not install it as a shared production root.
`)
}
