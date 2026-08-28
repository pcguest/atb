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
	"github.com/pcguest/atb/internal/event"
	verifypkg "github.com/pcguest/atb/internal/verify"
	"github.com/pcguest/atb/pkg/otel"
)

// importOTelConfig holds the parsed flags for `atb import otel`.
type importOTelConfig struct {
	InputPath     string
	BundlePath    string
	SnapshotName  string
	Format        string
	MaxInputBytes int64
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

// runImportOTel ingests an OTLP/JSON trace export into a bundle: it decodes the
// payload and translates every span to an ATB event via pkg/otel
// (DecodeTraceJSON -> Receiver.ReceiveJSON), then appends the events with their
// W3C trace linkage preserved. It is the documented OTLP ingest path; the
// This command intentionally accepts files or standard input, not gRPC.
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
				return fail(exitUserError, fmt.Sprintf("input file not found: %s", cfg.InputPath))
			}
			return fail(exitSystemError, fmt.Sprintf("cannot open input file: %v", openErr))
		}
		defer file.Close()
		rawReader = file
	}

	data, err := io.ReadAll(&cappedReader{r: rawReader, max: cfg.MaxInputBytes})
	if err != nil {
		if errors.Is(err, errInputTooLarge) {
			return fail(exitUserError, fmt.Sprintf("input exceeds maximum size (%d bytes); use --max-input-size to increase", cfg.MaxInputBytes))
		}
		return fail(exitSystemError, fmt.Sprintf("read input: %v", err))
	}

	receiver := &otel.Receiver{Translator: otel.DefaultTranslator{}}
	result, err := receiver.ReceiveJSON(ctx, data)
	if err != nil {
		// Malformed OTLP/JSON, or a span that does not map to an ATB event type
		// (the ingest is strict — it records only spans it can attribute).
		return fail(exitUserError, fmt.Sprintf("decode/translate OTLP: %v", err))
	}
	if len(result.Events) == 0 {
		return fail(exitUserError, "no translatable spans found in OTLP payload")
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
		if err := stampManifestProvenance(b, "bundle_provenance", bundle.BundleProvenanceRetrospective); err != nil {
			return fail(exitSystemError, fmt.Sprintf("manifest provenance: %v", err))
		}
	}

	written := 0
	for _, ev := range result.Events {
		if appendErr := b.AppendWithOptions(ev.Type, ev.Data, &bundle.AppendOptions{
			Timestamp:    ev.Timestamp,
			TraceID:      ev.TraceID,
			SpanID:       ev.SpanID,
			ParentSpanID: ev.ParentSpanID,
		}); appendErr != nil {
			return fail(exitSystemError, fmt.Sprintf("failed appending event %d/%d: %v", written+1, len(result.Events), appendErr))
		}
		written++
	}

	if cfg.SnapshotName != "" {
		snapshotAt := time.Now().UTC().Format(time.RFC3339Nano)
		bundleHash, hashErr := verifypkg.SnapshotBundleHash(b.Records)
		if hashErr != nil {
			return fail(snapshotExitCode(hashErr), fmt.Sprintf("events not persisted because snapshot step failed: %v", hashErr))
		}
		snap := snapshotEventData{
			Name:        cfg.SnapshotName,
			BundleHash:  bundleHash,
			RecordCount: len(b.Records),
			SnapshotAt:  snapshotAt,
		}
		if err := b.AppendWithOptions(event.TypeSnapshot, snap, &bundle.AppendOptions{Timestamp: snapshotAt}); err != nil {
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
		res := importOTelResult{
			EventsWritten:    written,
			SpansSkipped:     result.SkippedCount,
			BundlePath:       cfg.BundlePath,
			SnapshotAppended: cfg.SnapshotName != "",
			SnapshotName:     cfg.SnapshotName,
		}
		if err := json.NewEncoder(stdout).Encode(res); err != nil {
			fmt.Fprintf(stderr, "atb import otel: encode json: %v\n", err)
			return exitSystemError
		}
		return exitSuccess
	}

	fmt.Fprintf(stdout, "imported: %d events into %s", written, cfg.BundlePath)
	if result.SkippedCount > 0 {
		fmt.Fprintf(stdout, " (%d spans skipped)", result.SkippedCount)
	}
	if cfg.SnapshotName != "" {
		fmt.Fprintf(stdout, "; snapshot %s appended", cfg.SnapshotName)
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
