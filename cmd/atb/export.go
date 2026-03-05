package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	archiveledger "github.com/pcguest/atb/internal/archive"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/trust"
)

const (
	exportFormatCompliance = "compliance"
)

type exportConfig struct {
	Format string
	Output string
	DryRun bool
}

type exportManifest struct {
	GeneratedAt    string                   `json:"generated_at"`
	ATBVersion     string                   `json:"atb_version"`
	BundleHeadHash string                   `json:"bundle_head_hash,omitempty"`
	RetentionDays  *int                     `json:"retention_days,omitempty"`
	ArchiveCount   int                      `json:"archive_count"`
	LedgerHeadHash string                   `json:"ledger_head_hash,omitempty"`
	IncludedFiles  []string                 `json:"included_files"`
	Verification   exportVerificationStatus `json:"verification"`
	Warnings       []string                 `json:"warnings,omitempty"`
}

type exportVerificationStatus struct {
	ActiveVerified   int  `json:"active_verified"`
	ArchivedVerified int  `json:"archived_verified"`
	LedgerVerified   bool `json:"ledger_verified"`
}

type exportLedgerReport struct {
	Path        string `json:"path,omitempty"`
	Entries     int    `json:"entries"`
	Verified    bool   `json:"verified"`
	HeadHash    string `json:"head_hash,omitempty"`
	GeneratedAt string `json:"generated_at"`
}

type exportBundleInfo struct {
	ZipPath  string
	Source   string
	HeadHash string
	SHA256   string
	Archived bool
}

type exportFileEntry struct {
	ZipPath string
	Data    []byte
}

type exportBuildResult struct {
	Manifest      exportManifest
	Files         []exportFileEntry
	BundleFiles   []exportBundleInfo
	ChecksumLines []string
	OutputPath    string
}

func cmdExport() {
	cfg, err := parseExportArgs(os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb export: %v\n", err)
		printExportUsage()
		os.Exit(exitUserError)
	}

	result, err := buildComplianceExport(time.Now().UTC(), cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb export: %v\n", err)
		exitCode := exitSystemError
		if strings.Contains(strings.ToLower(err.Error()), "verification") {
			exitCode = exitIntegrityFailure
		}
		os.Exit(exitCode)
	}

	if cfg.DryRun {
		fmt.Printf("~ Dry run: would create %s with %d file(s)\n", cfg.Output, len(result.Files))
		for _, path := range result.Manifest.IncludedFiles {
			fmt.Printf("  - %s\n", path)
		}
		if len(result.Manifest.Warnings) > 0 {
			fmt.Println("Warnings:")
			for _, warning := range result.Manifest.Warnings {
				fmt.Printf("  - %s\n", warning)
			}
		}
		fmt.Printf(
			"Verification: active_verified=%d archived_verified=%d ledger_verified=%t\n",
			result.Manifest.Verification.ActiveVerified,
			result.Manifest.Verification.ArchivedVerified,
			result.Manifest.Verification.LedgerVerified,
		)
		return
	}

	if err := writeComplianceZip(result); err != nil {
		fmt.Fprintf(os.Stderr, "atb export: write zip: %v\n", err)
		os.Exit(exitSystemError)
	}
	fmt.Printf("✓ Exported compliance evidence: %s\n", result.OutputPath)
}

func parseExportArgs(args []string) (exportConfig, error) {
	cfg := exportConfig{Format: exportFormatCompliance}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--format":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --format (expected compliance)")
			}
			cfg.Format = strings.ToLower(strings.TrimSpace(args[i+1]))
			i++
		case strings.HasPrefix(arg, "--format="):
			cfg.Format = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--format=")))
		case arg == "--output":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --output")
			}
			cfg.Output = filepath.Clean(args[i+1])
			i++
		case strings.HasPrefix(arg, "--output="):
			cfg.Output = filepath.Clean(strings.TrimSpace(strings.TrimPrefix(arg, "--output=")))
		case arg == "--dry-run":
			cfg.DryRun = true
		case strings.HasPrefix(arg, "--"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			return cfg, fmt.Errorf("unexpected argument %q", arg)
		}
	}
	if cfg.Format != exportFormatCompliance {
		return cfg, fmt.Errorf("unsupported format %q (expected compliance)", cfg.Format)
	}
	if cfg.Output == "" {
		return cfg, fmt.Errorf("--output is required")
	}
	return cfg, nil
}

