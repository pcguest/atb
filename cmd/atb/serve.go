package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/pcguest/atb/internal/mcp"
)

const serveLongDescription = `Starts ATB as a Model Context Protocol server.
Communicates over stdin/stdout using JSON-RPC 2.0.
Configure in Claude Desktop's claude_desktop_config.json:

  {
    "mcpServers": {
      "atb": {
        "command": "atb",
        "args": ["serve"]
      }
    }
  }
`

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

	srv := mcp.New(version, os.Stdin, os.Stdout)
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
