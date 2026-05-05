// SPDX-License-Identifier: MIT
package verify

import (
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
)

var (
	// ErrBundleNotFound is returned when the bundle path does not exist.
	ErrBundleNotFound = errors.New("bundle not found")
	// ErrChainInvalid is returned when hash-chain verification fails.
	ErrChainInvalid = errors.New("hash chain invalid")
	// ErrProfileUnknown is returned when the requested profile ID is not registered.
	ErrProfileUnknown = errors.New("unknown profile")
)

type evaluateOpts struct {
	corroboration *CorroborationPolicy
}

// EvaluateOption configures optional behaviour for EvaluateBundle.
type EvaluateOption func(*evaluateOpts)

// WithCorroborationPolicy applies a CorroborationPolicy when computing the
// effective CAS score. A nil policy is a no-op (backward-compatible).
func WithCorroborationPolicy(p *CorroborationPolicy) EvaluateOption {
	return func(c *evaluateOpts) { c.corroboration = p }
}

// EvaluateConfig describes one bundle evaluation request.
type EvaluateConfig struct {
	BundlePath        string
	Records           []bundle.Record
	Profiles          []Profile
	AllApplicable     bool
	AnchorRoots       *x509.CertPool
	AnchorRequired    bool
	RequireValidChain bool
}

// EvaluateBundle loads the requested bundle source and evaluates it with the
// selected profiles. Use WithCorroborationPolicy to apply a verified-evidence
// bonus to the base CAS score.
func EvaluateBundle(cfg EvaluateConfig, opts ...EvaluateOption) (*Report, error) {
	eopts := &evaluateOpts{}
	for _, o := range opts {
		o(eopts)
	}
	if len(cfg.Profiles) == 0 && !cfg.AllApplicable {
		return nil, errors.New("verify: no profiles supplied")
	}
	if len(cfg.Records) == 0 && strings.TrimSpace(cfg.BundlePath) == "" {
		return nil, errors.New("verify: bundle path or records required")
	}

	selected := normaliseProfiles(cfg.Profiles)
	if len(cfg.Profiles) > 0 && len(selected) == 0 && !cfg.AllApplicable {
		return nil, fmt.Errorf("verify: all supplied profiles were nil: %w", ErrProfileUnknown)
	}

	var b *bundle.Bundle
	if len(cfg.Records) > 0 {
		b = &bundle.Bundle{
			Records: append([]bundle.Record(nil), cfg.Records...),
		}
	} else {
		loaded, err := bundle.Load(cfg.BundlePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("verify: %s: %w", cfg.BundlePath, ErrBundleNotFound)
			}
			return nil, err
		}
		b = loaded
	}

	report := evaluateLoadedBundle(
		b,
		cfg.BundlePath,
		selected,
		cfg.AllApplicable,
		cfg.AnchorRoots,
		cfg.AnchorRequired,
	)

	if cfg.RequireValidChain && !report.Integrity.ChainValid {
		return nil, fmt.Errorf("verify: %s: %w", cfg.BundlePath, ErrChainInvalid)
	}

	if eopts.corroboration != nil && report.CAS != nil {
		anchorVerified := report.Anchoring.TSAVerified
		signatureVerified := report.BundleSignature != nil && report.BundleSignature.Verified
		snapshotVerified := hasVerifiedSnapshot(b.Records)
		bonus := computeCorroborationBonus(eopts.corroboration, anchorVerified, signatureVerified, snapshotVerified)
		effective := math.Min(1.0, report.CAS.Overall+bonus)
		report.CAS.CorroborationBonus = bonus
		report.CAS.EffectiveScore = effective
		report.CAS.Grade = gradeFromScore(effective)
	}

	return &report, nil
}

