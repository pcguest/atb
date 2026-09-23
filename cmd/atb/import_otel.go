// SPDX-License-Identifier: MIT
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/acquisition"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

// importOTelConfig holds the parsed flags for `atb import otel`.
type importOTelConfig struct {
	InputPath      string
	BundlePath     string
	SnapshotName   string
	Format         string
	MaxInputBytes  int64
	Continue       bool
	Reconcile      bool
	CheckpointPath string
}

type importOTelResult struct {
	EventsWritten    int    `json:"events_written"`
	SpansSkipped     int    `json:"spans_skipped"`
	BundlePath       string `json:"bundle_path"`
	SnapshotAppended bool   `json:"snapshot_appended"`
	SnapshotName     string `json:"snapshot_name,omitempty"`
}

type importOTelError struct {
	Error         string `json:"error"`
	EventsWritten int    `json:"events_written"`
}

// runImportOTel ingests an OTLP/JSON trace export into a bundle using the
// acquisition package for checkpoint and reconciliation support.
func runImportOTel(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg, err := parseImportOTelArgs(args)
	if err != nil {
		if errors.Is(err, errImportHelp) {
			printImportCommandUsage(stdout)
			return exitSuccess
		}
		fmt.Fprintf(stderr, "atb import otel: %v\n", err)
		printImportCommandUsage(stderr)
		return exitUserError
	}

	fail := func(code int, msg string) int {
		fmt.Fprintf(stderr, "atb import otel: %s\n", msg)
		if cfg.Format == formatJSON {
			_ = json.NewEncoder(stdout).Encode(importOTelError{Error: msg})
		}
		return code
	}

	// Use the acquisition package for core import logic
	acqOpts := acquisition.ImportOptions{
		Format:         "otel",
		InputPath:      cfg.InputPath,
		BundlePath:     cfg.BundlePath,
		SnapshotName:   cfg.SnapshotName,
		OutputFormat:   cfg.Format,
		MaxInputBytes:  cfg.MaxInputBytes,
		Continue:       cfg.Continue,
		Reconcile:      cfg.Reconcile || cfg.Continue, // --continue implies --reconcile
		CheckpointPath: cfg.CheckpointPath,
	}
	if cfg.InputPath == "-" {
		acqOpts.Stdin = stdin
	}

	result, err := acquisition.ImportOTel(ctx, acqOpts)
	if err != nil {
		// Classify error for appropriate exit code
		errStr := err.Error()
		switch {
		case strings.Contains(errStr, "input file not found"),
			strings.Contains(errStr, "no such file"),
			errors.Is(err, os.ErrNotExist):
			return fail(exitUserError, fmt.Sprintf("input file not found: %s", cfg.InputPath))
		case strings.Contains(errStr, "decode OTLP/JSON"),
			errors.Is(err, acquisition.ErrCheckpointNotFound),
			errors.Is(err, acquisition.ErrCheckpointSourceMismatch),
			errors.Is(err, acquisition.ErrCheckpointAdapterMismatch),
			errors.Is(err, acquisition.ErrNoTranslatableSpans):
			return fail(exitUserError, err.Error())
		default:
			return fail(exitSystemError, err.Error())
		}
	}

	// Handle snapshot if requested (acquisition package doesn't handle snapshots yet)
	if cfg.SnapshotName != "" {
		if err := validateSnapshotName(cfg.SnapshotName); err != nil {
			return fail(exitUserError, err.Error())
		}
		b, err := bundle.Load(result.BundlePath)
		if err != nil {
			return fail(exitSystemError, fmt.Sprintf("load bundle for snapshot: %v", err))
		}
		snapshotAt := time.Now().UTC().Format(time.RFC3339Nano)
		bundleHash, err := verifypkg.SnapshotBundleHash(b.Records)
		if err != nil {
			return fail(snapshotExitCode(err), fmt.Sprintf("events not persisted because snapshot step failed: %v", err))
		}
		data := snapshotEventData{
			Name:        cfg.SnapshotName,
			BundleHash:  bundleHash,
			RecordCount: len(b.Records),
			SnapshotAt:  snapshotAt,
		}
		if err := b.AppendWithOptions(event.TypeSnapshot, data, &bundle.AppendOptions{Timestamp: snapshotAt}); err != nil {
			return fail(snapshotExitCode(err), fmt.Sprintf("events not persisted because snapshot step failed: %v", err))
		}
		if err := b.Save(ctx, result.BundlePath); err != nil {
			if isBundleLocked(err) {
				return fail(exitLockContention, bundleLockedMessage(err))
			}
			return fail(exitSystemError, fmt.Sprintf("save: %v", err))
		}
		result.SnapshotAppended = true
		result.SnapshotName = cfg.SnapshotName
	}

	if cfg.Format == formatJSON {
		jsonResult := importOTelResult{
			EventsWritten:    result.EventsWritten,
			SpansSkipped:     result.SkippedRecords,
			BundlePath:       result.BundlePath,
			SnapshotAppended: result.SnapshotAppended,
			SnapshotName:     result.SnapshotName,
		}
		if err := json.NewEncoder(stdout).Encode(jsonResult); err != nil {
			fmt.Fprintf(stderr, "atb import otel: encode json: %v\n", err)
			return exitSystemError
		}
		return exitSuccess
	}

	fmt.Fprintf(stdout, "imported: %d events into %s", result.EventsWritten, result.BundlePath)
	if result.SkippedRecords > 0 {
		fmt.Fprintf(stdout, " (%d spans skipped)", result.SkippedRecords)
	}
	if result.NewCount > 0 || result.ChangedCount > 0 || result.UnknownCount > 0 {
		fmt.Fprintf(stdout, " (new: %d, changed: %d, unchanged: %d, unknown: %d)", result.NewCount, result.ChangedCount, result.UnchangedCount, result.UnknownCount)
	}
	if result.SnapshotAppended {
		fmt.Fprintf(stdout, "; snapshot %s appended", result.SnapshotName)
	}
	fmt.Fprintln(stdout)
	return exitSuccess
}

