// SPDX-License-Identifier: MIT
package verify

// ProvabilityLayerChecklist summarises which provability layers are satisfied.
func ProvabilityLayerChecklist(report Report) map[string]bool {
	checklist := map[string]bool{
		"L1_integrity": report.Integrity.ChainValid,
	}

	if len(report.Profiles) > 0 {
		checklist["L2_profile_pass"] = report.Profiles[0].Pass
	}
	if report.CAS != nil {
		if sc, ok := report.CAS.SubScores["SC"]; ok {
			checklist["L3_source_binding"] = sc >= 0.5
		}
		if xc, ok := report.CAS.SubScores["XC"]; ok {
			checklist["L4_external_witness"] = xc >= 1.0
		}
		if ac, ok := report.CAS.SubScores["AC"]; ok {
			checklist["L4_anchor"] = ac > 0 && report.Anchoring.TSAVerified
		}
	}
	if report.BundleSignature != nil && report.BundleSignature.Verified {
		checklist["L3_bundle_signature"] = true
	}
	checklist["L5_live_capture"] = !report.Retrospective

	return checklist
}

// ExportVerificationSidecar is written beside compliance export ZIPs.
type ExportVerificationSidecar struct {
	Report            Report          `json:"report"`
	ProvabilityLayers map[string]bool `json:"provability_layers"`
}
