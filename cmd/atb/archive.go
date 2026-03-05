package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	archiveledger "github.com/pcguest/atb/internal/archive"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/hash"
)

type archiveConfig struct {
	BeforeSet bool
	Before    time.Time
	DryRun    bool
}

type archiveSettings struct {
	Cutoff       time.Time
	ArchiveDir   string
	Scope        []string
	CutoffSource string
	CutoffBasis  string
}

type archiveSummary struct {
	Candidates      int
	WouldArchive    int
	Archived        int
	SkippedRecent   int
	SkippedInvalid  int
	SkippedErrors   int
	ArchivedDefault bool
	LedgerHeadHash  string
}

func cmdArchive() {
	cfg, err := parseArchiveArgs(os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb archive: %v\n", err)
		printArchiveUsage()
		os.Exit(exitUserError)
	}

	summary, err := runArchive(time.Now().UTC(), cfg, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb archive: %v\n", err)
		os.Exit(exitSystemError)
	}
	if summary.SkippedInvalid > 0 {
		// Archive command is operationally successful when invalid bundles are skipped.
		// We keep exit code 0 and rely on text output + summary counts for visibility.
	}
}

func parseArchiveArgs(args []string) (archiveConfig, error) {
	cfg := archiveConfig{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--before":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --before (expected YYYY-MM-DD)")
			}
			before, err := parseArchiveBeforeDate(args[i+1])
			if err != nil {
				return cfg, err
			}
			cfg.BeforeSet = true
			cfg.Before = before
			i++
		case strings.HasPrefix(arg, "--before="):
			before, err := parseArchiveBeforeDate(strings.TrimPrefix(arg, "--before="))
			if err != nil {
				return cfg, err
			}
			cfg.BeforeSet = true
			cfg.Before = before
		case arg == "--dry-run":
			cfg.DryRun = true
		case strings.HasPrefix(arg, "--"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			return cfg, fmt.Errorf("unexpected argument %q", arg)
		}
	}
	return cfg, nil
}

func parseArchiveBeforeDate(raw string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --before value %q (expected YYYY-MM-DD)", raw)
	}
	return parsed.UTC(), nil
}

