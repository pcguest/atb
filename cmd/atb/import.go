// SPDX-License-Identifier: MIT
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	capturepkg "github.com/pcguest/atb/internal/capture"
	"github.com/pcguest/atb/internal/event"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

var errImportHelp = errors.New("import help requested")

// errInputTooLarge is returned by cappedReader when the cumulative number of
// bytes read exceeds the configured cap. It surfaces through scanner.Err() and
// is mapped to exitUserError at the CLI boundary.
var errInputTooLarge = errors.New("import: input exceeds maximum size")

const defaultMaxImportBytes = 256 * 1024 * 1024 // 256 MiB

type importChatlogConfig struct {
	From          string
	InputPath     string
	BundlePath    string
	SnapshotName  string
	Format        string
	MaxInputBytes int64
}

type cappedReader struct {
	r   io.Reader
	n   int64
	max int64
}

func (c *cappedReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	if c.n > c.max {
		return n, errInputTooLarge
	}
	return n, err
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
	_ = stdin
	source := ""
	format := "json"

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--source":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "import: --source is required")
				return exitUserError
			}
			i++
			source = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--source="):
			source = strings.TrimSpace(strings.TrimPrefix(arg, "--source="))
		case arg == "--format":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "import: --format must be json or csv")
				return exitUserError
			}
			i++
			format = strings.ToLower(strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "--format="):
			format = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--format=")))
		default:
			fmt.Fprintln(stderr, "import: --source is required")
			return exitUserError
		}
	}

	if source == "" {
		fmt.Fprintln(stderr, "import: --source is required")
		return exitUserError
	}
	if format != "json" && format != "csv" {
		fmt.Fprintln(stderr, "import: --format must be json or csv")
		return exitUserError
	}

	fmt.Fprintln(stdout, "import: not yet implemented (roadmap Q3 2026)")
	return exitSuccess
}

func printImportCommandUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb import chatlog --from <provider-type> --input <path|-> [--bundle <path>] [--snapshot <name>] [--format text|json] [--max-input-size <bytes>]")
	fmt.Fprintln(w, "  --input -                read chatlog from stdin")
	fmt.Fprintln(w, "  --format text            default; single-line summary on stdout")
	fmt.Fprintln(w, "  --format json            structured JSON object on stdout")
	fmt.Fprintf(w, "  --max-input-size <bytes> reject inputs larger than this (default %d)\n", defaultMaxImportBytes)
	fmt.Fprintln(w, "Supported provider types:")
	fmt.Fprintf(w, "  %s\n", capturepkg.FormatGenericJSONL)
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

	var rawReader io.Reader
	if cfg.InputPath == "-" {
		if stdin == nil {
			return fail(exitUserError, "open input: stdin not available")
		}
		rawReader = stdin
	} else {
		file, openErr := os.Open(filepath.Clean(cfg.InputPath))
		if openErr != nil {
			if errors.Is(openErr, fs.ErrNotExist) {
				fmt.Fprintf(stderr, "atb: input file not found: %s\n", cfg.InputPath)
				if cfg.Format == formatJSON {
					_ = json.NewEncoder(stdout).Encode(importChatlogError{Error: fmt.Sprintf("input file not found: %s", cfg.InputPath)})
				}
				return exitUserError
			}
			fmt.Fprintf(stderr, "atb: cannot open input file: %v\n", openErr)
			if cfg.Format == formatJSON {
				_ = json.NewEncoder(stdout).Encode(importChatlogError{Error: fmt.Sprintf("cannot open input file: %v", openErr)})
			}
			return exitSystemError
		}
		defer file.Close()
		rawReader = file
	}

	// Size cap is enforced by the capture package parser before any bundle IO;
	// cappedReader adds a total-input cap so a stream of small lines cannot
	// exhaust memory or disk.
	reader := &cappedReader{r: rawReader, max: cfg.MaxInputBytes}

	messages, err := capturepkg.ParseChatlog(cfg.From, reader)
	if err != nil {
		switch {
		case errors.Is(err, errInputTooLarge):
			return fail(exitUserError, fmt.Sprintf("input exceeds maximum size (%d bytes); use --max-input-size to increase", cfg.MaxInputBytes))
		case errors.Is(err, capturepkg.ErrUnsupportedProvider), errors.Is(err, capturepkg.ErrMalformedChatlog):
			return fail(exitUserError, err.Error())
		default:
			return fail(exitSystemError, err.Error())
		}
	}
	mapped, err := capturepkg.MapMessagesToEvents(messages)
	if errors.Is(err, capturepkg.ErrUnknownTurn) {
		var unknown []int
		messages, unknown = capturepkg.FilterKnownTurns(messages)
		for _, index := range unknown {
			fmt.Fprintf(stderr, "skipping unrecognised turn %d\n", index)
		}
		mapped, err = capturepkg.MapMessagesToEvents(messages)
		if err == nil {
			mapped.SkippedRecords += len(unknown)
		}
	}
	if err != nil {
		if errors.Is(err, capturepkg.ErrMalformedChatlog) {
			return fail(exitUserError, err.Error())
		}
		return fail(exitSystemError, err.Error())
	}

	if cfg.SnapshotName != "" {
		if err := validateSnapshotName(cfg.SnapshotName); err != nil {
			return fail(exitUserError, err.Error())
		}
	}

	b, created, err := loadSnapshotBundle(ctx, cfg.BundlePath, false)
	if err != nil {
		var loadErr mutationLoadError
		if errors.As(err, &loadErr) {
			return fail(classifyBundleLoadError(err), err.Error())
		}
		return fail(exitSystemError, err.Error())
	}
	if created {
		fmt.Fprintf(stderr, "atb: created new bundle at %s\n", cfg.BundlePath)
	}

	totalEvents := len(mapped.Events)
	written, err := capturepkg.AppendEventsToBundleInMemory(b, mapped.Events)
	if err != nil {
		return fail(exitSystemError, fmt.Sprintf("failed appending event %d/%d: %v", written+1, totalEvents, err))
	}

	if cfg.SnapshotName != "" {
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
	}

	if err := b.Save(ctx, cfg.BundlePath); err != nil {
		if isBundleLocked(err) {
			return fail(exitLockContention, bundleLockedMessage(err))
		}
		return fail(exitSystemError, fmt.Sprintf("save: %v", err))
	}

	if cfg.Format == formatJSON {
		result := importChatlogResult{
			EventsWritten:    written,
			SkippedRecords:   mapped.SkippedRecords,
			BundlePath:       cfg.BundlePath,
			SnapshotAppended: cfg.SnapshotName != "",
			SnapshotName:     cfg.SnapshotName,
		}
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			fmt.Fprintf(stderr, "atb import chatlog: encode json: %v\n", err)
			return exitSystemError
		}
		return exitSuccess
	}

	fmt.Fprintf(stdout, "imported: %d events into %s", written, cfg.BundlePath)
	if mapped.SkippedRecords > 0 {
		fmt.Fprintf(stdout, " (%d source records skipped)", mapped.SkippedRecords)
	}
	if cfg.SnapshotName != "" {
		fmt.Fprintf(stdout, "; snapshot %s appended", cfg.SnapshotName)
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
