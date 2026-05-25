// SPDX-License-Identifier: MIT
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pcguest/atb/internal/event"
	profiledsl "github.com/pcguest/atb/internal/profiles"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
)

type verifyTextRenderer struct {
	colour bool
}

type verifyLineStatus int

const (
	verifyLinePass verifyLineStatus = iota
	verifyLineFail
	verifyLineWarn
	verifyLineNote
)

type verifyOutputLine struct {
	status verifyLineStatus
	text   string
}

type verifyObligationSpec struct {
	eventType string
	message   string
	warning   bool
}

type verifyRelationSpec struct {
	from string
	to   string
}

// renderVerifyTerminalReport renders the human-readable verify report only.
// Exit-code decisions stay in verify.go so CI-facing trust-boundary behaviour
// is audited separately from presentation text.
func renderVerifyTerminalReport(w io.Writer, report verifypkg.Report) {
	renderer := verifyTextRenderer{colour: useVerifyANSI(w)}

	fmt.Fprintln(w, "ATB Verification Report")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Bundle: %s\n", report.BundlePath)
	fmt.Fprintf(w, "Profile: %s\n", renderVerifyProfileLine(report))
	renderer.renderVerifySummary(w, report)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Obligations")
	fmt.Fprintln(w)
	for _, line := range obligationLines(report) {
		renderer.renderLine(w, line)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Relations")
	fmt.Fprintln(w)
	for _, line := range relationLines(report) {
		renderer.renderLine(w, line)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Anchoring")
	fmt.Fprintln(w)
	fmt.Fprintln(w, report.Anchoring.Summary)
	if report.Anchoring.AnchorHash != "" {
		fmt.Fprintf(w, "Anchor hash: %s\n", report.Anchoring.AnchorHash)
	}
	if report.Anchoring.Reason != "" && report.Anchoring.Status != "verified" {
		fmt.Fprintf(w, "Reason: %s\n", report.Anchoring.Reason)
	}
	for _, err := range report.Anchoring.Errors {
		if err == report.Anchoring.Reason {
			continue
		}
		fmt.Fprintf(w, "  anchor warning: %s\n", err)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Bundle Signature")
	fmt.Fprintln(w)
	if report.BundleSignature == nil {
		fmt.Fprintln(w, "Signature present: no")
	} else {
		fmt.Fprintf(w, "Signature present: %s\n", yesNo(report.BundleSignature.Present))
		fmt.Fprintf(w, "Signature verified: %s\n", yesNo(report.BundleSignature.Verified))
		if report.BundleSignature.BundleHash != "" {
			fmt.Fprintf(w, "Bundle hash: %s\n", report.BundleSignature.BundleHash)
		}
		if report.BundleSignature.Error != "" {
			fmt.Fprintf(w, "Signature error: %s\n", report.BundleSignature.Error)
		}
	}
	if len(report.Signatures) > 0 {
		for _, sig := range report.Signatures {
			fmt.Fprintln(w, renderSignatureProvenanceLine(sig))
		}
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "CAS Scores")
	fmt.Fprintln(w)
	renderer.renderCASScores(w, report.CAS)
	if report.CAS != nil && len(report.Profiles) > 0 && !report.Profiles[0].Pass {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Note: CAS is diagnostic only when obligation checks fail; it does not overturn FAIL.")
	}

	if len(report.InformationalNotes) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Notes")
		fmt.Fprintln(w)
		for _, note := range report.InformationalNotes {
			renderer.renderLine(w, verifyOutputLine{status: verifyLineNote, text: note})
		}
	}

	if len(report.Exclusions) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Exclusions")
		fmt.Fprintln(w)
		for _, exclusion := range report.Exclusions {
			renderer.renderLine(w, verifyOutputLine{status: verifyLineNote, text: exclusion})
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Verification: %s\n", renderer.renderVerificationStatus(report))
}

func useVerifyANSI(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	return isTerminal(w)
}

func isTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	// Fall back to a simple character-device check when x/term is unavailable.
	return info.Mode()&os.ModeCharDevice != 0
}

const maxVerifySummaryIssues = 7

func (r verifyTextRenderer) renderVerifySummary(w io.Writer, report verifypkg.Report) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Summary")
	fmt.Fprintln(w)

	integrityLabel := "FAIL"
	integrityColour := ansiRed
	if report.Integrity.ChainValid {
		integrityLabel = "PASS"
		integrityColour = ansiGreen
	}
	fmt.Fprintf(w, "Integrity: %s\n", r.colourise(integrityLabel, integrityColour))

	profileLabel := "NOT EVALUATED"
	profileColour := ansiYellow
	if len(report.Profiles) > 0 {
		if report.Profiles[0].Pass {
			profileLabel = "PASS"
			profileColour = ansiGreen
		} else {
			profileLabel = "FAIL"
			profileColour = ansiRed
		}
	}
	fmt.Fprintf(w, "Profile:   %s\n", r.colourise(profileLabel, profileColour))

	if report.CAS != nil {
		fmt.Fprintf(w, "CAS:       %.2f (%s)\n", report.CAS.Overall, report.CAS.Grade)
	} else {
		fmt.Fprintf(w, "CAS:       n/a\n")
	}

	issues := collectVerifySummaryIssues(report, maxVerifySummaryIssues)
	if len(issues) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Issues")
		for _, issue := range issues {
			r.renderLine(w, issue)
		}
	}

	if len(report.Exclusions) > 0 {
		fmt.Fprintf(w, "\nExclusions: %d declared blind spot(s) (see below)\n", len(report.Exclusions))
	}
}

