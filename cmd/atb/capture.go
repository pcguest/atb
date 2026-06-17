// SPDX-License-Identifier: MIT
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

var (
	errCaptureHelp   = errors.New("capture help requested")
	envPrefixPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
)

type captureRunConfig struct {
	BundlePath   string
	SnapshotName string
	EnvPrefix    string
	EnvPrefixSet bool
	ProfileID    string
	Command      []string
	LockWait     time.Duration
}

func cmdCapture() {
	os.Exit(runCapture(os.Args[2:], os.Stdin, os.Stdout, os.Stderr))
}

func runCapture(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "atb capture: missing sub-command")
		printCaptureCommandUsage(stderr)
		return exitUserError
	}

	switch args[0] {
	case "run":
		return runCaptureRun(args[1:], stdin, stdout, stderr)
	case "-h", "--help", "help":
		printCaptureCommandUsage(stdout)
		return exitSuccess
	default:
		fmt.Fprintf(stderr, "atb capture: unknown sub-command %q\n", args[0])
		printCaptureCommandUsage(stderr)
		return exitUserError
	}
}

func printCaptureCommandUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb capture run [--bundle <path>] [--snapshot <name>] [--env-prefix <NAME>] [--profile <id>] [--lock-wait <duration>] -- <command> [args...]")
	fmt.Fprintln(w, "The child receives ATB_BUNDLE_PATH, ATB_CAPTURE_RUN_ID, and ATB_CAPTURE_MODE=run. The child or an ATB SDK integration must emit workflow events; capture run does not inspect arbitrary process traffic.")
}

func runCaptureRun(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	const opTimeout = 5 * time.Minute // Guard against hung bundle file operations.
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	cfg, err := parseCaptureRunArgs(args)
	if err != nil {
		if errors.Is(err, errCaptureHelp) {
			printCaptureCommandUsage(stdout)
			return exitSuccess
		}
		fmt.Fprintf(stderr, "atb capture run: %v\n", err)
		printCaptureCommandUsage(stderr)
		return exitUserError
	}

	resolvedBundlePath, err := filepath.Abs(cfg.BundlePath)
	if err != nil {
		fmt.Fprintf(stderr, "atb capture run: resolve bundle path: %v\n", err)
		return exitSystemError
	}

	runID, err := newCaptureRunID()
	if err != nil {
		fmt.Fprintf(stderr, "atb capture run: generate run id: %v\n", err)
		return exitSystemError
	}

	if err := ensureBundleExists(ctx, resolvedBundlePath, cfg.LockWait, stderr, runID); err != nil {
		if isBundleLocked(err) {
			fmt.Fprintf(stderr, "atb capture run: %s\n", bundleLockedMessage(err))
			return exitLockContention
		}
		fmt.Fprintf(stderr, "atb capture run: prepare bundle: %v\n", err)
		return exitSystemError
	}
	if cfg.EnvPrefixSet && cfg.EnvPrefix == "ATB" {
		fmt.Fprintln(stderr, "atb capture run: warning: --env-prefix=ATB duplicates the canonical prefix")
	}

	cmd := exec.Command(cfg.Command[0], cfg.Command[1:]...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = withCaptureEnvironment(os.Environ(), resolvedBundlePath, runID, cfg.EnvPrefix)

	childExitCode := exitSuccess
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if ws, ok := exitErr.ProcessState.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
				fmt.Fprintf(stderr, "atb capture run: child killed by signal %s\n", ws.Signal())
				childExitCode = 128 + int(ws.Signal())
			} else {
				childExitCode = exitErr.ExitCode()
			}
		} else {
			fmt.Fprintf(stderr, "atb capture run: start child command: %v\n", err)
			return exitSystemError
		}
	}

	if cfg.SnapshotName != "" {
		if _, err := appendSnapshot(snapshotConfig{
			Context:    ctx,
			Name:       cfg.SnapshotName,
			BundlePath: resolvedBundlePath,
			Quiet:      true,
			LockWait:   cfg.LockWait,
		}); err != nil {
			if isBundleLocked(err) {
				fmt.Fprintf(stderr, "atb capture run: %s\n", bundleLockedMessage(err))
				return snapshotExitCode(err)
			}
			fmt.Fprintf(stderr, "atb capture run: append snapshot: %v\n", err)
			return snapshotExitCode(err)
		}
	}

	if childExitCode != exitSuccess {
		return childExitCode
	}
	if strings.TrimSpace(cfg.ProfileID) == "" {
		return exitSuccess
	}

	return runVerifyWithConfigContext(ctx, verifyCLIConfig{
		BundlePath:   resolvedBundlePath,
		ProfileID:    cfg.ProfileID,
		LegacyFormat: formatJSON,
	}, false, stdout, stderr)
}

