// Package verify evaluates ATB bundles against obligation profiles and
// produces structured verification reports.
package verify

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	profiledsl "github.com/pcguest/atb/internal/profiles"
	signpkg "github.com/pcguest/atb/internal/sign"
)

// IntegrityResult holds the outcome of chain integrity verification.
type IntegrityResult struct {
	ChainValid       bool   `json:"chain_valid"`
	Canonicalization string `json:"canonicalization"` // always "rfc8785"
	HashAlgo         string `json:"hash_algo"`        // always "sha256"
	FirstSeq         int    `json:"first_seq"`
	LastSeq          int    `json:"last_seq"`
	Error            string `json:"error,omitempty"`
}

// AnchoringResult holds the outcome of TSA anchor checking.
type AnchoringResult struct {
	AnchorRequired         bool     `json:"anchor_required"`
	AnchorPresent          bool     `json:"anchor_present"`
	Status                 string   `json:"status"`
	Summary                string   `json:"summary"`
	MessageImprintVerified bool     `json:"message_imprint_verified"`
	SignatureVerified      bool     `json:"signature_verified"`
	TSAVerified            bool     `json:"tsa_verified"`
	CertChainVerified      bool     `json:"cert_chain_verified"`
	AnchorHash             string   `json:"anchor_hash,omitempty"`
	Reason                 string   `json:"reason,omitempty"`
	Errors                 []string `json:"errors,omitempty"`
}

// CriticalFailure describes a specific critical-obligation failure.
type CriticalFailure struct {
	Kind   string `json:"kind"` // "missing_event" | "missing_field" | "relation_violation" | "temporal_violation"
	Detail string `json:"detail"`
}

// ProfileResult holds the outcome of evaluating one obligation profile.
// A critical obligation failure forces Pass=false regardless of CAS; CAS is
// diagnostic only in that case and must not be treated as exculpatory.
type ProfileResult struct {
	ProfileID          string            `json:"profile_id"`
	Version            int               `json:"version"`
	WorkflowClass      string            `json:"workflow_class"`
	Pass               bool              `json:"pass"` // false if any critical obligation fails
	CriticalFailures   []CriticalFailure `json:"critical_failures"`
	RequiredWarnings   []string          `json:"required_warnings"`
	InformationalNotes []string          `json:"informational_notes"`
}

// CASResult holds the Completeness Assurance Score and its decomposition.
// It does not override obligation outcome: a profile can FAIL while Overall is
// non-zero; treat CAS as diagnostic completeness evidence only in that case.
type CASResult struct {
	Overall      float64            `json:"overall"`
	Grade        string             `json:"grade"` // "High" >=0.85 | "Medium" >=0.60 | "Low" >=0.30 | "Insufficient" <0.30
	SubScores    map[string]float64 `json:"sub_scores"`
	WeightVector map[string]float64 `json:"weight_vector"`
}

// ResidualRisk summarises the main outstanding concerns.
type ResidualRisk struct {
	Level                   string   `json:"level"` // "Low" | "Medium" | "High" | "Critical"
	Drivers                 []string `json:"drivers"`
	RecommendedNextEvidence []string `json:"recommended_next_evidence"`
}

// Report is the top-level verification output for a bundle.
type Report struct {
	BundlePath         string                 `json:"bundle_path"`
	Integrity          IntegrityResult        `json:"integrity"`
	Anchoring          AnchoringResult        `json:"anchoring"`
	BundleSignature    *BundleSignatureResult `json:"bundle_signature,omitempty"`
	Profiles           []ProfileResult        `json:"profiles"`
	CAS                *CASResult             `json:"cas,omitempty"` // nil if integrity fails or no profile matched
	InformationalNotes []string               `json:"informational_notes,omitempty"`
	Exclusions         []string               `json:"exclusions"`
	ResidualRisk       ResidualRisk           `json:"residual_risk"`
}