func runArchive(now time.Time, cfg archiveConfig, out io.Writer) (archiveSummary, error) {
	summary := archiveSummary{}
	settings, err := resolveArchiveSettings(now, cfg)
	if err != nil {
		return summary, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return summary, fmt.Errorf("resolve cwd: %w", err)
	}

	candidates, err := discoverArchiveCandidates(settings.Scope, settings.ArchiveDir)
	if err != nil {
		return summary, err
	}
	summary.Candidates = len(candidates)

	ledgerPath := filepath.Join(settings.ArchiveDir, archiveledger.LedgerFile)
	ledgerEntries, err := archiveledger.LoadOrEmpty(ledgerPath)
	if err != nil {
		return summary, fmt.Errorf("load archive ledger: %w", err)
	}
	ledgerHead, err := archiveledger.Verify(ledgerEntries)
	if err != nil {
		return summary, fmt.Errorf("verify archive ledger: %w", err)
	}
	summary.LedgerHeadHash = ledgerHead

	fmt.Fprintf(out, "Archive cutoff (%s): %s\n", settings.CutoffSource, settings.Cutoff.Format(time.RFC3339))
	fmt.Fprintf(out, "Archive scope: %s\n", strings.Join(settings.Scope, ", "))

	defaultBundle := filepath.Clean(bundle.DefaultPath())
	archivedDefault := false
	wouldArchiveDefault := false

	for _, sourcePath := range candidates {
		info, err := os.Stat(sourcePath)
		if err != nil {
			summary.SkippedErrors++
			fmt.Fprintf(out, "- skip %s (stat error: %v)\n", filepath.ToSlash(sourcePath), err)
			continue
		}
		if !info.ModTime().Before(settings.Cutoff) {
			summary.SkippedRecent++
			fmt.Fprintf(out, "- skip %s (not older than cutoff)\n", filepath.ToSlash(sourcePath))
			continue
		}

		b, err := bundle.Load(sourcePath)
		if err != nil {
			summary.SkippedInvalid++
			fmt.Fprintf(out, "- skip %s (load failed: %v)\n", filepath.ToSlash(sourcePath), err)
			continue
		}
		if err := b.Verify(); err != nil {
			summary.SkippedInvalid++
			fmt.Fprintf(out, "- skip %s (verification failed: %v)\n", filepath.ToSlash(sourcePath), err)
			continue
		}

		headHash := hash.GenesisHash
		if len(b.Records) > 0 {
			headHash = b.Records[len(b.Records)-1].Hash
		}
		fileHash, err := computeFileSHA256(sourcePath)
		if err != nil {
			summary.SkippedErrors++
			fmt.Fprintf(out, "- skip %s (sha256 failed: %v)\n", filepath.ToSlash(sourcePath), err)
			continue
		}

		sourceDisplay, destPath, destDisplay, err := buildArchiveDestination(cwd, sourcePath, settings.ArchiveDir, now)
		if err != nil {
			summary.SkippedErrors++
			fmt.Fprintf(out, "- skip %s (destination failed: %v)\n", filepath.ToSlash(sourcePath), err)
			continue
		}

		if cfg.DryRun {
			summary.WouldArchive++
			fmt.Fprintf(out, "~ would archive %s -> %s\n", sourceDisplay, destDisplay)
			if filepath.Clean(sourcePath) == defaultBundle {
				wouldArchiveDefault = true
			}
			continue
		}

		if err := moveFile(sourcePath, destPath); err != nil {
			summary.SkippedErrors++
			fmt.Fprintf(out, "- skip %s (move failed: %v)\n", sourceDisplay, err)
			continue
		}

		entry, err := archiveledger.NextEntry(
			ledgerEntries,
			now.UTC().Format(time.RFC3339),
			sourceDisplay,
			destDisplay,
			fileHash,
			headHash,
		)
		if err != nil {
			summary.SkippedErrors++
			fmt.Fprintf(out, "- skip %s (ledger link failed: %v)\n", sourceDisplay, err)
			_ = moveFile(destPath, sourcePath)
			continue
		}
		if err := archiveledger.Append(ledgerPath, entry); err != nil {
			summary.SkippedErrors++
			fmt.Fprintf(out, "- skip %s (ledger append failed: %v)\n", sourceDisplay, err)
			_ = moveFile(destPath, sourcePath)
			continue
		}
		ledgerEntries = append(ledgerEntries, entry)
		ledgerHead, err = archiveledger.Verify(ledgerEntries)
		if err == nil {
			summary.LedgerHeadHash = ledgerHead
		}

		summary.Archived++
		fmt.Fprintf(out, "✓ archived %s -> %s\n", sourceDisplay, destDisplay)

		if filepath.Clean(sourcePath) == defaultBundle {
			archivedDefault = true
		}
	}

	if cfg.DryRun && wouldArchiveDefault {
		fmt.Fprintln(out, "~ would recreate run.atb/bundle.atb after archiving default bundle")
	}

	if !cfg.DryRun && archivedDefault {
		fresh := bundle.New()
		if err := fresh.Save(bundle.DefaultPath()); err != nil {
			return summary, fmt.Errorf("recreate default bundle: %w", err)
		}
		summary.ArchivedDefault = true
		fmt.Fprintln(out, "✓ recreated empty default bundle at run.atb/bundle.atb")
	}

	fmt.Fprintf(
		out,
		"Archive summary: candidates=%d archived=%d would_archive=%d skipped_recent=%d skipped_invalid=%d skipped_errors=%d\n",
		summary.Candidates,
		summary.Archived,
		summary.WouldArchive,
		summary.SkippedRecent,
		summary.SkippedInvalid,
		summary.SkippedErrors,
	)

	return summary, nil
}

