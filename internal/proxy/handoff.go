// SPDX-License-Identifier: MIT
package proxy

import (
	"fmt"
	"path/filepath"

	"github.com/pcguest/atb/internal/bundle"
)

// Handoff is a portable, Mortise-independent incident handoff. It carries the
// exact evidence identity and the capture limitations so an investigation can
// move into a ticket, case-management, GRC, or audit workflow without losing
// identity or overclaiming completeness.
type Handoff struct {
	EvidenceReference  string   `json:"evidence_reference"`
	BundlePath         string   `json:"bundle_path"`
	BundleHead         string   `json:"bundle_head"`
	Sequence           int      `json:"sequence"`
	RecordHash         string   `json:"record_hash"`
	RecordType         string   `json:"record_type,omitempty"`
	CaptureState       string   `json:"capture_state"`
	SourceSystem       string   `json:"source_system,omitempty"`
	SourceIncarnation  string   `json:"source_incarnation,omitempty"`
	KnownGaps          []string `json:"known_gaps"`
	Limitations        []string `json:"limitations"`
	VerificationState  string   `json:"verification_state"`
	ExportInstructions string   `json:"export_instructions"`
}

// BuildHandoff builds a portable handoff for one record (or the bundle head when
// sequence is 0) from a verified bundle.
func BuildHandoff(bundlePath string, sequence int) (*Handoff, error) {
	b, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("handoff: load bundle: %w", err)
	}
	head := committedHead(b)
	if head == "" {
		return nil, fmt.Errorf("handoff: bundle has no records")
	}

	h := &Handoff{
		BundlePath:        filepath.Clean(bundlePath),
		BundleHead:        head,
		VerificationState: "verified",
		Limitations: []string{
			"ATB proves recorded integrity, not truth, completeness, or causation.",
			"Absence of a recorded event does not prove the event did not occur.",
			"Recorded order is append order, not causal order.",
			"A digest identifies a captured representation, not the provider-original object.",
		},
		ExportInstructions: "atb export --format soc2 --bundle " + filepath.Clean(bundlePath) + " --output <file>  (offline; no capture service or Mortise required)",
	}

	rec := b.Records[len(b.Records)-1]
	if sequence > 0 {
		found := false
		for _, r := range b.Records {
			if r.Event.Sequence == sequence {
				rec = r
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("handoff: sequence %d not found", sequence)
		}
	}
	h.Sequence = rec.Event.Sequence
	h.RecordHash = rec.Hash
	h.RecordType = rec.Event.Type
	h.EvidenceReference = fmt.Sprintf("atb://evidence/1/%s?seq=%d", head, rec.Event.Sequence)
	if rec.Event.Acquisition != nil {
		h.SourceSystem = rec.Event.Acquisition.SourceSystem
	}

	if st, err := ReadCaptureStatus(bundlePath); err == nil {
		h.CaptureState = st.CaptureState
		h.SourceIncarnation = st.SourceIncarnation
		if st.KnownGap || st.PossibleUnknownGap {
			detail := st.Detail
			if detail == "" {
				detail = "capture continuity is not fully established"
			}
			h.KnownGaps = append(h.KnownGaps, detail)
		}
		if st.JournalBacklog > 0 {
			h.KnownGaps = append(h.KnownGaps, fmt.Sprintf("%d durable observation(s) not yet committed to evidence", st.JournalBacklog))
		}
	}
	if h.KnownGaps == nil {
		h.KnownGaps = []string{}
	}
	return h, nil
}
