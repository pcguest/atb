// SPDX-License-Identifier: MIT
package apiv1

import (
	"github.com/pcguest/atb/internal/capabilities"
	"github.com/pcguest/atb/internal/contextlineage"
	"github.com/pcguest/atb/internal/incident"
)

// InvestigationOverviewResponse is the human-first summary for the loaded bundle.
type InvestigationOverviewResponse struct {
	BundlePath       string                `json:"bundle_path"`
	EventCount       int                   `json:"event_count"`
	IntegrityValid   bool                  `json:"integrity_valid"`
	IntegrityStatus  string                `json:"integrity_status"`
	Profile          *ProfileReportSummary `json:"profile,omitempty"`
	FindingCount     int                   `json:"finding_count"`
	CriticalFindings int                   `json:"critical_findings"`
	CustodyState     string                `json:"custody_state"`
	Summary          string                `json:"summary"`
}

// InvestigationFindingsResponse contains bounded, evidence-linked findings.
type InvestigationFindingsResponse struct {
	Findings []incident.Finding `json:"findings"`
}

// TimelineEventDTO is a human-labelled chronology entry. Chronology alone is
// never represented as causal evidence.
type TimelineEventDTO struct {
	Sequence   int    `json:"seq"`
	Type       string `json:"type"`
	Label      string `json:"label"`
	Timestamp  string `json:"timestamp,omitempty"`
	Hash       string `json:"hash"`
	Family     string `json:"family"`
	CausalEdge bool   `json:"causal_edge"`
}

type InvestigationTimelineResponse struct {
	Events []TimelineEventDTO `json:"events"`
}

type InvestigationContextResponse struct {
	Lineage      contextlineage.Lineage  `json:"lineage"`
	Capabilities []capabilities.Evidence `json:"capabilities"`
}

// RelationshipDTO is a semantic relationship supported by a shared captured identifier.
type RelationshipDTO struct {
	ID             string `json:"id"`
	SourceSequence int    `json:"source_seq"`
	TargetSequence int    `json:"target_seq"`
	Kind           string `json:"kind"`
	EvidenceValue  string `json:"evidence_value"`
	Strength       string `json:"strength"`
}

type InvestigationRelationshipsResponse struct {
	Relationships []RelationshipDTO `json:"relationships"`
}

// InvestigationTrustResponse states both the proofs present and their limits.
type InvestigationTrustResponse struct {
	ProofStatement        string   `json:"proof_statement"`
	IntegrityValid        bool     `json:"integrity_valid"`
	Canonicalisation      string   `json:"canonicalisation"`
	SignatureStatus       string   `json:"signature_status"`
	AnchorStatus          string   `json:"anchor_status"`
	ProfileID             string   `json:"profile_id,omitempty"`
	ProfilePass           bool     `json:"profile_pass"`
	CoverageScore         float64  `json:"coverage_score,omitempty"`
	CoverageGrade         string   `json:"coverage_grade,omitempty"`
	AssessmentCoverage    float64  `json:"assessment_coverage,omitempty"`
	AssuranceValid        bool     `json:"assurance_valid"`
	ExternalCorroboration bool     `json:"external_corroboration"`
	CustodyState          string   `json:"custody_state"`
	Limitations           []string `json:"limitations"`
}