func buildComplianceExport(now time.Time, cfg exportConfig) (exportBuildResult, error) {
	result := exportBuildResult{OutputPath: cfg.Output}
	manifest := exportManifest{
		GeneratedAt:   now.Format(time.RFC3339),
		ATBVersion:    version,
		IncludedFiles: []string{},
		Verification:  exportVerificationStatus{},
		Warnings:      []string{},
	}

	cwd, err := os.Getwd()
	if err != nil {
		return result, fmt.Errorf("resolve cwd: %w", err)
	}

	activePaths, err := discoverSortedGlob("run.atb/*.atb")
	if err != nil {
		return result, err
	}
	archivedPaths, err := discoverArchivedBundles(retentionDefaultArchive)
	if err != nil {
		return result, err
	}

	bundleInfos := make([]exportBundleInfo, 0, len(activePaths)+len(archivedPaths))
	for _, path := range activePaths {
		info, err := verifyBundleForExport(path)
		if err != nil {
			return result, err
		}
		info.ZipPath = filepath.ToSlash(filepath.Join("evidence", "bundles", "active", filepath.Clean(path)))
		info.Source = path
		info.Archived = false
		bundleInfos = append(bundleInfos, info)
		manifest.Verification.ActiveVerified++
		if filepath.Clean(path) == filepath.Clean(bundle.DefaultPath()) {
			manifest.BundleHeadHash = info.HeadHash
		}
	}
	for _, path := range archivedPaths {
		info, err := verifyBundleForExport(path)
		if err != nil {
			return result, err
		}
		info.ZipPath = filepath.ToSlash(filepath.Join("evidence", "bundles", "archived", filepath.Clean(path)))
		info.Source = path
		info.Archived = true
		bundleInfos = append(bundleInfos, info)
		manifest.Verification.ArchivedVerified++
	}
	manifest.ArchiveCount = manifest.Verification.ArchivedVerified

	files := make([]exportFileEntry, 0)
	for _, info := range bundleInfos {
		data, err := os.ReadFile(info.Source)
		if err != nil {
			return result, fmt.Errorf("read bundle file %s: %w", info.Source, err)
		}
		files = append(files, exportFileEntry{ZipPath: info.ZipPath, Data: data})
		manifest.IncludedFiles = append(manifest.IncludedFiles, info.ZipPath)
	}

	ledgerPath := filepath.Join(retentionDefaultArchive, archiveledger.LedgerFile)
	ledgerReport := exportLedgerReport{GeneratedAt: now.Format(time.RFC3339)}
	if entries, loadErr := archiveledger.Load(ledgerPath); loadErr == nil {
		head, err := archiveledger.Verify(entries)
		if err != nil {
			return result, fmt.Errorf("archive ledger verification failed: %w", err)
		}
		ledgerReport.Path = ledgerPath
		ledgerReport.Verified = true
		ledgerReport.Entries = len(entries)
		ledgerReport.HeadHash = head
		manifest.Verification.LedgerVerified = true
		manifest.LedgerHeadHash = head

		data, err := os.ReadFile(ledgerPath)
		if err != nil {
			return result, fmt.Errorf("read archive ledger: %w", err)
		}
		zipPath := filepath.ToSlash(filepath.Join("evidence", "bundles", "archived", ledgerPath))
		files = append(files, exportFileEntry{ZipPath: zipPath, Data: data})
		manifest.IncludedFiles = append(manifest.IncludedFiles, zipPath)
	} else if !os.IsNotExist(loadErr) {
		return result, fmt.Errorf("load archive ledger: %w", loadErr)
	}

	verifyReportData, err := buildVerifyReport(activePaths, archivedPaths)
	if err != nil {
		return result, err
	}
	verifyReportZip := filepath.ToSlash(filepath.Join("evidence", "reports", "verify.json"))
	files = append(files, exportFileEntry{ZipPath: verifyReportZip, Data: verifyReportData})
	manifest.IncludedFiles = append(manifest.IncludedFiles, verifyReportZip)

	trustReport := trust.BuildReport(cwd, bundle.DefaultPath())
	trustData, err := json.MarshalIndent(trustReport, "", "  ")
	if err != nil {
		return result, fmt.Errorf("marshal trust report: %w", err)
	}
	trustData = append(trustData, '\n')
	trustReportZip := filepath.ToSlash(filepath.Join("evidence", "reports", "trust-report.json"))
	files = append(files, exportFileEntry{ZipPath: trustReportZip, Data: trustData})
	manifest.IncludedFiles = append(manifest.IncludedFiles, trustReportZip)

	ledgerReportData, err := json.MarshalIndent(ledgerReport, "", "  ")
	if err != nil {
		return result, fmt.Errorf("marshal ledger report: %w", err)
	}
	ledgerReportData = append(ledgerReportData, '\n')
	ledgerReportZip := filepath.ToSlash(filepath.Join("evidence", "reports", "archive-ledger.json"))
	files = append(files, exportFileEntry{ZipPath: ledgerReportZip, Data: ledgerReportData})
	manifest.IncludedFiles = append(manifest.IncludedFiles, ledgerReportZip)

	if cfgData, cfgPath, retentionDays, err := loadConfigForExport(); err != nil {
		return result, err
	} else if cfgData != nil {
		zipPath := filepath.ToSlash(filepath.Join("evidence", "config", "atb-config.json"))
		files = append(files, exportFileEntry{ZipPath: zipPath, Data: cfgData})
		manifest.IncludedFiles = append(manifest.IncludedFiles, zipPath)
		manifest.RetentionDays = retentionDays
		_ = cfgPath
	}

	for _, doc := range []string{"docs/spec-v1.0.md", "docs/security.md", "docs/incident-response.md"} {
		if data, err := os.ReadFile(doc); err == nil {
			zipPath := filepath.ToSlash(filepath.Join("evidence", "docs", strings.TrimPrefix(filepath.ToSlash(doc), "docs/")))
			files = append(files, exportFileEntry{ZipPath: zipPath, Data: data})
			manifest.IncludedFiles = append(manifest.IncludedFiles, zipPath)
		} else if os.IsNotExist(err) {
			manifest.Warnings = append(manifest.Warnings, fmt.Sprintf("optional doc missing: %s", doc))
		} else {
			return result, fmt.Errorf("read doc %s: %w", doc, err)
		}
	}

	complianceDocs, err := discoverComplianceDocs("docs/compliance")
	if err != nil {
		return result, err
	}
	if len(complianceDocs) == 0 {
		manifest.Warnings = append(manifest.Warnings, "optional docs/compliance/*.md not found")
	}
	for _, doc := range complianceDocs {
		data, err := os.ReadFile(doc)
		if err != nil {
			return result, fmt.Errorf("read compliance doc %s: %w", doc, err)
		}
		zipPath := filepath.ToSlash(filepath.Join("evidence", "docs", strings.TrimPrefix(filepath.ToSlash(doc), "docs/")))
		files = append(files, exportFileEntry{ZipPath: zipPath, Data: data})
		manifest.IncludedFiles = append(manifest.IncludedFiles, zipPath)
	}

	sort.Strings(manifest.IncludedFiles)
	sort.Slice(files, func(i, j int) bool {
		return files[i].ZipPath < files[j].ZipPath
	})

	checksums := buildChecksumsFile(files)
	checksumsZip := filepath.ToSlash(filepath.Join("evidence", "checksums.sha256"))
	files = append(files, exportFileEntry{ZipPath: checksumsZip, Data: []byte(checksums)})
	manifest.IncludedFiles = append(manifest.IncludedFiles, checksumsZip)

	checksumMeta := buildChecksumMetaFile(files, bundleInfos, manifest.LedgerHeadHash)
	checksumMetaZip := filepath.ToSlash(filepath.Join("evidence", "checksums.chain"))
	files = append(files, exportFileEntry{ZipPath: checksumMetaZip, Data: []byte(checksumMeta)})
	manifest.IncludedFiles = append(manifest.IncludedFiles, checksumMetaZip)
	sort.Strings(manifest.IncludedFiles)

	manifestZip := filepath.ToSlash(filepath.Join("evidence", "manifest.json"))
	manifest.IncludedFiles = append(manifest.IncludedFiles, manifestZip)
	sort.Strings(manifest.IncludedFiles)

	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return result, fmt.Errorf("marshal manifest: %w", err)
	}
	manifestData = append(manifestData, '\n')
	files = append(files, exportFileEntry{ZipPath: manifestZip, Data: manifestData})
	sort.Slice(files, func(i, j int) bool {
		return files[i].ZipPath < files[j].ZipPath
	})

	result.Manifest = manifest
	result.Files = files
	result.BundleFiles = bundleInfos
	result.ChecksumLines = strings.Split(strings.TrimSuffix(checksums, "\n"), "\n")
	return result, nil
}

