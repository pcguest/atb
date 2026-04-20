package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	verifypkg "github.com/pcguest/atb/internal/verify"
)

const pushUsageLine = "Usage: atb push [<s3://bucket/prefix>] [--queue <endpoint-url> --hmac-key <hex-key>] [--bundle <path>] [--lock-until YYYY-MM-DD] [--dry-run] [--format text|json]"

var errPushHelp = errors.New("push help requested")

// atb push contract
//
// CLI surface:
//   - Explicit operator action only. No background pushes.
//   - Primary form is `atb push <s3://bucket/prefix>`.
//   - Queue-only form is `atb push --queue <endpoint-url> --hmac-key <hex-key>`.
//   - `--bundle` selects the local bundle path. Default is run.atb/bundle.atb.
//   - `--lock-until` requests S3 Object Lock headers for the upload.
//   - `--queue` publishes a signed JSON envelope after any S3 upload completes.
//   - `--hmac-key` is required when `--queue` is set.
//   - `--dry-run` resolves the target object key and queue envelope without contacting remotes.
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
//   - The local bundle is loaded from disk and pushed unchanged.
//   - The S3 object key is content addressed as `sha256-<bundle-head-hash>.atb`.
//   - Success means the remote returned a 2xx response to the S3 PUT request.
//   - Success means the remote returned a 2xx response to the queue POST request.
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
	// QueueEndpoint is the optional HTTP endpoint that receives a signed queue envelope.
	QueueEndpoint string
	// HMACKeyHex is the caller-supplied hex key used to sign queue envelopes.
	HMACKeyHex string
}