func collectVerifySummaryIssues(report verifypkg.Report, limit int) []verifyOutputLine {
	if limit <= 0 {
		return nil
	}

	lines := make([]verifyOutputLine, 0, limit)

	if !report.Integrity.ChainValid && report.Integrity.Error != "" {
		lines = append(lines, verifyOutputLine{
			status: verifyLineFail,
			text:   report.Integrity.Error,
		})
	}

	if len(report.Profiles) > 0 {
		profile := report.Profiles[0]
		for _, failure := range profile.CriticalFailures {
			if len(lines) >= limit {
				return lines
			}
			lines = append(lines, verifyOutputLine{
				status: verifyLineFail,
				text:   failure.Detail,
			})
		}
		for _, warning := range profile.RequiredWarnings {
			if len(lines) >= limit {
				return lines
			}
			lines = append(lines, verifyOutputLine{
				status: verifyLineWarn,
				text:   warning,
			})
		}
	}

	for _, driver := range report.ResidualRisk.Drivers {
		if len(lines) >= limit {
			return lines
		}
		lines = append(lines, verifyOutputLine{
			status: verifyLineNote,
			text:   "residual risk driver: " + driver,
		})
	}

	return lines
}

func renderVerifyProfileLine(report verifypkg.Report) string {
	if len(report.Profiles) == 0 {
		return "none matched"
	}

	profile := report.Profiles[0]
	return fmt.Sprintf("%s (%s)", profile.ProfileID, humaniseWorkflowClass(profile.WorkflowClass))
}

func humaniseWorkflowClass(workflowClass string) string {
	words := strings.Fields(strings.ReplaceAll(workflowClass, "_", " "))
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	if len(words) == 0 {
		return workflowClass
	}
	return strings.Join(words, " ")
}

func obligationLines(report verifypkg.Report) []verifyOutputLine {
	if len(report.Profiles) == 0 {
		if report.CAS != nil && report.CAS.SubScores["SC"] > 0 {
			return []verifyOutputLine{{
				status: verifyLinePass,
				text:   fmt.Sprintf("%s Source commitment evidence present", event.TypeBundleSignature),
			}}
		}
		return []verifyOutputLine{{status: verifyLineNote, text: "No profile-specific obligations evaluated."}}
	}

	profile := report.Profiles[0]
	specs := obligationSpecs(profile.ProfileID)
	if len(specs) == 0 {
		return genericObligationLines(profile)
	}

	failures := obligationFailures(profile)
	warnings := append([]string{}, profile.RequiredWarnings...)
	lines := make([]verifyOutputLine, 0, len(specs)+len(failures)+len(warnings))
	usedFailures := make([]bool, len(failures))
	usedWarnings := make([]bool, len(warnings))

	for _, spec := range specs {
		display := fmt.Sprintf("%s %s", spec.eventType, spec.message)
		if idx := matchingFailureIndex(failures, usedFailures, spec.eventType); idx >= 0 {
			usedFailures[idx] = true
			lines = append(lines, verifyOutputLine{
				status: verifyLineFail,
				text:   display + " [CRITICAL FAIL]",
			})
			continue
		}
		if idx := matchingWarningIndex(warnings, usedWarnings, spec.eventType); idx >= 0 {
			usedWarnings[idx] = true
			lines = append(lines, verifyOutputLine{
				status: verifyLineWarn,
				text:   display + " [warning]",
			})
			continue
		}
		lines = append(lines, verifyOutputLine{status: verifyLinePass, text: display})
	}

	for i, failure := range failures {
		if usedFailures[i] {
			continue
		}
		lines = append(lines, verifyOutputLine{
			status: verifyLineFail,
			text:   failure.Detail + " [CRITICAL FAIL]",
		})
	}
	for i, warning := range warnings {
		if usedWarnings[i] {
			continue
		}
		lines = append(lines, verifyOutputLine{
			status: verifyLineWarn,
			text:   warning + " [warning]",
		})
	}

	return lines
}