var recommendationBySubScore = map[string]string{
	"AC": "Add RFC 3161 TSA anchoring for exports and privileged actions.",
	"EC": "Ensure all critical event types are emitted at workflow call sites.",
	"FC": "Add missing required fields to event data payloads.",
	"GC": "Ensure precommit->execute->commit event chain is present for ACP gating.",
	"RC": "Verify that cross-event relation IDs are consistent (e.g. action_id).",
	"SC": "Add source signatures from policy engine or action gate.",
	"TC": "Check event timestamps conform to RFC 3339 and are in causal order.",
	"XC": "Configure external corroboration: gateway log, DB receipt, or queue dequeue.",
}

// Verify evaluates the bundle against the matching registered profiles and
// returns a structured verification report.
func Verify(b *bundle.Bundle, bundlePath string, profileID string) Report {
	report, ok := prepareVerificationReport(b, bundlePath)
	if !ok {
		return report
	}
	report.BundleSignature = inspectBundleSignature(b, bundlePath)

	profiles := matchingProfiles(b.Records, profileID)
	if len(profiles) == 0 {
		// Root cause: Report.CAS is a pointer tagged with omitempty, so when
		// auto-detect matched no profile this early return left report.CAS nil
		// and JSON dropped the entire block. The profile sub-score maps already
		// use concrete map literals, so there is no nil-map write to fix here.
		if profileID == "" && report.CAS == nil {
			report.CAS = &CASResult{
				SubScores: map[string]float64{
					"SC": computeSC(b, profileIDPrivilegedToolAction),
				},
			}
			if sc := report.CAS.SubScores["SC"]; sc > 0 && report.Integrity.ChainValid {
				// Partial CAS: only SC is available when no profile matched.
				// PA and IC remain 0.0; overall reflects source-commitment only.
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

	anchorResult := ClassifyAnchor(b, bundlePath)
	signatureWarnings, signatureNotes, signatureFailures := inspectPolicyDecisionSignatures(b.Records)
	for i, profile := range profiles {
		result := profile.Evaluate(b.Records)
		applyPolicyDecisionSignatureChecks(&result, signatureWarnings, signatureNotes, signatureFailures)
		report.Profiles = append(report.Profiles, result)
		report.Exclusions = appendUniqueStrings(report.Exclusions, profile.BlindSpots()...)
		if i != 0 {
			continue
		}

		if !profileSupportsCAS(profile) {
			if !report.Integrity.ChainValid {
				report.ResidualRisk = integrityFailureResidualRisk()
			} else {
				report.ResidualRisk = deriveResidualRiskWithoutCAS(result)
			}
			continue
		}

		subScores := subScoresForBundleWithAnchorResult(b, profile.ID(), anchorResult)
		cas := ComputeCAS(subScores, profile.DefaultWeights(), report.Integrity.ChainValid)
		report.CAS = &cas
		if !report.Integrity.ChainValid {
			report.ResidualRisk = integrityFailureResidualRisk()
		} else {
			report.ResidualRisk = deriveResidualRisk(cas, result)
		}
	}

	return report
}

// VerifyWithProfile evaluates a bundle against a specific explicit profile.
func VerifyWithProfile(b *bundle.Bundle, bundlePath string, profile Profile) Report {
	report, ok := prepareVerificationReport(b, bundlePath)
	if !ok {
		return report
	}
	report.BundleSignature = inspectBundleSignature(b, bundlePath)
	if profile == nil {
		if !report.Integrity.ChainValid {
			report.ResidualRisk = integrityFailureResidualRisk()
		} else {
			report.ResidualRisk = residualRiskNoMatchingProfile()
		}
		return report
	}

	result := profile.Evaluate(b.Records)
	signatureWarnings, signatureNotes, signatureFailures := inspectPolicyDecisionSignatures(b.Records)
	applyPolicyDecisionSignatureChecks(&result, signatureWarnings, signatureNotes, signatureFailures)
	report.Profiles = append(report.Profiles, result)
	report.Exclusions = appendUniqueStrings(report.Exclusions, profile.BlindSpots()...)

	if profileSupportsCAS(profile) {
		anchorResult := ClassifyAnchor(b, bundlePath)
		subScores := subScoresForBundleWithAnchorResult(b, profile.ID(), anchorResult)
		cas := ComputeCAS(subScores, profile.DefaultWeights(), report.Integrity.ChainValid)
		report.CAS = &cas
		if !report.Integrity.ChainValid {
			report.ResidualRisk = integrityFailureResidualRisk()
		} else {
			report.ResidualRisk = deriveResidualRisk(cas, result)
		}
		return report
	}

	if !report.Integrity.ChainValid {
		report.ResidualRisk = integrityFailureResidualRisk()
	} else {
		report.ResidualRisk = deriveResidualRiskWithoutCAS(result)
	}
	return report
}

func inspectPolicyDecisionSignatures(records []bundle.Record) ([]string, []string, []CriticalFailure) {
	warnings := []string{}
	notes := []string{}
	failures := []CriticalFailure{}

	for _, record := range records {
		if record.Event.Type != event.TypeAIPolicyDecision {
			continue
		}

		fields, ok := record.Event.Data.(map[string]any)
		if !ok {
			failures = append(failures, CriticalFailure{
				Kind:   "signature_verification",
				Detail: "ai.policy.decision: signature verification failed: policy decision data is not an object",
			})
			continue
		}

		err := signpkg.VerifyPolicyDecision(fields)
		switch {
		case err == nil:
			notes = append(notes, "ai.policy.decision: signature verified")
		case errors.Is(err, signpkg.ErrSignatureAbsent):
			warnings = append(warnings, "ai.policy.decision: policy_signature absent")
		default:
			failures = append(failures, CriticalFailure{
				Kind:   "signature_verification",
				Detail: "ai.policy.decision: signature verification failed: " + err.Error(),
			})
		}
	}

	return warnings, notes, failures
}

func applyPolicyDecisionSignatureChecks(result *ProfileResult, warnings []string, notes []string, failures []CriticalFailure) {
	if result == nil {
		return
	}
	result.RequiredWarnings = append(result.RequiredWarnings, warnings...)
	result.InformationalNotes = append(result.InformationalNotes, notes...)
	result.CriticalFailures = append(result.CriticalFailures, failures...)
	finaliseProfileResult(result)
}

// computeSC computes the source-commitment (SC) sub-score for a bundle.
// SC measures the completeness of the originating-request, signing, and anchor chain.
// Result is clamped to [0.0, 1.0].
func computeSC(b *bundle.Bundle, profileID string) float64 {
	if b == nil {
		return 0.0
	}
	schema, ok := loadSchemaIfAvailable(profileID)
	if !ok {
		return 0.0
	}
	if !schema.SupportsCAS {
		return 0.0
	}

	hasAnchor := false
	hasManifestAtSequenceZero := false
	hasRequestReceived := false
	hasPolicyDecision := false
	hasRetrievalExecuted := false
	hasSignature := false

	for _, record := range b.Records {
		switch record.Event.Type {
		case event.TypeBundleAnchor:
			hasAnchor = true
		case event.TypeBundleSignature:
			hasSignature = true
		case event.TypeBundleManifest:
			if record.Event.Sequence == 0 {
				hasManifestAtSequenceZero = true
			}
		case event.TypeAIRequestReceived:
			hasRequestReceived = true
		case event.TypeAIPolicyDecision:
			hasPolicyDecision = true
		case event.TypeAIRetrievalExecuted:
			hasRetrievalExecuted = true
		}
	}

	total := 0.0
	if hasAnchor {
		total += 0.40
	}
	if hasManifestAtSequenceZero {
		total += 0.25
	}
	if hasRequestReceived {
		total += 0.20
	}
	if hasSignature {
		total += 0.10
	}

	switch schema.SCMode {
	case "policy_decision":
		if hasPolicyDecision {
			total += 0.15
		}
	case "retrieval_executed":
		if hasRetrievalExecuted {
			total += 0.15
		}
	}

	return math.Min(total, 1.0)
}

func scanAnchoring(records []bundle.Record) AnchoringResult {
	result := AnchoringResult{
		AnchorRequired:    false,
		AnchorPresent:     false,
		Status:            string(anchorStatusFailed),
		Summary:           anchorFailedSummary("anchor record not present"),
		TSAVerified:       false,
		CertChainVerified: false,
	}
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].Event.Type != bundle.AnchorEventType {
			continue
		}
		result.AnchorPresent = true

		raw, ok := records[i].Event.Data.(string)
		if !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("anchor record at index %d: data is %T, want string", i, records[i].Event.Data))
			continue
		}

		var payload map[string]any
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("anchor record at index %d: invalid JSON payload: %v", i, err))
			continue
		}
		if anchorHash, ok := payload["bundle_hash"].(string); ok && anchorHash != "" && result.AnchorHash == "" {
			result.AnchorHash = anchorHash
			continue
		}
		if anchorHash, ok := payload["tsr_hash"].(string); ok && anchorHash != "" && result.AnchorHash == "" {
			result.AnchorHash = anchorHash
		}
	}
	return result
}