func buildVerifyReport(activePaths, archivedPaths []string) ([]byte, error) {
	type verifyBundle struct {
		Path     string `json:"path"`
		HeadHash string `json:"head_hash"`
		Archived bool   `json:"archived"`
	}
	report := struct {
		GeneratedAt string         `json:"generated_at"`
		Bundles     []verifyBundle `json:"bundles"`
	}{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Bundles:     make([]verifyBundle, 0, len(activePaths)+len(archivedPaths)),
	}

	for _, path := range activePaths {
		info, err := verifyBundleForExport(path)
		if err != nil {
			return nil, err
		}
		report.Bundles = append(report.Bundles, verifyBundle{Path: path, HeadHash: info.HeadHash, Archived: false})
	}
	for _, path := range archivedPaths {
		info, err := verifyBundleForExport(path)
		if err != nil {
			return nil, err
		}
		report.Bundles = append(report.Bundles, verifyBundle{Path: path, HeadHash: info.HeadHash, Archived: true})
	}

	sort.Slice(report.Bundles, func(i, j int) bool {
		return report.Bundles[i].Path < report.Bundles[j].Path
	})

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal verify report: %w", err)
	}
	data = append(data, '\n')
	return data, nil
}

func verifyBundleForExport(path string) (exportBundleInfo, error) {
	b, err := bundle.Load(path)
	if err != nil {
		return exportBundleInfo{}, fmt.Errorf("bundle verification failed for %s: %w", path, err)
	}
	if err := b.Verify(); err != nil {
		return exportBundleInfo{}, fmt.Errorf("bundle verification failed for %s: %w", path, err)
	}
	head := ""
	if len(b.Records) > 0 {
		head = b.Records[len(b.Records)-1].Hash
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return exportBundleInfo{}, fmt.Errorf("read bundle %s: %w", path, err)
	}
	return exportBundleInfo{
		HeadHash: head,
		SHA256:   sha256Hex(raw),
	}, nil
}

