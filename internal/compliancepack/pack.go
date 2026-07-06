// SPDX-License-Identifier: MIT
// Package compliancepack builds deterministic, profile-aware offline evidence
// packages for regulatory review.
package compliancepack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	atbembed "github.com/pcguest/atb"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/incident"
	"github.com/pcguest/atb/internal/retentionaudit"
	"github.com/pcguest/atb/internal/verify"
	custodypkg "github.com/pcguest/atb/pkg/custody"
)

const RegimeEUAIAct = "eu-ai-act"

// File is one artifact in a compliance pack.
type File struct {
	Name    string
	Content []byte
}

// FileEntry describes one artifact covered by MANIFEST.json.
type FileEntry struct {
	Name        string `json:"name"`
	SHA256      string `json:"sha256"`
	SizeBytes   int    `json:"size_bytes"`
	Description string `json:"description"`
}

// Manifest is the deterministic root inventory for a compliance pack.
type Manifest struct {
	PackVersion   string      `json:"pack_version"`
	GeneratedAt   string      `json:"generated_at"`
	ATBVersion    string      `json:"atb_version"`
	Regime        string      `json:"regime"`
	ProfileID     string      `json:"profile_id"`
	BundleFile    string      `json:"bundle_file"`
	BundleHash    string      `json:"bundle_hash"`
	IntegrityPass bool        `json:"integrity_pass"`
	ProfilePass   bool        `json:"profile_pass"`
	CASScore      float64     `json:"cas_score"`
	CASGrade      string      `json:"cas_grade,omitempty"`
	Files         []FileEntry `json:"files"`
}

// Pack is a complete in-memory compliance package.
type Pack struct {
	GeneratedAt time.Time
	Files       []File
	Manifest    Manifest
}

type casDocument struct {
	ProfileID string             `json:"profile_id"`
	Score     float64            `json:"score"`
	Grade     string             `json:"grade"`
	SubScores map[string]float64 `json:"sub_scores,omitempty"`
}

const readme = `# ATB compliance evidence pack

This offline package contains one authoritative ATB bundle and profile-scoped
review artifacts generated from it.

Start with:
- reports/verify.report.json: integrity, profile obligations, CAS, residual risk.
- reports/trust-report.md: reviewer-oriented interpretation.
- reports/obligations.json: observed, missing, and warning-level evidence.
- incidents/: session-scoped forensic reports when the bundle contains sessions.
- retention/: project operations evidence relevant to the bundle, when present.
- MANIFEST.json and SHA256SUMS: artifact inventory and content digests.

ATB proves integrity of recorded evidence. It does not prove capture
completeness, the truth of caller-provided identity evidence, or regulatory
compliance by itself.
`