func prepareVerificationReport(b *bundle.Bundle, bundlePath string) (Report, bool) {
	report := Report{
		BundlePath: bundlePath,
		Integrity: IntegrityResult{
			Canonicalization: "rfc8785",
			HashAlgo:         "sha256",
		},
		Anchoring:          scanAnchoring(nil),
		Profiles:           []ProfileResult{},
		InformationalNotes: []string{},
		Exclusions:         []string{},
		ResidualRisk:       ResidualRisk{Level: "Critical"},
	}
	if b == nil {
		report.Integrity.Error = "bundle is nil"
		report.ResidualRisk = integrityFailureResidualRisk()
		return report, false
	}

	report.Anchoring = scanAnchoring(b.Records)
	anchorInspection := inspectAnchor(b, bundlePath)
	report.Anchoring.AnchorPresent = anchorInspection.AnchorPresent
	report.Anchoring.Status = string(anchorInspection.Status)
	report.Anchoring.Summary = anchorInspection.Summary
	report.Anchoring.MessageImprintVerified = anchorInspection.MessageImprintVerified
	report.Anchoring.SignatureVerified = anchorInspection.SignatureVerified
	report.Anchoring.TSAVerified = anchorInspection.Result == AnchorVerified
	report.Anchoring.CertChainVerified = anchorInspection.CertChainVerified
	report.Anchoring.Reason = anchorInspection.Reason
	if report.Anchoring.AnchorHash == "" {
		report.Anchoring.AnchorHash = anchorInspection.AnchorHash
	}
	report.InformationalNotes = appendUniqueStrings(report.InformationalNotes, ValidateTimestamps(b.Records)...)
	setIntegritySequenceBounds(&report.Integrity, b.Records)

	if err := b.Verify(); err != nil {
		report.Integrity.Error = err.Error()
		report.ResidualRisk = integrityFailureResidualRisk()
		return report, true
	}
	report.Integrity.ChainValid = true
	return report, true
}

