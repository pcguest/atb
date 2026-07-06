// SPDX-License-Identifier: MIT
package main

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pcguest/atb/internal/compliancepack"
	"github.com/pcguest/atb/internal/mortise"
)

var errComplianceHelp = errors.New("compliance help requested")

type compliancePackConfig struct {
	BundlePath      string
	Profile         string
	Regime          string
	Output          string
	MortiseEndpoint string
}

func cmdCompliance() {
	os.Exit(runCompliance(os.Args[2:], os.Stdout, os.Stderr))
}

func runCompliance(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseComplianceArgs(args)
	if err != nil {
		if errors.Is(err, errComplianceHelp) {
			printComplianceUsage(stdout)
			return exitSuccess
		}
		fmt.Fprintf(stderr, "atb compliance: %v\n", err)
		printComplianceUsage(stderr)
		return exitUserError
	}
	pack, err := compliancepack.Build(
		context.Background(),
		cfg.BundlePath,
		cfg.Profile,
		cfg.Regime,
		version,
	)
	if err != nil {
		fmt.Fprintf(stderr, "atb compliance pack: %v\n", err)
		return exitUserError
	}
	if cfg.MortiseEndpoint != "" {
		var bundleBytes []byte
		for _, artifact := range pack.Files {
			if artifact.Name == "bundle.atb" {
				bundleBytes = append([]byte(nil), artifact.Content...)
				break
			}
		}
		token, _ := mortiseTokenFromEnv()
		client, clientErr := mortise.NewHTTPClient(cfg.MortiseEndpoint, token)
		if clientErr != nil {
			fmt.Fprintf(stderr, "atb compliance pack: Mortise endpoint: %v\n", clientErr)
			return exitUserError
		}
		receipt, sendErr := client.SendBundle(context.Background(), bundleBytes)
		if sendErr != nil {
			fmt.Fprintf(stderr, "atb compliance pack: push to Mortise: %v\n", sendErr)
			return exitSystemError
		}
		pack, err = compliancepack.AddArtifact(pack, compliancepack.File{
			Name:    "mortise/receipt.json",
			Content: receipt.Raw,
		})
		if err != nil {
			fmt.Fprintf(stderr, "atb compliance pack: %v\n", err)
			return exitSystemError
		}
		fmt.Fprintf(stdout, "lodged bundle with Mortise %s: receipt %s (bundle hash %s)\n",
			cfg.MortiseEndpoint, receipt.ReceiptID, receipt.BundleHash)
	}
	if strings.EqualFold(filepath.Ext(cfg.Output), ".zip") {
		err = writeComplianceZip(cfg.Output, pack)
	} else {
		err = writeComplianceDirectory(cfg.Output, pack)
	}
	if err != nil {
		fmt.Fprintf(stderr, "atb compliance pack: %v\n", err)
		return exitSystemError
	}
	fmt.Fprintf(stdout, "wrote compliance evidence pack %s (%d files)\n", cfg.Output, len(pack.Files))
	return exitSuccess
}

