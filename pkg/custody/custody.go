// SPDX-License-Identifier: MIT
// Package custody exposes a minimal, stable ingest surface for Mortise and other
// hosted custody services. It wraps ATB's verifier without reimplementing
// bundle parsing, hashing, or profile evaluation.
//
// Custody artefact (frozen contract):
//   - Raw .atb bundle bytes (content-addressed by head hash)
//   - verify.report.v1 JSON produced by ReportFromVerify
//
// Mortise Phase 1 stores both artefacts immutably and must not mutate or
// re-score the report.
package custody

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pcguest/atb/v2/internal/bundle"
	"github.com/pcguest/atb/v2/internal/hash"
	"github.com/pcguest/atb/v2/internal/verify"
)

// VerifierReport is the stable verify.report.v1 automation contract.
type VerifierReport = verify.VerifierReport

// VerifyReportVersion is the report_version value written by ATB.
const VerifyReportVersion = verify.VerifyReportVersion

// EvalResult holds the content-addressed head hash and verifier output for one bundle.
type EvalResult struct {
	HeadHash string
	Report   VerifierReport
}

// HeadHash returns the SHA-256 head hash of b (last record hash, or genesis when empty).
func HeadHash(b *bundle.Bundle) string {
	if b == nil || len(b.Records) == 0 {
		return hash.GenesisHash
	}
	return b.Records[len(b.Records)-1].Hash
}

// LoadBundle parses bundle bytes without verifying the hash chain.
func LoadBundle(data []byte) (*bundle.Bundle, error) {
	b, err := bundle.LoadReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("custody: load bundle: %w", err)
	}
	if len(b.Records) == 0 {
		return nil, fmt.Errorf("custody: empty bundle")
	}
	return b, nil
}

// Evaluate loads bundle bytes and evaluates them through the same full path as
// `atb verify` (verify.EvaluateBundle followed by
// verify.ReportFromVerifyWithBundle), so custody ingest and the CLI share one
// evaluation implementation. It returns the head hash plus verify.report.v1
// output.
//
// Profile IDs may be given as "atb.profile.<name>" or bare "<name>". An unknown
// profile fails closed with an error wrapping verify.ErrProfileUnknown instead
// of producing a report with no selected profile.
//
// Custody exposes no anchor, strict-source-signature, or corroboration options.
// Evaluator-only settings such as EvaluateConfig.AnchorRequired therefore
// remain CLI-only and are not reachable through this API.
//
// bundlePath is recorded in the report only; use a synthetic path such as
// "mortise://ingest" when none exists.
func Evaluate(data []byte, profileID, bundlePath string) (EvalResult, error) {
	b, err := LoadBundle(data)
	if err != nil {
		return EvalResult{}, err
	}
	if bundlePath == "" {
		bundlePath = "mortise://ingest"
	}
	profile, err := resolveProfile(profileID)
	if err != nil {
		return EvalResult{}, err
	}
	report, err := verify.EvaluateBundle(verify.EvaluateConfig{
		BundlePath: bundlePath,
		Records:    b.Records,
		Profiles:   []verify.Profile{profile},
	})
	if err != nil {
		return EvalResult{}, fmt.Errorf("custody: evaluate bundle: %w", err)
	}
	vr := verify.ReportFromVerifyWithBundle(*report, b)
	return EvalResult{
		HeadHash: HeadHash(b),
		Report:   vr,
	}, nil
}

// resolveProfile accepts a registered profile ID, with or without the
// "atb.profile." prefix, matching the profile resolution used by `atb verify`.
func resolveProfile(profileID string) (verify.Profile, error) {
	spec := strings.TrimSpace(profileID)
	profile := verify.ProfileByID(spec)
	if profile == nil && !strings.HasPrefix(spec, "atb.profile.") {
		profile = verify.ProfileByID("atb.profile." + spec)
	}
	if profile == nil {
		return nil, fmt.Errorf("custody: profile %q not found: %w", spec, verify.ErrProfileUnknown)
	}
	return profile, nil
}
