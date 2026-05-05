// SPDX-License-Identifier: MIT
package verify

type VerifierReport struct {
	BundlePath    string                `json:"bundle_path"`
	Retrospective bool                  `json:"retrospective,omitempty"`
	ProfileID     string                `json:"profile_id"`
	Pass          bool                  `json:"pass"`
	CASScore      float64               `json:"cas_score,omitempty"`
	CASGrade      string                `json:"cas_grade,omitempty"`
	SubScores     map[string]float64    `json:"sub_scores,omitempty"`
	Failures      []ReportFailure       `json:"critical_failures"`
	Warnings      []string              `json:"required_warnings"`
	Notes         []string              `json:"informational_notes"`
	Exclusions    []string              `json:"exclusions,omitempty"`
	Signatures    []SignatureProvenance `json:"signatures,omitempty"`
	ResidualRisk  string                `json:"residual_risk"`
}

type ReportFailure struct {
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}

func ReportFromVerify(r Report) VerifierReport {
	report := VerifierReport{
		BundlePath:    r.BundlePath,
		Retrospective: r.Retrospective,
		Failures:      []ReportFailure{},
		Warnings:      []string{},
		Notes:         []string{},
		ResidualRisk:  r.ResidualRisk.Level,
	}

	if len(r.Exclusions) > 0 {
		report.Exclusions = append([]string(nil), r.Exclusions...)
	}

	if len(r.Signatures) > 0 {
		report.Signatures = append([]SignatureProvenance(nil), r.Signatures...)
	}

	if r.CAS != nil {
		report.CASScore = r.CAS.Overall
		report.CASGrade = r.CAS.Grade
		if len(r.CAS.SubScores) > 0 {
			report.SubScores = make(map[string]float64, len(r.CAS.SubScores))
			for key, value := range r.CAS.SubScores {
				report.SubScores[key] = value
			}
		}
	}

	if len(r.Profiles) == 0 {
		return report
	}

	profile := r.Profiles[0]
	report.ProfileID = profile.ProfileID
	report.Pass = r.Integrity.ChainValid && profile.Pass
	report.Failures = make([]ReportFailure, 0, len(profile.CriticalFailures))
	for _, failure := range profile.CriticalFailures {
		report.Failures = append(report.Failures, ReportFailure(failure))
	}
	report.Warnings = append([]string(nil), profile.RequiredWarnings...)
	report.Notes = append([]string(nil), profile.InformationalNotes...)

	return report
}