func setIntegritySequenceBounds(result *IntegrityResult, records []bundle.Record) {
	if len(records) == 0 {
		return
	}
	result.FirstSeq = records[0].Event.Sequence
	result.LastSeq = records[len(records)-1].Event.Sequence
}

func residualRiskNoMatchingProfile() ResidualRisk {
	return ResidualRisk{
		Level:   "High",
		Drivers: []string{"No matching obligation profile evaluated."},
		RecommendedNextEvidence: []string{
			"Select an obligation profile explicitly or emit profile-identifying events.",
		},
	}
}

func integrityFailureResidualRisk() ResidualRisk {
	return ResidualRisk{
		Level:   "Critical",
		Drivers: []string{"Chain integrity invalid; completeness scoring is diagnostic only."},
	}
}

func matchingProfiles(records []bundle.Record, profileID string) []Profile {
	if profileID != "" {
		profile := ProfileByID(profileID)
		if profile == nil && !strings.HasPrefix(profileID, "atb.profile.") {
			profile = ProfileByID("atb.profile." + profileID)
		}
		if profile == nil {
			return nil
		}
		return []Profile{profile}
	}

	if profile := profileFromPurposeTag(records); profile != nil {
		return []Profile{profile}
	}

	hasJobScheduled := false
	hasJobStarted := false
	hasPrecommit := false
	hasModelInvoked := false
	for _, record := range records {
		switch record.Event.Type {
		case event.TypeAIJobScheduled:
			hasJobScheduled = true
		case event.TypeAIJobStarted:
			hasJobStarted = true
		case "ai.action.precommit":
			hasPrecommit = true
		case "ai.model.invoked":
			hasModelInvoked = true
		}
	}

	switch {
	case hasJobScheduled || hasJobStarted:
		if profile := ProfileByID(profileIDBackgroundAutomation); profile != nil {
			return []Profile{profile}
		}
	case hasPrecommit:
		if profile := ProfileByID(profileIDPrivilegedToolAction); profile != nil {
			return []Profile{profile}
		}
	case hasModelInvoked:
		if profile := ProfileByID(profileIDRAGAnswer); profile != nil {
			return []Profile{profile}
		}
	}

	return nil
}

