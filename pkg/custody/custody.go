// SPDX-License-Identifier: MIT
// Package custody exposes a minimal, stable ingest surface for Custos and other
// hosted custody services. It wraps ATB's verifier without reimplementing
// bundle parsing, hashing, or profile evaluation.
//
// Custody artefact (frozen contract):
//   - Raw .atb bundle bytes (content-addressed by head hash)
//   - verify.report.v1 JSON produced by ReportFromVerify
//
// Custos Phase 1 stores both artefacts immutably and must not mutate or
// re-score the report.
package custody

import (
	"bytes"
	"fmt"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/hash"
	"github.com/pcguest/atb/internal/verify"
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

// Evaluate loads bundle bytes, runs ATB verification for profileID, and returns
// the head hash plus verify.report.v1 output. bundlePath is recorded in the
// report only; use a synthetic path such as "custos://ingest" when none exists.
func Evaluate(data []byte, profileID, bundlePath string) (EvalResult, error) {
	b, err := LoadBundle(data)
	if err != nil {
		return EvalResult{}, err
	}
	if bundlePath == "" {
		bundlePath = "custos://ingest"
	}
	report := verify.Verify(b, bundlePath, profileID)
	vr := verify.ReportFromVerify(report)
	return EvalResult{
		HeadHash: HeadHash(b),
		Report:   vr,
	}, nil
}
