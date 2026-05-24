// SPDX-License-Identifier: MIT
package verify

import (
	"sort"
	"strings"
)

// ProvabilityGap describes a conditional limitation and how to close it.
type ProvabilityGap struct {
	Gap        string `json:"gap"`
	Layer      string `json:"layer"`
	Mitigation string `json:"mitigation"`
	ClosedWhen string `json:"closed_when"`
}

var provabilityGapBySubScore = map[string]ProvabilityGap{
	"EC": {
		Gap:        "event_coverage",
		Layer:      "L2",
		Mitigation: "Emit all required event types for the selected profile at workflow call sites.",
		ClosedWhen: "Profile pass is true and EC sub-score >= 0.85.",
	},
	"FC": {
		Gap:        "field_completeness",
		Layer:      "L2",
		Mitigation: "Populate required fields on each event payload.",
		ClosedWhen: "No missing_field critical failures and FC sub-score >= 0.85.",
	},
	"RC": {
		Gap:        "relation_consistency",
		Layer:      "L2",
		Mitigation: "Keep cross-event IDs consistent (request_id, action_id).",
		ClosedWhen: "No relation_violation failures and RC sub-score >= 0.85.",
	},
	"TC": {
		Gap:        "temporal_consistency",
		Layer:      "L2",
		Mitigation: "Record events in causal order with RFC 3339 timestamps.",
		ClosedWhen: "No temporal_violation failures and TC sub-score >= 0.85.",
	},
	"SC": {
		Gap:        "source_binding",
		Layer:      "L3",
		Mitigation: "Add policy_signature and policy_signer_pubkey to ai.policy.decision events.",
		ClosedWhen: "SC sub-score >= 0.5 and no policy_signature absent warnings.",
	},
	"XC": {
		Gap:        "external_corroboration",
		Layer:      "L4",
		Mitigation: "Append atb.corroboration.external via atb corroborate or an adapter.",
		ClosedWhen: "XC sub-score >= 1.0 with at least one valid corroboration event.",
	},
	"AC": {
		Gap:        "timestamp_anchor",
		Layer:      "L4",
		Mitigation: "Run atb anchor after snapshot for RFC 3161 timestamp commitment.",
		ClosedWhen: "Anchor status is verified and AC sub-score receives anchor credit.",
	},
	"GC": {
		Gap:        "gating_completeness",
		Layer:      "L5",
		Mitigation: "Record precommit, execute, and commit events through the ACP gate path.",
		ClosedWhen: "GC sub-score >= 0.85 for gated workflow profiles.",
	},
}

const provabilityGapScoreThreshold = 0.70

// DeriveProvabilityGaps returns actionable gaps from a verification report.
func DeriveProvabilityGaps(report Report) []ProvabilityGap {
	if !report.Integrity.ChainValid {
		return []ProvabilityGap{{
			Gap:        "integrity",
			Layer:      "L1",
			Mitigation: "Restore bundle from trusted backup or re-capture workflow evidence.",
			ClosedWhen: "integrity.chain_valid is true.",
		}}
	}

	gaps := make([]ProvabilityGap, 0, 8)
	seen := map[string]struct{}{}

	addGap := func(gap ProvabilityGap) {
		if gap.Gap == "" {
			return
		}
		if _, ok := seen[gap.Gap]; ok {
			return
		}
		seen[gap.Gap] = struct{}{}
		gaps = append(gaps, gap)
	}

	if report.CAS != nil {
		for key, score := range report.CAS.SubScores {
			if score >= provabilityGapScoreThreshold {
				continue
			}
			if template, ok := provabilityGapBySubScore[key]; ok {
				addGap(template)
			}
		}
	}

	if len(report.Profiles) > 0 {
		profile := report.Profiles[0]
		for _, warning := range profile.RequiredWarnings {
			if strings.Contains(warning, "policy_signature absent") {
				addGap(provabilityGapBySubScore["SC"])
			}
		}
		if report.Retrospective {
			addGap(ProvabilityGap{
				Gap:        "retrospective_capture",
				Layer:      "L5",
				Mitigation: "Prefer live SDK or capture run instrumentation over chatlog import for primary evidence.",
				ClosedWhen: "Bundle provenance is live capture, not retrospective import.",
			})
		}
	}

	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].Layer == gaps[j].Layer {
			return gaps[i].Gap < gaps[j].Gap
		}
		return gaps[i].Layer < gaps[j].Layer
	})

	return gaps
}