func parseImportOTelArgs(args []string) (importOTelConfig, error) {
	cfg := importOTelConfig{
		BundlePath:    bundle.DefaultPath(),
		Format:        formatText,
		MaxInputBytes: defaultMaxImportBytes,
	}
	inputSet := false
	bundleSet := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errImportHelp
		case arg == "--input":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --input")
			}
			i++
			cfg.InputPath = normalizeInputPath(args[i])
			inputSet = true
		case strings.HasPrefix(arg, "--input="):
			cfg.InputPath = normalizeInputPath(strings.TrimPrefix(arg, "--input="))
			inputSet = true
		case arg == "--format":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --format (expected text|json)")
			}
			i++
			cfg.Format = strings.ToLower(strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "--format="):
			cfg.Format = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--format=")))
		case arg == "--max-input-size":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --max-input-size")
			}
			i++
			n, err := strconv.ParseInt(strings.TrimSpace(args[i]), 10, 64)
			if err != nil || n <= 0 {
				return cfg, fmt.Errorf("invalid --max-input-size %q (expected positive integer)", args[i])
			}
			cfg.MaxInputBytes = n
		case strings.HasPrefix(arg, "--max-input-size="):
			v := strings.TrimSpace(strings.TrimPrefix(arg, "--max-input-size="))
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil || n <= 0 {
				return cfg, fmt.Errorf("invalid --max-input-size %q (expected positive integer)", v)
			}
			cfg.MaxInputBytes = n
		case arg == "--bundle":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --bundle")
			}
			if bundleSet {
				return cfg, fmt.Errorf("bundle path already set")
			}
			i++
			cfg.BundlePath = normalizeBundlePath(args[i])
			bundleSet = true
		case strings.HasPrefix(arg, "--bundle="):
			if bundleSet {
				return cfg, fmt.Errorf("bundle path already set")
			}
			cfg.BundlePath = normalizeBundlePath(strings.TrimPrefix(arg, "--bundle="))
			bundleSet = true
		case arg == "--snapshot":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --snapshot")
			}
			i++
			cfg.SnapshotName = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--snapshot="):
			cfg.SnapshotName = strings.TrimSpace(strings.TrimPrefix(arg, "--snapshot="))
		case arg == "--continue":
			cfg.Continue = true
		case arg == "--reconcile":
			cfg.Reconcile = true
		case arg == "--checkpoint":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --checkpoint")
			}
			i++
			cfg.CheckpointPath = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--checkpoint="):
			cfg.CheckpointPath = strings.TrimSpace(strings.TrimPrefix(arg, "--checkpoint="))
		default:
			return cfg, fmt.Errorf("unknown flag %q", arg)
		}
	}

	if !inputSet || cfg.InputPath == "" {
		return cfg, fmt.Errorf("missing required --input <path|->")
	}
	if cfg.Format != formatText && cfg.Format != formatJSON {
		return cfg, fmt.Errorf("invalid --format %q (expected text|json)", cfg.Format)
	}
	return cfg, nil
}