func resolveArchiveSettings(now time.Time, cfg archiveConfig) (archiveSettings, error) {
	policy, err := loadRetentionPolicy(defaultConfigPath())
	if err != nil {
		return archiveSettings{}, fmt.Errorf("load retention policy: %w", err)
	}

	settings := archiveSettings{
		ArchiveDir:  retentionDefaultArchive,
		Scope:       []string{"run.atb/*.atb"},
		CutoffBasis: retentionDefaultCutoff,
	}
	if policy != nil {
		if policy.ArchiveDir != "" {
			settings.ArchiveDir = filepath.Clean(policy.ArchiveDir)
		}
		if len(policy.Scope) > 0 {
			settings.Scope = append([]string(nil), policy.Scope...)
		}
		if policy.CutoffBasis != "" {
			settings.CutoffBasis = policy.CutoffBasis
		}
	}
	if settings.CutoffBasis != retentionDefaultCutoff {
		return archiveSettings{}, fmt.Errorf("unsupported cutoff_basis %q (only %q supported)", settings.CutoffBasis, retentionDefaultCutoff)
	}

	if cfg.BeforeSet {
		settings.Cutoff = cfg.Before
		settings.CutoffSource = "--before"
		return settings, nil
	}

	if policy == nil || policy.Days <= 0 {
		return archiveSettings{}, fmt.Errorf("missing cutoff: provide --before YYYY-MM-DD or set retention days via `atb config retention --days <n>`")
	}
	settings.Cutoff = now.UTC().AddDate(0, 0, -policy.Days)
	settings.CutoffSource = fmt.Sprintf("retention.days=%d", policy.Days)
	return settings, nil
}

func discoverArchiveCandidates(scope []string, archiveDir string) ([]string, error) {
	set := map[string]struct{}{}
	for _, pattern := range scope {
		converted := filepath.FromSlash(pattern)
		matches, err := filepath.Glob(converted)
		if err != nil {
			return nil, fmt.Errorf("invalid archive scope pattern %q: %w", pattern, err)
		}
		for _, m := range matches {
			clean := filepath.Clean(m)
			info, err := os.Stat(clean)
			if err != nil || info.IsDir() {
				continue
			}
			if isPathWithin(clean, archiveDir) {
				continue
			}
			set[clean] = struct{}{}
		}
	}
	paths := make([]string, 0, len(set))
	for p := range set {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths, nil
}

func buildArchiveDestination(cwd, sourcePath, archiveDir string, now time.Time) (string, string, string, error) {
	sourceClean := filepath.Clean(sourcePath)
	sourceAbs := sourceClean
	if !filepath.IsAbs(sourceAbs) {
		sourceAbs = filepath.Join(cwd, sourceAbs)
	}
	rel, err := filepath.Rel(cwd, sourceAbs)
	if err != nil || strings.HasPrefix(rel, "..") {
		rel = filepath.Base(sourceClean)
	}
	sourceDisplay := filepath.ToSlash(filepath.Clean(rel))

	destPath := filepath.Join(
		archiveDir,
		fmt.Sprintf("%04d", now.UTC().Year()),
		fmt.Sprintf("%02d", int(now.UTC().Month())),
		fmt.Sprintf("%02d", now.UTC().Day()),
		rel,
	)
	destPath = filepath.Clean(destPath)
	destDisplay := filepath.ToSlash(destPath)

	return sourceDisplay, destPath, destDisplay, nil
}

func computeFileSHA256(path string) (string, error) {
	f, err := os.Open(filepath.Clean(path)) // #nosec G304 -- caller provides local bundle path
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func moveFile(sourcePath, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0750); err != nil { // #nosec G301 -- project-local archive dir
		return fmt.Errorf("mkdir archive destination: %w", err)
	}
	renameErr := os.Rename(filepath.Clean(sourcePath), filepath.Clean(destPath))
	if renameErr == nil {
		return nil
	}

	// Fall back to copy+remove for cross-device and platform-specific rename failures.
	if err := copyFile(sourcePath, destPath); err != nil {
		return fmt.Errorf("rename failed (%v); copy source to archive destination: %w", renameErr, err)
	}
	if err := os.Remove(filepath.Clean(sourcePath)); err != nil {
		return fmt.Errorf("remove original after copy: %w", err)
	}
	return nil
}

func copyFile(sourcePath, destPath string) error {
	src, err := os.Open(filepath.Clean(sourcePath)) // #nosec G304 -- caller provides local bundle path
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(filepath.Clean(destPath), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600) // #nosec G304 -- caller provides local destination path
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}

func isPathWithin(candidate, base string) bool {
	candidateClean := filepath.Clean(candidate)
	baseClean := filepath.Clean(base)
	rel, err := filepath.Rel(baseClean, candidateClean)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != "" && !strings.HasPrefix(rel, "..")
}

func printArchiveUsage() {
	fmt.Println("Usage: atb archive [--before YYYY-MM-DD] [--dry-run]")
}
