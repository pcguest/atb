package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/push"
)

const pushUsageLine = "Usage: atb push <s3://bucket/prefix> [--bundle <path>] [--lock-until YYYY-MM-DD] [--dry-run] [--format text|json]"

var errPushHelp = errors.New("push help requested")

// atb push contract
//
// CLI surface:
//   - Explicit operator action only. No background uploads.
//   - Primary form is `atb push <s3://bucket/prefix>`.
//   - `--bundle` selects the local bundle path. Default is run.atb/bundle.atb.
//   - `--lock-until` requests S3 Object Lock headers for the upload.
//   - `--dry-run` resolves the target object key without contacting the remote.
//   - `--format text|json` controls output shape only.
//
// Configuration and credential source:
//   - The target object location is taken from the explicit S3 URI. Minimal
//     defaults for S3-compatible endpoints may also come from ./.atb/config.json.
//   - AWS credentials continue to come from the existing standard sources only:
//     environment variables and ~/.aws/credentials. atb push does not add new
//     credential resolution logic or make implicit network calls.
//
// Success criteria:
//   - The local bundle is loaded from disk and uploaded unchanged.
//   - The object key is content addressed as `sha256-<bundle-head-hash>.atb`.
//   - Success means the remote returned a 2xx response to the PUT request.
//   - When lock headers are requested, ATB only sends the headers. It does not
//     inspect or validate bucket-side WORM policy.
//
// Error reporting:
//   - Usage and local bundle problems are user errors.
//   - Credential, transport, and non-2xx remote responses are system errors.
//   - Text mode prints a single-line `atb push: ...` failure.
//   - JSON mode returns the pushResult error payload with the mapped exit code.

// pushConfig holds parsed CLI arguments plus optional defaults loaded from
// ./.atb/config.json.
type pushConfig struct {
	// Target is the destination URI, e.g. s3://bucket/prefix.
	Target string
	// BundlePath is the local bundle to push. Defaults to run.atb/bundle.atb.
	BundlePath string
	// LockUntil is the WORM retain-until date in YYYY-MM-DD format.
	// Only applied when the target bucket has Object Lock enabled in COMPLIANCE mode.
	// ATB requests the lock header; the bucket policy enforces WORM.
	LockUntil string
	// DryRun previews the operation without uploading.
	DryRun bool
	// Format is "text" or "json".
	Format string
	// EndpointURL overrides the S3 API base URL for S3-compatible targets.
	EndpointURL string
	// Region overrides the AWS region used for SigV4 signing.
	Region string
	// LockMode is the object-lock mode to request when LockUntil is set.
	LockMode string
	// CredentialsSource documents which existing resolver path to use.
	CredentialsSource string
}

// pushResult is the JSON output shape for atb push.
// Defined here to lock in the JSON contract (docs/spec/bundle-push.md § JSON output schema).
type pushResult struct {
	Status     string `json:"status"`
	Action     string `json:"action"`
	DryRun     bool   `json:"dry_run"`
	Target     string `json:"target"`
	BundlePath string `json:"bundle_path"`
	BundleHash string `json:"bundle_hash,omitempty"`
	ObjectKey  string `json:"object_key,omitempty"`
	LockUntil  string `json:"lock_until,omitempty"`
	Message    string `json:"message,omitempty"`
	Error      string `json:"error,omitempty"`
	ExitCode   int    `json:"exit_code,omitempty"`
}

func cmdPush() {
	os.Exit(runPush(os.Args[2:], os.Stdout, os.Stderr))
}

func runPush(args []string, stdout, stderr io.Writer) int {
	return runPushWithUploader(args, stdout, stderr, nil)
}