func evaluateLoadedBundle(
	b *bundle.Bundle,
	bundlePath string,
	profiles []Profile,
	allApplicable bool,
	roots *x509.CertPool,
	anchorRequired bool,
) Report {
	report, ok := prepareVerificationReport(b, bundlePath, roots)
	report.Anchoring.AnchorRequired = anchorRequired
	if !ok {
		return report
	}

	report.BundleSignature = inspectBundleSignature(b, bundlePath)
	report.Signatures = inspectAllBundleSignatures(b, bundlePath)

	selectedProfiles := normaliseProfiles(profiles)
	if allApplicable {
		selectedProfiles = matchingProfiles(b.Records, "")
	}

	if len(selectedProfiles) == 0 {
		if allApplicable && report.CAS == nil {
			sc := computeSC(b, profileIDPrivilegedToolAction)
			report.CAS = &CASResult{
				SubScores: map[string]float64{
					"SC": sc,
				},
			}
			if sc > 0 && report.Integrity.ChainValid {
				report.CAS.Overall = sc
				report.CAS.EffectiveScore = sc
				report.CAS.Grade = gradeFromScore(sc)
			}
		}
		if !report.Integrity.ChainValid {
			report.ResidualRisk = integrityFailureResidualRisk()
		} else {
			report.ResidualRisk = residualRiskNoMatchingProfile()
		}
		applyRetrospectiveProvenance(&report, b)
		return report
	}

	anchorResult := ClassifyAnchor(b, bundlePath, roots)
	signatureWarnings, signatureNotes, signatureFailures := inspectPolicyDecisionSignatures(b.Records)
	for i, profile := range selectedProfiles {
		result := profile.Evaluate(b.Records)
		stampProfileResult(&result, profile)
		applyPolicyDecisionSignatureChecks(&result, signatureWarnings, signatureNotes, signatureFailures)

		report.Profiles = append(report.Profiles, result)
		report.Exclusions = appendUniqueStrings(report.Exclusions, profile.BlindSpots()...)
		if i != 0 {
			continue
		}

		if profileSupportsCAS(profile) {
			subScores := subScoresForProfile(profile, b.Records, anchorResult)
			cas := ComputeCAS(subScores, profile.DefaultWeights(), report.Integrity.ChainValid)
			report.CAS = &cas
			if !report.Integrity.ChainValid {
				report.ResidualRisk = integrityFailureResidualRisk()
			} else {
				report.ResidualRisk = deriveResidualRisk(cas, result)
			}
			continue
		}

		if !report.Integrity.ChainValid {
			report.ResidualRisk = integrityFailureResidualRisk()
		} else {
			report.ResidualRisk = deriveResidualRiskWithoutCAS(result)
		}
	}

	applyRetrospectiveProvenance(&report, b)
	return report
}

func normaliseProfiles(profiles []Profile) []Profile {
	if len(profiles) == 0 {
		return nil
	}

	out := make([]Profile, 0, len(profiles))
	for _, profile := range profiles {
		if profile == nil {
			continue
		}
		out = append(out, profile)
	}
	return out
}

func stampProfileResult(result *ProfileResult, profile Profile) {
	if result == nil || profile == nil {
		return
	}
	if strings.TrimSpace(result.ProfileID) == "" {
		result.ProfileID = profile.ID()
	}
	if result.Version == 0 {
		result.Version = profile.Version()
	}
	if strings.TrimSpace(result.WorkflowClass) == "" {
		result.WorkflowClass = profile.WorkflowClass()
	}
}

func applyRetrospectiveProvenance(report *Report, b *bundle.Bundle) {
	if report == nil || !isRetrospectiveImportBundle(b) {
		return
	}
	report.Retrospective = true
	report.ResidualRisk.RecommendedNextEvidence = appendUniqueStrings(
		report.ResidualRisk.RecommendedNextEvidence,
		"This bundle was constructed from imported logs. TSA anchoring reflects import time, not original event time.",
	)
}

func isRetrospectiveImportBundle(b *bundle.Bundle) bool {
	if b == nil || len(b.Records) == 0 {
		return false
	}
	if b.Records[0].Event.Type != bundle.ManifestEventType {
		return false
	}

	const provenanceKey = "bundle_provenance"

	switch data := b.Records[0].Event.Data.(type) {
	case map[string]any:
		meta, _ := data["metadata"].(map[string]any)
		provenance, _ := meta[provenanceKey].(string)
		return strings.TrimSpace(provenance) == bundle.BundleProvenanceRetrospective
	case string:
		var payload map[string]any
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			return false
		}
		meta, _ := payload["metadata"].(map[string]any)
		provenance, _ := meta[provenanceKey].(string)
		return strings.TrimSpace(provenance) == bundle.BundleProvenanceRetrospective
	default:
		return false
	}
}
