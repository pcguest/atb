package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/trust"
)

var errTrustReportHelp = errors.New("trust-report help requested")

type trustReportConfig struct {
	BundlePath string
	Format     string
}

func cmdTrustReport() {
	cfg, err := parseTrustReportArgs(os.Args[2:])
	if err != nil {
		if errors.Is(err, errTrustReportHelp) {
			printTrustReportUsage()
			return
		}
		fmt.Fprintf(os.Stderr, "atb trust-report: %v\n", err)
		printTrustReportUsage()
		os.Exit(exitUserError)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb trust-report: current directory: %v\n", err)
		os.Exit(exitSystemError)
	}

	report := trust.BuildReport(cwd, cfg.BundlePath)
	switch cfg.Format {
	case "json":
		if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "atb trust-report: encode json: %v\n", err)
			os.Exit(exitSystemError)
		}
	case "markdown":
		printTrustReportMarkdown(report)
	default:
		fmt.Fprintf(os.Stderr, "atb trust-report: unsupported format %q\n", cfg.Format)
		os.Exit(exitUserError)
	}
}

func parseTrustReportArgs(args []string) (trustReportConfig, error) {
	cfg := trustReportConfig{
		BundlePath: bundle.DefaultPath(),
		Format:     "markdown",
	}
	pathSet := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errTrustReportHelp
		case arg == "--format":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --format (expected markdown|json)")
			}
			cfg.Format = strings.ToLower(strings.TrimSpace(args[i+1]))
			i++
		case strings.HasPrefix(arg, "--format="):
			cfg.Format = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--format=")))
		case strings.HasPrefix(arg, "--"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			if pathSet {
				return cfg, fmt.Errorf("expected at most one bundle path")
			}
			cfg.BundlePath = normalizeBundlePath(arg)
			pathSet = true
		}
	}
	if cfg.Format != "markdown" && cfg.Format != "json" {
		return cfg, fmt.Errorf("invalid format %q (expected markdown|json)", cfg.Format)
	}
	return cfg, nil
}

func printTrustReportUsage() {
	fmt.Println("Usage: atb trust-report [bundle_path] [--format markdown|json]")
}

func printTrustReportMarkdown(report trust.Report) {
	fmt.Println("# ATB Trust Report")
	fmt.Println()
	fmt.Printf("- Verdict: **%s**\n", strings.ToUpper(report.Status))
	fmt.Printf("- Generated: %s\n", report.GeneratedAt)
	fmt.Printf("- Bundle: `%s`\n", report.BundlePath)
	fmt.Printf("- Chain length: %d\n", report.ChainLength)
	if report.HeadHash != "" {
		fmt.Printf("- Head hash: `%s`\n", report.HeadHash)
	}
	fmt.Println()
	for _, category := range report.Categories {
		fmt.Printf("## %s (%s)\n\n", category.Title, strings.ToUpper(category.Status))
		for _, check := range category.Checks {
			fmt.Printf("- [%s] %s: %s\n", strings.ToUpper(check.Status), check.Title, check.Details)
			if len(check.Evidence) > 0 {
				fmt.Printf("  Evidence: `%s`\n", relativeOrOriginal(report.BundlePath, check.Evidence[0]))
			}
		}
		fmt.Println()
	}
}

func relativeOrOriginal(bundlePath string, candidate string) string {
	if !filepath.IsAbs(candidate) {
		return candidate
	}
	base := filepath.Dir(bundlePath)
	rel, err := filepath.Rel(base, candidate)
	if err != nil {
		return candidate
	}
	return rel
}