// runPushWithUploader is the testable form of runPush.
// When uploader is nil, NewHTTPClient is called to create the real S3 client.
func runPushWithUploader(args []string, stdout, stderr io.Writer, uploader push.S3Uploader) int {
	cfg, err := parsePushArgs(args)
	if err != nil {
		if errors.Is(err, errPushHelp) {
			fmt.Fprintln(stdout, pushUsageLine)
			return exitSuccess
		}
		if cfg.Format == verifyFormatJSON {
			writePushError(stdout, stderr, cfg, err.Error(), exitUserError)
			return exitUserError
		}
		fmt.Fprintf(stderr, "atb push: %v\n", err)
		fmt.Fprintln(stderr, pushUsageLine)
		return exitUserError
	}

	cfg, err = mergePushConfigWithDefaults(cfg, defaultConfigPath())
	if err != nil {
		if cfg.Format == verifyFormatJSON {
			writePushError(stdout, stderr, cfg, err.Error(), exitUserError)
			return exitUserError
		}
		fmt.Fprintf(stderr, "atb push: %v\n", err)
		return exitUserError
	}
	if strings.TrimSpace(cfg.Target) == "" {
		err = fmt.Errorf("target URI required (e.g. s3://bucket/prefix)")
		if cfg.Format == verifyFormatJSON {
			writePushError(stdout, stderr, cfg, err.Error(), exitUserError)
			return exitUserError
		}
		fmt.Fprintf(stderr, "atb push: %v\n", err)
		fmt.Fprintln(stderr, pushUsageLine)
		return exitUserError
	}

	// Parse and validate the S3 target URI.
	bucket, prefix, err := push.ParseS3URI(cfg.Target)
	if err != nil {
		if cfg.Format == verifyFormatJSON {
			writePushError(stdout, stderr, cfg, err.Error(), exitUserError)
			return exitUserError
		}
		fmt.Fprintf(stderr, "atb push: %v\n", err)
		return exitUserError
	}

	// Validate and normalise --lock-until → RFC 3339 datetime.
	lockUntil := ""
	if cfg.LockUntil != "" {
		lockUntil, err = parseLockUntil(cfg.LockUntil)
		if err != nil {
			if cfg.Format == verifyFormatJSON {
				writePushError(stdout, stderr, cfg, err.Error(), exitUserError)
				return exitUserError
			}
			fmt.Fprintf(stderr, "atb push: %v\n", err)
			return exitUserError
		}
	}

	// Load and validate the local bundle.
	b, err := bundle.Load(cfg.BundlePath)
	if err != nil {
		code := classifyBundleLoadError(err)
		if cfg.Format == verifyFormatJSON {
			writePushError(stdout, stderr, cfg, err.Error(), code)
			return code
		}
		fmt.Fprintf(stderr, "atb push: %v\n", err)
		return code
	}
	if len(b.Records) == 0 {
		const msg = "bundle has no records; cannot push an empty bundle"
		if cfg.Format == verifyFormatJSON {
			writePushError(stdout, stderr, cfg, msg, exitUserError)
			return exitUserError
		}
		fmt.Fprintf(stderr, "atb push: %s\n", msg)
		return exitUserError
	}

	headHash := b.Records[len(b.Records)-1].Hash
	key := push.ObjectKey(prefix, headHash)

	// Dry-run: print resolved key and headers; do not upload.
	if cfg.DryRun {
		return runPushDryRun(cfg, bucket, key, headHash, lockUntil, stdout, stderr)
	}

	// Read bundle bytes for upload.
	bundleBytes, err := os.ReadFile(filepath.Clean(cfg.BundlePath)) // #nosec G304 -- path validated by bundle.Load
	if err != nil {
		if cfg.Format == verifyFormatJSON {
			writePushError(stdout, stderr, cfg, err.Error(), exitSystemError)
			return exitSystemError
		}
		fmt.Fprintf(stderr, "atb push: read bundle: %v\n", err)
		return exitSystemError
	}

	// Resolve the S3 client (real or injected fake for tests).
	if uploader == nil {
		uploader, err = push.NewHTTPClientWithConfig(push.ClientConfig{
			EndpointURL:       cfg.EndpointURL,
			Region:            cfg.Region,
			CredentialsSource: cfg.CredentialsSource,
		})
		if err != nil {
			msg := "credential error: " + err.Error()
			if cfg.Format == verifyFormatJSON {
				writePushError(stdout, stderr, cfg, msg, exitSystemError)
				return exitSystemError
			}
			fmt.Fprintf(stderr, "atb push: %s\n", msg)
			return exitSystemError
		}
	}

	// Upload.
	lockMode := ""
	if cfg.LockUntil != "" {
		lockMode = cfg.LockMode
		if lockMode == "" {
			lockMode = "COMPLIANCE"
		}
	}
	out, err := uploader.PutObject(context.Background(), push.PutObjectInput{
		Bucket:    bucket,
		Key:       key,
		Body:      bundleBytes,
		LockMode:  lockMode,
		LockUntil: lockUntil,
	})
	if err != nil {
		code := exitSystemError
		msg := classifyPushError(err)
		if cfg.Format == verifyFormatJSON {
			writePushError(stdout, stderr, cfg, msg, code)
			return code
		}
		fmt.Fprintf(stderr, "atb push: %s\n", msg)
		return code
	}
	remoteURI := "s3://" + bucket + "/" + key

	// Success output.
	if cfg.Format == verifyFormatJSON {
		res := pushResult{
			Status:     "ok",
			Action:     "push",
			DryRun:     false,
			Target:     cfg.Target,
			BundlePath: cfg.BundlePath,
			BundleHash: headHash,
			ObjectKey:  key,
			LockUntil:  cfg.LockUntil,
			Message:    successMessage(remoteURI, key, out.ETag),
			ExitCode:   exitSuccess,
		}
		if err := json.NewEncoder(stdout).Encode(res); err != nil {
			fmt.Fprintf(stderr, "atb push: encode json output: %v\n", err)
		}
		return exitSuccess
	}

	fmt.Fprintf(stdout, "pushed  %s\n", remoteURI)
	fmt.Fprintf(stdout, "key     %s\n", key)
	fmt.Fprintf(stdout, "hash    %s\n", headHash)
	if cfg.LockUntil != "" {
		fmt.Fprintf(stdout, "locked  COMPLIANCE until %s\n", cfg.LockUntil)
	}
	return exitSuccess
}

