package main

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/push"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

var errVerifyHelp = errors.New("verify help requested")

type verifyCLIConfig struct {
	BundlePath        string
	RemoteURI         string // --remote s3://bucket/key; mutually exclusive with BundlePath
	ProfileID         string
	JSON              bool
	LegacyFormat      string
	Quiet             bool
	Trace             bool
	WithAnchor        bool
	WithSnapshotCheck bool
	RootsPath         string
}

func cmdVerify() {
	os.Exit(runVerify(os.Args[2:], os.Stdout, os.Stderr))
}

// remoteVerifyUploader allows tests to inject a fake S3 client for --remote verify.
var remoteVerifyUploader push.S3Uploader

func runVerify(args []string, stdout, stderr io.Writer) int {
	cfg, dryRun, err := parseVerifyArgsWithDryRun(args)
	if err != nil {
		if errors.Is(err, errVerifyHelp) {
			if !cfg.Quiet {
				printVerifyUsage(stdout)
			}
			return exitSuccess
		}
		if !cfg.Quiet {
			fmt.Fprintf(stderr, "atb verify: %v\n", err)
			printVerifyUsage(stderr)
		}
		return exitUserError
	}

	// --remote: download from S3 and stream-verify, then check key hash.
	if cfg.RemoteURI != "" {
		return runVerifyRemote(cfg, stdout, stderr)
	}

	return runVerifyWithConfig(cfg, dryRun, stdout, stderr)
}

func runVerifyWithConfig(cfg verifyCLIConfig, dryRun bool, stdout, stderr io.Writer) int {
	if cfg.Quiet && !dryRun && !verifyWantsStructuredJSON(cfg) && !cfg.JSON {
		stdout = io.Discard
		stderr = io.Discard
	}

	var profile verifypkg.Profile
	var err error
	resolvedProfileID := ""
	if cfg.ProfileID != "" {
		if isVerifyProfilePath(cfg.ProfileID) {
			profile, err = verifypkg.ResolveProfile(cfg.ProfileID)
			if err != nil {
				fmt.Fprintf(stderr, "atb verify: %v\n", err)
				printVerifyUsage(stderr)
				return exitUserError
			}
		} else {
			profile = verifypkg.ProfileByID(cfg.ProfileID)
			if profile == nil {
				fmt.Fprintf(stderr, "unknown profile: %s\n", cfg.ProfileID)
				return exitIntegrityFailure
			}
		}
		resolvedProfileID = profile.ID()
	}

	if dryRun {
		report := verifypkg.VerifierReport{
			BundlePath:   cfg.BundlePath,
			ProfileID:    resolvedProfileID,
			Pass:         false,
			Failures:     []verifypkg.ReportFailure{},
			Warnings:     []string{},
			Notes:        []string{"dry-run: no evaluation performed"},
			ResidualRisk: "Unknown",
		}
		if err := writeVerifierReportJSON(stdout, report); err != nil {
			fmt.Fprintf(stderr, "atb verify: encode json output: %v\n", err)
			return exitSystemError
		}
		return exitSuccess
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
	snapshotCheckFailed := false
	if cfg.WithSnapshotCheck {
		snapshotFailures := verifypkg.VerifySnapshotHashes(b.Records)
		if len(snapshotFailures) > 0 {
			report.InformationalNotes = append(report.InformationalNotes, snapshotFailures...)
			snapshotCheckFailed = true
			report.ResidualRisk.Level = "Critical"
		}
	}
	if cfg.WithAnchor && report.Integrity.ChainValid {
		roots, err := loadVerifyRoots(cfg.RootsPath)
		if err != nil {
			if isLegacyJSONMode(cfg) {
				result := newVerifyResult(cfg.BundlePath, b, "invalid")
				_ = writeLegacyVerifyJSON(stdout, result, err)
			}
			fmt.Fprintf(stderr, "atb verify: %v\n", err)
			return exitIntegrityFailure
		}
		prevRoots := verifyBundleAnchorRoots
		verifyBundleAnchorRoots = roots
		defer func() {
			verifyBundleAnchorRoots = prevRoots
		}()

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

	if verifyWantsStructuredJSON(cfg) {
		verifierReport := verifypkg.ReportFromVerify(report)
		if err := writeVerifierReportJSON(stdout, verifierReport); err != nil {
			fmt.Fprintf(stderr, "atb verify: encode json output: %v\n", err)
			return exitSystemError
		}
		if snapshotCheckFailed {
			return exitIntegrityFailure
		}
		return verificationExitCode(report)
	}

	if cfg.JSON {
		if err := json.NewEncoder(stdout).Encode(report); err != nil {
			fmt.Fprintf(stderr, "atb verify: encode json output: %v\n", err)
			return exitSystemError
		}
		if snapshotCheckFailed {
			return exitIntegrityFailure
		}
		return verificationExitCode(report)
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

	renderVerifyText(stdout, report)
	if snapshotCheckFailed {
		return exitIntegrityFailure
	}
	return verificationExitCode(report)
}

func parseVerifyArgsWithDryRun(args []string) (verifyCLIConfig, bool, error) {
	filtered := make([]string, 0, len(args))
	dryRun := false
	for _, arg := range args {
		if arg == "--dry-run" {
			dryRun = true
			continue
		}
		filtered = append(filtered, arg)
	}

	cfg, err := parseVerifyCommandArgs(filtered)
	return cfg, dryRun, err
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
		case arg == "-f":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for -f (expected text|json)")
			}
			i++
			cfg.LegacyFormat = strings.ToLower(strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "--format="):
			cfg.LegacyFormat = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--format=")))
		case strings.HasPrefix(arg, "-f="):
			cfg.LegacyFormat = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "-f=")))
		case arg == "--trace":
			cfg.Trace = true
		case arg == "--with-anchor":
			cfg.WithAnchor = true
		case arg == "--with-snapshot-check":
			cfg.WithSnapshotCheck = true
		case arg == "--roots":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --roots")
			}
			i++
			cfg.RootsPath = filepath.Clean(strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "--roots="):
			cfg.RootsPath = filepath.Clean(strings.TrimSpace(strings.TrimPrefix(arg, "--roots=")))
		case arg == "--remote":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --remote (expected s3://bucket/key)")
			}
			i++
			cfg.RemoteURI = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--remote="):
			cfg.RemoteURI = strings.TrimSpace(strings.TrimPrefix(arg, "--remote="))
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
	if cfg.RemoteURI != "" && bundlePathSet {
		return cfg, fmt.Errorf("--remote and --bundle are mutually exclusive")
	}
	return cfg, nil
}

func printVerifyUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb verify [bundle_path] [--bundle path/to/file.atb] [--remote s3://bucket/key] [--profile <id|path>] [--json] [--format text|json] [-f text|json] [--dry-run] [--quiet] [--trace] [--with-anchor] [--with-snapshot-check] [--roots <pem-file>]")
	fmt.Fprintln(w, "  --remote s3://bucket/prefix/sha256-<hash>.atb  stream-verify a bundle stored on S3; checks key hash against computed head hash")
	fmt.Fprintln(w, "  --with-anchor  verify RFC 3161 timestamp token: digest, cert chain, and signature")
	fmt.Fprintln(w, "  --with-snapshot-check  verify each atb.snapshot bundle_hash against the recorded bundle prefix")
	fmt.Fprintln(w, "  --roots <pem-file>   PEM file containing trusted root certificates for TSA chain verification (default: system roots)")
}

func verificationExitCode(report verifypkg.Report) int {
	if !report.Integrity.ChainValid {
		return exitIntegrityFailure
	}
	if report.BundleSignature != nil && !report.BundleSignature.Verified {
		return exitIntegrityFailure
	}
	for _, profile := range report.Profiles {
		if !profile.Pass {
			return exitVerifyFailure
		}
	}
	return exitSuccess
}

func isLegacyJSONMode(cfg verifyCLIConfig) bool {
	return !cfg.JSON && cfg.LegacyFormat == verifyFormatJSON
}

func verifyWantsStructuredJSON(cfg verifyCLIConfig) bool {
	return !cfg.JSON && cfg.LegacyFormat == verifyFormatJSON
}

