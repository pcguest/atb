// SPDX-License-Identifier: MIT
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/mcp"
)

const serveLongDescription = `Starts ATB as a Model Context Protocol server.
Communicates over stdin/stdout using JSON-RPC 2.0.
Configure in Claude Desktop's claude_desktop_config.json:

  {
    "mcpServers": {
      "atb": {
        "command": "atb",
        "args": ["mcp", "serve"]
      }
    }
  }
`

const mcpLongDescription = `ATB Model Context Protocol commands.

Usage:
  atb mcp serve
`

func cmdMCP() {
	os.Exit(runMCPCommand(os.Args[2:], os.Stdout, os.Stderr))
}

func runMCPCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printMCPHelp(stderr)
		return exitUserError
	}

	switch args[0] {
	case "serve":
		return runServeCommand(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		printMCPHelp(stdout)
		return exitSuccess
	default:
		fmt.Fprintf(stderr, "atb mcp: unknown sub-command %q\n", args[0])
		printMCPHelp(stderr)
		return exitUserError
	}
}

func cmdServe() {
	os.Exit(runServeCommand(os.Args[2:], os.Stdout, os.Stderr))
}

func runServeCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "-h", "--help", "help":
			printServeHelp(stdout)
			return exitSuccess
		default:
			fmt.Fprintf(stderr, "atb serve: unknown argument %q\n", args[0])
			return exitUserError
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := mcp.NewWithHandlers(version, os.Stdin, stdout, mcp.ToolHandlers{
		Verify: mcpVerifyHandler,
		Init:   mcpInitHandler,
		Append: mcpAppendHandler,
	})
	if err := srv.Serve(ctx); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(stderr, "atb serve: %v\n", err)
		return exitSystemError
	}

	return exitSuccess
}

func printServeHelp(w io.Writer) {
	fmt.Fprintln(w, "Start ATB as an MCP server (stdio transport)")
	fmt.Fprintln(w)
	fmt.Fprint(w, serveLongDescription)
}

func printMCPHelp(w io.Writer) {
	fmt.Fprintln(w, "ATB Model Context Protocol commands")
	fmt.Fprintln(w)
	fmt.Fprint(w, mcpLongDescription)
}

func mcpVerifyHandler(_ context.Context, input mcp.VerifyInput, stdout, stderr io.Writer) int {
	cfg := verifyCLIConfig{
		BundlePath:   bundle.DefaultPath(),
		ProfileID:    input.Profile,
		LegacyFormat: formatJSON,
		Quiet:        input.Quiet,
		Trace:        input.Trace,
		WithAnchor:   input.WithAnchor,
	}
	if input.Path != "" {
		cfg.BundlePath = input.Path
	}
	return runVerifyWithConfig(cfg, false, stdout, stderr)
}

func mcpInitHandler(_ context.Context, stdout, stderr io.Writer) int {
	return runInitWithOptions(initRunOptions{
		OutputFormat: formatJSON,
	}, stdout, stderr)
}

func mcpAppendHandler(_ context.Context, eventType string, data map[string]any) (bundle.Record, error) {
	return appendToDefaultBundle(eventType, data, false)
}