// pushResult is the JSON output shape for atb push.
type pushResult struct {
	Status     string `json:"status"`
	Action     string `json:"action"`
	DryRun     bool   `json:"dry_run"`
	Target     string `json:"target"`
	BundlePath string `json:"bundle_path"`
	BundleHash string `json:"bundle_hash,omitempty"`
	ObjectKey  string `json:"object_key,omitempty"`
	LockUntil  string `json:"lock_until,omitempty"`
	QueueURL   string `json:"queue_url,omitempty"`
	Envelope   any    `json:"envelope,omitempty"`
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

	queueRequested := strings.TrimSpace(cfg.QueueEndpoint) != ""
	var hmacKey []byte
	if queueRequested {
		if strings.TrimSpace(cfg.HMACKeyHex) == "" {
			err = fmt.Errorf("--hmac-key is required when --queue is set")
			if cfg.Format == verifyFormatJSON {
				writePushError(stdout, stderr, cfg, err.Error(), exitUserError)
				return exitUserError
			}
			fmt.Fprintf(stderr, "atb push: %v\n", err)
			return exitUserError
		}
		hmacKey, err = parseHMACKey(cfg.HMACKeyHex)
		if err != nil {
			if cfg.Format == verifyFormatJSON {
				writePushError(stdout, stderr, cfg, err.Error(), exitUserError)
				return exitUserError
			}
			fmt.Fprintf(stderr, "atb push: %v\n", err)
			return exitUserError
		}
	}

	if strings.TrimSpace(cfg.Target) == "" && !queueRequested {
		err = fmt.Errorf("target URI required (e.g. s3://bucket/prefix); use --queue <endpoint-url> for queue-only push")
		if cfg.Format == verifyFormatJSON {
			writePushError(stdout, stderr, cfg, err.Error(), exitUserError)
			return exitUserError
		}
		fmt.Fprintf(stderr, "atb push: %v\n", err)
		fmt.Fprintln(stderr, pushUsageLine)
		return exitUserError
	}

	// Validate and normalise --lock-until -> RFC 3339 datetime.
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

	var bucket string
	var key string
	if strings.TrimSpace(cfg.Target) != "" {
		parsedBucket, prefix, parseErr := push.ParseS3URI(cfg.Target)
		if parseErr != nil {
			if cfg.Format == verifyFormatJSON {
				writePushError(stdout, stderr, cfg, parseErr.Error(), exitUserError)
				return exitUserError
			}
			fmt.Fprintf(stderr, "atb push: %v\n", parseErr)
			return exitUserError
		}
		bucket = parsedBucket
		key = push.ObjectKey(prefix, headHash)
	}

	// Read bundle bytes for push and queue metadata.
	bundleBytes, err := os.ReadFile(filepath.Clean(cfg.BundlePath)) // #nosec G304 -- path validated by bundle.Load
	if err != nil {
		if cfg.Format == verifyFormatJSON {
			writePushError(stdout, stderr, cfg, err.Error(), exitSystemError)
			return exitSystemError
		}
		fmt.Fprintf(stderr, "atb push: read bundle: %v\n", err)
		return exitSystemError
	}

	meta := buildPushMeta(b, bundleBytes, cfg.BundlePath)

	var envelopeJSON json.RawMessage
	if queueRequested {
		queuePusher := push.QueuePusher{
			EndpointURL: cfg.QueueEndpoint,
			HMACKey:     hmacKey,
			ATBVersion:  version,
		}
		body, marshalErr := queuePusher.MarshalEnvelope(meta)
		if marshalErr != nil {
			if cfg.Format == verifyFormatJSON {
				writePushError(stdout, stderr, cfg, marshalErr.Error(), exitUserError)
				return exitUserError
			}
			fmt.Fprintf(stderr, "atb push: %v\n", marshalErr)
			return exitUserError
		}
		envelopeJSON = json.RawMessage(body)
	}

	if cfg.DryRun {
		return runPushDryRun(cfg, bucket, key, headHash, lockUntil, envelopeJSON, stdout, stderr)
	}

	var remoteURI string
	if strings.TrimSpace(cfg.Target) != "" {
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

		lockMode := ""
		if cfg.LockUntil != "" {
			lockMode = cfg.LockMode
			if lockMode == "" {
				lockMode = "COMPLIANCE"
			}
		}

		s3Pusher := push.S3Pusher{
			Uploader:  uploader,
			Bucket:    bucket,
			Key:       key,
			LockMode:  lockMode,
			LockUntil: lockUntil,
		}
		if err := s3Pusher.Push(context.Background(), bundleBytes, meta); err != nil {
			code := exitSystemError
			msg := classifyPushError(err)
			if cfg.Format == verifyFormatJSON {
				writePushError(stdout, stderr, cfg, msg, code)
				return code
			}
			fmt.Fprintf(stderr, "atb push: %s\n", msg)
			return code
		}

		remoteURI = "s3://" + bucket + "/" + key
	}

	if queueRequested {
		queuePusher := push.QueuePusher{
			EndpointURL: cfg.QueueEndpoint,
			HMACKey:     hmacKey,
			ATBVersion:  version,
		}
		if err := queuePusher.Push(context.Background(), bundleBytes, meta); err != nil {
			if cfg.Format == verifyFormatJSON {
				writePushError(stdout, stderr, cfg, err.Error(), exitSystemError)
				return exitSystemError
			}
			fmt.Fprintf(stderr, "atb push: %v\n", err)
			return exitSystemError
		}
	}

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
			QueueURL:   cfg.QueueEndpoint,
			Message:    successMessage(remoteURI, key, cfg.QueueEndpoint),
			ExitCode:   exitSuccess,
		}
		if err := json.NewEncoder(stdout).Encode(res); err != nil {
			fmt.Fprintf(stderr, "atb push: encode json output: %v\n", err)
		}
		return exitSuccess
	}

	if remoteURI != "" {
		fmt.Fprintf(stdout, "pushed  %s\n", remoteURI)
		fmt.Fprintf(stdout, "key     %s\n", key)
	}
	fmt.Fprintf(stdout, "hash    %s\n", headHash)
	if remoteURI != "" && cfg.LockUntil != "" {
		fmt.Fprintf(stdout, "locked  COMPLIANCE until %s\n", cfg.LockUntil)
	}
	if queueRequested {
		fmt.Fprintf(stdout, "queue   %s\n", cfg.QueueEndpoint)
	}
	return exitSuccess
}

