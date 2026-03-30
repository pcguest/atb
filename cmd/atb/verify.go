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
	Quiet        bool
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
			if !cfg.Quiet {
				printVerifyCommandUsage(stdout)
			}
			return exitSuccess
		}
		if !cfg.Quiet {
			fmt.Fprintf(stderr, "atb verify: %v\n", err)
			printVerifyCommandUsage(stderr)
		}
		return exitUserError
	}

	if cfg.Quiet {
		stdout = io.Discard
		stderr = io.Discard
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
		case arg == "--quiet":
			cfg.Quiet = true
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
	fmt.Fprintln(w, "Usage: atb verify [bundle_path] [--bundle path/to/file.atb] [--profile <id|path>] [--json] [--format text|json] [--quiet] [--trace] [--with-anchor]")
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
	renderVerifyTerminalReport(w, report)
}
