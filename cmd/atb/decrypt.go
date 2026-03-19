package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pcguest/atb/internal/encrypt"
)

type decryptConfig struct {
	InputPath  string
	OutputPath string
	Password   string
}

func parseDecryptArgs(args []string) (decryptConfig, error) {
	cfg := decryptConfig{}
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
			if cfg.InputPath != "" {
				return cfg, fmt.Errorf("decrypt accepts exactly one encrypted bundle path")
			}
			cfg.InputPath = arg
		}
	}

	if cfg.InputPath == "" {
		return cfg, fmt.Errorf("encrypted bundle path is required")
	}
	if cfg.Password == "" {
		cfg.Password = strings.TrimSpace(os.Getenv("ATB_PASSWORD"))
	}
	if cfg.Password == "" {
		return cfg, fmt.Errorf("--password is required (or set ATB_PASSWORD)")
	}
	if cfg.OutputPath == "" {
		cfg.OutputPath = defaultDecryptOutputPath(cfg.InputPath)
	}
	return cfg, nil
}

func defaultDecryptOutputPath(input string) string {
	if strings.HasSuffix(input, ".enc") {
		return strings.TrimSuffix(input, ".enc")
	}
	return input + ".decrypted.atb"
}

func cmdDecrypt() {
	cfg, err := parseDecryptArgs(os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb decrypt: %v\n", err)
		fmt.Fprintln(os.Stderr, "Usage: atb decrypt <encrypted_path> [--output <path>] [--password <password>] (or set ATB_PASSWORD)")
		os.Exit(exitUserError)
	}

	encryptedBytes, err := os.ReadFile(cfg.InputPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "atb decrypt: encrypted bundle not found: %s\n", cfg.InputPath)
			os.Exit(exitUserError)
		}
		fmt.Fprintf(os.Stderr, "atb decrypt: read encrypted bundle: %v\n", err)
		os.Exit(exitSystemError)
	}

	plaintext, err := encrypt.Decrypt(encryptedBytes, cfg.Password)
	if err != nil {
		switch {
		case errors.Is(err, encrypt.ErrInvalidFormat), errors.Is(err, encrypt.ErrUnsupportedVersion):
			fmt.Fprintf(os.Stderr, "atb decrypt: %v\n", err)
			os.Exit(exitUserError)
		case errors.Is(err, encrypt.ErrDecryptFailed):
			fmt.Fprintf(os.Stderr, "atb decrypt: wrong password or tampered ciphertext\n")
			os.Exit(exitIntegrityFailure)
		default:
			fmt.Fprintf(os.Stderr, "atb decrypt: %v\n", err)
			os.Exit(exitSystemError)
		}
	}

	b, err := bundleFromCanonicalPayload(plaintext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb decrypt: %v\n", err)
		os.Exit(exitIntegrityFailure)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.OutputPath), 0750); err != nil { // #nosec G301 -- tightened to 0750 per gosec
		fmt.Fprintf(os.Stderr, "atb decrypt: mkdir output dir: %v\n", err)
		os.Exit(exitSystemError)
	}
	if err := b.Save(cfg.OutputPath); err != nil {
		fmt.Fprintf(os.Stderr, "atb decrypt: save decrypted bundle: %v\n", err)
		os.Exit(exitSystemError)
	}

	fmt.Printf("✓ Decrypted bundle: %s -> %s\n", cfg.InputPath, cfg.OutputPath)
}
