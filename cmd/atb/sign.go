// SPDX-License-Identifier: MIT
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/signer"
)

var errSignHelp = errors.New("sign help requested")
var errSignBundleIntegrity = errors.New("bundle integrity invalid")

const (
	signBackendLocal    = "local"
	signBackendHTTPHTTP = "https-http"
	signBackendAWSKMS   = "aws-kms"
	signBackendGCPKMS   = "gcp-kms"
	signBackendVault    = "vault"
)

type signConfig struct {
	BundlePath string
	KeyPath    string
	OutputPath string

	Backend       string
	KeyID         string
	Endpoint      string
	APIKey        string
	FallbackLocal bool
	LockWait      time.Duration
	LockWaitSet   bool
}

type signResult struct {
	Algorithm  string
	BundleHash string
	OutputPath string
}

func cmdSign() {
	os.Exit(runSign(os.Args[2:], os.Stdout, os.Stderr))
}

func runSign(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseSignArgs(args)
	if err != nil {
		if errors.Is(err, errSignHelp) {
			printSignCommandUsage(stdout)
			return exitSuccess
		}
		fmt.Fprintf(stderr, "atb sign: %v\n", err)
		printSignCommandUsage(stderr)
		return exitUserError
	}

	// Apply env defaults for any backend/lock flag the user did not pass.
	cfg = applySignBackendEnv(cfg, os.Environ())
	if !cfg.LockWaitSet {
		wait, err := lockWaitFromEnv()
		if err != nil {
			fmt.Fprintf(stderr, "atb sign: %v\n", err)
			printSignCommandUsage(stderr)
			return exitUserError
		}
		cfg.LockWait = wait
	}

	if err := validateSignConfig(&cfg, stderr); err != nil {
		fmt.Fprintf(stderr, "atb sign: %v\n", err)
		printSignCommandUsage(stderr)
		return exitUserError
	}

	result, err := signBundle(cfg, stderr)
	if err != nil {
		exitCode := classifySignError(err)
		if isBundleLocked(err) {
			fmt.Fprintf(stderr, "atb sign: %s\n", bundleLockedMessage(err))
			return exitCode
		}
		fmt.Fprintf(stderr, "atb sign: %v\n", err)
		return exitCode
	}

	fmt.Fprintln(stdout, "Bundle signed")
	fmt.Fprintf(stdout, "  algorithm:   %s\n", result.Algorithm)
	fmt.Fprintf(stdout, "  bundle_hash: %s\n", result.BundleHash)
	fmt.Fprintf(stdout, "  output:      %s\n", result.OutputPath)
	return exitSuccess
}

func parseSignArgs(args []string) (signConfig, error) {
	cfg := signConfig{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errSignHelp
		case arg == "--bundle" || arg == "-b":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for %s", arg)
			}
			i++
			cfg.BundlePath = normalizeBundlePath(args[i])
		case strings.HasPrefix(arg, "--bundle="):
			cfg.BundlePath = normalizeBundlePath(strings.TrimPrefix(arg, "--bundle="))
		case arg == "--key" || arg == "-k":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for %s", arg)
			}
			i++
			cfg.KeyPath = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--key="):
			cfg.KeyPath = strings.TrimSpace(strings.TrimPrefix(arg, "--key="))
		case arg == "--out" || arg == "-o":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for %s", arg)
			}
			i++
			cfg.OutputPath = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--out="):
			cfg.OutputPath = strings.TrimSpace(strings.TrimPrefix(arg, "--out="))
		case arg == "--backend":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --backend")
			}
			i++
			cfg.Backend = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--backend="):
			cfg.Backend = strings.TrimSpace(strings.TrimPrefix(arg, "--backend="))
		case arg == "--key-id":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --key-id")
			}
			i++
			cfg.KeyID = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--key-id="):
			cfg.KeyID = strings.TrimSpace(strings.TrimPrefix(arg, "--key-id="))
		case arg == "--sign-endpoint":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --sign-endpoint")
			}
			i++
			cfg.Endpoint = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--sign-endpoint="):
			cfg.Endpoint = strings.TrimSpace(strings.TrimPrefix(arg, "--sign-endpoint="))
		case arg == "--sign-api-key":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --sign-api-key")
			}
			i++
			cfg.APIKey = args[i] // do not trim — secret may legitimately have whitespace
		case strings.HasPrefix(arg, "--sign-api-key="):
			cfg.APIKey = strings.TrimPrefix(arg, "--sign-api-key=")
		case arg == "--fallback-local":
			cfg.FallbackLocal = true
		case arg == "--lock-wait":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --lock-wait")
			}
			i++
			wait, err := parseLockWaitDuration(args[i])
			if err != nil {
				return cfg, fmt.Errorf("invalid --lock-wait: %w", err)
			}
			cfg.LockWait = wait
			cfg.LockWaitSet = true
		case strings.HasPrefix(arg, "--lock-wait="):
			wait, err := parseLockWaitDuration(strings.TrimPrefix(arg, "--lock-wait="))
			if err != nil {
				return cfg, fmt.Errorf("invalid --lock-wait: %w", err)
			}
			cfg.LockWait = wait
			cfg.LockWaitSet = true
		case strings.HasPrefix(arg, "-"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			return cfg, fmt.Errorf("unexpected argument %q", arg)
		}
	}

	if cfg.BundlePath == "" {
		return cfg, fmt.Errorf("--bundle is required")
	}
	if cfg.KeyPath != "" {
		cfg.KeyPath = filepath.Clean(cfg.KeyPath)
	}
	if cfg.OutputPath == "" {
		cfg.OutputPath = cfg.BundlePath
	} else {
		cfg.OutputPath = filepath.Clean(cfg.OutputPath)
	}
	return cfg, nil
}

