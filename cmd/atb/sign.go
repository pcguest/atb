package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	signpkg "github.com/pcguest/atb/internal/sign"
)

var errSignHelp = errors.New("sign help requested")
var errSignBundleIntegrity = errors.New("bundle integrity invalid")

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

	rawBundle, err := os.ReadFile(cfg.BundlePath) // #nosec G304 -- bundle path is supplied explicitly by the user
	if err != nil {
		return result, fmt.Errorf("read bundle: %w", err)
	}

	b, err := bundle.Load(cfg.BundlePath)
	if err != nil {
		return result, err
	}
	if err := b.Verify(); err != nil {
		return result, fmt.Errorf("%w: %v", errSignBundleIntegrity, err)
	}

	digest := sha256.Sum256(rawBundle)
	result.BundleHash = hex.EncodeToString(digest[:])

	privateKey, err := loadEd25519PrivateKey(cfg.KeyPath)
	if err != nil {
		return result, err
	}

	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return result, fmt.Errorf("derive public key: loaded key is not Ed25519")
	}

	signature := ed25519.Sign(privateKey, digest[:])
	if len(signature) == 0 {
		return result, fmt.Errorf("sign bundle digest: empty signature")
	}

	if err := b.AppendWithOptions(event.TypeBundleSignature, signpkg.NewBundleSignatureRecord(digest[:], publicKey, signature), &bundle.AppendOptions{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return result, err
	}

	signedRecord := b.Records[len(b.Records)-1]
	if err := appendSignedRecord(cfg, rawBundle, signedRecord); err != nil {
		return result, err
	}

	return result, nil
}

func appendSignedRecord(cfg signConfig, rawBundle []byte, record bundle.Record) error {
	encodedRecord, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode signature record: %w", err)
	}

	info, err := os.Stat(cfg.BundlePath)
	if err != nil {
		return fmt.Errorf("stat source bundle: %w", err)
	}

	payload := make([]byte, 0, len(rawBundle)+len(encodedRecord)+2)
	payload = append(payload, rawBundle...)
	if len(payload) > 0 && payload[len(payload)-1] != '\n' {
		payload = append(payload, '\n')
	}
	payload = append(payload, encodedRecord...)
	payload = append(payload, '\n')

	if err := os.MkdirAll(filepath.Dir(cfg.OutputPath), 0750); err != nil { // #nosec G301 -- CLI-created directory; tightened permissions
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(cfg.OutputPath, payload, info.Mode().Perm()); err != nil { // #nosec G304 G703 -- output path is filepath.Clean-sanitised at parse time and chosen explicitly by the operator
		return fmt.Errorf("write signed bundle: %w", err)
	}
	return nil
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
	case errors.Is(err, errSignBundleIntegrity):
		return exitIntegrityFailure
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
