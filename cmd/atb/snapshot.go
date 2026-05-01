// SPDX-License-Identifier: MIT
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

var errSnapshotHelp = errors.New("snapshot help requested")
var errInvalidOversightNote = errors.New("invalid oversight note")

// errBundleMissing is returned by loadSnapshotBundle when requireExisting is
// true and the bundle path does not exist. It maps to exitUserError.
var errBundleMissing = errors.New("bundle not found")

const snapshotUsageLine = "Usage: atb snapshot <name> [--dry-run] [--format text|json] [--lock-wait <duration>] [--actor-id <id>] [--actor-role <role>] [--oversight-note <text>]\nThe bundle must already exist. Use 'atb init' or 'atb capture run' to create one."

// isBundleLocked returns true if err wraps bundle.ErrBundleLocked.
func isBundleLocked(err error) bool {
	return errors.Is(err, bundle.ErrBundleLocked)
}

func bundleLockedMessage(err error) string {
	message := "bundle is locked by another process; retry after the other operation completes"
	if err != nil && strings.Contains(err.Error(), "locked after waiting") {
		return fmt.Sprintf("%s: %v", message, err)
	}
	return message
}

// validateSnapshotName returns a non-nil error if name is not a valid snapshot name.
// Rules enforced:
//   - Non-empty after strings.TrimSpace
//   - Length <= 128 runes (after trim)
//   - No ASCII control characters (< 0x20 or == 0x7F)
//   - No forward slash, backslash, or null byte
func validateSnapshotName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("snapshot name must not be empty")
	}
	if len([]rune(name)) > 128 {
		return errors.New("snapshot name must be 128 characters or fewer")
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7F || r == '/' || r == '\\' || r == 0x00 {
			return errors.New("snapshot name contains invalid characters (no control chars, /, \\, or newlines)")
		}
	}
	return nil
}

type snapshotConfig struct {
	Context         context.Context
	Name            string
	BundlePath      string
	Quiet           bool
	Format          string
	DryRun          bool
	LockWait        time.Duration
	RequireExisting bool
	Stderr          io.Writer
	ActorID         string
	ActorRole       string
	OversightNote   string
	// snapshotClock optionally overrides the time source used to stamp the
	// snapshot. The zero value means time.Now. Tests inject a fake clock to
	// pin SnapshotAt deterministically.
	snapshotClock func() time.Time
}

type snapshotEventData struct {
	Name        string `json:"name"`
	BundleHash  string `json:"bundle_hash"`
	RecordCount int    `json:"record_count"`
	SnapshotAt  string `json:"snapshot_at"`
	// Oversight fields are optional Article 14 attribution evidence.
	ActorID       string `json:"actor_id,omitempty"`
	ActorRole     string `json:"actor_role,omitempty"`
	OversightNote string `json:"oversight_note,omitempty"`
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
		if cfg.Format == formatJSON {
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

	const opTimeout = 5 * time.Minute // Guard against hung bundle file operations.
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	cfg.Context = ctx
	cfg.RequireExisting = true
	cfg.Stderr = stderr
	result, err := appendSnapshot(cfg)
	if err != nil {
		exitCode := snapshotExitCode(err)
		displayError := err.Error()
		if isBundleLocked(err) {
			displayError = bundleLockedMessage(err)
		}
		if cfg.Format == formatJSON {
			if err := writeSnapshotJSON(stdout, mutationResult{
				Status:   "error",
				Action:   "snapshot",
				DryRun:   cfg.DryRun,
				Path:     cfg.BundlePath,
				Error:    displayError,
				ExitCode: exitCode,
			}, stderr); err != nil {
				return exitSystemError
			}
			return exitCode
		}
		if !cfg.Quiet {
			fmt.Fprintf(stderr, "atb snapshot: %s\n", displayError)
		}
		return exitCode
	}

	if cfg.Format == formatJSON {
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

// snapshotExitCode maps an error returned by appendSnapshot to the
// appropriate CLI exit code. It mirrors the classification in runSnapshot
// and must be kept in sync with it.
func snapshotExitCode(err error) int {
	if err == nil {
		return exitSuccess
	}
	var loadErr mutationLoadError
	switch {
	case isBundleLocked(err):
		return exitLockContention
	case errors.Is(err, errBundleMissing):
		return exitUserError
	case errors.Is(err, errInvalidOversightNote):
		return exitUserError
	case errors.As(err, &loadErr):
		return classifyBundleLoadError(err)
	default:
		return exitSystemError
	}
}

func parseSnapshotArgs(args []string) (snapshotConfig, error) {
	cfg := snapshotConfig{
		BundlePath: bundle.DefaultPath(),
		Format:     formatText,
	}
	nameSet := false
	bundlePathSet := false
	lockWaitSet := false

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
		case arg == "--actor-id":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --actor-id")
			}
			i++
			cfg.ActorID = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--actor-id="):
			cfg.ActorID = strings.TrimSpace(strings.TrimPrefix(arg, "--actor-id="))
		case arg == "--actor-role":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --actor-role")
			}
			i++
			cfg.ActorRole = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--actor-role="):
			cfg.ActorRole = strings.TrimSpace(strings.TrimPrefix(arg, "--actor-role="))
		case arg == "--oversight-note":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --oversight-note")
			}
			i++
			cfg.OversightNote = args[i]
		case strings.HasPrefix(arg, "--oversight-note="):
			cfg.OversightNote = strings.TrimPrefix(arg, "--oversight-note=")
		case arg == "--lock-wait":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --lock-wait")
			}
			i++
			wait, err := parseLockWaitDuration(args[i])
			if err != nil {
				return cfg, fmt.Errorf("invalid --lock-wait: %w", err)
			}
			cfg.LockWait = wait
			lockWaitSet = true
		case strings.HasPrefix(arg, "--lock-wait="):
			wait, err := parseLockWaitDuration(strings.TrimPrefix(arg, "--lock-wait="))
			if err != nil {
				return cfg, fmt.Errorf("invalid --lock-wait: %w", err)
			}
			cfg.LockWait = wait
			lockWaitSet = true
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

	if !nameSet {
		return cfg, fmt.Errorf("snapshot name cannot be empty")
	}
	if err := validateSnapshotName(cfg.Name); err != nil {
		return cfg, err
	}
	if cfg.Format != formatText && cfg.Format != formatJSON {
		return cfg, fmt.Errorf("invalid format %q (expected text|json)", cfg.Format)
	}
	if !lockWaitSet {
		wait, err := lockWaitFromEnv()
		if err != nil {
			return cfg, err
		}
		cfg.LockWait = wait
	}
	return cfg, nil
}

