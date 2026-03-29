package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

var errVerifyHelp = errors.New("verify help requested")

type verifyCLIConfig struct {
	BundlePath   string
	ProfileID    string
	JSON         bool
	LegacyFormat string
	Trace        bool
	WithAnchor   bool
}

func cmdVerifyProfile() {
	os.Exit(runVerify(os.Args[2:], os.Stdout, os.Stderr))
}

func runVerify(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseVerifyCommandArgs(args)
	if err != nil {
		if errors.Is(err, errVerifyHelp) {
			printVerifyCommandUsage(stdout)
			return exitSuccess
		}
		fmt.Fprintf(stderr, "atb verify: %v\n", err)
		printVerifyCommandUsage(stderr)
		return exitUserError
	}

	var profile verifypkg.Profile
	if cfg.ProfileID != "" {
		profile, err = verifypkg.ResolveProfile(cfg.ProfileID)
		if err != nil {
			fmt.Fprintf(stderr, "atb verify: %v\n", err)
			printVerifyCommandUsage(stderr)
			return exitUserError
		}
	}

	b, err := bundle.Load(cfg.BundlePath)
	if err != nil {
		exitCode := exitUserError
		if isLegacyJSONMode(cfg) {
			exitCode = classifyBundleLoadError(err)
			_ = writeLegacyVerifyJSON(stdout, newVerifyResult(cfg.BundlePath, nil, "error"), err)
		}
		fmt.Fprintf(stderr, "atb verify: %v\n", err)
		return exitCode
	}

	if cfg.Trace {
		_ = verifyWithTrace(b, stderr)
	}

	report := verifypkg.Verify(b, cfg.BundlePath, cfg.ProfileID)
	if profile != nil {
		report = verifypkg.VerifyWithProfile(b, cfg.BundlePath, profile)
	}
	if cfg.WithAnchor && report.Integrity.ChainValid {
		anchorOut := stdout
		if cfg.JSON || isLegacyJSONMode(cfg) {
			anchorOut = io.Discard
		}
		if err := verifyBundleAnchor(cfg.BundlePath, b, anchorOut); err != nil {
			if isLegacyJSONMode(cfg) {
				result := newVerifyResult(cfg.BundlePath, b, "invalid")
				_ = writeLegacyVerifyJSON(stdout, result, err)
			}
			fmt.Fprintf(stderr, "atb verify: %v\n", err)
			return exitIntegrityFailure
		}
	}

	if isLegacyJSONMode(cfg) {
		status := "valid"
		if len(b.Records) == 0 {
			status = "empty"
		}
		if !report.Integrity.ChainValid {
			status = "invalid"
		}

		result := newVerifyResult(cfg.BundlePath, b, status)
		if len(b.Records) == 0 {
			result.Message = "bundle is empty - nothing to verify"
		}
		var verifyErr error
		if report.Integrity.Error != "" {
			verifyErr = errors.New(report.Integrity.Error)
		}
		_ = writeLegacyVerifyJSON(stdout, result, verifyErr)
		if !report.Integrity.ChainValid {
			return exitIntegrityFailure
		}
		return exitSuccess
	}

	if cfg.JSON {
		if err := json.NewEncoder(stdout).Encode(report); err != nil {
			fmt.Fprintf(stderr, "atb verify: encode json output: %v\n", err)
			return exitSystemError
		}
		return verificationExitCode(report)
	}

	renderVerifyText(stdout, report)
	return verificationExitCode(report)
}

func parseVerifyCommandArgs(args []string) (verifyCLIConfig, error) {
	cfg := verifyCLIConfig{
		BundlePath:   bundle.DefaultPath(),
		LegacyFormat: verifyFormatText,
	}
	bundlePathSet := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errVerifyHelp
		case arg == "--bundle" || arg == "-b":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for %s", arg)
			}
			if bundlePathSet {
				return cfg, fmt.Errorf("bundle path already set")
			}
			i++
			cfg.BundlePath = normalizeBundlePath(args[i])
			bundlePathSet = true
		case strings.HasPrefix(arg, "--bundle="):
			if bundlePathSet {
				return cfg, fmt.Errorf("bundle path already set")
			}
			cfg.BundlePath = normalizeBundlePath(strings.TrimPrefix(arg, "--bundle="))
			bundlePathSet = true
		case arg == "--profile" || arg == "-p":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for %s", arg)
			}
			i++
			cfg.ProfileID = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--profile="):
			cfg.ProfileID = strings.TrimSpace(strings.TrimPrefix(arg, "--profile="))
		case arg == "--json":
			cfg.JSON = true
		case arg == "--format":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --format (expected text|json)")
			}
			i++
			cfg.LegacyFormat = strings.ToLower(strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "--format="):
			cfg.LegacyFormat = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--format=")))
		case arg == "--trace":
			cfg.Trace = true
		case arg == "--with-anchor":
			cfg.WithAnchor = true
		case strings.HasPrefix(arg, "-"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			if bundlePathSet {
				return cfg, fmt.Errorf("expected at most one bundle path")
			}
			cfg.BundlePath = normalizeBundlePath(arg)
			bundlePathSet = true
		}
	}

	if cfg.LegacyFormat != verifyFormatText && cfg.LegacyFormat != verifyFormatJSON {
		return cfg, fmt.Errorf("invalid format %q (expected text|json)", cfg.LegacyFormat)
	}
	if cfg.ProfileID != "" && strings.TrimSpace(cfg.ProfileID) == "" {
		return cfg, fmt.Errorf("--profile cannot be empty")
	}
	return cfg, nil
}

func printVerifyCommandUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb verify [bundle_path] [--bundle path/to/file.atb] [--profile <id|path>] [--json] [--format text|json] [--trace] [--with-anchor]")
}

func isLegacyJSONMode(cfg verifyCLIConfig) bool {
	return !cfg.JSON && cfg.LegacyFormat == verifyFormatJSON
}

func writeLegacyVerifyJSON(w io.Writer, result verifyResult, verifyErr error) error {
	if verifyErr != nil && verifyErr.Error() != "" {
		result.Error = verifyErr.Error()
	}
	return json.NewEncoder(w).Encode(result)
}

func verificationExitCode(report verifypkg.Report) int {
	if !report.Integrity.ChainValid {
		return exitIntegrityFailure
	}
	for _, profile := range report.Profiles {
		if !profile.Pass {
			return exitVerifyFailure
		}
	}
	return exitSuccess
}

func renderVerifyText(w io.Writer, report verifypkg.Report) {
	fmt.Fprintln(w, "ATB Verification Report")
	fmt.Fprintln(w, "=======================")
	fmt.Fprintf(w, "Bundle:    %s\n", report.BundlePath)

	if report.Integrity.ChainValid {
		fmt.Fprintf(
			w,
			"Integrity: PASS  [%s / %s / seq %d-%d]\n",
			report.Integrity.Canonicalization,
			report.Integrity.HashAlgo,
			report.Integrity.FirstSeq,
			report.Integrity.LastSeq,
		)
	} else {
		fmt.Fprintln(w, "Integrity: FAIL")
		if report.Integrity.Error != "" {
			fmt.Fprintf(w, "  Error: %s\n", report.Integrity.Error)
		}
	}

	if len(report.Profiles) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Profile:   none matched")
		fmt.Fprintln(w, "Result:    NOT EVALUATED")
	} else {
		profile := report.Profiles[0]
		status := "FAIL"
		if profile.Pass {
			status = "PASS"
		}

		fmt.Fprintln(w)
		fmt.Fprintf(w, "Profile:   %s v%d\n", profile.ProfileID, profile.Version)
		fmt.Fprintf(w, "Result:    %s\n", status)
		if report.CAS != nil {
			fmt.Fprintf(w, "CAS:       %.4f  [%s]\n", report.CAS.Overall, report.CAS.Grade)
			fmt.Fprintf(
				w,
				"  EC: %.2f  FC: %.2f  RC: %.2f  TC: %.2f\n",
				report.CAS.SubScores["EC"],
				report.CAS.SubScores["FC"],
				report.CAS.SubScores["RC"],
				report.CAS.SubScores["TC"],
			)
			fmt.Fprintf(
				w,
				"  SC: %.2f  XC: %.2f  AC: %.2f  GC: %.2f\n",
				report.CAS.SubScores["SC"],
				report.CAS.SubScores["XC"],
				report.CAS.SubScores["AC"],
				report.CAS.SubScores["GC"],
			)
		}

		if len(profile.CriticalFailures) > 0 {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "Critical failures (%d):\n", len(profile.CriticalFailures))
			for _, failure := range profile.CriticalFailures {
				fmt.Fprintf(w, "  - %s: %s\n", failure.Kind, failure.Detail)
			}
		}

		warnings := append([]string{}, profile.RequiredWarnings...)
		warnings = append(warnings, profile.InformationalNotes...)
		if len(warnings) > 0 {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "Warnings (%d):\n", len(warnings))
			for _, warning := range warnings {
				fmt.Fprintf(w, "  - %s\n", warning)
			}
		}
	}

	if len(report.Exclusions) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Exclusions:")
		for _, exclusion := range report.Exclusions {
			fmt.Fprintf(w, "  - %s\n", exclusion)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Residual risk: %s\n", report.ResidualRisk.Level)

	switch {
	case !report.Integrity.ChainValid:
		fmt.Fprintln(w, "  - Chain integrity invalid; no further evaluation possible.")
	case len(report.ResidualRisk.RecommendedNextEvidence) == 0 && report.ResidualRisk.Level == "Low":
		fmt.Fprintln(w, "  No high-priority evidence gaps.")
	default:
		for _, evidence := range report.ResidualRisk.RecommendedNextEvidence {
			fmt.Fprintf(w, "  - %s\n", evidence)
		}
		if len(report.ResidualRisk.RecommendedNextEvidence) == 0 {
			for _, driver := range report.ResidualRisk.Drivers {
				fmt.Fprintf(w, "  - %s\n", driver)
			}
		}
	}
}