func profileFromPurposeTag(records []bundle.Record) Profile {
	for _, record := range records {
		if record.Event.Type != event.TypeAIRequestReceived {
			continue
		}

		fields, ok := record.Event.Data.(map[string]any)
		if !ok {
			continue
		}

		purposeTag, _ := fields["purpose_tag"].(string)
		purposeTag = strings.TrimSpace(purposeTag)
		if purposeTag == "" {
			continue
		}

		for _, candidateID := range []string{
			profileIDPrivilegedToolAction,
			profileIDDataExport,
			profileIDPolicyDecision,
			profileIDHumanOverride,
			profileIDRAGAnswer,
		} {
			profile := ProfileByID(candidateID)
			if profile == nil {
				continue
			}

			schema, ok := loadSchemaIfAvailable(candidateID)
			if !ok {
				continue
			}
			if purposeTag == schema.WorkflowClass || purposeTag == candidateID {
				return profile
			}
		}
	}

	return nil
}

func deriveResidualRisk(cas CASResult, profile ProfileResult) ResidualRisk {
	level := "Low"
	switch {
	case !profile.Pass || cas.Overall < 0.60:
		level = "High"
	case cas.Overall < 0.85 || len(profile.RequiredWarnings) > 0:
		level = "Medium"
	}

	drivers := []string{}
	for key, score := range cas.SubScores {
		if score < 0.70 {
			drivers = append(drivers, key)
		}
	}
	sort.Strings(drivers)

	recommended := []string{}
	for _, driver := range drivers {
		if next := recommendationBySubScore[driver]; next != "" {
			recommended = appendUniqueStrings(recommended, next)
		}
	}

	return ResidualRisk{
		Level:                   level,
		Drivers:                 drivers,
		RecommendedNextEvidence: recommended,
	}
}

func deriveResidualRiskWithoutCAS(profile ProfileResult) ResidualRisk {
	level := "Low"
	switch {
	case !profile.Pass:
		level = "High"
	case len(profile.RequiredWarnings) > 0:
		level = "Medium"
	}

	recommended := make([]string, 0, len(profile.CriticalFailures)+len(profile.RequiredWarnings))
	for _, failure := range profile.CriticalFailures {
		recommended = append(recommended, failure.Detail)
	}
	recommended = append(recommended, profile.RequiredWarnings...)
	recommended = appendUniqueStrings(nil, recommended...)
	sort.Strings(recommended)

	return ResidualRisk{
		Level:                   level,
		RecommendedNextEvidence: recommended,
	}
}

func profileSupportsCAS(profile Profile) bool {
	if profile == nil {
		return false
	}
	return SupportsCAS(profile.ID())
}

// SupportsCAS reports whether a registered profile opts into CAS scoring.
func SupportsCAS(profileID string) bool {
	schema, ok := loadSchemaIfAvailable(profileID)
	return ok && schema.SupportsCAS
}

func loadSchemaIfAvailable(id string) (schema profiledsl.ProfileSchema, ok bool) {
	defer func() {
		if recover() != nil {
			schema = profiledsl.ProfileSchema{}
			ok = false
		}
	}()

	schema = loadSchema(id)
	return schema, true
}

func appendUniqueStrings(dst []string, values ...string) []string {
	seen := make(map[string]struct{}, len(dst))
	for _, value := range dst {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		dst = append(dst, value)
		seen[value] = struct{}{}
	}
	return dst
}
