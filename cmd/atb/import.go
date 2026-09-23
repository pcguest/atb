// SPDX-License-Identifier: MIT
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/acquisition"
	"github.com/pcguest/atb/internal/bundle"
	capturepkg "github.com/pcguest/atb/internal/capture"
	"github.com/pcguest/atb/internal/event"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

var errImportHelp = errors.New("import help requested")

const defaultMaxImportBytes = 256 * 1024 * 1024 // 256 MiB

type importChatlogConfig struct {
	From           string
	InputPath      string
	BundlePath     string
	SnapshotName   string
	Format         string
	MaxInputBytes  int64
	Continue       bool
	Reconcile      bool
	CheckpointPath string
}

type importChatlogResult struct {
	EventsWritten    int    `json:"events_written"`
	SkippedRecords   int    `json:"skipped_records"`
	BundlePath       string `json:"bundle_path"`
	SnapshotAppended bool   `json:"snapshot_appended"`
	SnapshotName     string `json:"snapshot_name,omitempty"`
}

type importChatlogError struct {
	Error         string `json:"error"`
	EventsWritten int    `json:"events_written"`
}

func cmdImport() {
	os.Exit(runImport(os.Args[2:], os.Stdin, os.Stdout, os.Stderr))
}

func runImport(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "chatlog" {
		chatlogArgs := make([]string, 0, len(args)-1)
		chatlogArgs = append(chatlogArgs, args[1:]...)
		fromSet := false
		for i := 0; i < len(chatlogArgs); i++ {
			arg := chatlogArgs[i]
			switch {
			case arg == "--source":
				chatlogArgs[i] = "--input"
			case strings.HasPrefix(arg, "--source="):
				chatlogArgs[i] = "--input=" + strings.TrimPrefix(arg, "--source=")
			case arg == "--from", strings.HasPrefix(arg, "--from="):
				fromSet = true
			}
		}
		if !fromSet {
			chatlogArgs = append([]string{"--from", capturepkg.FormatGenericJSONL}, chatlogArgs...)
		}
		const opTimeout = 5 * time.Minute // Guard against hung bundle file operations.
		ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
		defer cancel()
		return runImportChatlogWithContext(ctx, chatlogArgs, stdin, stdout, stderr)
	}

	if len(args) > 0 && args[0] == "otel" {
		const opTimeout = 5 * time.Minute // Guard against hung bundle file operations.
		ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
		defer cancel()
		return runImportOTel(ctx, args[1:], stdin, stdout, stderr)
	}

	_ = stdin
	_ = stdout
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printImportCommandUsage(stderr)
		if len(args) > 0 {
			return exitSuccess
		}
		return exitUserError
	}
	fmt.Fprintf(stderr, "import: unknown subcommand %q (supported: chatlog, otel)\n", args[0])
	printImportCommandUsage(stderr)
	return exitUserError
}

func printImportCommandUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb import chatlog --from <provider-type> --input <path|-> [--bundle <path>] [--snapshot <name>] [--format text|json] [--max-input-size <bytes>] [--continue] [--reconcile] [--checkpoint <path>]")
	fmt.Fprintln(w, "       atb import otel --input <path|-> [--bundle <path>] [--snapshot <name>] [--format text|json] [--max-input-size <bytes>] [--continue] [--reconcile] [--checkpoint <path>]")
	fmt.Fprintln(w, "  --input -                read input from stdin")
	fmt.Fprintln(w, "  --format text            default; single-line summary on stdout")
	fmt.Fprintln(w, "  --format json            structured JSON object on stdout")
	fmt.Fprintf(w, "  --max-input-size <bytes> reject inputs larger than this (default %d)\n", defaultMaxImportBytes)
	fmt.Fprintln(w, "  --continue               continue from previous checkpoint (implies --reconcile)")
	fmt.Fprintln(w, "  --reconcile              enable reconciliation mode for re-import")
	fmt.Fprintln(w, "  --checkpoint <path>      override default checkpoint path")
	fmt.Fprintln(w, "chatlog provider types:")
	fmt.Fprintf(w, "  %s\n", capturepkg.FormatGenericJSONL)
	fmt.Fprintf(w, "  %s\n", capturepkg.FormatOpenAIJSONL)
	fmt.Fprintln(w, "otel: ingest an OTLP/JSON trace export (file or standard input; not a gRPC collector)")
}

func runImportChatlog(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	const opTimeout = 5 * time.Minute // Guard against hung bundle file operations.
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	return runImportChatlogWithContext(ctx, args, stdin, stdout, stderr)
}

