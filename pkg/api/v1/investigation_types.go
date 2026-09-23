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
	Findings            []incident.Finding      `json:"findings"`
	AcquisitionFindings []AcquisitionFindingDTO `json:"acquisition_findings,omitempty"`
}

// AcquisitionFindingDTO is a bounded acquisition-continuity finding. It states a
// source-representation change relationship; it does not establish tampering,
// intent, or truth.
type AcquisitionFindingDTO struct {
	Flag               string `json:"flag"`
	Severity           string `json:"severity"`
	Title              string `json:"title"`
	Detail             string `json:"detail"`
	SourceSystem       string `json:"source_system,omitempty"`
	SourceRecordID     string `json:"source_record_id,omitempty"`
	PreviousDigest     string `json:"previous_digest,omitempty"`
	CurrentDigest      string `json:"current_digest,omitempty"`
	PreviousAcquiredAt string `json:"previous_acquired_at,omitempty"`
	CurrentAcquiredAt  string `json:"current_acquired_at,omitempty"`
	Adapter            string `json:"adapter,omitempty"`
	AdapterVersion     string `json:"adapter_version,omitempty"`
	EventSeq           int    `json:"event_seq"`
	Boundedness        string `json:"boundedness"`
	WhatCanConclude    string `json:"what_atb_can_conclude"`
	WhatCannotConclude string `json:"what_atb_cannot_conclude"`
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
	// Acquisition continuity is bounded operational/provenance context, not truth.
	AcquisitionRecords   int    `json:"acquisition_records"`
	SourceChangeFindings int    `json:"source_change_findings"`
	AcquisitionNote      string `json:"acquisition_note,omitempty"`
}
