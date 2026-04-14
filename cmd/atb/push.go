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
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/push"
)

const pushUsageLine = "Usage: atb push <s3://bucket/prefix> [--bundle <path>] [--lock-until YYYY-MM-DD] [--dry-run] [--format text|json]"

var errPushHelp = errors.New("push help requested")

// pushConfig holds parsed arguments for the push command.
// Fields map 1:1 to the CLI surface in docs/spec/bundle-push.md.
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

// bundlePushedData is the event payload for atb.bundle.pushed records.
type bundlePushedData struct {
	RemoteURI  string `json:"remote_uri"`
	ObjectKey  string `json:"object_key"`
	BundleHash string `json:"bundle_hash"`
	PushedAt   string `json:"pushed_at"` // RFC 3339 UTC
	ETag       string `json:"etag,omitempty"`
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
		uploader, err = push.NewHTTPClient()
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
		lockMode = "COMPLIANCE"
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

	// Record the push event in the local bundle so the bundle reflects the export.
	remoteURI := "s3://" + bucket + "/" + key
	pushedAt := time.Now().UTC().Format(time.RFC3339)
	pushEventData := bundlePushedData{
		RemoteURI:  remoteURI,
		ObjectKey:  key,
		BundleHash: headHash,
		PushedAt:   pushedAt,
		ETag:       out.ETag,
	}
	if eventJSON, merr := json.Marshal(pushEventData); merr != nil {
		fmt.Fprintf(stderr, "atb push: warning: marshal push event: %v\n", merr)
	} else if aerr := b.Append(event.TypeBundlePushed, string(eventJSON)); aerr != nil {
		fmt.Fprintf(stderr, "atb push: warning: append push event: %v\n", aerr)
	} else if serr := b.Save(cfg.BundlePath); serr != nil {
		fmt.Fprintf(stderr, "atb push: warning: save bundle after push event: %v\n", serr)
	}

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
			Message:    "bundle pushed",
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

	if !targetSet || cfg.Target == "" {
		return cfg, fmt.Errorf("target URI required (e.g. s3://bucket/prefix)")
	}
	if cfg.Format != verifyFormatText && cfg.Format != verifyFormatJSON {
		return cfg, fmt.Errorf("invalid format %q (expected text|json)", cfg.Format)
	}
	return cfg, nil
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