// runPushDryRun handles --dry-run: prints what would be uploaded without contacting S3.
func runPushDryRun(cfg pushConfig, bucket, key, headHash, lockUntil string, stdout, stderr io.Writer) int {
	remoteURI := "s3://" + bucket + "/" + key

	if cfg.Format == verifyFormatJSON {
		res := pushResult{
			Status:     "ok",
			Action:     "preview_push",
			DryRun:     true,
			Target:     cfg.Target,
			BundlePath: cfg.BundlePath,
			BundleHash: headHash,
			ObjectKey:  key,
			LockUntil:  cfg.LockUntil,
			Message:    "dry-run: no upload performed",
			ExitCode:   exitSuccess,
		}
		if err := json.NewEncoder(stdout).Encode(res); err != nil {
			fmt.Fprintf(stderr, "atb push: encode json output: %v\n", err)
		}
		return exitSuccess
	}

	fmt.Fprintf(stdout, "dry-run  no upload performed\n")
	fmt.Fprintf(stdout, "target   %s\n", remoteURI)
	fmt.Fprintf(stdout, "key      %s\n", key)
	fmt.Fprintf(stdout, "hash     %s\n", headHash)
	if lockUntil != "" {
		fmt.Fprintf(stdout, "header   x-amz-object-lock-mode: COMPLIANCE\n")
		fmt.Fprintf(stdout, "header   x-amz-object-lock-retain-until-date: %s\n", lockUntil)
	}
	return exitSuccess
}

// parseLockUntil validates and converts a YYYY-MM-DD date to the RFC 3339 datetime
// required by the x-amz-object-lock-retain-until-date header.
func parseLockUntil(s string) (string, error) {
	_, err := time.Parse("2006-01-02", s)
	if err != nil {
		return "", fmt.Errorf("--lock-until %q must be in YYYY-MM-DD format", s)
	}
	return s + "T00:00:00Z", nil
}

