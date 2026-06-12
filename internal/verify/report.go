// SPDX-License-Identifier: MIT
package verify

import (
	"strconv"
	"strings"
)

const VerifyReportVersion = "verify.report.v1"

type VerifierReport struct {
	ReportVersion      string                `json:"report_version"`
	BundlePath         string                `json:"bundle_path"`
	Retrospective      bool                  `json:"retrospective,omitempty"`
	ProfileID          string                `json:"profile_id"`
	ProfileVersion     int                   `json:"profile_version,omitempty"`
	Pass               bool                  `json:"pass"`
	GateResult         GateResult            `json:"gate_result"`
	CASScore           float64               `json:"cas_score"`
	CASGrade           string                `json:"cas_grade,omitempty"`
	CorroborationBonus float64               `json:"corroboration_bonus,omitempty"`
	EffectiveScore     float64               `json:"effective_score,omitempty"`
	SubScores          map[string]float64    `json:"sub_scores,omitempty"`
	Failures           []ReportFailure       `json:"critical_failures"`
	Obligations        []ObligationResult    `json:"obligations,omitempty"`
	Warnings           []string              `json:"required_warnings"`
	Notes              []string              `json:"informational_notes"`
	Exclusions         []string              `json:"exclusions,omitempty"`
	Signatures         []SignatureProvenance `json:"signatures,omitempty"`
	ProvabilityGaps    []ProvabilityGap      `json:"provability_gaps,omitempty"`
	ResidualRisk       ResidualRiskReport    `json:"residual_risk"`
}

type ReportFailure struct {
	ID     string `json:"id,omitempty"`
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}

type ResidualRiskReport struct {
	Level                   string   `json:"level"`
	Drivers                 []string `json:"drivers,omitempty"`
	RecommendedNextEvidence []string `json:"recommended_next_evidence,omitempty"`
}

func ReportFromVerify(r Report) VerifierReport {
	report := VerifierReport{
		ReportVersion: VerifyReportVersion,
		BundlePath:    r.BundlePath,
		Retrospective: r.Retrospective,
		GateResult:    gateResultFromVerify(r, false),
		Failures:      []ReportFailure{},
		Warnings:      []string{},
		Notes:         []string{},
		ResidualRisk: ResidualRiskReport{
			Level:                   r.ResidualRisk.Level,
			Drivers:                 append([]string(nil), r.ResidualRisk.Drivers...),
			RecommendedNextEvidence: append([]string(nil), r.ResidualRisk.RecommendedNextEvidence...),
		},
	}

	if len(r.Exclusions) > 0 {
		report.Exclusions = append([]string(nil), r.Exclusions...)
	}

	if len(r.Signatures) > 0 {
		report.Signatures = append([]SignatureProvenance(nil), r.Signatures...)
	}

	if !r.Integrity.ChainValid {
		detail := strings.TrimSpace(r.Integrity.Error)
		if detail == "" {
			detail = "bundle hash chain is invalid"
		}
		report.Failures = append(report.Failures, ReportFailure{
			ID:     "integrity:chain",
			Kind:   "integrity_failure",
			Detail: detail,
		})
	}

	if r.CAS != nil {
		report.CASScore = r.CAS.Overall
		report.CASGrade = r.CAS.Grade
		if r.CAS.CorroborationBonus != 0 {
			report.CorroborationBonus = r.CAS.CorroborationBonus
			report.EffectiveScore = r.CAS.EffectiveScore
		}
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
	report.GateResult = gateResultFromVerify(r, profile.Pass)
	report.ProfileID = profile.ProfileID
	report.ProfileVersion = profile.Version
	report.Pass = report.GateResult.Pass
	for _, failure := range profile.CriticalFailures {
		report.Failures = append(report.Failures, ReportFailure(failure))
	}
	report.Obligations = reportObligations(profile)
	report.Warnings = append([]string(nil), profile.RequiredWarnings...)
	report.Notes = append([]string(nil), profile.InformationalNotes...)
	report.ProvabilityGaps = append([]ProvabilityGap(nil), r.ProvabilityGaps...)

	return report
}

func gateResultFromVerify(r Report, profilePass bool) GateResult {
	result := GateResult{
		Pass:           r.Integrity.ChainValid && profilePass,
		ChainValid:     r.Integrity.ChainValid,
		ProfilePass:    profilePass,
		AnchorRequired: r.Anchoring.AnchorRequired,
	}
	if r.Anchoring.AnchorRequired {
		anchorPass := r.Anchoring.TSAVerified || r.Anchoring.Status == string(anchorStatusVerified)
		result.AnchorPass = &anchorPass
		result.Pass = result.Pass && anchorPass
	}
	return result
}

func reportObligations(profile ProfileResult) []ObligationResult {
	obligations := make([]ObligationResult, 0,
		len(profile.Obligations)+len(profile.CriticalFailures)+len(profile.RequiredWarnings))
	obligations = append(obligations, profile.Obligations...)

	for i, failure := range profile.CriticalFailures {
		obligation := ObligationResult{
			ID:        failureObligationID(failure, i),
			Kind:      failure.Kind,
			EventType: failureEventType(failure.ID),
			Severity:  "critical",
			Status:    "fail",
			Message:   failure.Detail,
		}
		obligations = upsertObligation(obligations, obligation)
	}

	for i, warning := range profile.RequiredWarnings {
		obligation := ObligationResult{
			ID:       warningObligationID(i),
			Kind:     "required_warning",
			Severity: "warning",
			Status:   "warning",
			Message:  warning,
		}
		obligations = upsertObligation(obligations, obligation)
	}

	return obligations
}

func upsertObligation(obligations []ObligationResult, obligation ObligationResult) []ObligationResult {
	for i := range obligations {
		if obligations[i].ID != obligation.ID {
			continue
		}
		obligations[i] = mergeObligation(obligations[i], obligation)
		return obligations
	}
	return append(obligations, obligation)
}

func mergeObligation(current ObligationResult, next ObligationResult) ObligationResult {
	if next.Kind != "" {
		current.Kind = next.Kind
	}
	if next.EventType != "" {
		current.EventType = next.EventType
	}
	if next.Severity != "" {
		current.Severity = next.Severity
	}
	if next.Status != "" {
		current.Status = next.Status
	}
	if next.Message != "" {
		current.Message = next.Message
	}
	return current
}

func failureObligationID(failure CriticalFailure, index int) string {
	if strings.TrimSpace(failure.ID) != "" {
		return failure.ID
	}
	return "failure:" + failure.Kind + ":" + strconv.Itoa(index+1)
}

func failureEventType(id string) string {
	switch {
	case strings.HasPrefix(id, "required:"):
		value := strings.TrimPrefix(id, "required:")
		if eventType, _, ok := strings.Cut(value, ":field:"); ok {
			return eventType
		}
		return value
	case strings.HasPrefix(id, "optional:"):
		return strings.TrimPrefix(id, "optional:")
	case strings.HasPrefix(id, "required_when:"):
		value := strings.TrimPrefix(id, "required_when:")
		if eventType, _, ok := strings.Cut(value, ":when:"); ok {
			return eventType
		}
	}
	return ""
}
