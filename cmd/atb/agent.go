// SPDX-License-Identifier: MIT
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pcguest/atb/internal/agent"
)

const agentLongDescription = `ATB Agent commands.

The agent is a long-running local service for workspace management,
capture sessions, and (future) viewer and MCP coordination.

Usage:
  atb agent run
`

func cmdAgent() {
	os.Exit(runAgentCommand(os.Args[2:], os.Stdout, os.Stderr))
}

func runAgentCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printAgentHelp(stdout)
		return exitUserError
	}

	switch args[0] {
	case "run":
		return runAgentRun(args[1:], stderr)
	case "-h", "--help", "help":
		printAgentHelp(stdout)
		return exitSuccess
	default:
		fmt.Fprintf(stderr, "atb agent: unknown sub-command %q\n", args[0])
		printAgentHelp(stderr)
		return exitUserError
	}
}

func runAgentRun(args []string, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "-h", "--help", "help":
			printAgentRunHelp(stderr)
			return exitSuccess
		default:
			fmt.Fprintf(stderr, "atb agent run: unknown argument %q\n", args[0])
			return exitUserError
		}
	}

	cfg, err := agent.LoadConfig(version)
	if err != nil {
		fmt.Fprintf(stderr, "atb agent run: %v\n", err)
		return exitSystemError
	}

	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := agent.Run(ctx, cfg, logger); err != nil {
		fmt.Fprintf(stderr, "atb agent run: %v\n", err)
		return exitSystemError
	}
	return exitSuccess
}

func printAgentHelp(w io.Writer) {
	fmt.Fprint(w, agentLongDescription)
}

func printAgentRunHelp(w io.Writer) {
	fmt.Fprint(w, `Usage: atb agent run

Environment:
  ATB_AGENT_LISTEN_ADDR  Loopback listen address (default 127.0.0.1:6180)
  ATB_AGENT_DATA_DIR     Workspace data directory (default ~/.atb/agent, or ./data/agent when HOME is unset)

Config file (optional, lower priority than environment):
  ~/.atb/config.json or ./.atb/config.json with an "agent" object:
    listen_addr, data_dir
`)
}
