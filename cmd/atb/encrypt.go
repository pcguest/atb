// SPDX-License-Identifier: MIT
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/encrypt"
)

type encryptConfig struct {
	InputPath  string
	OutputPath string
	Password   string
}

func parseEncryptArgs(args []string) (encryptConfig, error) {
	cfg := encryptConfig{
		InputPath: bundle.DefaultPath(),
	}
	var bundlePath string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--output":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --output")
			}
			cfg.OutputPath = args[i+1]
			i++
		case strings.HasPrefix(arg, "--output="):
			cfg.OutputPath = strings.TrimPrefix(arg, "--output=")
		case arg == "--password":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --password")
			}
			cfg.Password = args[i+1]
			i++
		case strings.HasPrefix(arg, "--password="):
			cfg.Password = strings.TrimPrefix(arg, "--password=")
		case strings.HasPrefix(arg, "--"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			if bundlePath != "" {
				return cfg, fmt.Errorf("encrypt accepts at most one bundle path")
			}
			bundlePath = normalizeBundlePath(arg)
		}
	}

	if bundlePath != "" {
		cfg.InputPath = bundlePath
	}
	if cfg.Password == "" {
		cfg.Password = strings.TrimSpace(os.Getenv("ATB_PASSWORD"))
	}
	if cfg.Password == "" {
		return cfg, fmt.Errorf("--password is required (or set ATB_PASSWORD)")
	}
	if cfg.OutputPath == "" {
		cfg.OutputPath = cfg.InputPath + ".enc"
	}
	return cfg, nil
}

func cmdEncrypt() {
	cfg, err := parseEncryptArgs(os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb encrypt: %v\n", err)
		fmt.Fprintln(os.Stderr, "Usage: atb encrypt [bundle_path] [--output <path>] [--password <password>] (or set ATB_PASSWORD)")
		os.Exit(exitUserError)
	}

	b, err := bundle.Load(cfg.InputPath)
	if err != nil {
		exitCode := classifyBundleLoadError(err)
		fmt.Fprintf(os.Stderr, "atb encrypt: load bundle: %v\n", err)
		os.Exit(exitCode)
	}
	if err := b.Verify(); err != nil {
		fmt.Fprintf(os.Stderr, "atb encrypt: verify bundle before encrypt: %v\n", err)
		os.Exit(exitIntegrityFailure)
	}

	plaintext, err := canonicalPayloadFromBundle(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb encrypt: %v\n", err)
		os.Exit(exitSystemError)
	}
	encryptedBytes, err := encrypt.Encrypt(plaintext, cfg.Password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb encrypt: %v\n", err)
		os.Exit(exitSystemError)
	}

	if err := os.MkdirAll(filepath.Dir(cfg.OutputPath), 0750); err != nil { // #nosec G301 -- tightened to 0750 per gosec
		fmt.Fprintf(os.Stderr, "atb encrypt: mkdir output dir: %v\n", err)
		os.Exit(exitSystemError)
	}
	if err := os.WriteFile(cfg.OutputPath, encryptedBytes, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "atb encrypt: write encrypted bundle: %v\n", err)
		os.Exit(exitSystemError)
	}

	fmt.Printf("✓ Encrypted bundle: %s -> %s\n", cfg.InputPath, cfg.OutputPath)
}