func runImportChatlogWithContext(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg, err := parseImportChatlogArgs(args)
	if err != nil {
		if errors.Is(err, errImportHelp) {
			printImportCommandUsage(stdout)
			return exitSuccess
		}
		fmt.Fprintf(stderr, "atb import chatlog: %v\n", err)
		printImportCommandUsage(stderr)
		return exitUserError
	}

	fail := func(exitCode int, msg string) int {
		fmt.Fprintf(stderr, "atb import chatlog: %s\n", msg)
		if cfg.Format == formatJSON {
			_ = json.NewEncoder(stdout).Encode(importChatlogError{Error: msg, EventsWritten: 0})
		}
		return exitCode
	}

	// Use the acquisition package for core import logic
	acqOpts := acquisition.ImportOptions{
		Format:         cfg.From,
		InputPath:      cfg.InputPath,
		BundlePath:     cfg.BundlePath,
		SnapshotName:   cfg.SnapshotName,
		OutputFormat:   cfg.Format,
		MaxInputBytes:  cfg.MaxInputBytes,
		Continue:       cfg.Continue,
		Reconcile:      cfg.Reconcile || cfg.Continue, // --continue implies --reconcile
		CheckpointPath: cfg.CheckpointPath,
	}

	result, err := acquisition.ImportChatlog(ctx, acqOpts)
	if err != nil {
		// Classify error for appropriate exit code
		errStr := err.Error()
		switch {
		case strings.Contains(errStr, "input file not found"),
			strings.Contains(errStr, "no such file"),
			errors.Is(err, os.ErrNotExist):
			return fail(exitUserError, fmt.Sprintf("input file not found: %s", cfg.InputPath))
		case errors.Is(err, capturepkg.ErrUnsupportedProvider), errors.Is(err, capturepkg.ErrMalformedChatlog):
			return fail(exitUserError, err.Error())
		case errors.Is(err, acquisition.ErrCheckpointNotFound),
			errors.Is(err, acquisition.ErrCheckpointSourceMismatch),
			errors.Is(err, acquisition.ErrCheckpointAdapterMismatch):
			return fail(exitUserError, err.Error())
		default:
			return fail(exitSystemError, err.Error())
		}
	}

	// Print unknown turn messages
	for _, idx := range result.UnknownTurnIndices {
		fmt.Fprintf(stderr, "skipping unrecognised turn %d\n", idx)
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
		jsonResult := importChatlogResult{
			EventsWritten:    result.EventsWritten,
			SkippedRecords:   result.SkippedRecords,
			BundlePath:       result.BundlePath,
			SnapshotAppended: result.SnapshotAppended,
			SnapshotName:     result.SnapshotName,
		}
		if err := json.NewEncoder(stdout).Encode(jsonResult); err != nil {
			fmt.Fprintf(stderr, "atb import chatlog: encode json: %v\n", err)
			return exitSystemError
		}
		return exitSuccess
	}

	fmt.Fprintf(stdout, "imported: %d events into %s", result.EventsWritten, result.BundlePath)
	if result.SkippedRecords > 0 {
		fmt.Fprintf(stdout, " (%d source records skipped)", result.SkippedRecords)
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

func parseImportChatlogArgs(args []string) (importChatlogConfig, error) {
	cfg := importChatlogConfig{
		BundlePath:    bundle.DefaultPath(),
		Format:        formatText,
		MaxInputBytes: defaultMaxImportBytes,
	}
	fromSet := false
	inputSet := false
	bundleSet := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errImportHelp
		case arg == "--from":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --from")
			}
			i++
			cfg.From = strings.TrimSpace(args[i])
			fromSet = true
		case strings.HasPrefix(arg, "--from="):
			cfg.From = strings.TrimSpace(strings.TrimPrefix(arg, "--from="))
			fromSet = true
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

	if !fromSet || cfg.From == "" {
		return cfg, fmt.Errorf("--from is required")
	}
	if !inputSet || cfg.InputPath == "" {
		return cfg, fmt.Errorf("--input is required")
	}
	if cfg.Format != formatText && cfg.Format != formatJSON {
		return cfg, fmt.Errorf("invalid format %q (expected text|json)", cfg.Format)
	}
	return cfg, nil
}

func normalizeInputPath(value string) string {
	v := strings.TrimSpace(value)
	if v == "-" {
		return "-"
	}
	if v == "" {
		return ""
	}
	return filepath.Clean(v)
}
