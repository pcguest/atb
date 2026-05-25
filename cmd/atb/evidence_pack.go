// SPDX-License-Identifier: MIT
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	evidencepack "github.com/pcguest/atb/internal/evidencepack"
)

var errEvidencePackHelp = errors.New("evidence pack help requested")

type evidencePackConfig struct {
	Output       string
	WorkspaceDir string
	Paths        []string
}

func runEvidencePack(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseEvidencePackArgs(args)
	if err != nil {
		if errors.Is(err, errEvidencePackHelp) {
			printEvidencePackUsage(stdout)
			return exitSuccess
		}
		fmt.Fprintf(stderr, "atb evidence pack: %v\n", err)
		printEvidencePackUsage(stderr)
		return exitUserError
	}
	paths := cfg.Paths
	if cfg.WorkspaceDir != "" {
		discovered, err := evidencepack.DiscoverWorkspaceBundles(cfg.WorkspaceDir)
		if err != nil {
			fmt.Fprintf(stderr, "atb evidence pack: discover workspace bundles: %v\n", err)
			return exitSystemError
		}
		paths = discovered
	} else if len(cfg.Paths) == 0 {
		fmt.Fprintln(stderr, "atb evidence pack: at least one bundle path is required")
		printEvidencePackUsage(stderr)
		return exitUserError
	}

	pack, anyErrors, userError := evidencepack.PackPaths(context.Background(), paths)
	switch cfg.Output {
	case "json":
		data, err := json.MarshalIndent(pack, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "atb evidence pack: encode json output: %v\n", err)
			return exitSystemError
		}
		data = append(data, '\n')
		if _, err := stdout.Write(data); err != nil {
			fmt.Fprintf(stderr, "atb evidence pack: write output: %v\n", err)
			return exitSystemError
		}
	case "markdown", "md":
		if err := evidencepack.RenderMarkdown(pack, stdout, time.Now().UTC(), version); err != nil {
			fmt.Fprintf(stderr, "atb evidence pack: render markdown output: %v\n", err)
			return exitSystemError
		}
	default:
		fmt.Fprintf(stderr, "atb evidence pack: unsupported output %q\n", cfg.Output)
		return exitUserError
	}
	if anyErrors {
		if userError {
			return exitUserError
		}
		return exitSystemError
	}
	return exitSuccess
}

func parseEvidencePackArgs(args []string) (evidencePackConfig, error) {
	cfg := evidencePackConfig{Output: "json"}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errEvidencePackHelp
		case arg == "--output":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --output (expected json or markdown)")
			}
			i++
			cfg.Output = strings.ToLower(strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "--output="):
			cfg.Output = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--output=")))
		case arg == "--workspace":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --workspace")
			}
			i++
			cfg.WorkspaceDir = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--workspace="):
			cfg.WorkspaceDir = strings.TrimSpace(strings.TrimPrefix(arg, "--workspace="))
		case strings.HasPrefix(arg, "-"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			cfg.Paths = append(cfg.Paths, normalizeBundlePath(arg))
		}
	}

	if cfg.Output != "json" && cfg.Output != "markdown" && cfg.Output != "md" {
		return cfg, fmt.Errorf("unsupported output %q (expected json or markdown)", cfg.Output)
	}
	if cfg.WorkspaceDir != "" && len(cfg.Paths) > 0 {
		return cfg, fmt.Errorf("--workspace cannot be used with explicit bundle paths")
	}
	return cfg, nil
}

func printEvidencePackUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb evidence pack [--output json|markdown] [--workspace <dir>] <bundle.atb> [<bundle2.atb> ...]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Verify one or more local bundles and emit a combined evidence summary.")
	fmt.Fprintln(w, "Use --output json (default) or --output markdown.")
	fmt.Fprintln(w, "Use --workspace <dir> to scan an Agent-style workspace for sessions/*/bundle.atb (mutually exclusive with explicit bundle paths).")
	fmt.Fprintln(w, "Profile selection follows the implicit all-applicable verify behaviour.")
}
