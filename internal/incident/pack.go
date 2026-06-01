// SPDX-License-Identifier: MIT
package incident

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/verify"
)

// PackFile is one named artifact in an incident evidence package.
type PackFile struct {
	Name    string
	Content []byte
}

// FileDigest records the SHA-256 of a packed artifact.
type FileDigest struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

// Manifest is the chain-of-custody manifest at the root of an evidence package.
type Manifest struct {
	GeneratedAt        string                       `json:"generated_at"`
	ToolVersion        string                       `json:"tool_version"`
	BundleFile         string                       `json:"bundle_file"`
	SessionID          string                       `json:"session_id"`
	IntegrityValid     bool                         `json:"integrity_valid"`
	ChainHeadHash      string                       `json:"chain_head_hash"`
	Signatures         []verify.SignatureProvenance `json:"signatures,omitempty"`
	CaptureScope       *CaptureScope                `json:"capture_scope,omitempty"`
	Files              []FileDigest                 `json:"files"`
	VerifyInstructions string                       `json:"verify_instructions"`
}

const packReadme = `# ATB incident evidence package

This package is a self-contained, independently verifiable record of one agent
session.

Contents:
- bundle.atb            The authoritative, tamper-evident hash-chained bundle.
- incident-report.md    Human-readable session report.
- incident-report.json  Machine-readable session report.
- verify-report.json    Full verifier output (integrity, signatures, anchoring).
- MANIFEST.json         Chain-of-custody manifest: bundle hash, signature
                        provenance, capture scope, and a SHA-256 of every file
                        in this package.

To verify independently (no trust in this package required):

    atb verify bundle.atb

The hash chain proves the records were not altered; the signature provenance in
MANIFEST.json attributes the bundle to a signing key; each event in the report
carries its record hash, checkable against bundle.atb.
`

// BuildPack assembles a self-contained incident evidence package for one
// session: the signed bundle, the incident report (markdown + JSON), the full
// verifier report, and a chain-of-custody manifest digesting every file. The
// MANIFEST.json is produced last so it covers all other artifacts.
func BuildPack(ctx context.Context, bundlePath, sessionID, toolVersion string) ([]PackFile, error) {
	report, err := Build(ctx, bundlePath, sessionID)
	if err != nil {
		return nil, err
	}
	if !report.Found {
		return nil, fmt.Errorf("incident: no events found for session %q", sessionID)
	}

	bundleBytes, err := os.ReadFile(bundlePath) // #nosec G304 -- operator-supplied bundle path
	if err != nil {
		return nil, fmt.Errorf("incident: read bundle %q: %w", bundlePath, err)
	}

	reportJSON, err := report.JSON()
	if err != nil {
		return nil, err
	}

	files := []PackFile{
		{Name: "bundle.atb", Content: bundleBytes},
		{Name: "incident-report.md", Content: []byte(report.Markdown())},
		{Name: "incident-report.json", Content: reportJSON},
		{Name: "README.md", Content: []byte(packReadme)},
	}

	// Full verifier report (best effort; integrity is already in the manifest).
	if b, lerr := bundle.Load(bundlePath); lerr == nil {
		if vr, verr := verify.EvaluateBundle(verify.EvaluateConfig{
			BundlePath:    bundlePath,
			Records:       b.Records,
			AllApplicable: true,
		}); verr == nil && vr != nil {
			if vj, merr := json.MarshalIndent(vr, "", "  "); merr == nil {
				files = append(files, PackFile{Name: "verify-report.json", Content: vj})
			}
		}
	}

	manifest := Manifest{
		GeneratedAt:        time.Now().UTC().Format(time.RFC3339),
		ToolVersion:        toolVersion,
		BundleFile:         filepath.Base(bundlePath),
		SessionID:          sessionID,
		IntegrityValid:     report.IntegrityValid,
		ChainHeadHash:      report.ChainHeadHash,
		Signatures:         report.Signatures,
		CaptureScope:       report.CaptureScope,
		VerifyInstructions: "Recompute integrity independently with: atb verify bundle.atb",
	}
	for _, f := range files {
		sum := sha256.Sum256(f.Content)
		manifest.Files = append(manifest.Files, FileDigest{Name: f.Name, SHA256: hex.EncodeToString(sum[:])})
	}
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	// MANIFEST.json last: it digests every other artifact.
	files = append(files, PackFile{Name: "MANIFEST.json", Content: manifestJSON})
	return files, nil
}
