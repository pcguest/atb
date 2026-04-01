package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/trust"
)

var errTrustReportHelp = errors.New("trust-report help requested")

type trustReportConfig struct {
	BundlePath string
	Format     string
	ProfileID  string
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

	report := trust.BuildReport(cwd, cfg.BundlePath, cfg.ProfileID)
	switch cfg.Format {
	case "json":
		if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "atb trust-report: encode json: %v\n", err)
			os.Exit(exitSystemError)
		}
	case "text":
		renderTrustReportText(os.Stdout, report)
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
				return cfg, fmt.Errorf("missing value for --format (expected markdown|json|text)")
			}
			cfg.Format = strings.ToLower(strings.TrimSpace(args[i+1]))
			i++
		case arg == "--profile" || arg == "-p":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for %s", arg)
			}
			cfg.ProfileID = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(arg, "--format="):
			cfg.Format = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--format=")))
		case strings.HasPrefix(arg, "--profile="):
			cfg.ProfileID = strings.TrimSpace(strings.TrimPrefix(arg, "--profile="))
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
	if cfg.Format != "markdown" && cfg.Format != "json" && cfg.Format != "text" {
		return cfg, fmt.Errorf("invalid format %q (expected markdown|json|text)", cfg.Format)
	}
	return cfg, nil
}

func printTrustReportUsage() {
	fmt.Println("Usage: atb trust-report [bundle_path] [--format markdown|json] [--profile <id>]")
}

func printTrustReportMarkdown(report trust.Report) {
	fmt.Println("# ATB Trust Report")
	fmt.Println()
	fmt.Printf("- Verdict: **%s**\n", strings.ToUpper(report.Status))
	fmt.Printf("- CI Gate: **%s**\n", strings.ToUpper(report.Gate.Status))
	fmt.Printf("- Blocking failures: %d\n", report.Gate.BlockingFailures)
	fmt.Printf("- Generated: %s\n", report.GeneratedAt)
	fmt.Printf("- Bundle: `%s`\n", report.BundlePath)
	fmt.Printf("- Chain length: %d\n", report.ChainLength)
	if report.HeadHash != "" {
		fmt.Printf("- Head hash: `%s`\n", report.HeadHash)
	}
	fmt.Printf("- Checks: total=%d pass=%d warn=%d fail=%d\n", report.Summary.Total, report.Summary.Pass, report.Summary.Warn, report.Summary.Fail)
	if len(report.Gate.FailedChecks) > 0 {
		fmt.Printf("- Failed blocking checks: `%s`\n", strings.Join(report.Gate.FailedChecks, "`, `"))
	}
	fmt.Println()
	if report.CAS != nil {
		fmt.Println("## Completeness Assurance")
		fmt.Println()
		fmt.Printf("- Profile: `%s`\n", report.CAS.ProfileID)
		fmt.Printf("- Workflow class: `%s`\n", report.CAS.WorkflowClass)
		fmt.Printf("- Overall: %.3f (%s)\n", report.CAS.Overall, report.CAS.Grade)
		fmt.Printf("- Anchor quality: `%s` (XC=%.3f, AC=%.3f)\n", report.CAS.AnchorQuality.Label, report.CAS.AnchorQuality.XC, report.CAS.AnchorQuality.AC)

		scoreKeys := sortedFloatKeys(report.CAS.SubScores)
		if len(scoreKeys) > 0 {
			fmt.Printf("- Sub-scores: `%s`\n", formatFloatMap(report.CAS.SubScores, scoreKeys))
		}

		weightKeys := sortedFloatKeys(report.CAS.Weights)
		if len(weightKeys) > 0 {
			fmt.Printf("- Weights: `%s`\n", formatFloatMap(report.CAS.Weights, weightKeys))
		}
		fmt.Println()
	}
	for _, category := range report.Categories {
		fmt.Printf("## %s (%s)\n\n", category.Title, strings.ToUpper(category.Status))
		for _, check := range category.Checks {
			blocking := "non-blocking"
			if check.Blocking {
				blocking = "blocking"
			}
			fmt.Printf("- [%s] (%s, %s) %s: %s\n", strings.ToUpper(check.Status), check.Severity, blocking, check.Title, check.Details)
			if len(check.Evidence) > 0 {
				fmt.Printf("  Evidence: `%s`\n", relativeOrOriginal(report.BundlePath, check.Evidence[0]))
			}
		}
		fmt.Println()
	}
}

func renderTrustReportText(w io.Writer, report trust.Report) {
	renderer := verifyTextRenderer{colour: useVerifyANSI(w)}

	fmt.Fprintf(w, "Bundle:   %s\n", report.BundlePath)
	fmt.Fprintf(w, "Status:   %s\n", renderTrustReportStatus(renderer, report.Status))
	fmt.Fprintf(w, "Gate:     %s\n", strings.ToUpper(report.Gate.Status))
	fmt.Fprintf(w, "Summary:  total=%d pass=%d warn=%d fail=%d\n", report.Summary.Total, report.Summary.Pass, report.Summary.Warn, report.Summary.Fail)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Categories:")
	for _, category := range report.Categories {
		fmt.Fprintf(w, "  %s: %s\n", category.Key, strings.ToUpper(category.Status))
	}

	if report.CAS == nil {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "CAS:")
	fmt.Fprintf(w, "  Profile:   %s\n", report.CAS.ProfileID)
	fmt.Fprintf(w, "  Class:     %s\n", report.CAS.WorkflowClass)
	fmt.Fprintf(w, "  Grade:     %s  (%.2f)\n", report.CAS.Grade, report.CAS.Overall)
	fmt.Fprintf(w, "  Anchor:    %s  (XC=%.2f AC=%.2f)\n", report.CAS.AnchorQuality.Label, report.CAS.AnchorQuality.XC, report.CAS.AnchorQuality.AC)
	fmt.Fprintln(w, "  Sub-scores:")
	fmt.Fprintf(
		w,
		"    EC  %.2f   FC  %.2f   RC  %.2f   TC  %.2f\n",
		report.CAS.SubScores["EC"],
		report.CAS.SubScores["FC"],
		report.CAS.SubScores["RC"],
		report.CAS.SubScores["TC"],
	)
	fmt.Fprintf(
		w,
		"    SC  %.2f   XC  %.2f   AC  %.2f   GC  %.2f\n",
		report.CAS.SubScores["SC"],
		report.CAS.SubScores["XC"],
		report.CAS.SubScores["AC"],
		report.CAS.SubScores["GC"],
	)
}

func renderTrustReportStatus(renderer verifyTextRenderer, status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case trust.StatusPass:
		return renderer.colourise("PASS", ansiGreen)
	case trust.StatusFail:
		return renderer.colourise("FAIL", ansiRed)
	case trust.StatusWarn:
		return renderer.colourise("WARN", ansiYellow)
	default:
		return strings.ToUpper(strings.TrimSpace(status))
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

func sortedFloatKeys(values map[string]float64) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func formatFloatMap(values map[string]float64, keys []string) string {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%.3f", key, values[key]))
	}
	return strings.Join(parts, ", ")
}
