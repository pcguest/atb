// SPDX-License-Identifier: MIT
// Package evidencepack builds multi-bundle verification summaries for the
// atb evidence pack command without altering verify.report.v1 or CAS contracts.
package evidencepack

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/verify"
	custodypkg "github.com/pcguest/atb/pkg/custody"
)

const defaultPackTimeout = 5 * time.Minute

// ResidualRiskSummary mirrors residual risk fields exposed in evidence pack JSON.
type ResidualRiskSummary struct {
	Level                   string   `json:"level,omitempty"`
	Drivers                 []string `json:"drivers,omitempty"`
	RecommendedNextEvidence []string `json:"recommended_next_evidence,omitempty"`
}

// BundleEvidenceSummary is one bundle entry in an evidence pack document.
type BundleEvidenceSummary struct {
	BundlePath              string               `json:"bundle_path"`
	ProfileID               string               `json:"profile_id,omitempty"`
	ProfileVersion          int                  `json:"profile_version,omitempty"`
	HeadHash                string               `json:"head_hash,omitempty"`
	IntegrityPass           bool                 `json:"integrity_pass"`
	ProfilePass             bool                 `json:"profile_pass"`
	CASScore                float64              `json:"cas_score,omitempty"`
	CASGrade                string               `json:"cas_grade,omitempty"`
	ResidualRisk            *ResidualRiskSummary `json:"residual_risk,omitempty"`
	RecommendedNextEvidence []string             `json:"recommended_next_evidence,omitempty"`
	Exclusions              []string             `json:"exclusions,omitempty"`
	Error                   string               `json:"error,omitempty"`
}

// Pack is the top-level evidence pack JSON document.
type Pack struct {
	Bundles []BundleEvidenceSummary `json:"bundles"`
}

// PackPaths verifies each bundle path and returns a combined evidence pack.
// The returned bool is true when any bundle entry carries an error (IO or internal).
// When userError is true, at least one failure was a missing/invalid user path.
func PackPaths(ctx context.Context, paths []string) (Pack, bool, bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, defaultPackTimeout)
	defer cancel()

	pack := Pack{
		Bundles: make([]BundleEvidenceSummary, 0, len(paths)),
	}
	anyErrors := false
	userError := false
	for _, rawPath := range paths {
		summary, err := packOne(ctx, rawPath)
		if err != nil {
			anyErrors = true
			if isUserPathError(err) {
				userError = true
			}
			summary.Error = err.Error()
		}
		pack.Bundles = append(pack.Bundles, summary)
	}
	return pack, anyErrors, userError
}

func packOne(ctx context.Context, rawPath string) (BundleEvidenceSummary, error) {
	path := strings.TrimSpace(rawPath)
	summary := BundleEvidenceSummary{BundlePath: path}
	if path == "" {
		return summary, fmt.Errorf("bundle path is required")
	}

	b, err := bundle.Load(ctx, path)
	if err != nil {
		return summary, err
	}
	if err := validateBundleManifest(b); err != nil {
		return summary, err
	}

	summary.HeadHash = custodypkg.HeadHash(b)

	report, err := verify.EvaluateBundle(verify.EvaluateConfig{
		BundlePath:    path,
		Records:       b.Records,
		AllApplicable: true,
	})
	if err != nil {
		return summary, err
	}

	verifierReport := verify.ReportFromVerify(*report)
	summary.IntegrityPass = verifierReport.GateResult.ChainValid
	summary.ProfilePass = verifierReport.GateResult.ProfilePass
	summary.ProfileID = verifierReport.ProfileID
	summary.ProfileVersion = verifierReport.ProfileVersion
	summary.CASScore = verifierReport.CASScore
	summary.CASGrade = verifierReport.CASGrade

	if len(verifierReport.Exclusions) > 0 {
		summary.Exclusions = append([]string(nil), verifierReport.Exclusions...)
	}

	if verifierReport.ResidualRisk.Level != "" ||
		len(verifierReport.ResidualRisk.Drivers) > 0 ||
		len(verifierReport.ResidualRisk.RecommendedNextEvidence) > 0 {
		summary.ResidualRisk = &ResidualRiskSummary{
			Level:                   verifierReport.ResidualRisk.Level,
			Drivers:                 append([]string(nil), verifierReport.ResidualRisk.Drivers...),
			RecommendedNextEvidence: append([]string(nil), verifierReport.ResidualRisk.RecommendedNextEvidence...),
		}
	}
	if len(verifierReport.ResidualRisk.RecommendedNextEvidence) > 0 {
		summary.RecommendedNextEvidence = append(
			[]string(nil),
			verifierReport.ResidualRisk.RecommendedNextEvidence...,
		)
	}

	return summary, nil
}

func validateBundleManifest(b *bundle.Bundle) error {
	if b == nil || len(b.Records) == 0 {
		return fmt.Errorf("bundle: missing manifest record")
	}
	if b.Records[0].Event.Type != bundle.ManifestEventType {
		return fmt.Errorf("bundle: missing manifest record")
	}
	if _, err := b.ManifestE(); err != nil {
		return err
	}
	return nil
}

func isUserPathError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, verify.ErrBundleNotFound) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "no such file")
}
