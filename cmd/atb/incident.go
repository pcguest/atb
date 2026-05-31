// SPDX-License-Identifier: MIT
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pcguest/atb/internal/incident"
)

func cmdIncident() {
	os.Exit(runIncident(os.Args[2:], os.Stdout, os.Stderr))
}

func runIncident(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printIncidentUsage(stderr)
		return exitUserError
	}
	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "report":
		return runIncidentReport(args[1:], stdout, stderr)
	case "list":
		return runIncidentList(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		printIncidentUsage(stdout)
		return exitSuccess
	default:
		fmt.Fprintf(stderr, "atb incident: unknown subcommand %q\n", sub)
		printIncidentUsage(stderr)
		return exitUserError
	}
}

func runIncidentReport(args []string, stdout, stderr io.Writer) int {
	bundlePath := ""
	sessionID := ""
	format := "markdown"

	for i := 0; i < len(args); i++ {
		arg := args[i]
		next := func() (string, bool) {
			if i+1 >= len(args) {
				return "", false
			}
			i++
			return args[i], true
		}
		switch {
		case arg == "-h", arg == "--help":
			printIncidentUsage(stdout)
			return exitSuccess
		case arg == "--bundle":
			v, ok := next()
			if !ok {
				fmt.Fprintln(stderr, "atb incident report: missing value for --bundle")
				return exitUserError
			}
			bundlePath = strings.TrimSpace(v)
		case strings.HasPrefix(arg, "--bundle="):
			bundlePath = strings.TrimSpace(strings.TrimPrefix(arg, "--bundle="))
		case arg == "--session":
			v, ok := next()
			if !ok {
				fmt.Fprintln(stderr, "atb incident report: missing value for --session")
				return exitUserError
			}
			sessionID = strings.TrimSpace(v)
		case strings.HasPrefix(arg, "--session="):
			sessionID = strings.TrimSpace(strings.TrimPrefix(arg, "--session="))
		case arg == "--format":
			v, ok := next()
			if !ok {
				fmt.Fprintln(stderr, "atb incident report: missing value for --format")
				return exitUserError
			}
			format = strings.ToLower(strings.TrimSpace(v))
		case strings.HasPrefix(arg, "--format="):
			format = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--format=")))
		default:
			fmt.Fprintf(stderr, "atb incident report: unknown argument %q\n", arg)
			return exitUserError
		}
	}

	if bundlePath == "" {
		fmt.Fprintln(stderr, "atb incident report: --bundle is required")
		return exitUserError
	}
	if sessionID == "" {
		fmt.Fprintln(stderr, "atb incident report: --session is required")
		return exitUserError
	}
	if format != "markdown" && format != "json" {
		fmt.Fprintf(stderr, "atb incident report: invalid --format %q (expected markdown|json)\n", format)
		return exitUserError
	}

	report, err := incident.Build(context.Background(), bundlePath, sessionID)
	if err != nil {
		fmt.Fprintf(stderr, "atb incident report: %v\n", err)
		return exitSystemError
	}

	switch format {
	case "json":
		out, err := report.JSON()
		if err != nil {
			fmt.Fprintf(stderr, "atb incident report: render json: %v\n", err)
			return exitSystemError
		}
		fmt.Fprintln(stdout, string(out))
	default:
		fmt.Fprint(stdout, report.Markdown())
	}

	if !report.Found {
		fmt.Fprintf(stderr, "atb incident report: no events found for session %q\n", sessionID)
		return exitUserError
	}
	return exitSuccess
}

func runIncidentList(args []string, stdout, stderr io.Writer) int {
	bundlePath := ""
	format := "markdown"
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h", arg == "--help":
			printIncidentUsage(stdout)
			return exitSuccess
		case arg == "--bundle":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "atb incident list: missing value for --bundle")
				return exitUserError
			}
			i++
			bundlePath = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--bundle="):
			bundlePath = strings.TrimSpace(strings.TrimPrefix(arg, "--bundle="))
		case arg == "--format":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "atb incident list: missing value for --format")
				return exitUserError
			}
			i++
			format = strings.ToLower(strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "--format="):
			format = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--format=")))
		default:
			fmt.Fprintf(stderr, "atb incident list: unknown argument %q\n", arg)
			return exitUserError
		}
	}
	if bundlePath == "" {
		fmt.Fprintln(stderr, "atb incident list: --bundle is required")
		return exitUserError
	}
	if format != "markdown" && format != "json" {
		fmt.Fprintf(stderr, "atb incident list: invalid --format %q (expected markdown|json)\n", format)
		return exitUserError
	}

	entries, err := incident.ListSessions(context.Background(), bundlePath)
	if err != nil {
		fmt.Fprintf(stderr, "atb incident list: %v\n", err)
		return exitSystemError
	}
	switch format {
	case "json":
		out, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "atb incident list: render json: %v\n", err)
			return exitSystemError
		}
		fmt.Fprintln(stdout, string(out))
	default:
		fmt.Fprint(stdout, incident.SessionListMarkdown(bundlePath, entries))
	}
	return exitSuccess
}

func printIncidentUsage(w io.Writer) {
	fmt.Fprint(w, `atb incident list   --bundle <path> [--format markdown|json]
atb incident report --bundle <path> --session <id> [--format markdown|json]

Discover and review agent sessions captured in an ATB bundle.

The full signed bundle remains the authoritative, tamper-evident evidence; a
report scopes one session for review and lists each event with its sequence and
record hash so it is verifiable against that bundle.

list flags:
  --bundle <path>            Bundle to read (required)
  --format markdown|json     Output format (default markdown)

report flags:
  --bundle <path>            Bundle to read (required)
  --session <id>             Session identifier to report on (required)
  --format markdown|json     Output format (default markdown)
`)
}
