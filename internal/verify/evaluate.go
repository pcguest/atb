package verify

import (
	"crypto/x509"
	"errors"
	"fmt"
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

// EvaluateConfig describes one bundle evaluation request.
type EvaluateConfig struct {
	BundlePath       string
	Records          []bundle.Record
	Profiles         []Profile
	AllApplicable    bool
	AnchorRoots      *x509.CertPool
	AnchorRequired   bool
	RequireValidChain bool
}

// EvaluateBundle loads the requested bundle source and evaluates it with the
// selected profiles.
func EvaluateBundle(cfg EvaluateConfig) (*Report, error) {
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

	selectedProfiles := normaliseProfiles(profiles)
	if allApplicable {
		selectedProfiles = matchingProfiles(b.Records, "")
	}

	if len(selectedProfiles) == 0 {
		if allApplicable && report.CAS == nil {
			report.CAS = &CASResult{
				SubScores: map[string]float64{
					"SC": computeSC(b, profileIDPrivilegedToolAction),
				},
			}
			if sc := report.CAS.SubScores["SC"]; sc > 0 && report.Integrity.ChainValid {
				report.CAS.Overall = sc
				report.CAS.Grade = gradeFromScore(sc)
			}
		}
		if !report.Integrity.ChainValid {
			report.ResidualRisk = integrityFailureResidualRisk()
		} else {
			report.ResidualRisk = residualRiskNoMatchingProfile()
		}
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
