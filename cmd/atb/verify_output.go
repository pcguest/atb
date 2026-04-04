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

func renderVerifyTerminalReport(w io.Writer, report verifypkg.Report) {
	renderer := verifyTextRenderer{colour: useVerifyANSI(w)}

	fmt.Fprintln(w, "ATB Verification Report")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Bundle: %s\n", report.BundlePath)
	fmt.Fprintf(w, "Profile: %s\n", renderVerifyProfileLine(report))
	fmt.Fprintf(w, "Grade: %s\n", renderer.renderGrade(report))
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
	fmt.Fprintf(w, "Anchor present: %s\n", yesNo(report.Anchoring.AnchorPresent))
	fmt.Fprintf(w, "TSA verified: %s\n", yesNo(report.Anchoring.TSAVerified))
	if report.Anchoring.AnchorHash != "" {
		fmt.Fprintf(w, "Anchor hash: %s\n", report.Anchoring.AnchorHash)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "CAS Scores")
	fmt.Fprintln(w)
	renderer.renderCASScores(w, report.CAS)

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

func (r verifyTextRenderer) renderGrade(report verifypkg.Report) string {
	if report.CAS == nil {
		return fmt.Sprintf("n/a (residual risk: %s)", strings.ToLower(report.ResidualRisk.Level))
	}

	grade := displayLetterGrade(report.CAS)
	return fmt.Sprintf("%s (residual risk: %s)", r.colourGrade(grade), strings.ToLower(report.ResidualRisk.Level))
}

func displayLetterGrade(cas *verifypkg.CASResult) string {
	if cas == nil {
		return "N/A"
	}

	switch cas.Grade {
	case "High":
		return "A"
	case "Medium":
		return "B"
	case "Low":
		return "C"
	default:
		return "F"
	}
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

func (r verifyTextRenderer) colourGrade(grade string) string {
	switch grade {
	case "A", "B":
		return r.colourise(grade, ansiGreen)
	case "C":
		return r.colourise(grade, ansiYellow)
	case "D", "F":
		return r.colourise(grade, ansiRed)
	default:
		return grade
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