func mergePushConfigWithDefaults(cfg pushConfig, configPath string) (pushConfig, error) {
	settings, err := loadPushSettings(configPath)
	if err != nil {
		return cfg, fmt.Errorf("load push config: %w", err)
	}
	if settings == nil {
		return cfg, nil
	}

	if strings.TrimSpace(cfg.Target) == "" {
		cfg.Target = strings.TrimSpace(settings.Target)
	}
	if strings.TrimSpace(cfg.EndpointURL) == "" {
		cfg.EndpointURL = strings.TrimSpace(settings.EndpointURL)
	}
	if strings.TrimSpace(cfg.Region) == "" {
		cfg.Region = strings.TrimSpace(settings.Region)
	}
	if strings.TrimSpace(cfg.LockMode) == "" {
		cfg.LockMode = strings.ToUpper(strings.TrimSpace(settings.LockMode))
	}
	if strings.TrimSpace(cfg.LockUntil) == "" {
		cfg.LockUntil = strings.TrimSpace(settings.LockUntil)
	}
	if strings.TrimSpace(cfg.CredentialsSource) == "" {
		cfg.CredentialsSource = strings.TrimSpace(settings.CredentialsSource)
	}
	if cfg.LockMode != "" && cfg.LockMode != "COMPLIANCE" && cfg.LockMode != "GOVERNANCE" {
		return cfg, fmt.Errorf("invalid lock mode %q in push config", cfg.LockMode)
	}
	return cfg, nil
}

// classifyPushError maps an S3 error to a user-facing message.
func classifyPushError(err error) string {
	if push.IsAuthError(err) {
		return "credential/permission error: " + err.Error()
	}
	if push.IsNotFound(err) {
		return "bucket not found: " + err.Error()
	}
	return "upload failed: " + err.Error()
}

func parsePushArgs(args []string) (pushConfig, error) {
	cfg := pushConfig{
		BundlePath: bundle.DefaultPath(),
		Format:     verifyFormatText,
	}
	targetSet := false
	bundlePathSet := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errPushHelp
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
		case arg == "--lock-until":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --lock-until (expected YYYY-MM-DD)")
			}
			i++
			cfg.LockUntil = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--lock-until="):
			cfg.LockUntil = strings.TrimSpace(strings.TrimPrefix(arg, "--lock-until="))
		case arg == "--dry-run":
			cfg.DryRun = true
		case arg == "--format":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --format (expected text|json)")
			}
			i++
			cfg.Format = strings.ToLower(strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "--format="):
			cfg.Format = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--format=")))
		case strings.HasPrefix(arg, "-"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			if targetSet {
				return cfg, fmt.Errorf("unexpected argument %q (target already set to %q)", arg, cfg.Target)
			}
			cfg.Target = strings.TrimSpace(arg)
			targetSet = true
		}
	}

	if cfg.Format != verifyFormatText && cfg.Format != verifyFormatJSON {
		return cfg, fmt.Errorf("invalid format %q (expected text|json)", cfg.Format)
	}
	return cfg, nil
}

func successMessage(remoteURI, objectKey, etag string) string {
	msg := fmt.Sprintf("bundle pushed to %s (%s)", remoteURI, objectKey)
	if strings.TrimSpace(etag) != "" {
		msg += " etag=" + strings.TrimSpace(etag)
	}
	return msg
}

func writePushError(stdout, stderr io.Writer, cfg pushConfig, msg string, code int) {
	res := pushResult{
		Status:     "error",
		Action:     "push",
		DryRun:     cfg.DryRun,
		Target:     cfg.Target,
		BundlePath: cfg.BundlePath,
		LockUntil:  cfg.LockUntil,
		Error:      msg,
		ExitCode:   code,
	}
	if err := json.NewEncoder(stdout).Encode(res); err != nil {
		fmt.Fprintf(stderr, "atb push: encode json output: %v\n", err)
	}
}