// runPushDryRun handles --dry-run: prints what would be pushed without contacting remotes.
func runPushDryRun(cfg pushConfig, bucket, key, headHash, lockUntil string, envelopeJSON json.RawMessage, stdout, stderr io.Writer) int {
	remoteURI := ""
	if bucket != "" && key != "" {
		remoteURI = "s3://" + bucket + "/" + key
	}

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
			QueueURL:   cfg.QueueEndpoint,
			Envelope:   envelopeJSON,
			Message:    "dry-run: no push performed",
			ExitCode:   exitSuccess,
		}
		if err := json.NewEncoder(stdout).Encode(res); err != nil {
			fmt.Fprintf(stderr, "atb push: encode json output: %v\n", err)
		}
		return exitSuccess
	}

	fmt.Fprintf(stdout, "dry-run  no push performed\n")
	if remoteURI != "" {
		fmt.Fprintf(stdout, "target   %s\n", remoteURI)
		fmt.Fprintf(stdout, "key      %s\n", key)
	}
	fmt.Fprintf(stdout, "hash     %s\n", headHash)
	if lockUntil != "" {
		fmt.Fprintf(stdout, "header   x-amz-object-lock-mode: COMPLIANCE\n")
		fmt.Fprintf(stdout, "header   x-amz-object-lock-retain-until-date: %s\n", lockUntil)
	}
	if cfg.QueueEndpoint != "" {
		fmt.Fprintf(stdout, "queue    %s\n", cfg.QueueEndpoint)
		fmt.Fprintf(stdout, "envelope %s\n", string(envelopeJSON))
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
		case arg == "--queue":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --queue")
			}
			i++
			cfg.QueueEndpoint = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--queue="):
			cfg.QueueEndpoint = strings.TrimSpace(strings.TrimPrefix(arg, "--queue="))
		case arg == "--hmac-key":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --hmac-key")
			}
			i++
			cfg.HMACKeyHex = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--hmac-key="):
			cfg.HMACKeyHex = strings.TrimSpace(strings.TrimPrefix(arg, "--hmac-key="))
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

func successMessage(remoteURI, objectKey, queueEndpoint string) string {
	if remoteURI == "" {
		return fmt.Sprintf("queue envelope posted to %s", queueEndpoint)
	}
	msg := fmt.Sprintf("bundle pushed to %s (%s)", remoteURI, objectKey)
	if strings.TrimSpace(queueEndpoint) != "" {
		msg += "; queue envelope posted to " + strings.TrimSpace(queueEndpoint)
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
		QueueURL:   cfg.QueueEndpoint,
		Error:      msg,
		ExitCode:   code,
	}
	if err := json.NewEncoder(stdout).Encode(res); err != nil {
		fmt.Fprintf(stderr, "atb push: encode json output: %v\n", err)
	}
}

func parseHMACKey(value string) ([]byte, error) {
	key, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return nil, fmt.Errorf("--hmac-key must be valid hex")
	}
	if len(key) == 0 {
		return nil, fmt.Errorf("--hmac-key must not be empty")
	}
	return key, nil
}

func buildPushMeta(b *bundle.Bundle, bundleBytes []byte, bundlePath string) push.PushMeta {
	digest := sha256.Sum256(bundleBytes)
	meta := push.PushMeta{
		Digest:        hex.EncodeToString(digest[:]),
		SealTimestamp: inferSealTimestamp(b, bundlePath),
		ProfileID:     inferPushProfileID(b, bundlePath),
	}
	if manifest := b.Manifest(); manifest != nil {
		meta.BundleID = manifest.BundleID
	}
	return meta
}

func inferSealTimestamp(b *bundle.Bundle, bundlePath string) time.Time {
	for i := len(b.Records) - 1; i >= 0; i-- {
		ts := strings.TrimSpace(b.Records[i].Event.Timestamp)
		if ts == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, ts)
		if err == nil {
			return parsed.UTC()
		}
	}

	if manifest := b.Manifest(); manifest != nil {
		parsed, err := time.Parse(time.RFC3339, manifest.CreatedAt)
		if err == nil {
			return parsed.UTC()
		}
	}

	info, err := os.Stat(filepath.Clean(bundlePath))
	if err == nil {
		return info.ModTime().UTC()
	}

	return time.Now().UTC()
}

func inferPushProfileID(b *bundle.Bundle, bundlePath string) string {
	report := verifypkg.Verify(b, bundlePath, "")
	if len(report.Profiles) == 0 {
		return ""
	}
	return report.Profiles[0].ProfileID
}