func writeVerifierReportJSON(w io.Writer, report verifypkg.VerifierReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

func writeLegacyVerifyJSON(w io.Writer, result verifyResult, verifyErr error) error {
	if verifyErr != nil && verifyErr.Error() != "" {
		result.Error = verifyErr.Error()
	}
	return json.NewEncoder(w).Encode(result)
}

func isVerifyProfilePath(spec string) bool {
	if strings.ContainsAny(spec, `/\`) {
		return true
	}

	switch strings.ToLower(filepath.Ext(spec)) {
	case ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func loadVerifyRoots(path string) (*x509.CertPool, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}

	pemBytes, err := os.ReadFile(path) // #nosec G703 -- path is filepath.Clean-sanitised at parse time; supplied by operator via --roots CLI flag
	if err != nil {
		return nil, fmt.Errorf("roots: read %s: %w", path, err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemBytes) {
		return nil, fmt.Errorf("roots: no certificates found in %s", path)
	}
	return pool, nil
}

func renderVerifyText(w io.Writer, report verifypkg.Report) {
	renderVerifyTerminalReport(w, report)
}

// remoteVerifyReport wraps the normal VerifierReport with S3-specific fields for --remote.
type remoteVerifyReport struct {
	RemoteURI    string `json:"remote_uri"`
	KeyHashOK    bool   `json:"key_hash_ok"`
	KeyHashWarn  string `json:"key_hash_warn,omitempty"`
	verifypkg.VerifierReport
}

// runVerifyRemote implements atb verify --remote s3://bucket/key.
// It downloads the bundle, runs the normal verify pipeline, then checks
// that the computed head hash matches the hash encoded in the S3 key name.
func runVerifyRemote(cfg verifyCLIConfig, stdout, stderr io.Writer) int {
	bucket, key, err := parseRemoteVerifyURI(cfg.RemoteURI)
	if err != nil {
		fmt.Fprintf(stderr, "atb verify: %v\n", err)
		return exitUserError
	}

	// Extract the expected head hash from the object key filename.
	expectedHash, hasHash := push.KeyHash(cfg.RemoteURI)

	// Resolve the S3 client (real or injected fake for tests).
	uploader := remoteVerifyUploader
	if uploader == nil {
		uploader, err = push.NewHTTPClient()
		if err != nil {
			fmt.Fprintf(stderr, "atb verify: credential error: %v\n", err)
			return exitSystemError
		}
	}

	// Download the bundle from S3.
	out, err := uploader.GetObject(context.Background(), push.GetObjectInput{
		Bucket: bucket,
		Key:    key,
	})
	if err != nil {
		code := exitSystemError
		if push.IsNotFound(err) {
			code = exitUserError
		}
		fmt.Fprintf(stderr, "atb verify: download %s: %v\n", cfg.RemoteURI, err)
		return code
	}
	defer out.Body.Close()

	// Load bundle from the S3 response stream.
	b, err := bundle.LoadReader(out.Body)
	if err != nil {
		fmt.Fprintf(stderr, "atb verify: load remote bundle: %v\n", err)
		return exitSystemError
	}

	// Run the normal verification pipeline (profile, chain integrity, etc.).
	report := verifypkg.Verify(b, cfg.RemoteURI, cfg.ProfileID)
	if cfg.ProfileID != "" {
		if isVerifyProfilePath(cfg.ProfileID) {
			profile, perr := verifypkg.ResolveProfile(cfg.ProfileID)
			if perr != nil {
				fmt.Fprintf(stderr, "atb verify: %v\n", perr)
				return exitUserError
			}
			report = verifypkg.VerifyWithProfile(b, cfg.RemoteURI, profile)
		} else {
			profile := verifypkg.ProfileByID(cfg.ProfileID)
			if profile == nil {
				fmt.Fprintf(stderr, "unknown profile: %s\n", cfg.ProfileID)
				return exitIntegrityFailure
			}
			report = verifypkg.VerifyWithProfile(b, cfg.RemoteURI, profile)
		}
	}

	// Content-addressing check: computed head hash must match the hash in the key name.
	keyHashOK := true
	keyHashWarn := ""
	if !hasHash {
		keyHashWarn = "key does not follow sha256-<hash>.atb naming; content-address check skipped"
	} else if len(b.Records) > 0 {
		computedHead := b.Records[len(b.Records)-1].Hash
		if computedHead != expectedHash {
			keyHashOK = false
			keyHashWarn = fmt.Sprintf(
				"CRITICAL: key hash %s does not match computed head hash %s — bundle may have been replaced",
				expectedHash, computedHead)
		}
	}

	// Emit output.
	verifierReport := verifypkg.ReportFromVerify(report)

	if verifyWantsStructuredJSON(cfg) || cfg.JSON {
		wrapped := remoteVerifyReport{
			RemoteURI:      cfg.RemoteURI,
			KeyHashOK:      keyHashOK,
			KeyHashWarn:    keyHashWarn,
			VerifierReport: verifierReport,
		}
		data, merr := json.MarshalIndent(wrapped, "", "  ")
		if merr != nil {
			fmt.Fprintf(stderr, "atb verify: encode json output: %v\n", merr)
			return exitSystemError
		}
		data = append(data, '\n')
		if _, werr := stdout.Write(data); werr != nil {
			fmt.Fprintf(stderr, "atb verify: write output: %v\n", werr)
			return exitSystemError
		}
	} else {
		// Text output: print normal report then the remote-specific lines.
		renderVerifyText(stdout, report)
		fmt.Fprintln(stdout)
		fmt.Fprintf(stdout, "Remote:    %s\n", cfg.RemoteURI)
		if keyHashOK && hasHash {
			fmt.Fprintf(stdout, "Key hash:  OK (%s)\n", expectedHash)
		} else if keyHashWarn != "" {
			fmt.Fprintf(stdout, "Key hash:  %s\n", keyHashWarn)
		}
	}

	if !keyHashOK {
		return exitIntegrityFailure
	}
	return verificationExitCode(report)
}

// parseRemoteVerifyURI splits a full S3 object URI into (bucket, key).
// Unlike ParseS3URI which parses a prefix, here the entire path after the
// bucket is the object key.
func parseRemoteVerifyURI(uri string) (bucket, key string, err error) {
	if !strings.HasPrefix(uri, "s3://") {
		return "", "", fmt.Errorf("--remote must be an S3 URI (s3://bucket/key), got %q", uri)
	}
	rest := strings.TrimPrefix(uri, "s3://")
	idx := strings.Index(rest, "/")
	if idx == -1 || idx == len(rest)-1 {
		return "", "", fmt.Errorf("--remote URI must include an object key: %q", uri)
	}
	return rest[:idx], rest[idx+1:], nil
}