func appendSnapshot(cfg snapshotConfig) (snapshotResult, error) {
	if err := validateSnapshotName(cfg.Name); err != nil {
		return snapshotResult{}, fmt.Errorf("validate: %w", err)
	}
	ctx := cfg.Context
	if ctx == nil {
		ctx = context.Background()
	}

	now := cfg.snapshotClock
	if now == nil {
		now = time.Now
	}
	snapshotAt := now().UTC().Format(time.RFC3339Nano)

	b, created, err := loadSnapshotBundle(ctx, cfg.BundlePath, cfg.RequireExisting)
	if err != nil {
		return snapshotResult{}, err
	}
	if created && cfg.Stderr != nil && !cfg.Quiet {
		fmt.Fprintf(cfg.Stderr, "atb: created new bundle at %s\n", cfg.BundlePath)
	}
	if err := validateOversightNote(cfg.OversightNote); err != nil {
		return snapshotResult{}, err
	}

	bundleHash, err := verifypkg.SnapshotBundleHash(b.Records)
	if err != nil {
		return snapshotResult{}, fmt.Errorf("compute snapshot bundle hash: %w", err)
	}
	data := snapshotEventData{
		Name:          cfg.Name,
		BundleHash:    bundleHash,
		RecordCount:   len(b.Records),
		SnapshotAt:    snapshotAt,
		ActorID:       cfg.ActorID,
		ActorRole:     cfg.ActorRole,
		OversightNote: cfg.OversightNote,
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
	if err := b.SaveWithRetry(ctx, cfg.BundlePath, cfg.LockWait, bundle.DefaultLockRetryInterval); err != nil {
		if cfg.LockWait > 0 && isBundleLocked(err) {
			return snapshotResult{}, lockWaitError(cfg.LockWait)
		}
		return snapshotResult{}, fmt.Errorf("save: %w", err)
	}
	return snapshotResult{Data: data, Last: last}, nil
}

func validateOversightNote(note string) error {
	if note == "" {
		return nil
	}
	if len([]rune(note)) > 512 {
		return fmt.Errorf("%w: oversight note must be 512 characters or fewer", errInvalidOversightNote)
	}
	for _, r := range note {
		switch r {
		case '\t', '\n':
			continue
		case 0:
			return fmt.Errorf("%w: oversight note cannot contain null bytes", errInvalidOversightNote)
		default:
			if unicode.IsControl(r) {
				return fmt.Errorf("%w: oversight note cannot contain control characters other than tab or newline", errInvalidOversightNote)
			}
		}
	}
	return nil
}

func loadSnapshotBundle(ctx context.Context, path string, requireExisting bool) (*bundle.Bundle, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	b, err := bundle.Load(ctx, path)
	if err == nil {
		return b, false, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		if requireExisting {
			return nil, false, fmt.Errorf("%w at %q; create one first with: atb init %s", errBundleMissing, path, path)
		}
		nb, nerr := bundle.New()
		if nerr != nil {
			return nil, false, nerr
		}
		return nb, true, nil
	}
	return nil, false, mutationLoadError{err: err}
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