func loadConfigForExport() ([]byte, string, *int, error) {
	configPath := defaultConfigPath()
	raw, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", nil, nil
		}
		return nil, "", nil, fmt.Errorf("read config file: %w", err)
	}
	cfg, err := loadATBConfig(configPath)
	if err != nil {
		return nil, "", nil, err
	}
	var retentionDays *int
	if cfg.Retention != nil && cfg.Retention.Days > 0 {
		d := cfg.Retention.Days
		retentionDays = &d
	}
	return raw, configPath, retentionDays, nil
}

func discoverSortedGlob(pattern string) ([]string, error) {
	matches, err := filepath.Glob(filepath.FromSlash(pattern))
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}
	paths := make([]string, 0, len(matches))
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil || info.IsDir() {
			continue
		}
		paths = append(paths, filepath.Clean(m))
	}
	sort.Strings(paths)
	return paths, nil
}

func discoverArchivedBundles(archiveDir string) ([]string, error) {
	paths := make([]string, 0)
	walkErr := filepath.WalkDir(archiveDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".atb") {
			paths = append(paths, filepath.Clean(path))
		}
		return nil
	})
	if walkErr != nil {
		if os.IsNotExist(walkErr) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("walk archive directory: %w", walkErr)
	}
	sort.Strings(paths)
	return paths, nil
}

