// SPDX-License-Identifier: MIT
// events.go implements the "atb events" sub-command, which prints the
// canonical ATB event catalogue.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pcguest/atb/internal/event"
)

var errEventsHelp = errors.New("events help requested")

type eventsConfig struct {
	JSON    bool
	Profile string
}

func cmdEvents() {
	os.Exit(runEvents(os.Args[2:], os.Stdout, os.Stderr))
}

func runEvents(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseEventsArgs(args)
	if err != nil {
		if errors.Is(err, errEventsHelp) {
			printEventsUsage(stdout)
			return exitSuccess
		}
		fmt.Fprintf(stderr, "atb events: %v\n", err)
		printEventsUsage(stderr)
		return exitUserError
	}

	items := filterEventRegistry(cfg.Profile)
	if cfg.JSON {
		if err := json.NewEncoder(stdout).Encode(items); err != nil {
			fmt.Fprintf(stderr, "atb events: encode json output: %v\n", err)
			return exitSystemError
		}
		return exitSuccess
	}

	renderEventsText(stdout, items)
	return exitSuccess
}

func parseEventsArgs(args []string) (eventsConfig, error) {
	cfg := eventsConfig{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errEventsHelp
		case arg == "--json":
			cfg.JSON = true
		case arg == "--profile" || arg == "-p":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for %s", arg)
			}
			i++
			cfg.Profile = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--profile="):
			cfg.Profile = strings.TrimSpace(strings.TrimPrefix(arg, "--profile="))
		case strings.HasPrefix(arg, "-"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			return cfg, fmt.Errorf("unexpected argument %q", arg)
		}
	}
	return cfg, nil
}

func printEventsUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb events [--json] [--profile <id>]")
}

func filterEventRegistry(profileID string) []event.EventInfo {
	if strings.TrimSpace(profileID) == "" {
		return append([]event.EventInfo(nil), event.RegistryGenerated...)
	}

	items := make([]event.EventInfo, 0, len(event.RegistryGenerated))
	for _, item := range event.RegistryGenerated {
		if eventInfoMatchesProfile(item, profileID) {
			items = append(items, item)
		}
	}
	return items
}

func eventInfoMatchesProfile(item event.EventInfo, profileID string) bool {
	for _, candidate := range strings.Split(item.Profile, ",") {
		if strings.TrimSpace(candidate) == profileID {
			return true
		}
	}
	return false
}

func renderEventsText(w io.Writer, items []event.EventInfo) {
	fmt.Fprintln(w, "ATB Canonical Event Types")
	fmt.Fprintln(w, "=========================")

	currentSection := ""
	for _, item := range items {
		section := eventSectionTitle(item.Type)
		if section != currentSection {
			if currentSection != "" {
				fmt.Fprintln(w)
			}
			fmt.Fprintln(w, section)
			currentSection = section
		}
		fmt.Fprintf(w, "  %-22s [%-13s] %s\n", item.Type, item.Criticality, item.Description)
	}
}

func eventSectionTitle(eventType string) string {
	switch {
	case strings.HasPrefix(eventType, "atb."):
		return "Bundle lifecycle"
	case strings.HasPrefix(eventType, "ai.request.") || strings.HasPrefix(eventType, "ai.response."):
		return "AI request / response"
	case strings.HasPrefix(eventType, "ai.policy."):
		return "Policy"
	case strings.HasPrefix(eventType, "ai.retrieval.") || strings.HasPrefix(eventType, "ai.model."):
		return "RAG (Retrieval-Augmented Generation)"
	case strings.HasPrefix(eventType, "ai.action."):
		return "Privileged actions (ACP-gated)"
	case strings.HasPrefix(eventType, "ai.human.") || strings.HasPrefix(eventType, "ai.override."):
		return "Human oversight"
	case strings.HasPrefix(eventType, "ai.job."):
		return "Background automation"
	case strings.HasPrefix(eventType, "data."):
		return "Data export"
	case strings.HasPrefix(eventType, "dev.") || strings.HasPrefix(eventType, "snapshot."):
		return "Developer / tooling"
	default:
		return "Other"
	}
}