// Build creates an offline compliance pack for one bundle and profile.
func Build(ctx context.Context, bundlePath, profileSpec, regime, toolVersion string) (Pack, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(bundlePath) == "" {
		return Pack{}, fmt.Errorf("compliance pack: bundle path is required")
	}
	if strings.TrimSpace(profileSpec) == "" {
		return Pack{}, fmt.Errorf("compliance pack: profile is required")
	}
	if regime == "" {
		regime = RegimeEUAIAct
	}
	if regime != RegimeEUAIAct {
		return Pack{}, fmt.Errorf("compliance pack: unsupported regime %q", regime)
	}

	b, err := bundle.Load(ctx, bundlePath)
	if err != nil {
		return Pack{}, fmt.Errorf("compliance pack: load bundle: %w", err)
	}
	profile, err := resolveProfile(profileSpec)
	if err != nil {
		return Pack{}, err
	}
	report, err := verify.EvaluateBundle(verify.EvaluateConfig{
		BundlePath: bundlePath,
		Records:    b.Records,
		Profiles:   []verify.Profile{profile},
	})
	if err != nil {
		return Pack{}, fmt.Errorf("compliance pack: verify: %w", err)
	}
	verifierReport := verify.ReportFromVerifyWithBundle(*report, b)
	trustReport := verify.TrustReportFromVerify(*report, b)
	generatedAt := bundleTimestamp(b)

	bundleBytes, err := os.ReadFile(filepath.Clean(bundlePath)) // #nosec G304 -- validated by bundle.Load
	if err != nil {
		return Pack{}, fmt.Errorf("compliance pack: read bundle: %w", err)
	}
	verifyJSON, err := marshalJSON(verifierReport)
	if err != nil {
		return Pack{}, err
	}
	trustJSON, err := marshalJSON(trustReport)
	if err != nil {
		return Pack{}, err
	}
	obligationsJSON, err := marshalJSON(verifierReport.Obligations)
	if err != nil {
		return Pack{}, err
	}
	casJSON, err := marshalJSON(casDocument{
		ProfileID: verifierReport.ProfileID,
		Score:     verifierReport.CASScore,
		Grade:     verifierReport.CASGrade,
		SubScores: verifierReport.SubScores,
	})
	if err != nil {
		return Pack{}, err
	}

	files := []File{
		{Name: "README.md", Content: []byte(readme)},
		{Name: "bundle.atb", Content: bundleBytes},
		{Name: "reports/verify.report.json", Content: verifyJSON},
		{Name: "reports/trust-report.json", Content: trustJSON},
		{Name: "reports/trust-report.md", Content: []byte(renderTrustMarkdown(trustReport))},
		{Name: "reports/cas.json", Content: casJSON},
		{Name: "reports/obligations.json", Content: obligationsJSON},
	}

	files, err = appendIncidentFiles(ctx, files, bundlePath)
	if err != nil {
		return Pack{}, err
	}
	files, err = appendRetentionFiles(files, bundlePath, custodypkg.HeadHash(b))
	if err != nil {
		return Pack{}, err
	}
	for _, doc := range []string{
		"docs/compliance/eu-ai-act.md",
		"docs/profiles.md",
		"docs/cas-guide.md",
	} {
		content, readErr := atbembed.ReadExportDoc(doc)
		if readErr != nil {
			return Pack{}, fmt.Errorf("compliance pack: read embedded %s: %w", doc, readErr)
		}
		files = append(files, File{Name: doc, Content: content})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	manifest := Manifest{
		PackVersion:   "compliance.pack.v1",
		GeneratedAt:   generatedAt.Format(time.RFC3339),
		ATBVersion:    toolVersion,
		Regime:        regime,
		ProfileID:     profile.ID(),
		BundleFile:    "bundle.atb",
		BundleHash:    custodypkg.HeadHash(b),
		IntegrityPass: verifierReport.GateResult.ChainValid,
		ProfilePass:   verifierReport.GateResult.ProfilePass,
		CASScore:      verifierReport.CASScore,
		CASGrade:      verifierReport.CASGrade,
	}
	for _, file := range files {
		sum := sha256.Sum256(file.Content)
		manifest.Files = append(manifest.Files, FileEntry{
			Name:        file.Name,
			SHA256:      hex.EncodeToString(sum[:]),
			SizeBytes:   len(file.Content),
			Description: describe(file.Name),
		})
	}
	manifestJSON, err := marshalJSON(manifest)
	if err != nil {
		return Pack{}, err
	}
	files = append(files, File{Name: "MANIFEST.json", Content: manifestJSON})
	files = append(files, File{Name: "SHA256SUMS", Content: checksumFile(manifest.Files)})
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	return Pack{GeneratedAt: generatedAt, Files: files, Manifest: manifest}, nil
}

func resolveProfile(spec string) (verify.Profile, error) {
	spec = strings.TrimSpace(spec)
	if profile := verify.ProfileByID(spec); profile != nil {
		return profile, nil
	}
	if !strings.HasPrefix(spec, "atb.profile.") {
		if profile := verify.ProfileByID("atb.profile." + spec); profile != nil {
			return profile, nil
		}
	}
	profile, err := verify.ResolveProfile(spec)
	if err != nil {
		return nil, fmt.Errorf("compliance pack: resolve profile %q: %w", spec, err)
	}
	return profile, nil
}

func appendIncidentFiles(ctx context.Context, files []File, bundlePath string) ([]File, error) {
	sessions, err := incident.ListSessions(ctx, bundlePath)
	if err != nil {
		return nil, fmt.Errorf("compliance pack: list incidents: %w", err)
	}
	indexJSON, err := marshalJSON(sessions)
	if err != nil {
		return nil, err
	}
	files = append(files, File{Name: "incidents/index.json", Content: indexJSON})
	for _, session := range sessions {
		report, buildErr := incident.Build(ctx, bundlePath, session.SessionID)
		if buildErr != nil {
			return nil, fmt.Errorf("compliance pack: incident %s: %w", session.SessionID, buildErr)
		}
		if !report.Found {
			continue
		}
		reportJSON, marshalErr := report.JSON()
		if marshalErr != nil {
			return nil, marshalErr
		}
		reportJSON = append(reportJSON, '\n')
		reportNDJSON, marshalErr := report.NDJSON()
		if marshalErr != nil {
			return nil, marshalErr
		}
		base := "incidents/" + safeName(session.SessionID)
		files = append(files,
			File{Name: base + ".md", Content: []byte(report.Markdown())},
			File{Name: base + ".json", Content: reportJSON},
			File{Name: base + ".ndjson", Content: reportNDJSON},
		)
	}
	return files, nil
}

func appendRetentionFiles(files []File, bundlePath, bundleHash string) ([]File, error) {
	auditPath := retentionaudit.PathForBundle(bundlePath)
	auditBundle, err := bundle.LoadVerified(auditPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return files, nil
		}
		return nil, fmt.Errorf("compliance pack: load retention audit: %w", err)
	}
	var relevant []bundle.Record
	for _, record := range auditBundle.Records {
		switch record.Event.Type {
		case event.TypeDataRetentionPolicySet, event.TypeDataRetentionPolicyChanged:
			relevant = append(relevant, record)
		case event.TypeDataRetentionEnforced:
			data, _ := record.Event.Data.(map[string]any)
			recordBundleHash, _ := data["bundle_hash"].(string)
			if recordBundleHash == "" || recordBundleHash == bundleHash {
				relevant = append(relevant, record)
			}
		}
	}
	if len(relevant) == 0 {
		return files, nil
	}
	raw, err := os.ReadFile(filepath.Clean(auditPath)) // #nosec G304 -- derived project-local path
	if err != nil {
		return nil, fmt.Errorf("compliance pack: read retention audit: %w", err)
	}
	relevantJSON, err := marshalJSON(relevant)
	if err != nil {
		return nil, err
	}
	return append(files,
		File{Name: "retention/operations.atb", Content: raw},
		File{Name: "retention/events.json", Content: relevantJSON},
	), nil
}