func discoverComplianceDocs(path string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(path, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("discover compliance docs: %w", err)
	}
	paths := make([]string, 0, len(matches))
	for _, p := range matches {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			paths = append(paths, filepath.Clean(p))
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func buildChecksumsFile(files []exportFileEntry) string {
	lines := make([]string, 0, len(files))
	for _, f := range files {
		base := strings.ToLower(filepath.Base(f.ZipPath))
		if base == "checksums.sha256" || base == "checksums.chain" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s  %s", sha256Hex(f.Data), f.ZipPath))
	}
	return strings.Join(lines, "\n") + "\n"
}

func buildChecksumMetaFile(files []exportFileEntry, bundleInfos []exportBundleInfo, ledgerHeadHash string) string {
	lines := make([]string, 0, len(files)+len(bundleInfos)+4)
	lines = append(lines, "# ATB compliance export checksum metadata")
	for _, f := range files {
		base := strings.ToLower(filepath.Base(f.ZipPath))
		if base == "checksums.sha256" || base == "checksums.chain" {
			continue
		}
		lines = append(lines, fmt.Sprintf("file:%s  %s", sha256Hex(f.Data), f.ZipPath))
	}
	for _, info := range bundleInfos {
		if info.HeadHash == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("chain:%s  %s", info.HeadHash, info.ZipPath))
	}
	if ledgerHeadHash != "" {
		lines = append(lines, fmt.Sprintf("chain:%s  %s", ledgerHeadHash, filepath.ToSlash(filepath.Join("evidence", "bundles", "archived", retentionDefaultArchive, archiveledger.LedgerFile))))
	}
	return strings.Join(lines, "\n") + "\n"
}

func writeComplianceZip(result exportBuildResult) error {
	if err := os.MkdirAll(filepath.Dir(result.OutputPath), 0750); err != nil { // #nosec G301 -- project-local output path
		return fmt.Errorf("mkdir export output dir: %w", err)
	}
	f, err := os.Create(filepath.Clean(result.OutputPath)) // #nosec G304 -- output path is user-provided
	if err != nil {
		return fmt.Errorf("create export zip: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for _, file := range result.Files {
		hdr := &zip.FileHeader{
			Name:   filepath.ToSlash(file.ZipPath),
			Method: zip.Deflate,
		}
		hdr.SetModTime(time.Now())
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			_ = zw.Close()
			return fmt.Errorf("create zip entry %s: %w", file.ZipPath, err)
		}
		if _, err := w.Write(file.Data); err != nil {
			_ = zw.Close()
			return fmt.Errorf("write zip entry %s: %w", file.ZipPath, err)
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("close zip writer: %w", err)
	}
	return nil
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func printExportUsage() {
	fmt.Println("Usage: atb export --format compliance --output <path.zip> [--dry-run]")
}
