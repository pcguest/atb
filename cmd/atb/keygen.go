// SPDX-License-Identifier: MIT
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var errKeygenHelp = errors.New("keygen help requested")

type keygenConfig struct {
	OutDir string
}

func cmdKeygen() {
	os.Exit(runKeygen(os.Args[2:], os.Stdout, os.Stderr))
}

func runKeygen(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseKeygenArgs(args)
	if err != nil {
		if errors.Is(err, errKeygenHelp) {
			printKeygenCommandUsage(stdout)
			return exitSuccess
		}
		fmt.Fprintf(stderr, "atb keygen: %v\n", err)
		printKeygenCommandUsage(stderr)
		return exitUserError
	}

	if isRepoRoot(cfg.OutDir) {
		fmt.Fprintln(stderr, "atb keygen: warning: writing key to the repository root; ensure .gitignore covers atb-key.pem and atb-key.pub.pem before committing")
	}

	result, err := generateKeypair(cfg)
	if err != nil {
		exitCode := exitUserError
		if errors.Is(err, os.ErrPermission) {
			exitCode = exitSystemError
		}
		fmt.Fprintf(stderr, "atb keygen: %v\n", err)
		return exitCode
	}

	fmt.Fprintln(stdout, "Generated Ed25519 keypair")
	fmt.Fprintf(stdout, "  private key: %s\n", result.PrivateKeyPath)
	fmt.Fprintf(stdout, "  public key:  %s\n", result.PublicKeyPath)
	fmt.Fprintln(stdout, "Keep atb-key.pem secret. Add it to .gitignore.")
	return exitSuccess
}

type keygenResult struct {
	PrivateKeyPath string
	PublicKeyPath  string
}

func parseKeygenArgs(args []string) (keygenConfig, error) {
	cfg := keygenConfig{OutDir: "."}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errKeygenHelp
		case arg == "--out-dir":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --out-dir")
			}
			i++
			cfg.OutDir = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--out-dir="):
			cfg.OutDir = strings.TrimSpace(strings.TrimPrefix(arg, "--out-dir="))
		case strings.HasPrefix(arg, "-"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			return cfg, fmt.Errorf("unexpected argument %q", arg)
		}
	}

	if cfg.OutDir == "" {
		return cfg, fmt.Errorf("--out-dir cannot be empty")
	}
	cfg.OutDir = filepath.Clean(cfg.OutDir)
	return cfg, nil
}

func printKeygenCommandUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb keygen [--out-dir <dir>]")
}

func generateKeypair(cfg keygenConfig) (keygenResult, error) {
	result := keygenResult{
		PrivateKeyPath: displayOutputPath(cfg.OutDir, "atb-key.pem"),
		PublicKeyPath:  displayOutputPath(cfg.OutDir, "atb-key.pub.pem"),
	}

	if err := os.MkdirAll(cfg.OutDir, 0750); err != nil { // #nosec G301 -- CLI-created directory; tightened permissions
		return result, fmt.Errorf("create output directory: %w", err)
	}

	privateKeyPath := filepath.Join(cfg.OutDir, "atb-key.pem")
	publicKeyPath := filepath.Join(cfg.OutDir, "atb-key.pub.pem")

	for _, path := range []string{privateKeyPath, publicKeyPath} {
		if _, err := os.Stat(path); err == nil {
			return result, fmt.Errorf("refusing to overwrite existing key file %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return result, fmt.Errorf("check key output path %s: %w", path, err)
		}
	}

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return result, fmt.Errorf("generate Ed25519 keypair: %w", err)
	}

	privatePEM, err := marshalEd25519PrivateKeyPEM(privateKey)
	if err != nil {
		return result, err
	}
	publicPEM, err := marshalEd25519PublicKeyPEM(publicKey)
	if err != nil {
		return result, err
	}

	if err := os.WriteFile(privateKeyPath, privatePEM, 0600); err != nil { // #nosec G304 G703 -- privateKeyPath is the cleaned user-selected --out-dir plus a fixed filename
		return result, fmt.Errorf("write private key: %w", err)
	}
	if err := os.WriteFile(publicKeyPath, publicPEM, 0600); err != nil { // #nosec G304 G703 -- publicKeyPath is the cleaned user-selected --out-dir plus a fixed filename
		return result, fmt.Errorf("write public key: %w", err)
	}

	// Belt-and-braces: confirm the umask did not widen the private key mode (Unix only).
	if runtime.GOOS != "windows" {
		if info, err := os.Stat(privateKeyPath); err == nil && info.Mode().Perm() != 0600 {
			return result, fmt.Errorf("keygen: private key was written with mode %o; expected 0600", info.Mode().Perm())
		}
	}

	return result, nil
}

// isRepoRoot reports whether dir contains a go.mod file, i.e. is the working
// tree root of a Go module. Used to warn when keygen would write keys into a
// repository checkout.
func isRepoRoot(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}

func marshalEd25519PrivateKeyPEM(privateKey ed25519.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	}), nil
}

func marshalEd25519PublicKeyPEM(publicKey ed25519.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	}), nil
}

func displayOutputPath(dir, name string) string {
	if dir == "." {
		return "." + string(os.PathSeparator) + name
	}
	return filepath.Join(dir, name)
}