// applySignBackendEnv fills in any backend-selection field that was not
// supplied via flags from the corresponding ATB_SIGN_* environment
// variable. Flags always override env. env is the os.Environ slice (or a
// fake injected by tests).
func applySignBackendEnv(cfg signConfig, env []string) signConfig {
	lookup := func(key string) (string, bool) {
		prefix := key + "="
		for _, kv := range env {
			if strings.HasPrefix(kv, prefix) {
				return kv[len(prefix):], true
			}
		}
		return "", false
	}

	if cfg.Backend == "" {
		if v, ok := lookup("ATB_SIGN_BACKEND"); ok {
			cfg.Backend = strings.TrimSpace(v)
		}
	}
	if cfg.KeyID == "" {
		if v, ok := lookup("ATB_SIGN_KEY_ID"); ok {
			cfg.KeyID = strings.TrimSpace(v)
		}
	}
	if cfg.Endpoint == "" {
		if v, ok := lookup("ATB_SIGN_ENDPOINT"); ok {
			cfg.Endpoint = strings.TrimSpace(v)
		}
	}
	if cfg.APIKey == "" {
		if v, ok := lookup("ATB_SIGN_API_KEY"); ok {
			cfg.APIKey = v
		}
	}
	if !cfg.FallbackLocal {
		if v, ok := lookup("ATB_SIGN_FALLBACK_LOCAL"); ok {
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "1", "true", "yes", "on":
				cfg.FallbackLocal = true
			}
		}
	}
	if cfg.Backend == "" {
		cfg.Backend = signBackendLocal
	}
	return cfg
}

func validateSignConfig(cfg *signConfig, stderr io.Writer) error {
	switch cfg.Backend {
	case signBackendLocal:
		if cfg.KeyPath == "" {
			return fmt.Errorf("--key is required for backend=%s", signBackendLocal)
		}
	case signBackendHTTPHTTP:
		if cfg.Endpoint == "" {
			return fmt.Errorf("--sign-endpoint is required for backend=%s", signBackendHTTPHTTP)
		}
		if cfg.FallbackLocal && cfg.KeyPath == "" {
			return fmt.Errorf("--key is required when --fallback-local is set")
		}
	case signBackendAWSKMS, signBackendGCPKMS, signBackendVault:
		if cfg.KeyID == "" {
			return fmt.Errorf("--key-id is required for backend=%s", cfg.Backend)
		}
		if cfg.FallbackLocal && cfg.KeyPath == "" {
			return fmt.Errorf("--key is required when --fallback-local is set")
		}
	default:
		return fmt.Errorf(
			"unknown --backend %q (expected: %s, %s, %s, %s, %s)",
			cfg.Backend,
			signBackendLocal,
			signBackendHTTPHTTP,
			signBackendAWSKMS,
			signBackendGCPKMS,
			signBackendVault,
		)
	}
	return nil
}

func printSignCommandUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb sign --bundle <path> [--key <path>] [--out <path>]")
	fmt.Fprintln(w, "             [--backend local|https-http|aws-kms|gcp-kms|vault]")
	fmt.Fprintln(w, "             [--sign-endpoint <url>] [--sign-api-key <token>]")
	fmt.Fprintln(w, "             [--key-id <id>] [--fallback-local] [--lock-wait <duration>]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Backends:")
	fmt.Fprintln(w, "  local       Use a local Ed25519 PEM key (default).")
	fmt.Fprintln(w, "  https-http  POST the bundle digest to a remote signing service.")
	fmt.Fprintln(w, "  aws-kms     Sign with AWS KMS (requires a binary built with -tags awskms).")
	fmt.Fprintln(w, "  gcp-kms     Sign with GCP Cloud KMS (requires a binary built with -tags gcpkms).")
	fmt.Fprintln(w, "  vault       Sign with HashiCorp Vault transit (requires a binary built with -tags vault).")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Environment overrides (used when the matching flag is absent):")
	fmt.Fprintln(w, "  ATB_SIGN_BACKEND, ATB_SIGN_KEY_ID, ATB_SIGN_ENDPOINT,")
	fmt.Fprintln(w, "  ATB_SIGN_API_KEY, ATB_SIGN_FALLBACK_LOCAL=1")
	fmt.Fprintln(w, "  ATB_LOCK_WAIT=5s")
}