func genericObligationLines(profile verifypkg.ProfileResult) []verifyOutputLine {
	lines := make([]verifyOutputLine, 0, len(profile.CriticalFailures)+len(profile.RequiredWarnings)+1)
	for _, failure := range obligationFailures(profile) {
		lines = append(lines, verifyOutputLine{
			status: verifyLineFail,
			text:   failure.Detail + " [CRITICAL FAIL]",
		})
	}
	for _, warning := range profile.RequiredWarnings {
		lines = append(lines, verifyOutputLine{
			status: verifyLineWarn,
			text:   warning + " [warning]",
		})
	}
	if len(lines) == 0 {
		lines = append(lines, verifyOutputLine{status: verifyLinePass, text: "All recorded obligations satisfied."})
	}
	return lines
}

func relationLines(report verifypkg.Report) []verifyOutputLine {
	if len(report.Profiles) == 0 {
		return []verifyOutputLine{{status: verifyLineNote, text: "No profile-specific relation checks evaluated."}}
	}

	profile := report.Profiles[0]
	relationFailures := relationFailures(profile)
	if len(relationFailures) > 0 {
		lines := make([]verifyOutputLine, 0, len(relationFailures))
		for _, failure := range relationFailures {
			lines = append(lines, verifyOutputLine{
				status: verifyLineFail,
				text:   failure.Detail + " [CRITICAL FAIL]",
			})
		}
		return lines
	}
	if len(obligationFailures(profile)) > 0 {
		return []verifyOutputLine{{
			status: verifyLineNote,
			text:   "Relation checks incomplete because required events are missing.",
		}}
	}

	specs := relationSpecs(profile.ProfileID)
	if len(specs) == 0 {
		return []verifyOutputLine{{status: verifyLinePass, text: "No relation issues detected."}}
	}

	lines := make([]verifyOutputLine, 0, len(specs))
	for _, spec := range specs {
		lines = append(lines, verifyOutputLine{
			status: verifyLinePass,
			text:   fmt.Sprintf("%s → %s", spec.from, spec.to),
		})
	}
	return lines
}

func obligationSpecs(profileID string) []verifyObligationSpec {
	schema, ok := loadOutputSchema(profileID)
	if !ok {
		return nil
	}

	specs := make([]verifyObligationSpec, 0, len(schema.Required)+len(schema.Optional))
	for _, rule := range schema.Required {
		specs = append(specs, verifyObligationSpec{
			eventType: rule.Type,
			message:   schemaRuleMessage(rule),
			warning:   strings.EqualFold(rule.Severity, "warning"),
		})
	}
	for _, rule := range schema.Optional {
		specs = append(specs, verifyObligationSpec{
			eventType: rule.Type,
			message:   schemaRuleMessage(rule),
			warning:   strings.EqualFold(rule.Severity, "warning"),
		})
	}
	return specs
}

func relationSpecs(profileID string) []verifyRelationSpec {
	schema, ok := loadOutputSchema(profileID)
	if !ok {
		return nil
	}

	specs := make([]verifyRelationSpec, 0, len(schema.Relations))
	for _, rule := range schema.Relations {
		specs = append(specs, verifyRelationSpec{from: rule.From, to: rule.To})
	}
	return specs
}

func loadOutputSchema(profileID string) (profiledsl.ProfileSchema, bool) {
	if !profiledsl.HasSchema(profileID) {
		return profiledsl.ProfileSchema{}, false
	}
	return profiledsl.MustLoadSchema(profileID), true
}

func schemaRuleMessage(rule profiledsl.EventRule) string {
	if strings.TrimSpace(rule.Message) != "" {
		return rule.Message
	}
	if strings.EqualFold(rule.Severity, "warning") {
		return "Recommended"
	}
	return "Required"
}

