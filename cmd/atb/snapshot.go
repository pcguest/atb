package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

var errSnapshotHelp = errors.New("snapshot help requested")

const snapshotUsageLine = "Usage: atb snapshot <name> [--dry-run] [--format text|json]"

type snapshotConfig struct {
	Name       string
	BundlePath string
	Quiet      bool
	Format     string
	DryRun     bool
}

type snapshotEventData struct {
	Name        string `json:"name"`
	BundleHash  string `json:"bundle_hash"`
	RecordCount int    `json:"record_count"`
	SnapshotAt  string `json:"snapshot_at"`
}

type snapshotResult struct {
	Data snapshotEventData
	Last bundle.Record
}

func runSnapshot(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseSnapshotArgs(args)
	if err != nil {
		if errors.Is(err, errSnapshotHelp) {
			fmt.Fprintln(stdout, snapshotUsageLine)
			return exitSuccess
		}
		if cfg.Format == verifyFormatJSON {
			if err := writeSnapshotJSON(stdout, mutationResult{
				Status:   "error",
				Action:   "snapshot",
				DryRun:   cfg.DryRun,
				Path:     cfg.BundlePath,
				Error:    err.Error(),
				ExitCode: exitUserError,
			}, stderr); err != nil {
				return exitSystemError
			}
			return exitUserError
		}
		if !cfg.Quiet {
			fmt.Fprintf(stderr, "atb snapshot: %v\n", err)
			fmt.Fprintln(stderr, snapshotUsageLine)
		}
		return exitUserError
	}

	result, err := appendSnapshot(cfg)
	if err != nil {
		exitCode := exitSystemError
		var loadErr mutationLoadError
		if errors.As(err, &loadErr) {
			exitCode = classifyBundleLoadError(err)
		}
		if cfg.Format == verifyFormatJSON {
			if err := writeSnapshotJSON(stdout, mutationResult{
				Status:   "error",
				Action:   "snapshot",
				DryRun:   cfg.DryRun,
				Path:     cfg.BundlePath,
				Error:    err.Error(),
				ExitCode: exitCode,
			}, stderr); err != nil {
				return exitSystemError
			}
			return exitCode
		}
		if !cfg.Quiet {
			fmt.Fprintf(stderr, "atb snapshot: %v\n", err)
		}
		return exitCode
	}

	if cfg.Format == verifyFormatJSON {
		action := "snapshot"
		message := "snapshot appended"
		if cfg.DryRun {
			action = "preview_snapshot"
			message = "snapshot would be appended"
		}
		if err := writeSnapshotJSON(stdout, mutationResult{
			Status:    "ok",
			Action:    action,
			DryRun:    cfg.DryRun,
			Path:      cfg.BundlePath,
			Sequence:  result.Last.Event.Sequence,
			EventType: result.Last.Event.Type,
			Hash:      result.Last.Hash,
			Message:   message,
		}, stderr); err != nil {
			return exitSystemError
		}
		return exitSuccess
	}

	if !cfg.Quiet {
		fmt.Fprintf(stdout, "snapshot %s  records=%d  hash=%s...\n", cfg.Name, result.Data.RecordCount, shortSnapshotHash(result.Data.BundleHash))
	}
	return exitSuccess
}

func parseSnapshotArgs(args []string) (snapshotConfig, error) {
	cfg := snapshotConfig{
		BundlePath: bundle.DefaultPath(),
		Format:     verifyFormatText,
	}
	nameSet := false
	bundlePathSet := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errSnapshotHelp
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
		case arg == "--quiet":
			cfg.Quiet = true
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
			if nameSet {
				return cfg, fmt.Errorf("expected exactly one snapshot name")
			}
			cfg.Name = strings.TrimSpace(arg)
			nameSet = true
		}
	}

	if !nameSet || cfg.Name == "" {
		return cfg, fmt.Errorf("snapshot name cannot be empty")
	}
	if cfg.Format != verifyFormatText && cfg.Format != verifyFormatJSON {
		return cfg, fmt.Errorf("invalid format %q (expected text|json)", cfg.Format)
	}
	return cfg, nil
}

func appendSnapshot(cfg snapshotConfig) (snapshotResult, error) {
	b, err := loadSnapshotBundle(cfg.BundlePath)
	if err != nil {
		return snapshotResult{}, err
	}

	snapshotAt := time.Now().UTC().Format(time.RFC3339)
	bundleHash, err := verifypkg.SnapshotBundleHash(b.Records)
	if err != nil {
		return snapshotResult{}, fmt.Errorf("compute snapshot bundle hash: %w", err)
	}
	data := snapshotEventData{
		Name:        cfg.Name,
		BundleHash:  bundleHash,
		RecordCount: len(b.Records),
		SnapshotAt:  snapshotAt,
	}

	if err := b.AppendWithOptions(event.TypeSnapshot, data, &bundle.AppendOptions{
		Timestamp: snapshotAt,
	}); err != nil {
		return snapshotResult{}, err
	}

	last := b.Records[len(b.Records)-1]
	if cfg.DryRun {
		return snapshotResult{Data: data, Last: last}, nil
	}
	if err := b.Save(cfg.BundlePath); err != nil {
		return snapshotResult{}, fmt.Errorf("save: %w", err)
	}
	return snapshotResult{Data: data, Last: last}, nil
}

func loadSnapshotBundle(path string) (*bundle.Bundle, error) {
	b, err := bundle.Load(path)
	if err == nil {
		return b, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return bundle.New()
	}
	return nil, mutationLoadError{err: err}
}

func serializeBundleSnapshot(b *bundle.Bundle) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, record := range b.Records {
		if err := enc.Encode(record); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func shortSnapshotHash(value string) string {
	if len(value) < 12 {
		return value
	}
	return value[:12]
}

func writeSnapshotJSON(w io.Writer, result mutationResult, stderr io.Writer) error {
	if err := json.NewEncoder(w).Encode(result); err != nil {
		fmt.Fprintf(stderr, "atb snapshot: encode json output: %v\n", err)
		return err
	}
	return nil
}