func parseCaptureRunArgs(args []string) (captureRunConfig, error) {
	cfg := captureRunConfig{BundlePath: bundle.DefaultPath()}
	bundleSet := false
	lockWaitSet := false
	separatorSeen := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			cfg.Command = append([]string(nil), args[i+1:]...)
			separatorSeen = true
			break
		}

		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errCaptureHelp
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
			if err := validateSnapshotName(cfg.SnapshotName); err != nil {
				return cfg, err
			}
		case strings.HasPrefix(arg, "--snapshot="):
			cfg.SnapshotName = strings.TrimSpace(strings.TrimPrefix(arg, "--snapshot="))
			if err := validateSnapshotName(cfg.SnapshotName); err != nil {
				return cfg, err
			}
		case arg == "--env-prefix":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --env-prefix")
			}
			i++
			cfg.EnvPrefix = strings.ToUpper(strings.TrimSpace(args[i]))
			cfg.EnvPrefixSet = true
		case strings.HasPrefix(arg, "--env-prefix="):
			cfg.EnvPrefix = strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(arg, "--env-prefix=")))
			cfg.EnvPrefixSet = true
		case arg == "--profile":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --profile")
			}
			i++
			cfg.ProfileID = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--profile="):
			cfg.ProfileID = strings.TrimSpace(strings.TrimPrefix(arg, "--profile="))
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
		default:
			return cfg, fmt.Errorf("unknown flag %q (use -- before the child command)", arg)
		}
	}

	if !separatorSeen {
		return cfg, fmt.Errorf("missing -- before child command")
	}
	if len(cfg.Command) == 0 {
		return cfg, fmt.Errorf("missing child command")
	}
	if cfg.EnvPrefix != "" && !envPrefixPattern.MatchString(cfg.EnvPrefix) {
		return cfg, fmt.Errorf("invalid --env-prefix %q", cfg.EnvPrefix)
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

func ensureBundleExists(ctx context.Context, path string, lockWait time.Duration, stderr io.Writer, captureRunID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	b, err := bundle.NewWithOptions(bundle.NewOptions{CaptureRunID: captureRunID})
	if err != nil {
		return err
	}
	if err := b.SaveWithRetry(ctx, path, lockWait, bundle.DefaultLockRetryInterval); err != nil {
		if lockWait > 0 && isBundleLocked(err) {
			return lockWaitError(lockWait)
		}
		return err
	}
	if stderr != nil {
		fmt.Fprintf(stderr, "atb: created new bundle at %s\n", path)
	}
	return nil
}

func withCaptureEnvironment(env []string, bundlePath string, runID string, prefix string) []string {
	out := append([]string(nil), env...)
	out = appendCapturePrefixEnvironment(out, "ATB", bundlePath, runID)
	prefix = strings.ToUpper(strings.TrimSpace(prefix))
	if prefix != "" && prefix != "ATB" {
		out = appendCapturePrefixEnvironment(out, prefix, bundlePath, runID)
	}
	return out
}

func appendCapturePrefixEnvironment(env []string, prefix string, bundlePath string, runID string) []string {
	prefix = strings.ToUpper(strings.TrimSpace(prefix))
	return append(env,
		prefix+"_BUNDLE_PATH="+bundlePath,
		prefix+"_CAPTURE_RUN_ID="+runID,
		prefix+"_CAPTURE_MODE=run",
	)
}

func newCaptureRunID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "cap-" + hex.EncodeToString(buf), nil
}