func obligationFailures(profile verifypkg.ProfileResult) []verifypkg.CriticalFailure {
	failures := make([]verifypkg.CriticalFailure, 0, len(profile.CriticalFailures))
	for _, failure := range profile.CriticalFailures {
		if failure.Kind == "relation_violation" {
			continue
		}
		failures = append(failures, failure)
	}
	return failures
}

func relationFailures(profile verifypkg.ProfileResult) []verifypkg.CriticalFailure {
	failures := make([]verifypkg.CriticalFailure, 0, len(profile.CriticalFailures))
	for _, failure := range profile.CriticalFailures {
		if failure.Kind != "relation_violation" {
			continue
		}
		failures = append(failures, failure)
	}
	return failures
}

func matchingFailureIndex(failures []verifypkg.CriticalFailure, used []bool, eventType string) int {
	for i, failure := range failures {
		if used[i] {
			continue
		}
		if strings.Contains(failure.Detail, eventType) {
			return i
		}
	}
	return -1
}

func matchingWarningIndex(warnings []string, used []bool, eventType string) int {
	for i, warning := range warnings {
		if used[i] {
			continue
		}
		if strings.Contains(warning, eventType) {
			return i
		}
	}
	return -1
}

func (r verifyTextRenderer) renderCASScores(w io.Writer, cas *verifypkg.CASResult) {
	if cas == nil {
		fmt.Fprintln(w, "CAS unavailable")
		return
	}

	scoreLabels := []struct {
		key   string
		label string
	}{
		{key: "EC", label: "Event Coverage"},
		{key: "FC", label: "Field Completeness"},
		{key: "RC", label: "Relation Consistency"},
		{key: "TC", label: "Temporal Consistency"},
		{key: "SC", label: "Source Commitment"},
		{key: "XC", label: "External Corroboration"},
		{key: "AC", label: "Anchor Coverage"},
		{key: "GC", label: "Gating Completeness"},
	}

	for _, score := range scoreLabels {
		value, ok := cas.SubScores[score.key]
		if !ok {
			continue
		}
		fmt.Fprintf(w, "%s (%s): %.2f\n", score.key, score.label, value)
	}
	fmt.Fprintf(w, "Overall: %.2f\n", cas.Overall)
}

func (r verifyTextRenderer) renderVerificationStatus(report verifypkg.Report) string {
	switch {
	case !report.Integrity.ChainValid:
		return r.colourise("FAIL", ansiRed)
	case report.BundleSignature != nil && !report.BundleSignature.Verified:
		return r.colourise("FAIL", ansiRed)
	case len(report.Profiles) == 0:
		return r.colourise("NOT EVALUATED", ansiYellow)
	case report.Profiles[0].Pass:
		return r.colourise("PASS", ansiGreen)
	default:
		return r.colourise("FAIL", ansiRed)
	}
}

func (r verifyTextRenderer) renderLine(w io.Writer, line verifyOutputLine) {
	fmt.Fprintf(w, "%s %s\n", r.linePrefix(line.status), line.text)
}

func (r verifyTextRenderer) linePrefix(status verifyLineStatus) string {
	switch status {
	case verifyLinePass:
		return r.colourise("✓", ansiGreen)
	case verifyLineFail:
		return r.colourise("✗", ansiRed)
	case verifyLineWarn:
		return r.colourise("⚠", ansiYellow)
	default:
		return "-"
	}
}

func (r verifyTextRenderer) colourise(text, code string) string {
	if !r.colour {
		return text
	}
	return code + text + ansiReset
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

// renderSignatureProvenanceLine produces the canonical one-liner used in
// `atb verify` text output for each atb.bundle.signature record:
//
//	signature: backend=<b> key_id=<k> signed_at=<t> pubkey=<p> valid=<bool>
//
// Empty fields are rendered as "-" so the line stays uniform across local
// and remote-signed bundles.
func renderSignatureProvenanceLine(sig verifypkg.SignatureProvenance) string {
	dash := func(s string) string {
		if s == "" {
			return "-"
		}
		return s
	}
	return fmt.Sprintf(
		"signature: backend=%s key_id=%s signed_at=%s pubkey=%s valid=%t",
		dash(sig.Backend),
		dash(sig.KeyID),
		dash(sig.SignedAt),
		dash(sig.PubKey),
		sig.Valid,
	)
}