func safeName(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "session"
	}
	return b.String()
}

func bundleTimestamp(b *bundle.Bundle) time.Time {
	if b != nil && len(b.Records) > 0 {
		if parsed, err := time.Parse(time.RFC3339Nano, b.Records[0].Event.Timestamp); err == nil {
			return parsed.UTC()
		}
	}
	return time.Unix(0, 0).UTC()
}

func renderTrustMarkdown(report verify.TrustReport) string {
	var b strings.Builder
	fmt.Fprintln(&b, "# ATB trust report")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "- Bundle: `%s`\n", report.BundlePath)
	fmt.Fprintf(&b, "- Profile: `%s`\n", report.ProfileID)
	fmt.Fprintf(&b, "- Result: **%s**\n", passLabel(report.Pass))
	fmt.Fprintf(&b, "- Hash chain: **%s**\n", passLabel(report.Chain.Valid))
	fmt.Fprintf(&b, "- CAS: %.3f (%s)\n", report.CASScore, report.CASGrade)
	fmt.Fprintf(&b, "- Residual risk: %s\n", report.ResidualRisk)
	for _, section := range report.Sections {
		fmt.Fprintf(&b, "\n## %s\n\n", section.Title)
		fmt.Fprintf(&b, "- Result: %s\n", passLabel(section.Pass))
		keys := make([]string, 0, len(section.Fields))
		for key := range section.Fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(&b, "- `%s`: `%s`\n", key, section.Fields[key])
		}
		for _, note := range section.Notes {
			fmt.Fprintf(&b, "- %s\n", note)
		}
	}
	if len(report.ReviewerIdentities) > 0 {
		fmt.Fprintln(&b, "\n## Reviewer identity evidence")
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "Caller-provided and hash-chained; not independently verified by ATB.")
		for _, evidence := range report.ReviewerIdentities {
			fmt.Fprintf(&b, "- provider=`%s` subject=`%s` assertion=`%s` digest=`%s`\n",
				evidence.IdentityProvider, evidence.Subject, evidence.AssertionType, evidence.AssertionDigest)
		}
	}
	return b.String()
}

func passLabel(pass bool) string {
	if pass {
		return "PASS"
	}
	return "FAIL"
}

func marshalJSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("compliance pack: marshal JSON: %w", err)
	}
	return append(data, '\n'), nil
}

func checksumFile(entries []FileEntry) []byte {
	var b strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&b, "%s  %s\n", entry.SHA256, entry.Name)
	}
	return []byte(b.String())
}

func describe(name string) string {
	switch {
	case name == "bundle.atb":
		return "Authoritative tamper-evident ATB bundle."
	case name == "reports/verify.report.json":
		return "Stable verify.report.v1 profile evaluation."
	case name == "reports/cas.json":
		return "CAS score, grade, and sub-scores."
	case name == "reports/obligations.json":
		return "Machine-readable profile obligation outcomes."
	case strings.HasPrefix(name, "incidents/"):
		return "Session-scoped incident-forensics artifact."
	case strings.HasPrefix(name, "retention/"):
		return "Retention policy or enforcement audit evidence."
	case name == "mortise/receipt.json":
		return "Signed Mortise custody receipt for the authoritative bundle."
	case strings.HasPrefix(name, "docs/"):
		return "Reference documentation included for reviewer context."
	default:
		return "Compliance evidence pack artifact."
	}
}