func signBundle(cfg signConfig, stderr io.Writer) (signResult, error) {
	result := signResult{
		Algorithm:  "ed25519",
		OutputPath: cfg.OutputPath,
	}

	b, err := bundle.Load(cfg.BundlePath)
	if err != nil {
		return result, err
	}
	if err := b.Verify(); err != nil {
		return result, fmt.Errorf("%w: %v", errSignBundleIntegrity, err)
	}

	primary, fallback, err := buildSigners(cfg)
	if err != nil {
		return result, err
	}

	ctx := context.Background()
	bundleHash, err := bundle.SignToWithSignerRetry(
		ctx,
		cfg.BundlePath,
		cfg.OutputPath,
		primary,
		cfg.LockWait,
		bundle.DefaultLockRetryInterval,
	)
	if err != nil && fallback != nil && !isBundleLocked(err) {
		fmt.Fprintf(stderr, "atb sign: remote backend %q failed (%v), falling back to local\n", cfg.Backend, err)
		bundleHash, err = bundle.SignToWithSignerRetry(
			ctx,
			cfg.BundlePath,
			cfg.OutputPath,
			fallback,
			cfg.LockWait,
			bundle.DefaultLockRetryInterval,
		)
	}
	if cfg.LockWait > 0 && isBundleLocked(err) {
		return result, lockWaitError(cfg.LockWait)
	}
	if err != nil {
		return result, err
	}
	result.BundleHash = bundleHash

	return result, nil
}

// buildSigners returns the primary signer and, when fallback is enabled
// and the primary is non-local, a secondary local signer that records its
// backend as "local:fallback:<original-backend>" so the audit trail
// captures the deviation.
func buildSigners(cfg signConfig) (primary signer.Signer, fallback signer.Signer, err error) {
	switch cfg.Backend {
	case signBackendLocal:
		priv, err := loadEd25519PrivateKey(cfg.KeyPath)
		if err != nil {
			return nil, nil, err
		}
		return signer.NewLocalSigner(priv), nil, nil

	case signBackendHTTPHTTP:
		http, err := signer.NewHTTPRemoteSigner(signer.HTTPConfig{
			Endpoint: cfg.Endpoint,
			APIKey:   cfg.APIKey,
			Backend:  signBackendHTTPHTTP,
			KeyID:    cfg.KeyID,
		})
		if err != nil {
			return nil, nil, err
		}
		if !cfg.FallbackLocal {
			return http, nil, nil
		}
		priv, err := loadEd25519PrivateKey(cfg.KeyPath)
		if err != nil {
			return nil, nil, err
		}
		fallback := &fallbackLocalSigner{
			local:           signer.NewLocalSigner(priv),
			intendedBackend: cfg.Backend,
			intendedKeyID:   cfg.KeyID,
		}
		return http, fallback, nil

	case signBackendAWSKMS, signBackendGCPKMS, signBackendVault:
		primary, err := signer.Resolve(context.Background(), cfg.Backend, cfg.KeyID)
		if err != nil {
			return nil, nil, err
		}
		if !cfg.FallbackLocal {
			return primary, nil, nil
		}
		priv, err := loadEd25519PrivateKey(cfg.KeyPath)
		if err != nil {
			return nil, nil, err
		}
		fallback := &fallbackLocalSigner{
			local:           signer.NewLocalSigner(priv),
			intendedBackend: cfg.Backend,
			intendedKeyID:   cfg.KeyID,
		}
		return primary, fallback, nil
	}
	return nil, nil, fmt.Errorf("unknown backend %q", cfg.Backend)
}

// fallbackLocalSigner wraps a LocalSigner and overrides the returned
// backend label to "local:fallback:<intendedBackend>" so the on-disk
// signature record makes the deviation auditable.
type fallbackLocalSigner struct {
	local           *signer.LocalSigner
	intendedBackend string
	intendedKeyID   string
}

func (f *fallbackLocalSigner) Sign(ctx context.Context, digest []byte) (sig, pubKey []byte, keyID, backend, algorithm string, err error) {
	sig, pubKey, _, _, algorithm, err = f.local.Sign(ctx, digest)
	if err != nil {
		return nil, nil, "", "", "", err
	}
	return sig, pubKey, f.intendedKeyID, "local:fallback:" + f.intendedBackend, algorithm, nil
}

func loadEd25519PrivateKey(path string) (ed25519.PrivateKey, error) {
	keyBytes, err := os.ReadFile(path) // #nosec G304 -- key path is supplied explicitly by the user
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, fmt.Errorf("parse private key PEM: malformed PEM")
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key PEM: %w", err)
	}

	privateKey, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("parse private key PEM: key is not Ed25519")
	}
	return privateKey, nil
}

func classifySignError(err error) int {
	switch {
	case isBundleLocked(err):
		return exitLockContention
	case errors.Is(err, os.ErrNotExist):
		return exitUserError
	case errors.Is(err, os.ErrPermission):
		return exitSystemError
	case errors.Is(err, errSignBundleIntegrity):
		return exitIntegrityFailure
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "no signing key found") {
		return exitUserError
	}
	if strings.Contains(msg, "malformed pem") || strings.Contains(msg, "not ed25519") {
		return exitUserError
	}
	if strings.Contains(msg, "private key") {
		return exitUserError
	}
	return exitSystemError
}
