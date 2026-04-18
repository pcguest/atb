package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	atbembed "github.com/pcguest/atb"
	archiveledger "github.com/pcguest/atb/internal/archive"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/canonicalize"
	exportpkg "github.com/pcguest/atb/internal/export"
	"github.com/pcguest/atb/internal/trust"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

const (
	exportFormatCompliance = "compliance"
	exportFormatSOC2       = "soc2"
	exportFormatGDPR       = "gdpr"

	exportGDPRTypeDSR  = "dsr"
	exportGDPRTypeROPA = "ropa"

	unsignedSignatureStatus = "unsigned (encrypted handoff spec forthcoming)"
)

type exportConfig struct {
	Format     string
	Output     string
	DryRun     bool
	JSON       bool
	WithVerify bool
	BundlePath string
	GDPRType   string
	SubjectID  string
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

type exportCommandResult struct {
	Status       string                     `json:"status"`
	Format       string                     `json:"format"`
	Output       string                     `json:"output"`
	DryRun       bool                       `json:"dry_run"`
	FileCount    int                        `json:"file_count"`
	Verification *exportCommandVerification `json:"verification,omitempty"`
}

type exportCommandVerification struct {
	Passed       bool    `json:"passed"`
	Grade        string  `json:"grade"`
	ResidualRisk float64 `json:"residual_risk"`
	Sidecar      string  `json:"sidecar"`
}

type exportBaseEvidence struct {
	Manifest    exportManifest
	Files       []exportFileEntry
	BundleFiles []exportBundleInfo
	BundlePath  string
	BundleRaw   []byte
	BundleData  *bundle.Bundle
}

type soc2ControlDefinition struct {
	ControlID   string
	Description string
	EventTypes  map[string]struct{}
}

type soc2IntegrityProof struct {
	FirstEventHash string `json:"first_event_hash"`
	LastEventHash  string `json:"last_event_hash"`
	ChainValid     bool   `json:"chain_valid"`
}

type soc2ControlEvidence struct {
	ControlID      string             `json:"control_id"`
	Description    string             `json:"description"`
	EvidenceCount  int                `json:"evidence_count"`
	SampleIDs      []string           `json:"sample_ids"`
	IntegrityProof soc2IntegrityProof `json:"integrity_proof"`
}

type soc2EvidenceManifest struct {
	AuditPeriod struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"audit_period"`
	BundleHash        string                `json:"bundle_hash"`
	GeneratedAt       string                `json:"generated_at"`
	Controls          []soc2ControlEvidence `json:"controls"`
	VerifierSignature *string               `json:"verifier_signature"`
	SignatureStatus   string                `json:"signature_status"`
}

type soc2AuditTrailRecord struct {
	EventID   string      `json:"event_id"`
	Sequence  int         `json:"seq"`
	Type      string      `json:"type"`
	Hash      string      `json:"hash"`
	PrevHash  string      `json:"prev_hash"`
	Timestamp string      `json:"timestamp,omitempty"`
	Data      interface{} `json:"data"`
	ActorID   *string     `json:"actor_id,omitempty"`
	OrgID     *string     `json:"org_id,omitempty"`
	Workspace *string     `json:"workspace_id,omitempty"`
}

type soc2VerificationReport struct {
	BundlePath     string `json:"bundle_path"`
	BundleHash     string `json:"bundle_hash"`
	VerifiedAt     string `json:"verified_at"`
	ChainValid     bool   `json:"chain_valid"`
	FirstEventHash string `json:"first_event_hash,omitempty"`
	LastEventHash  string `json:"last_event_hash,omitempty"`
	EventCount     int    `json:"event_count"`
}

type gdprDSRRecord struct {
	EventID   string                 `json:"event_id"`
	Timestamp string                 `json:"timestamp"`
	Action    string                 `json:"action"`
	Context   map[string]interface{} `json:"context"`
}

type gdprDSRCategory struct {
	Category      string          `json:"category"`
	RecordCount   int             `json:"record_count"`
	RetentionRule string          `json:"retention_rule"`
	Records       []gdprDSRRecord `json:"records"`
}

type gdprDSRExport struct {
	RequestType    string            `json:"request_type"`
	SubjectID      string            `json:"subject_id"`
	ExportDate     string            `json:"export_date"`
	LegalBasis     string            `json:"legal_basis"`
	DataCategories []gdprDSRCategory `json:"data_categories"`
	Provenance     gdprDSRProvenance `json:"provenance"`
}

type gdprDSRProvenance struct {
	SourceBundleHash    string  `json:"source_bundle_hash"`
	ExtractionSignature *string `json:"extraction_signature"`
	SignatureStatus     string  `json:"signature_status"`
}

type gdprROPAExport struct {
	ControllerID         string                   `json:"controller_id"`
	ReportingPeriod      string                   `json:"reporting_period"`
	ProcessingActivities []gdprProcessingActivity `json:"processing_activities"`
}

type gdprProcessingActivity struct {
	Purpose             string   `json:"purpose"`
	LegalBasis          string   `json:"legal_basis"`
	DataCategories      []string `json:"data_categories"`
	RecipientCategories []string `json:"recipient_categories,omitempty"`
	RetentionSchedule   string   `json:"retention_schedule"`
	SecurityMeasures    []string `json:"security_measures"`
	EventVolume         int      `json:"event_volume"`
}

var errExportHelp = errors.New("export help requested")

var complianceManifestRegulatoryCoverage = []string{
	"EU AI Act Article 12",
	"NIST AI RMF GOVERN",
	"NIST AI RMF MANAGE",
	"UK DSIT AI Code of Practice",
}

var soc2Controls = []soc2ControlDefinition{
	{
		ControlID:   "CC6.1",
		Description: "Logical Access Security",
		EventTypes:  setOf("auth.login", "auth.logout", "auth.failure", "permission.change"),
	},
	{
		ControlID:   "CC6.6",
		Description: "System Boundaries",
		EventTypes:  setOf("system.config.change", "system.config_change", "network.policy.update", "network.policy_update"),
	},
	{
		ControlID:   "CC7.2",
		Description: "System Monitoring",
		EventTypes:  setOf("alert.triggered", "monitor.anomaly", "health.check.fail", "health.check_fail"),
	},
	{
		ControlID:   "CC8.1",
		Description: "Change Management",
		EventTypes:  setOf("deploy.start", "deploy.complete", "code.merge", "config.promote"),
	},
	{
		ControlID:   "CC9.1",
		Description: "Risk Mitigation",
		EventTypes:  setOf("backup.start", "backup.complete", "restore.init"),
	},
}

func cmdExport() {
	os.Exit(runExport(os.Args[2:], os.Stdout, os.Stderr))
}

func runExport(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseExportArgs(args)
	if err != nil {
		if errors.Is(err, errExportHelp) {
			printExportUsage(stdout)
			return exitSuccess
		}
		fmt.Fprintf(stderr, "atb export: %v\n", err)
		printExportUsage(stderr)
		return exitUserError
	}

	now := time.Now().UTC()
	result, err := buildExport(now, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "atb export: %v\n", err)
		exitCode := exitSystemError
		if strings.Contains(strings.ToLower(err.Error()), "verification") {
			exitCode = exitUserError
		}
		return exitCode
	}

	if cfg.Format == exportFormatCompliance && cfg.JSON {
		manifest, exitCode, err := buildComplianceJSONManifest(now, cfg, result)
		if err != nil {
			fmt.Fprintf(stderr, "atb export: %v\n", err)
			return exitSystemError
		}
		data, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "atb export: encode json output: %v\n", err)
			return exitSystemError
		}
		if _, err := stdout.Write(append(data, '\n')); err != nil {
			fmt.Fprintf(stderr, "atb export: write json output: %v\n", err)
			return exitSystemError
		}
		return exitCode
	}

	if cfg.DryRun {
		if cfg.JSON {
			payload := exportCommandResult{
				Status:    "dry_run",
				Format:    cfg.Format,
				Output:    result.OutputPath,
				DryRun:    true,
				FileCount: len(result.Files),
			}
			if err := json.NewEncoder(stdout).Encode(payload); err != nil {
				fmt.Fprintf(stderr, "atb export: encode json output: %v\n", err)
				return exitSystemError
			}
			return exitSuccess
		}

		fmt.Fprintf(stdout, "~ Dry run: would create %s with %d file(s)\n", cfg.Output, len(result.Files))
		for _, path := range result.Manifest.IncludedFiles {
			fmt.Fprintf(stdout, "  - %s\n", path)
		}
		if len(result.Manifest.Warnings) > 0 {
			fmt.Fprintln(stdout, "Warnings:")
			for _, warning := range result.Manifest.Warnings {
				fmt.Fprintf(stdout, "  - %s\n", warning)
			}
		}
		fmt.Fprintf(
			stdout,
			"Verification: active_verified=%d archived_verified=%d ledger_verified=%t\n",
			result.Manifest.Verification.ActiveVerified,
			result.Manifest.Verification.ArchivedVerified,
			result.Manifest.Verification.LedgerVerified,
		)
		return exitSuccess
	}

	if err := writeExportZip(result, now); err != nil {
		fmt.Fprintf(stderr, "atb export: write zip: %v\n", err)
		return exitSystemError
	}

	exitCode := exitSuccess
	var verificationSummary *exportCommandVerification
	if cfg.WithVerify {
		report, summary, err := writeExportVerificationSidecar(cfg, result, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "atb export: %v\n", err)
			return exitUserError
		}
		verificationSummary = summary
		if verificationSummary != nil {
			exitCode = verificationExitCode(report)
		}
	}

	if cfg.JSON {
		payload := exportCommandResult{
			Status:       "written",
			Format:       cfg.Format,
			Output:       result.OutputPath,
			DryRun:       false,
			FileCount:    len(result.Files),
			Verification: verificationSummary,
		}
		if err := json.NewEncoder(stdout).Encode(payload); err != nil {
			fmt.Fprintf(stderr, "atb export: encode json output: %v\n", err)
			return exitSystemError
		}
		return exitCode
	}

	fmt.Fprintf(stdout, "✓ Exported compliance evidence: %s\n", result.OutputPath)
	if verificationSummary != nil {
		status := "FAIL"
		if verificationSummary.Passed {
			status = "PASS"
		}
		fmt.Fprintf(stdout, "Verification: %s  grade=%s  residual_risk=%.2f\n", status, verificationSummary.Grade, verificationSummary.ResidualRisk)
		fmt.Fprintf(stdout, "Sidecar written: %s\n", verificationSummary.Sidecar)
	}
	return exitCode
}

func buildComplianceJSONManifest(now time.Time, cfg exportConfig, result exportBuildResult) (exportpkg.ComplianceManifest, int, error) {
	bundlePath, err := filepath.Abs(bundle.DefaultPath())
	if err != nil {
		return exportpkg.ComplianceManifest{}, exitSystemError, fmt.Errorf("resolve bundle path: %w", err)
	}

	manifest := exportpkg.ComplianceManifest{
		BundlePath:         bundlePath,
		ExportFormat:       exportFormatCompliance,
		GeneratedAt:        now.UTC().Format(time.RFC3339),
		Files:              complianceManifestFiles(result.Files),
		RegulatoryCoverage: append([]string(nil), complianceManifestRegulatoryCoverage...),
	}
	if cfg.DryRun {
		return manifest, exitSuccess, nil
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		return exportpkg.ComplianceManifest{}, exitSystemError, fmt.Errorf("load bundle %s for manifest verification: %w", bundlePath, err)
	}

	report := verifypkg.ReportFromVerify(verifypkg.Verify(b, bundlePath, ""))
	manifest.VerifyResult = &report
	if report.Pass {
		return manifest, exitSuccess, nil
	}
	return manifest, exitUserError, nil
}

func complianceManifestFiles(files []exportFileEntry) []exportpkg.ComplianceFile {
	manifestFiles := make([]exportpkg.ComplianceFile, 0, len(files))
	for _, file := range files {
		manifestFiles = append(manifestFiles, exportpkg.ComplianceFile{
			Name:        file.ZipPath,
			Description: complianceManifestFileDescription(file.ZipPath),
			SizeBytes:   int64(len(file.Data)),
		})
	}
	return manifestFiles
}

func complianceManifestFileDescription(name string) string {
	switch {
	case strings.HasPrefix(name, "evidence/bundles/active/"):
		return "Active bundle included in the compliance export."
	case strings.HasPrefix(name, "evidence/bundles/archived/") && strings.HasSuffix(name, archiveledger.LedgerFile):
		return "Archive ledger included in the compliance export."
	case strings.HasPrefix(name, "evidence/bundles/archived/"):
		return "Archived bundle included in the compliance export."
	case name == "evidence/reports/verify.json":
		return "Aggregate verification report for active and archived bundles."
	case name == "evidence/reports/trust-report.json":
		return "Trust report for the primary active bundle."
	case name == "evidence/reports/archive-ledger.json":
		return "Archive ledger verification summary."
	case name == "evidence/config/atb-config.json":
		return "ATB configuration snapshot."
	case strings.HasPrefix(name, "evidence/docs/"):
		return fmt.Sprintf("Documentation reference: %s.", path.Base(name))
	case name == "evidence/checksums.sha256":
		return "SHA-256 checksums for exported files."
	case name == "evidence/checksums.chain":
		return "Checksum chain metadata for exported files."
	case name == "evidence/manifest.json":
		return "Archive manifest that would be embedded in the zip export."
	default:
		return "Compliance export file."
	}
}

func parseExportArgs(args []string) (exportConfig, error) {
	cfg := exportConfig{Format: exportFormatCompliance}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errExportHelp
		case arg == "--format":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --format (expected compliance|soc2|gdpr)")
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
		case arg == "--bundle":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --bundle")
			}
			cfg.BundlePath = normalizeBundlePath(args[i+1])
			i++
		case strings.HasPrefix(arg, "--bundle="):
			cfg.BundlePath = normalizeBundlePath(strings.TrimSpace(strings.TrimPrefix(arg, "--bundle=")))
		case arg == "--type":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --type (expected dsr|ropa)")
			}
			cfg.GDPRType = strings.ToLower(strings.TrimSpace(args[i+1]))
			i++
		case strings.HasPrefix(arg, "--type="):
			cfg.GDPRType = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--type=")))
		case arg == "--subject-id":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --subject-id")
			}
			cfg.SubjectID = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(arg, "--subject-id="):
			cfg.SubjectID = strings.TrimSpace(strings.TrimPrefix(arg, "--subject-id="))
		case arg == "--dry-run":
			cfg.DryRun = true
		case arg == "--json":
			cfg.JSON = true
		case arg == "--with-verify":
			cfg.WithVerify = true
		case strings.HasPrefix(arg, "--"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			return cfg, fmt.Errorf("unexpected argument %q", arg)
		}
	}

	switch cfg.Format {
	case exportFormatCompliance, exportFormatSOC2, exportFormatGDPR:
	default:
		return cfg, fmt.Errorf("unsupported format %q (expected compliance|soc2|gdpr)", cfg.Format)
	}

	if cfg.Output == "" {
		return cfg, fmt.Errorf("--output is required")
	}

	switch cfg.Format {
	case exportFormatCompliance:
		if cfg.BundlePath != "" || cfg.GDPRType != "" || cfg.SubjectID != "" {
			return cfg, fmt.Errorf("--bundle, --type, and --subject-id are unsupported for --format compliance")
		}
	case exportFormatSOC2:
		if cfg.BundlePath == "" {
			cfg.BundlePath = bundle.DefaultPath()
		}
		if cfg.GDPRType != "" || cfg.SubjectID != "" {
			return cfg, fmt.Errorf("--type and --subject-id are only valid for --format gdpr")
		}
	case exportFormatGDPR:
		if cfg.BundlePath == "" {
			cfg.BundlePath = bundle.DefaultPath()
		}
		if cfg.GDPRType != exportGDPRTypeDSR && cfg.GDPRType != exportGDPRTypeROPA {
			return cfg, fmt.Errorf("--type is required for --format gdpr (expected dsr|ropa)")
		}
		if cfg.GDPRType == exportGDPRTypeDSR && strings.TrimSpace(cfg.SubjectID) == "" {
			return cfg, fmt.Errorf("--subject-id is required for --format gdpr --type dsr")
		}
		if cfg.GDPRType == exportGDPRTypeROPA && strings.TrimSpace(cfg.SubjectID) != "" {
			return cfg, fmt.Errorf("--subject-id is only valid for --format gdpr --type dsr")
		}
	}

	return cfg, nil
}

func buildExport(now time.Time, cfg exportConfig) (exportBuildResult, error) {
	switch cfg.Format {
	case exportFormatCompliance:
		return buildComplianceExport(now, cfg)
	case exportFormatSOC2:
		return buildSOC2Export(now, cfg)
	case exportFormatGDPR:
		return buildGDPRExport(now, cfg)
	default:
		return exportBuildResult{}, fmt.Errorf("unsupported format %q", cfg.Format)
	}
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
		data, err := os.ReadFile(info.Source) // #nosec G304 -- bundle paths come from verified local discovery
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

		data, err := os.ReadFile(ledgerPath) // #nosec G304 -- archive ledger path is a fixed project-local location
		if err != nil {
			return result, fmt.Errorf("read archive ledger: %w", err)
		}
		zipPath := filepath.ToSlash(filepath.Join("evidence", "bundles", "archived", ledgerPath))
		files = append(files, exportFileEntry{ZipPath: zipPath, Data: data})
		manifest.IncludedFiles = append(manifest.IncludedFiles, zipPath)
	} else if !os.IsNotExist(loadErr) {
		return result, fmt.Errorf("load archive ledger: %w", loadErr)
	}

	verifyReportData, err := buildVerifyReport(now, activePaths, archivedPaths)
	if err != nil {
		return result, err
	}
	verifyReportZip := filepath.ToSlash(filepath.Join("evidence", "reports", "verify.json"))
	files = append(files, exportFileEntry{ZipPath: verifyReportZip, Data: verifyReportData})
	manifest.IncludedFiles = append(manifest.IncludedFiles, verifyReportZip)

	trustReport := trust.BuildReport(cwd, bundle.DefaultPath(), "")
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

	if cfgData, _, retentionDays, err := loadConfigForExport(); err != nil {
		return result, err
	} else if cfgData != nil {
		zipPath := filepath.ToSlash(filepath.Join("evidence", "config", "atb-config.json"))
		files = append(files, exportFileEntry{ZipPath: zipPath, Data: cfgData})
		manifest.IncludedFiles = append(manifest.IncludedFiles, zipPath)
		manifest.RetentionDays = retentionDays
	}

	for _, doc := range []string{"docs/spec-v1.0.md", "docs/security.md", "docs/incident-response.md"} {
		if data, err := readExportDoc(doc); err == nil {
			zipPath := filepath.ToSlash(filepath.Join("evidence", "docs", strings.TrimPrefix(filepath.ToSlash(doc), "docs/")))
			files = append(files, exportFileEntry{ZipPath: zipPath, Data: data})
			manifest.IncludedFiles = append(manifest.IncludedFiles, zipPath)
		} else if errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err) {
			manifest.Warnings = append(manifest.Warnings, fmt.Sprintf("optional doc missing: %s", doc))
		} else {
			return result, fmt.Errorf("read doc %s: %w", doc, err)
		}
	}

	complianceDocs, err := discoverComplianceDocs()
	if err != nil {
		return result, err
	}
	if len(complianceDocs) == 0 {
		manifest.Warnings = append(manifest.Warnings, "optional embedded compliance docs not found")
	}
	for _, doc := range complianceDocs {
		data, err := readExportDoc(doc)
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
	result.OutputPath = cfg.Output
	return result, nil
}

func buildSOC2Export(now time.Time, cfg exportConfig) (exportBuildResult, error) {
	base, err := buildBaseExportEvidence(now, cfg)
	if err != nil {
		return exportBuildResult{}, err
	}

	relevant := make([]soc2AuditTrailRecord, 0)
	controlRecords := make(map[string][]soc2AuditTrailRecord, len(soc2Controls))
	for _, control := range soc2Controls {
		controlRecords[control.ControlID] = []soc2AuditTrailRecord{}
	}

	var periodStart *time.Time
	var periodEnd *time.Time
	for _, rec := range base.BundleData.Records {
		record := newSOC2AuditTrailRecord(rec)
		if !isSOC2RelevantType(record.Type) {
			continue
		}
		relevant = append(relevant, record)
		for _, control := range soc2Controls {
			if _, ok := control.EventTypes[record.Type]; ok {
				controlRecords[control.ControlID] = append(controlRecords[control.ControlID], record)
			}
		}
		if ts, ok := parseRFC3339(record.Timestamp); ok {
			if periodStart == nil || ts.Before(*periodStart) {
				start := ts
				periodStart = &start
			}
			if periodEnd == nil || ts.After(*periodEnd) {
				end := ts
				periodEnd = &end
			}
		}
	}

	if periodStart == nil || periodEnd == nil {
		fallback := now.UTC()
		periodStart = &fallback
		periodEnd = &fallback
	}

	trailData, err := buildSOC2AuditTrailFile(relevant)
	if err != nil {
		return exportBuildResult{}, err
	}

	soc2Manifest := soc2EvidenceManifest{
		BundleHash:        "sha256:" + sha256Hex(base.BundleRaw),
		GeneratedAt:       now.UTC().Format(time.RFC3339),
		Controls:          make([]soc2ControlEvidence, 0, len(soc2Controls)),
		VerifierSignature: nil,
		SignatureStatus:   unsignedSignatureStatus,
	}
	soc2Manifest.AuditPeriod.Start = periodStart.UTC().Format(time.RFC3339)
	soc2Manifest.AuditPeriod.End = periodEnd.UTC().Format(time.RFC3339)

	for _, control := range soc2Controls {
		recs := controlRecords[control.ControlID]
		item := soc2ControlEvidence{
			ControlID:     control.ControlID,
			Description:   control.Description,
			EvidenceCount: len(recs),
			SampleIDs:     []string{},
			IntegrityProof: soc2IntegrityProof{
				ChainValid: true,
			},
		}
		if len(recs) > 0 {
			item.IntegrityProof.FirstEventHash = "sha256:" + recs[0].Hash
			item.IntegrityProof.LastEventHash = "sha256:" + recs[len(recs)-1].Hash
			limit := len(recs)
			if limit > 3 {
				limit = 3
			}
			for i := 0; i < limit; i++ {
				item.SampleIDs = append(item.SampleIDs, recs[i].EventID)
			}
		}
		soc2Manifest.Controls = append(soc2Manifest.Controls, item)
	}

	manifestData, err := json.MarshalIndent(soc2Manifest, "", "  ")
	if err != nil {
		return exportBuildResult{}, fmt.Errorf("marshal soc2 manifest: %w", err)
	}
	manifestData = append(manifestData, '\n')

	verification := soc2VerificationReport{
		BundlePath: base.BundlePath,
		BundleHash: "sha256:" + sha256Hex(base.BundleRaw),
		VerifiedAt: now.UTC().Format(time.RFC3339),
		ChainValid: true,
		EventCount: len(base.BundleData.Records),
	}
	if len(base.BundleData.Records) > 0 {
		verification.FirstEventHash = "sha256:" + base.BundleData.Records[0].Hash
		verification.LastEventHash = "sha256:" + base.BundleData.Records[len(base.BundleData.Records)-1].Hash
	}
	verificationData, err := json.MarshalIndent(verification, "", "  ")
	if err != nil {
		return exportBuildResult{}, fmt.Errorf("marshal soc2 verification report: %w", err)
	}
	verificationData = append(verificationData, '\n')

	extra := []exportFileEntry{
		{ZipPath: filepath.ToSlash(filepath.Join("evidence", "soc2_evidence_manifest.json")), Data: manifestData},
		{ZipPath: filepath.ToSlash(filepath.Join("evidence", "audit_trail.jsonl")), Data: trailData},
		{ZipPath: filepath.ToSlash(filepath.Join("evidence", "verification_report.json")), Data: verificationData},
	}

	return finalizeExport(now, cfg, base, extra)
}

func buildGDPRExport(now time.Time, cfg exportConfig) (exportBuildResult, error) {
	base, err := buildBaseExportEvidence(now, cfg)
	if err != nil {
		return exportBuildResult{}, err
	}
	if err := validateGDPRRetention(now, cfg.BundlePath); err != nil {
		return exportBuildResult{}, err
	}

	extra := make([]exportFileEntry, 0, 1)
	salt := "atb-gdpr-v1:" + sha256Hex(base.BundleRaw)

	switch cfg.GDPRType {
	case exportGDPRTypeDSR:
		dsr, err := buildGDPRDSRDocument(now, cfg, base, salt)
		if err != nil {
			return exportBuildResult{}, err
		}
		dsrData, err := json.MarshalIndent(dsr, "", "  ")
		if err != nil {
			return exportBuildResult{}, fmt.Errorf("marshal gdpr dsr report: %w", err)
		}
		dsrData = append(dsrData, '\n')
		extra = append(extra, exportFileEntry{
			ZipPath: filepath.ToSlash(filepath.Join("evidence", fmt.Sprintf("dsr_%s.json", cfg.SubjectID))),
			Data:    dsrData,
		})
	case exportGDPRTypeROPA:
		ropa, err := buildGDPRROPADocument(now, base, salt)
		if err != nil {
			return exportBuildResult{}, err
		}
		ropaData, err := json.MarshalIndent(ropa, "", "  ")
		if err != nil {
			return exportBuildResult{}, fmt.Errorf("marshal gdpr ropa report: %w", err)
		}
		ropaData = append(ropaData, '\n')
		extra = append(extra, exportFileEntry{
			ZipPath: filepath.ToSlash(filepath.Join("evidence", "ropa_summary.json")),
			Data:    ropaData,
		})
	default:
		return exportBuildResult{}, fmt.Errorf("unsupported gdpr --type %q", cfg.GDPRType)
	}

	return finalizeExport(now, cfg, base, extra)
}

func buildBaseExportEvidence(now time.Time, cfg exportConfig) (exportBaseEvidence, error) {
	base := exportBaseEvidence{
		Manifest: exportManifest{
			GeneratedAt:   now.Format(time.RFC3339),
			ATBVersion:    version,
			IncludedFiles: []string{},
			Verification:  exportVerificationStatus{},
			Warnings:      []string{},
		},
		Files:       []exportFileEntry{},
		BundleFiles: []exportBundleInfo{},
		BundlePath:  filepath.Clean(cfg.BundlePath),
	}

	info, err := verifyBundleForExport(base.BundlePath)
	if err != nil {
		return base, err
	}
	info.Source = base.BundlePath
	info.ZipPath = filepath.ToSlash(filepath.Join("evidence", "bundles", "active", base.BundlePath))
	info.Archived = false

	raw, err := os.ReadFile(base.BundlePath) // #nosec G304 -- bundle path is a validated local CLI input
	if err != nil {
		return base, fmt.Errorf("read bundle file %s: %w", base.BundlePath, err)
	}
	b, err := bundle.Load(base.BundlePath)
	if err != nil {
		return base, fmt.Errorf("load bundle file %s: %w", base.BundlePath, err)
	}
	if err := b.Verify(); err != nil {
		return base, fmt.Errorf("bundle verification failed for %s: %w", base.BundlePath, err)
	}

	base.BundleRaw = raw
	base.BundleData = b
	base.BundleFiles = append(base.BundleFiles, info)
	base.Manifest.BundleHeadHash = info.HeadHash
	base.Manifest.Verification.ActiveVerified = 1
	base.Manifest.ArchiveCount = 0

	base.Files = append(base.Files, exportFileEntry{ZipPath: info.ZipPath, Data: raw})
	base.Manifest.IncludedFiles = append(base.Manifest.IncludedFiles, info.ZipPath)

	if cfgData, _, retentionDays, err := loadConfigForExport(); err != nil {
		return base, err
	} else if cfgData != nil {
		zipPath := filepath.ToSlash(filepath.Join("evidence", "config", "atb-config.json"))
		base.Files = append(base.Files, exportFileEntry{ZipPath: zipPath, Data: cfgData})
		base.Manifest.IncludedFiles = append(base.Manifest.IncludedFiles, zipPath)
		base.Manifest.RetentionDays = retentionDays
	}

	for _, doc := range []string{"docs/spec-v1.0.md", "docs/security.md", "docs/incident-response.md"} {
		if data, err := readExportDoc(doc); err == nil {
			zipPath := filepath.ToSlash(filepath.Join("evidence", "docs", strings.TrimPrefix(filepath.ToSlash(doc), "docs/")))
			base.Files = append(base.Files, exportFileEntry{ZipPath: zipPath, Data: data})
			base.Manifest.IncludedFiles = append(base.Manifest.IncludedFiles, zipPath)
		} else if errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err) {
			base.Manifest.Warnings = append(base.Manifest.Warnings, fmt.Sprintf("optional doc missing: %s", doc))
		} else {
			return base, fmt.Errorf("read doc %s: %w", doc, err)
		}
	}

	requiredComplianceDoc := path.Join("docs", "compliance", cfg.Format+".md")
	complianceData, err := readExportDoc(requiredComplianceDoc)
	if err != nil {
		return base, fmt.Errorf("read required compliance doc %s: %w", requiredComplianceDoc, err)
	}
	complianceZipPath := filepath.ToSlash(filepath.Join("evidence", "docs", strings.TrimPrefix(filepath.ToSlash(requiredComplianceDoc), "docs/")))
	base.Files = append(base.Files, exportFileEntry{ZipPath: complianceZipPath, Data: complianceData})
	base.Manifest.IncludedFiles = append(base.Manifest.IncludedFiles, complianceZipPath)

	return base, nil
}

func finalizeExport(_ time.Time, cfg exportConfig, base exportBaseEvidence, extraReports []exportFileEntry) (exportBuildResult, error) {
	result := exportBuildResult{OutputPath: cfg.Output}
	manifest := base.Manifest
	files := append([]exportFileEntry{}, base.Files...)

	for _, extra := range extraReports {
		files = append(files, extra)
		manifest.IncludedFiles = append(manifest.IncludedFiles, extra.ZipPath)
	}

	sort.Strings(manifest.IncludedFiles)
	sort.Slice(files, func(i, j int) bool {
		return files[i].ZipPath < files[j].ZipPath
	})

	checksums := buildChecksumsFile(files)
	checksumsZip := filepath.ToSlash(filepath.Join("evidence", "checksums.sha256"))
	files = append(files, exportFileEntry{ZipPath: checksumsZip, Data: []byte(checksums)})
	manifest.IncludedFiles = append(manifest.IncludedFiles, checksumsZip)

	checksumMeta := buildChecksumMetaFile(files, base.BundleFiles, manifest.LedgerHeadHash)
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
	result.BundleFiles = base.BundleFiles
	result.ChecksumLines = strings.Split(strings.TrimSuffix(checksums, "\n"), "\n")
	result.OutputPath = cfg.Output
	return result, nil
}

func buildVerifyReport(now time.Time, activePaths, archivedPaths []string) ([]byte, error) {
	type verifyBundle struct {
		Path     string `json:"path"`
		HeadHash string `json:"head_hash"`
		Archived bool   `json:"archived"`
	}
	report := struct {
		GeneratedAt string         `json:"generated_at"`
		Bundles     []verifyBundle `json:"bundles"`
	}{
		GeneratedAt: now.UTC().Format(time.RFC3339),
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

func buildSOC2AuditTrailFile(records []soc2AuditTrailRecord) ([]byte, error) {
	if len(records) == 0 {
		return []byte{}, nil
	}
	lines := make([]string, 0, len(records))
	for _, rec := range records {
		canonical, err := canonicalize.Marshal(rec)
		if err != nil {
			return nil, fmt.Errorf("canonicalize soc2 audit trail record: %w", err)
		}
		lines = append(lines, string(canonical))
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

func newSOC2AuditTrailRecord(rec bundle.Record) soc2AuditTrailRecord {
	return soc2AuditTrailRecord{
		EventID:   eventIdentifier(rec),
		Sequence:  rec.Event.Sequence,
		Type:      rec.Event.Type,
		Hash:      rec.Hash,
		PrevHash:  rec.Event.PrevHash,
		Timestamp: exportEventTimestamp(rec, ""),
		Data:      rec.Event.Data,
		ActorID:   rec.Event.ActorID,
		OrgID:     rec.Event.OrgID,
		Workspace: rec.Event.WorkspaceID,
	}
}

func buildGDPRDSRDocument(now time.Time, cfg exportConfig, base exportBaseEvidence, salt string) (gdprDSRExport, error) {
	categories := map[string][]gdprDSRRecord{}
	foundSubject := false

	retentionRule := "retention_not_configured"
	if base.Manifest.RetentionDays != nil {
		retentionRule = fmt.Sprintf("delete_after_%dd", *base.Manifest.RetentionDays)
	}

	for _, rec := range base.BundleData.Records {
		if !recordContainsSubject(rec, cfg.SubjectID) {
			continue
		}
		foundSubject = true

		action := rec.Event.Type
		if idx := strings.LastIndex(action, "."); idx >= 0 && idx < len(action)-1 {
			action = action[idx+1:]
		}
		context := map[string]interface{}{}
		if dataMap, ok := sanitizeGDPRContext(rec.Event.Data, exportGDPRTypeDSR, cfg.SubjectID, salt).(map[string]interface{}); ok {
			context = extractPreferredContext(dataMap)
		}
		if len(context) == 0 {
			context = map[string]interface{}{}
		}

		category := dsrCategoryForEventType(rec.Event.Type)
		categories[category] = append(categories[category], gdprDSRRecord{
			EventID:   eventIdentifier(rec),
			Timestamp: exportEventTimestamp(rec, now.UTC().Format(time.RFC3339)),
			Action:    action,
			Context:   context,
		})
	}

	if !foundSubject {
		return gdprDSRExport{}, fmt.Errorf("--subject-id %q not found in bundle", cfg.SubjectID)
	}

	keys := make([]string, 0, len(categories))
	for key := range categories {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	dsrCategories := make([]gdprDSRCategory, 0, len(keys))
	for _, category := range keys {
		records := categories[category]
		sort.Slice(records, func(i, j int) bool {
			return records[i].EventID < records[j].EventID
		})
		dsrCategories = append(dsrCategories, gdprDSRCategory{
			Category:      category,
			RecordCount:   len(records),
			RetentionRule: retentionRule,
			Records:       records,
		})
	}

	return gdprDSRExport{
		RequestType:    "GDPR_ARTICLE_15",
		SubjectID:      cfg.SubjectID,
		ExportDate:     now.UTC().Format(time.RFC3339),
		LegalBasis:     "contract_performance",
		DataCategories: dsrCategories,
		Provenance: gdprDSRProvenance{
			SourceBundleHash:    "sha256:" + sha256Hex(base.BundleRaw),
			ExtractionSignature: nil,
			SignatureStatus:     unsignedSignatureStatus,
		},
	}, nil
}

func buildGDPRROPADocument(now time.Time, base exportBaseEvidence, salt string) (gdprROPAExport, error) {
	activityMap := map[string]*gdprProcessingActivity{}
	for _, rec := range base.BundleData.Records {
		purpose := ropaPurposeForEventType(rec.Event.Type)
		activity, ok := activityMap[purpose]
		if !ok {
			activity = &gdprProcessingActivity{
				Purpose:           purpose,
				LegalBasis:        ropaLegalBasisForPurpose(purpose),
				DataCategories:    []string{},
				RetentionSchedule: ropaRetentionScheduleForPurpose(purpose),
				SecurityMeasures:  ropaSecurityMeasuresForPurpose(purpose),
			}
			if recipients := ropaRecipientsForPurpose(purpose); len(recipients) > 0 {
				activity.RecipientCategories = recipients
			}
			activityMap[purpose] = activity
		}

		activity.EventVolume++
		for _, category := range ropaDataCategoriesFromEvent(rec.Event.Data, salt) {
			if !containsStringValue(activity.DataCategories, category) {
				activity.DataCategories = append(activity.DataCategories, category)
			}
		}
		sort.Strings(activity.DataCategories)
	}

	if len(activityMap) == 0 {
		activityMap["service_delivery"] = &gdprProcessingActivity{
			Purpose:             "service_delivery",
			LegalBasis:          "contract_performance",
			DataCategories:      []string{"usage_metrics"},
			RecipientCategories: []string{"internal_engineering", "cloud_provider_aws"},
			RetentionSchedule:   "24_months_from_last_activity",
			SecurityMeasures: []string{
				"AES-256-GCM encryption at rest",
				"Hash-chained audit logs",
				"Role-based access control",
			},
			EventVolume: 0,
		}
	}

	purposes := make([]string, 0, len(activityMap))
	for purpose := range activityMap {
		purposes = append(purposes, purpose)
	}
	sort.Strings(purposes)

	activities := make([]gdprProcessingActivity, 0, len(purposes))
	for _, purpose := range purposes {
		activities = append(activities, *activityMap[purpose])
	}

	controllerID := "org_local"
	for _, rec := range base.BundleData.Records {
		if rec.Event.OrgID != nil && strings.TrimSpace(*rec.Event.OrgID) != "" {
			controllerID = *rec.Event.OrgID
			break
		}
	}

	return gdprROPAExport{
		ControllerID:         controllerID,
		ReportingPeriod:      quarterString(now.UTC()),
		ProcessingActivities: activities,
	}, nil
}

func recordContainsSubject(rec bundle.Record, subjectID string) bool {
	subject := strings.TrimSpace(subjectID)
	if subject == "" {
		return false
	}
	if rec.Event.ActorID != nil && strings.TrimSpace(*rec.Event.ActorID) == subject {
		return true
	}
	if rec.Event.OrgID != nil && strings.TrimSpace(*rec.Event.OrgID) == subject {
		return true
	}
	if rec.Event.WorkspaceID != nil && strings.TrimSpace(*rec.Event.WorkspaceID) == subject {
		return true
	}
	return valueContainsSubject(rec.Event.Data, subject)
}

func valueContainsSubject(value interface{}, subject string) bool {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == subject
	case map[string]interface{}:
		for _, v := range typed {
			if valueContainsSubject(v, subject) {
				return true
			}
		}
	case []interface{}:
		for _, v := range typed {
			if valueContainsSubject(v, subject) {
				return true
			}
		}
	}
	return false
}

func sanitizeGDPRContext(value interface{}, mode string, subjectID, salt string) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(typed))
		for k, v := range typed {
			out[k] = sanitizeGDPRField(k, v, mode, subjectID, salt)
		}
		return out
	case []interface{}:
		out := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			out = append(out, sanitizeGDPRContext(item, mode, subjectID, salt))
		}
		return out
	default:
		return typed
	}
}

func sanitizeGDPRField(key string, value interface{}, mode string, subjectID, salt string) interface{} {
	category := gdprFieldCategory(key)
	if category == "metadata" {
		return sanitizeGDPRContext(value, mode, subjectID, salt)
	}

	switch mode {
	case exportGDPRTypeDSR:
		if category == "direct" || category == "sensitive" || category == "third_party" {
			if scalarMatchesSubject(value, subjectID) {
				return sanitizeGDPRContext(value, mode, subjectID, salt)
			}
			return "[REDACTED]"
		}
		return sanitizeGDPRContext(value, mode, subjectID, salt)
	case exportGDPRTypeROPA:
		if category == "sensitive" {
			return "[REDACTED]"
		}
		if category == "direct" || category == "third_party" {
			return hashGDPRValue(value, salt)
		}
		return sanitizeGDPRContext(value, mode, subjectID, salt)
	default:
		return sanitizeGDPRContext(value, mode, subjectID, salt)
	}
}

func gdprFieldCategory(key string) string {
	normalized := strings.ToLower(strings.TrimSpace(key))
	switch normalized {
	case "email", "ip", "user_id":
		return "direct"
	}
	if strings.Contains(normalized, "payment") || strings.Contains(normalized, "health") || strings.Contains(normalized, "bio") {
		return "sensitive"
	}
	if strings.Contains(normalized, "timestamp") || normalized == "ts" || strings.Contains(normalized, "hash") {
		return "metadata"
	}
	if strings.HasSuffix(normalized, "_id") || normalized == "id" {
		return "third_party"
	}
	return "other"
}

func hashGDPRValue(value interface{}, salt string) interface{} {
	switch typed := value.(type) {
	case string:
		return hashPIIString(salt + ":" + typed)
	case fmt.Stringer:
		return hashPIIString(salt + ":" + typed.String())
	case int:
		return hashPIIString(salt + ":" + strconv.Itoa(typed))
	case int64:
		return hashPIIString(fmt.Sprintf("%s:%d", salt, typed))
	case float64:
		return hashPIIString(fmt.Sprintf("%s:%f", salt, typed))
	case bool:
		return hashPIIString(fmt.Sprintf("%s:%t", salt, typed))
	case map[string]interface{}:
		out := make(map[string]interface{}, len(typed))
		for k, v := range typed {
			out[k] = hashGDPRValue(v, salt)
		}
		return out
	case []interface{}:
		out := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			out = append(out, hashGDPRValue(item, salt))
		}
		return out
	default:
		return hashPIIString(fmt.Sprintf("%s:%v", salt, typed))
	}
}

func scalarMatchesSubject(value interface{}, subjectID string) bool {
	subject := strings.TrimSpace(subjectID)
	if subject == "" {
		return false
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == subject
	}
	return false
}

func extractPreferredContext(data map[string]interface{}) map[string]interface{} {
	context := make(map[string]interface{})
	if value, ok := lookupKeyCaseInsensitive(data, "ip"); ok {
		context["ip"] = value
	}
	if value, ok := lookupKeyCaseInsensitive(data, "user_agent"); ok {
		context["user_agent"] = value
	}
	if len(context) > 0 {
		return context
	}
	return data
}

func lookupKeyCaseInsensitive(data map[string]interface{}, key string) (interface{}, bool) {
	for k, v := range data {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return nil, false
}

func dsrCategoryForEventType(eventType string) string {
	switch {
	case strings.HasPrefix(eventType, "auth."):
		return "access_logs"
	case strings.HasPrefix(eventType, "deploy.") || strings.HasPrefix(eventType, "code.") || strings.HasPrefix(eventType, "config."):
		return "change_logs"
	default:
		return "activity_logs"
	}
}

func ropaPurposeForEventType(eventType string) string {
	switch {
	case strings.HasPrefix(eventType, "alert."), strings.HasPrefix(eventType, "monitor."), strings.Contains(eventType, "fraud"):
		return "fraud_detection"
	default:
		return "service_delivery"
	}
}

func ropaLegalBasisForPurpose(purpose string) string {
	if purpose == "fraud_detection" {
		return "legitimate_interest"
	}
	return "contract_performance"
}

func ropaRetentionScheduleForPurpose(purpose string) string {
	if purpose == "fraud_detection" {
		return "90_days"
	}
	return "24_months_from_last_activity"
}

func ropaRecipientsForPurpose(purpose string) []string {
	if purpose == "fraud_detection" {
		return []string{"internal_security"}
	}
	return []string{"internal_engineering", "cloud_provider_aws"}
}

func ropaSecurityMeasuresForPurpose(purpose string) []string {
	if purpose == "fraud_detection" {
		return []string{"Automated anomaly detection", "Manual review workflow"}
	}
	return []string{"AES-256-GCM encryption at rest", "Hash-chained audit logs", "Role-based access control"}
}

func ropaDataCategoriesFromEvent(data interface{}, salt string) []string {
	set := map[string]struct{}{}
	collectROPADataCategories(data, salt, set)
	if len(set) == 0 {
		set["usage_metrics"] = struct{}{}
	}
	values := make([]string, 0, len(set))
	for v := range set {
		values = append(values, v)
	}
	sort.Strings(values)
	return values
}

func collectROPADataCategories(value interface{}, salt string, set map[string]struct{}) {
	switch typed := value.(type) {
	case map[string]interface{}:
		for k, v := range typed {
			category := gdprFieldCategory(k)
			switch category {
			case "direct":
				if strings.EqualFold(k, "ip") {
					set["ip_address"] = struct{}{}
				} else {
					set["identity"] = struct{}{}
				}
			case "sensitive":
				set["sensitive_attributes"] = struct{}{}
			case "metadata":
				set["usage_metrics"] = struct{}{}
			case "third_party":
				set["usage_metrics"] = struct{}{}
			default:
				set["usage_metrics"] = struct{}{}
			}
			collectROPADataCategories(v, salt, set)
		}
	case []interface{}:
		for _, item := range typed {
			collectROPADataCategories(item, salt, set)
		}
	}
}

func quarterString(now time.Time) string {
	quarter := ((int(now.Month()) - 1) / 3) + 1
	return fmt.Sprintf("%04d-Q%d", now.Year(), quarter)
}

func validateGDPRRetention(now time.Time, bundlePath string) error {
	policy, err := loadRetentionPolicy(defaultConfigPath())
	if err != nil {
		return fmt.Errorf("load retention policy: %w", err)
	}
	if policy == nil || policy.Days <= 0 {
		return nil
	}
	info, err := os.Stat(bundlePath)
	if err != nil {
		return fmt.Errorf("stat bundle for retention check: %w", err)
	}
	cutoff := now.UTC().AddDate(0, 0, -policy.Days)
	if info.ModTime().UTC().Before(cutoff) {
		return fmt.Errorf("bundle retention policy expired for requested data range")
	}
	return nil
}

func isSOC2RelevantType(eventType string) bool {
	for _, control := range soc2Controls {
		if _, ok := control.EventTypes[eventType]; ok {
			return true
		}
	}
	return false
}

func eventIdentifier(rec bundle.Record) string {
	if data, ok := rec.Event.Data.(map[string]interface{}); ok {
		if raw, ok := data["event_id"]; ok {
			if id, ok := raw.(string); ok && strings.TrimSpace(id) != "" {
				return id
			}
		}
		if raw, ok := data["id"]; ok {
			if id, ok := raw.(string); ok && strings.TrimSpace(id) != "" {
				return id
			}
		}
	}
	if data, ok := rec.Event.Data.(map[string]any); ok {
		if raw, ok := data["event_id"]; ok {
			if id, ok := raw.(string); ok && strings.TrimSpace(id) != "" {
				return id
			}
		}
		if raw, ok := data["id"]; ok {
			if id, ok := raw.(string); ok && strings.TrimSpace(id) != "" {
				return id
			}
		}
	}
	return fmt.Sprintf("evt_%06d", rec.Event.Sequence)
}

func exportEventTimestamp(rec bundle.Record, fallback string) string {
	if data, ok := rec.Event.Data.(map[string]interface{}); ok {
		for _, key := range []string{"timestamp", "ts"} {
			if raw, ok := data[key]; ok {
				if value, ok := raw.(string); ok {
					if parsed, ok := parseRFC3339(value); ok {
						return parsed.UTC().Format(time.RFC3339)
					}
				}
			}
		}
	}
	if data, ok := rec.Event.Data.(map[string]any); ok {
		for _, key := range []string{"timestamp", "ts"} {
			if raw, ok := data[key]; ok {
				if value, ok := raw.(string); ok {
					if parsed, ok := parseRFC3339(value); ok {
						return parsed.UTC().Format(time.RFC3339)
					}
				}
			}
		}
	}
	return fallback
}

func parseRFC3339(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func setOf(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func hashPIIString(input string) string {
	return "sha256:" + sha256Hex([]byte(input))
}

func containsStringValue(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
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
	raw, err := os.ReadFile(path) // #nosec G304 -- bundle path is provided by verified export inputs
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
	raw, err := os.ReadFile(configPath) // #nosec G304 -- config path is the project-local default config file
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

func discoverComplianceDocs() ([]string, error) {
	paths, err := atbembed.ListComplianceDocs()
	if err == nil && len(paths) > 0 {
		return paths, nil
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("discover embedded compliance docs: %w", err)
	}

	matches, err := filepath.Glob(filepath.Join("docs", "compliance", "*.md"))
	if err != nil {
		return nil, fmt.Errorf("discover fallback compliance docs: %w", err)
	}
	paths = make([]string, 0, len(matches))
	for _, p := range matches {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			paths = append(paths, filepath.Clean(p))
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func readExportDoc(doc string) ([]byte, error) {
	embeddedDocPath := path.Clean(filepath.ToSlash(doc))
	data, err := atbembed.ReadExportDoc(embeddedDocPath)
	if err == nil {
		return data, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	data, readErr := os.ReadFile(filepath.FromSlash(embeddedDocPath)) // #nosec G304 -- fallback reads from repo-controlled doc paths
	if readErr != nil {
		return nil, readErr
	}
	return data, nil
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

func writeExportZip(result exportBuildResult, modTime time.Time) error {
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
			Name:     filepath.ToSlash(file.ZipPath),
			Method:   zip.Deflate,
			Modified: modTime.UTC(),
		}
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

func writeExportVerificationSidecar(cfg exportConfig, result exportBuildResult, stderr io.Writer) (verifypkg.Report, *exportCommandVerification, error) {
	sidecarPath, notice := exportVerificationSidecarPath(result.OutputPath)
	if notice != "" {
		fmt.Fprintln(stderr, notice)
		return verifypkg.Report{}, nil, nil
	}

	bundlePath := exportVerificationBundlePath(cfg, result)
	b, err := bundle.Load(bundlePath)
	if err != nil {
		return verifypkg.Report{}, nil, fmt.Errorf("load bundle for verification: %w", err)
	}

	report := verifypkg.Verify(b, bundlePath, "")
	if report.Integrity.Error != "" {
		return verifypkg.Report{}, nil, fmt.Errorf("verify bundle for export: %s", report.Integrity.Error)
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return verifypkg.Report{}, nil, fmt.Errorf("marshal verification sidecar: %w", err)
	}
	if err := os.WriteFile(sidecarPath, data, 0o644); err != nil { // #nosec G703 — sidecarPath is derived from the cleaned --output path plus a fixed ".verify.json" suffix
		return verifypkg.Report{}, nil, fmt.Errorf("write verification sidecar: %w", err)
	}

	return report, &exportCommandVerification{
		Passed:       verificationExitCode(report) == exitSuccess,
		Grade:        exportVerificationGrade(report),
		ResidualRisk: exportVerificationResidualRisk(report),
		Sidecar:      sidecarPath,
	}, nil
}

func exportVerificationBundlePath(cfg exportConfig, result exportBuildResult) string {
	if strings.TrimSpace(cfg.BundlePath) != "" {
		return filepath.Clean(cfg.BundlePath)
	}
	for _, info := range result.BundleFiles {
		if !info.Archived && strings.TrimSpace(info.Source) != "" {
			return filepath.Clean(info.Source)
		}
	}
	return bundle.DefaultPath()
}

func exportVerificationSidecarPath(outputPath string) (string, string) {
	if strings.TrimSpace(outputPath) == "" {
		return "", "warning: --with-verify ignored when writing to stdout"
	}

	cleaned := filepath.Clean(outputPath)
	return cleaned + ".verify.json", ""
}

func exportVerificationGrade(report verifypkg.Report) string {
	if report.CAS == nil {
		return "D"
	}

	switch report.CAS.Grade {
	case "High":
		return "A"
	case "Medium":
		return "B"
	case "Low":
		return "C"
	default:
		return "D"
	}
}

func exportVerificationResidualRisk(report verifypkg.Report) float64 {
	if report.CAS == nil {
		return 1.0
	}

	risk := 1 - report.CAS.Overall
	if risk < 0 {
		risk = 0
	}
	if risk > 1 {
		risk = 1
	}
	return math.Round(risk*100) / 100
}

func printExportUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb export --format <compliance|soc2|gdpr> --output <path.zip> [--bundle <path>] [--type dsr|ropa] [--subject-id <id>] [--dry-run] [--json] [--with-verify]")
	fmt.Fprintln(w, "  --with-verify    write <output>.verify.json sidecar with full verify report")
}