func parseComplianceArgs(args []string) (compliancePackConfig, error) {
	cfg := compliancePackConfig{Regime: compliancepack.RegimeEUAIAct}
	mortiseEndpointFlag := ""
	setMortiseEndpoint := func(flag, value string) error {
		if mortiseEndpointFlag != "" {
			return fmt.Errorf("cannot combine %s with %s", mortiseEndpointFlag, flag)
		}
		mortiseEndpointFlag = flag
		cfg.MortiseEndpoint = strings.TrimSpace(value)
		return nil
	}
	if len(args) == 0 || strings.ToLower(strings.TrimSpace(args[0])) != "pack" {
		return cfg, fmt.Errorf("expected subcommand pack")
	}
	for i := 1; i < len(args); i++ {
		arg := args[i]
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("missing value for %s", arg)
			}
			i++
			return strings.TrimSpace(args[i]), nil
		}
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errComplianceHelp
		case arg == "--bundle":
			value, err := next()
			if err != nil {
				return cfg, err
			}
			cfg.BundlePath = normalizeBundlePath(value)
		case strings.HasPrefix(arg, "--bundle="):
			cfg.BundlePath = normalizeBundlePath(strings.TrimPrefix(arg, "--bundle="))
		case arg == "--profile":
			value, err := next()
			if err != nil {
				return cfg, err
			}
			cfg.Profile = value
		case strings.HasPrefix(arg, "--profile="):
			cfg.Profile = strings.TrimSpace(strings.TrimPrefix(arg, "--profile="))
		case arg == "--regime":
			value, err := next()
			if err != nil {
				return cfg, err
			}
			cfg.Regime = strings.ToLower(value)
		case strings.HasPrefix(arg, "--regime="):
			cfg.Regime = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--regime=")))
		case arg == "--out":
			value, err := next()
			if err != nil {
				return cfg, err
			}
			cfg.Output = filepath.Clean(value)
		case strings.HasPrefix(arg, "--out="):
			cfg.Output = filepath.Clean(strings.TrimSpace(strings.TrimPrefix(arg, "--out=")))
		case arg == "--mortise-endpoint" || arg == "--custos-endpoint":
			value, err := next()
			if err != nil {
				return cfg, err
			}
			if err := setMortiseEndpoint(arg, value); err != nil {
				return cfg, err
			}
		case strings.HasPrefix(arg, "--mortise-endpoint="):
			if err := setMortiseEndpoint("--mortise-endpoint", strings.TrimPrefix(arg, "--mortise-endpoint=")); err != nil {
				return cfg, err
			}
		case strings.HasPrefix(arg, "--custos-endpoint="):
			if err := setMortiseEndpoint("--custos-endpoint", strings.TrimPrefix(arg, "--custos-endpoint=")); err != nil {
				return cfg, err
			}
		default:
			return cfg, fmt.Errorf("unknown argument %q", arg)
		}
	}
	if cfg.BundlePath == "" {
		return cfg, fmt.Errorf("--bundle is required")
	}
	if cfg.Profile == "" {
		return cfg, fmt.Errorf("--profile is required")
	}
	if cfg.Output == "" {
		return cfg, fmt.Errorf("--out is required")
	}
	if cfg.Regime != compliancepack.RegimeEUAIAct {
		return cfg, fmt.Errorf("unsupported --regime %q", cfg.Regime)
	}
	return cfg, nil
}

func writeComplianceDirectory(output string, pack compliancepack.Pack) error {
	if err := os.MkdirAll(output, 0o750); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	root, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	for _, file := range pack.Files {
		target := filepath.Join(root, filepath.FromSlash(file.Name))
		cleanTarget := filepath.Clean(target)
		if cleanTarget != root && !strings.HasPrefix(cleanTarget, root+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe pack path %q", file.Name)
		}
		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(cleanTarget, file.Content, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func writeComplianceZip(output string, pack compliancepack.Pack) error {
	if err := os.MkdirAll(filepath.Dir(output), 0o750); err != nil {
		return err
	}
	file, err := os.OpenFile(
		filepath.Clean(output), // #nosec G304 -- explicit CLI output path
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o600,
	)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := zip.NewWriter(file)
	for _, artifact := range pack.Files {
		header := &zip.FileHeader{
			Name:     filepath.ToSlash(artifact.Name),
			Method:   zip.Deflate,
			Modified: pack.GeneratedAt,
		}
		entry, createErr := writer.CreateHeader(header)
		if createErr != nil {
			_ = writer.Close()
			return createErr
		}
		if _, writeErr := entry.Write(artifact.Content); writeErr != nil {
			_ = writer.Close()
			return writeErr
		}
	}
	return writer.Close()
}

func printComplianceUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb compliance pack --bundle <path> --profile <id-or-path> --regime eu-ai-act --out <directory-or-pack.zip> [--mortise-endpoint <url>]")
}
