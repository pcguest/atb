package main

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	anchorpkg "github.com/pcguest/atb/internal/anchor"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

var errSignHelp = errors.New("sign help requested")

type signConfig struct {
	BundlePath string
	KeyPath    string
	OutputPath string
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

	result, err := signBundle(cfg)
	if err != nil {
		exitCode := classifySignError(err)
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
		case strings.HasPrefix(arg, "-"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			return cfg, fmt.Errorf("unexpected argument %q", arg)
		}
	}

	if cfg.BundlePath == "" {
		return cfg, fmt.Errorf("--bundle is required")
	}
	if cfg.KeyPath == "" {
		return cfg, fmt.Errorf("--key is required")
	}
	cfg.KeyPath = filepath.Clean(cfg.KeyPath)
	if cfg.OutputPath == "" {
		cfg.OutputPath = cfg.BundlePath
	} else {
		cfg.OutputPath = filepath.Clean(cfg.OutputPath)
	}
	return cfg, nil
}

func printSignCommandUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb sign --bundle <path> --key <path> [--out <path>]")
}

func signBundle(cfg signConfig) (signResult, error) {
	result := signResult{
		Algorithm:  "ed25519",
		OutputPath: cfg.OutputPath,
	}

	b, err := bundle.Load(cfg.BundlePath)
	if err != nil {
		return result, err
	}

	digest, err := anchorpkg.HashBundle(cfg.BundlePath)
	if err != nil {
		return result, err
	}
	result.BundleHash = hex.EncodeToString(digest)

	privateKey, err := loadEd25519PrivateKey(cfg.KeyPath)
	if err != nil {
		return result, err
	}

	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return result, fmt.Errorf("derive public key: loaded key is not Ed25519")
	}

	signature := ed25519.Sign(privateKey, digest)
	if len(signature) == 0 {
		return result, fmt.Errorf("sign bundle digest: empty signature")
	}

	if err := b.AppendWithOptions(event.TypeBundleSignature, map[string]string{
		"algorithm":   result.Algorithm,
		"public_key":  base64.StdEncoding.EncodeToString(publicKey),
		"signature":   base64.StdEncoding.EncodeToString(signature),
		"bundle_hash": result.BundleHash,
	}, &bundle.AppendOptions{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return result, err
	}

	if err := b.Save(cfg.OutputPath); err != nil {
		return result, fmt.Errorf("save: %w", err)
	}

	return result, nil
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
	case errors.Is(err, os.ErrNotExist):
		return exitUserError
	case errors.Is(err, os.ErrPermission):
		return exitSystemError
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "malformed pem") || strings.Contains(msg, "not ed25519") {
		return exitUserError
	}
	if strings.Contains(msg, "private key") {
		return exitUserError
	}
	return exitSystemError
}
