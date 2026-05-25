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
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	evidencepkg "github.com/pcguest/atb/internal/evidence"
)

var errEvidenceHelp = errors.New("evidence help requested")

type evidenceConfig struct {
	BundlePath string
	Format     string
}

func cmdEvidence() {
	args := os.Args[2:]
	if len(args) > 0 && args[0] == "pack" {
		os.Exit(runEvidencePack(args[1:], os.Stdout, os.Stderr))
		return
	}
	os.Exit(runEvidence(args, os.Stdout, os.Stderr))
}

func runEvidence(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseEvidenceArgs(args)
	if err != nil {
		if errors.Is(err, errEvidenceHelp) {
			printEvidenceUsage(stdout)
			return exitSuccess
		}
		fmt.Fprintf(stderr, "atb evidence: %v\n", err)
		printEvidenceUsage(stderr)
		return exitUserError
	}

	ev, err := evidencepkg.Build(context.Background(), cfg.BundlePath)
	if err != nil && errors.Is(err, bundle.ErrBundleLocked) {
		fmt.Fprintln(stderr, "atb evidence: bundle is locked; retry after a short delay")
		return exitLockContention
	}
	if err != nil && !ev.Tampered {
		fmt.Fprintf(stderr, "atb evidence: %v\n", err)
		return classifyBundleLoadError(err)
	}

	if writeErr := writeEvidence(stdout, cfg.Format, ev); writeErr != nil {
		fmt.Fprintf(stderr, "atb evidence: write output: %v\n", writeErr)
		return exitSystemError
	}
	if err != nil {
		fmt.Fprintf(stderr, "atb evidence: %v\n", err)
		if ev.Tampered {
			return exitIntegrityFailure
		}
		return classifyBundleLoadError(err)
	}
	return exitSuccess
}

func parseEvidenceArgs(args []string) (evidenceConfig, error) {
	cfg := evidenceConfig{Format: formatText}
	bundlePathSet := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errEvidenceHelp
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
			return cfg, fmt.Errorf("unexpected argument %q", arg)
		}
	}

	if !bundlePathSet || strings.TrimSpace(cfg.BundlePath) == "" {
		return cfg, fmt.Errorf("--bundle is required")
	}
	if cfg.Format != formatText && cfg.Format != formatJSON {
		return cfg, fmt.Errorf("invalid format %q (expected text|json)", cfg.Format)
	}
	return cfg, nil
}

func printEvidenceUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb evidence --bundle <path> [--format text|json]")
	fmt.Fprintln(w, "       atb evidence pack [--output json] <bundle.atb> [<bundle2.atb> ...]")
}

func writeEvidence(w io.Writer, format string, ev evidencepkg.BundleEvidence) error {
	if format == formatJSON {
		data, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	return writeEvidenceText(w, ev)
}

func writeEvidenceText(w io.Writer, ev evidencepkg.BundleEvidence) error {
	absPath := ev.Path
	if absPath == "" {
		absPath = "-"
	} else if resolved, err := filepath.Abs(absPath); err == nil {
		absPath = resolved
	}

	var out strings.Builder
	fmt.Fprintf(&out, "Bundle:    %s\n", absPath)
	fmt.Fprintf(&out, "Tampered:  %t\n", ev.Tampered)
	fmt.Fprintf(
		&out,
		"Manifest:  version=%d bundle_id=%s created_at=%s\n",
		ev.Manifest.Version,
		dash(ev.Manifest.BundleID),
		dash(ev.Manifest.CreatedAt),
	)
	fmt.Fprintf(&out, "Records:   %d\n", ev.RecordCount)
	fmt.Fprintln(&out, "Snapshots:")
	for _, snapshot := range ev.Snapshots {
		fmt.Fprintf(
			&out,
			"  [%d] name=%q hash=%s records=%d at=%s\n",
			snapshot.Sequence,
			dash(snapshot.Name),
			shortEvidenceHash(snapshot.BundleHash),
			snapshot.RecordCount,
			dash(snapshot.SnapshotAt),
		)
	}
	fmt.Fprintln(&out, "Signatures:")
	for _, signature := range ev.Signatures {
		fmt.Fprintf(
			&out,
			"  [%d] backend=%s key_id=%s signed_at=%s pubkey=%s valid=%t\n",
			signature.Sequence,
			dash(signature.Backend),
			dash(signature.KeyID),
			dash(signature.SignedAt),
			shortEvidencePubKey(signature.PubKey),
			signature.Valid,
		)
	}
	_, err := io.WriteString(w, out.String())
	return err
}

func shortEvidenceHash(value string) string {
	if value == "" {
		return "-"
	}
	if len(value) <= 16 {
		return value
	}
	return value[:16] + "…"
}

func shortEvidencePubKey(value string) string {
	if value == "" {
		return "-"
	}
	if len(value) <= 12 {
		return value
	}
	return value[:12] + "…"
}

func dash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
