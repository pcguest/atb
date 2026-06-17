// SPDX-License-Identifier: MIT
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/trust"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

var errTrustReportHelp = errors.New("trust-report help requested")

const trustReportRAGAnswerProfileID = "atb.profile.rag_answer"

type trustReportConfig struct {
	BundlePath string
	Format     string
	ProfileID  string
}

type ragAnswerReportDetails struct {
	ModelProvider                string
	ModelID                      string
	ModelParametersDigestPresent bool
	RetrievalPresent             bool
	ResponsePresent              bool
	RequestIDBindingConfirmed    bool
}

func cmdTrustReport() {
	os.Exit(runTrustReport(os.Args[2:], os.Stdout, os.Stderr))
}

func runTrustReport(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseTrustReportArgs(args)
	if err != nil {
		if errors.Is(err, errTrustReportHelp) {
			printTrustReportUsage()
			return exitSuccess
		}
		fmt.Fprintf(stderr, "atb trust-report: %v\n", err)
		printTrustReportUsage()
		return exitUserError
	}

	if cfg.Format == "json" {
		return runTrustReportJSON(cfg, stdout, stderr)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "atb trust-report: current directory: %v\n", err)
		return exitSystemError
	}

	_, resolvedProfileID, err := resolveTrustReportProfile(cfg.BundlePath, cfg.ProfileID)
	if err != nil {
		fmt.Fprintf(stderr, "atb trust-report: %v\n", err)
		printTrustReportUsage()
		return exitUserError
	}

	buildProfileID := cfg.ProfileID
	if buildProfileID == "" && resolvedProfileID == trustReportRAGAnswerProfileID {
		buildProfileID = resolvedProfileID
	} else if buildProfileID != "" && isVerifyProfilePath(buildProfileID) {
		buildProfileID = resolvedProfileID
	}

	report := trust.BuildReport(cwd, cfg.BundlePath, buildProfileID)
	ragDetails, hasRAGDetails := loadRAGAnswerReportDetails(cfg.BundlePath, resolvedProfileID)
	switch cfg.Format {
	case "text":
		renderTrustReportText(stdout, report)
	case "markdown":
		renderTrustReportMarkdown(stdout, report, hasRAGDetails, ragDetails)
	default:
		fmt.Fprintf(stderr, "atb trust-report: unsupported format %q\n", cfg.Format)
		return exitUserError
	}
	return exitSuccess
}

func runTrustReportJSON(cfg trustReportConfig, stdout, stderr io.Writer) int {
	var profile verifypkg.Profile
	if strings.TrimSpace(cfg.ProfileID) != "" {
		var err error
		profile, _, err = resolveTrustReportProfile(cfg.BundlePath, cfg.ProfileID)
		if err != nil {
			fmt.Fprintf(stderr, "atb trust-report: %v\n", err)
			printTrustReportUsage()
			return exitUserError
		}
	}

	b, err := bundle.Load(cfg.BundlePath)
	if err != nil {
		fmt.Fprintf(stderr, "atb trust-report: %v\n", err)
		return classifyBundleLoadError(err)
	}

	report := verifypkg.Verify(b, cfg.BundlePath, "")
	if profile != nil {
		report = verifypkg.VerifyWithProfile(b, cfg.BundlePath, profile)
	}

	trustReport := verifypkg.TrustReportFromVerify(report, b)
	payload, err := json.MarshalIndent(trustReport, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "atb trust-report: marshal json: %v\n", err)
		return exitSystemError
	}
	if _, err := stdout.Write(append(payload, '\n')); err != nil {
		fmt.Fprintf(stderr, "atb trust-report: write json: %v\n", err)
		return exitSystemError
	}
	if trustReport.Pass {
		return exitSuccess
	}
	return exitUserError
}

func parseTrustReportArgs(args []string) (trustReportConfig, error) {
	cfg := trustReportConfig{
		BundlePath: bundle.DefaultPath(),
		Format:     "markdown",
	}
	pathSet := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errTrustReportHelp
		case arg == "--format":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --format (expected markdown|json|text)")
			}
			cfg.Format = strings.ToLower(strings.TrimSpace(args[i+1]))
			i++
		case arg == "--profile" || arg == "-p":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for %s", arg)
			}
			cfg.ProfileID = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(arg, "--format="):
			cfg.Format = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--format=")))
		case strings.HasPrefix(arg, "--profile="):
			cfg.ProfileID = strings.TrimSpace(strings.TrimPrefix(arg, "--profile="))
		case strings.HasPrefix(arg, "--"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			if pathSet {
				return cfg, fmt.Errorf("expected at most one bundle path")
			}
			cfg.BundlePath = normalizeBundlePath(arg)
			pathSet = true
		}
	}
	if cfg.Format != "markdown" && cfg.Format != "json" && cfg.Format != "text" {
		return cfg, fmt.Errorf("invalid format %q (expected markdown|json|text)", cfg.Format)
	}
	return cfg, nil
}

func printTrustReportUsage() {
	fmt.Println("Usage: atb trust-report [bundle_path] [--format markdown|json|text] [--profile <id>]")
}

func renderTrustReportMarkdown(w io.Writer, report trust.Report, includeRAGDetails bool, ragDetails ragAnswerReportDetails) {
	fmt.Fprintln(w, "# ATB Trust Report")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "- Verdict: **%s**\n", strings.ToUpper(report.Status))
	fmt.Fprintf(w, "- CI Gate: **%s**\n", strings.ToUpper(report.Gate.Status))
	fmt.Fprintf(w, "- Blocking failures: %d\n", report.Gate.BlockingFailures)
	fmt.Fprintf(w, "- Generated: %s\n", report.GeneratedAt)
	fmt.Fprintf(w, "- Bundle: `%s`\n", report.BundlePath)
	fmt.Fprintf(w, "- Chain length: %d\n", report.ChainLength)
	if report.HeadHash != "" {
		fmt.Fprintf(w, "- Head hash: `%s`\n", report.HeadHash)
	}
	fmt.Fprintf(w, "- Checks: total=%d pass=%d warn=%d fail=%d\n", report.Summary.Total, report.Summary.Pass, report.Summary.Warn, report.Summary.Fail)
	if len(report.Gate.FailedChecks) > 0 {
		fmt.Fprintf(w, "- Failed blocking checks: `%s`\n", strings.Join(report.Gate.FailedChecks, "`, `"))
	}
	fmt.Fprintln(w)
	if report.CAS != nil {
		fmt.Fprintln(w, "## Completeness Assurance")
		fmt.Fprintln(w)
		fmt.Fprintf(w, "- Profile: `%s`\n", report.CAS.ProfileID)
		fmt.Fprintf(w, "- Workflow class: `%s`\n", report.CAS.WorkflowClass)
		fmt.Fprintf(w, "- Overall: %.3f (%s)\n", report.CAS.Overall, report.CAS.Grade)
		fmt.Fprintf(w, "- Anchor quality: `%s` (XC=%.3f, AC=%.3f)\n", report.CAS.AnchorQuality.Label, report.CAS.AnchorQuality.XC, report.CAS.AnchorQuality.AC)

		scoreKeys := sortedFloatKeys(report.CAS.SubScores)
		if len(scoreKeys) > 0 {
			fmt.Fprintf(w, "- Sub-scores: `%s`\n", formatFloatMap(report.CAS.SubScores, scoreKeys))
		}

		weightKeys := sortedFloatKeys(report.CAS.Weights)
		if len(weightKeys) > 0 {
			fmt.Fprintf(w, "- Weights: `%s`\n", formatFloatMap(report.CAS.Weights, weightKeys))
		}
		fmt.Fprintln(w)
	}
	if len(report.ReviewerIdentities) > 0 {
		fmt.Fprintln(w, "## Reviewer identity evidence")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Caller-provided identity evidence preserved by ATB; ATB did not independently verify the asserted identity.")
		for _, evidence := range report.ReviewerIdentities {
			fmt.Fprintf(w, "- Seq %d: provider=`%s` subject=`%s` assertion=`%s` digest=`%s`\n",
				evidence.Sequence,
				evidence.IdentityProvider,
				evidence.Subject,
				evidence.AssertionType,
				evidence.AssertionDigest,
			)
		}
		fmt.Fprintln(w)
	}
	if includeRAGDetails {
		fmt.Fprintln(w, "## Model invocation")
		fmt.Fprintln(w)
		fmt.Fprintf(w, "- model_provider: `%s`\n", ragDetails.ModelProvider)
		fmt.Fprintf(w, "- model_id: `%s`\n", ragDetails.ModelID)
		fmt.Fprintf(w, "- model_parameters_digest present: `%t`\n", ragDetails.ModelParametersDigestPresent)
		fmt.Fprintln(w)
		fmt.Fprintln(w, "## Retrieval")
		fmt.Fprintln(w)
		fmt.Fprintf(w, "- ai.retrieval.executed present: `%t`\n", ragDetails.RetrievalPresent)
		fmt.Fprintln(w)
		fmt.Fprintln(w, "## Response")
		fmt.Fprintln(w)
		fmt.Fprintf(w, "- ai.response.sent present: `%t`\n", ragDetails.ResponsePresent)
		fmt.Fprintf(w, "- request_id binding confirmed: `%t`\n", ragDetails.RequestIDBindingConfirmed)
		fmt.Fprintln(w)
	}
	for _, category := range report.Categories {
		fmt.Fprintf(w, "## %s (%s)\n\n", category.Title, strings.ToUpper(category.Status))
		for _, check := range category.Checks {
			blocking := "non-blocking"
			if check.Blocking {
				blocking = "blocking"
			}
			fmt.Fprintf(w, "- [%s] (%s, %s) %s: %s\n", strings.ToUpper(check.Status), check.Severity, blocking, check.Title, check.Details)
			if len(check.Evidence) > 0 {
				fmt.Fprintf(w, "  Evidence: `%s`\n", relativeOrOriginal(report.BundlePath, check.Evidence[0]))
			}
		}
		fmt.Fprintln(w)
	}
}

func resolveTrustReportProfile(bundlePath string, profileSpec string) (verifypkg.Profile, string, error) {
	profileSpec = strings.TrimSpace(profileSpec)
	if profileSpec == "" {
		b, err := bundle.Load(bundlePath)
		if err != nil {
			return nil, "", nil
		}
		report := verifypkg.Verify(b, bundlePath, "")
		if len(report.Profiles) == 0 {
			return nil, "", nil
		}
		profileID := report.Profiles[0].ProfileID
		return verifypkg.ProfileByID(profileID), profileID, nil
	}

	if isVerifyProfilePath(profileSpec) {
		profile, err := verifypkg.ResolveProfile(profileSpec)
		if err != nil {
			return nil, "", err
		}
		if profile == nil {
			return nil, "", fmt.Errorf("unknown profile: %s", profileSpec)
		}
		return profile, profile.ID(), nil
	}

	profile := verifypkg.ProfileByID(profileSpec)
	if profile == nil {
		return nil, "", fmt.Errorf("unknown profile: %s", profileSpec)
	}
	return profile, profile.ID(), nil
}

func loadRAGAnswerReportDetails(bundlePath string, resolvedProfileID string) (ragAnswerReportDetails, bool) {
	if resolvedProfileID != trustReportRAGAnswerProfileID {
		return ragAnswerReportDetails{}, false
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		return ragAnswerReportDetails{}, false
	}

	details := ragAnswerReportDetails{
		ModelProvider:             "unknown",
		ModelID:                   "unknown",
		RequestIDBindingConfirmed: false,
	}
	requestIDs := map[string]struct{}{}
	responseRequestIDs := []string{}

	for _, record := range b.Records {
		switch record.Event.Type {
		case event.TypeAIRequestReceived:
			if requestID := trustReportFieldString(record, "request_id"); requestID != "" {
				requestIDs[requestID] = struct{}{}
			}
		case event.TypeAIModelInvoked:
			if fields := trustReportDataMap(record); fields != nil {
				if details.ModelProvider == "unknown" {
					if provider, ok := fields["model_provider"].(string); ok && strings.TrimSpace(provider) != "" {
						details.ModelProvider = provider
					}
				}
				if details.ModelID == "unknown" {
					if modelID, ok := fields["model_id"].(string); ok && strings.TrimSpace(modelID) != "" {
						details.ModelID = modelID
					}
				}
				if hasField := trustReportHasField(fields, "model_parameters_digest"); hasField {
					details.ModelParametersDigestPresent = true
				}
			}
		case event.TypeAIRetrievalExecuted:
			details.RetrievalPresent = true
		case event.TypeAIResponseSent:
			details.ResponsePresent = true
			if requestID := trustReportFieldString(record, "request_id"); requestID != "" {
				responseRequestIDs = append(responseRequestIDs, requestID)
			}
		}
	}

	if len(responseRequestIDs) > 0 {
		details.RequestIDBindingConfirmed = true
		for _, requestID := range responseRequestIDs {
			if _, ok := requestIDs[requestID]; !ok {
				details.RequestIDBindingConfirmed = false
				break
			}
		}
	}

	return details, true
}

func trustReportDataMap(record bundle.Record) map[string]any {
	fields, _ := record.Event.Data.(map[string]any)
	return fields
}

func trustReportFieldString(record bundle.Record, field string) string {
	fields := trustReportDataMap(record)
	if fields == nil {
		return ""
	}
	value, _ := fields[field].(string)
	return strings.TrimSpace(value)
}

func trustReportHasField(fields map[string]any, field string) bool {
	value, ok := fields[field]
	if !ok {
		return false
	}
	if stringValue, ok := value.(string); ok {
		return strings.TrimSpace(stringValue) != ""
	}
	return value != nil
}

func renderTrustReportText(w io.Writer, report trust.Report) {
	renderer := verifyTextRenderer{colour: useVerifyANSI(w)}

	fmt.Fprintf(w, "Bundle:   %s\n", report.BundlePath)
	fmt.Fprintf(w, "Status:   %s\n", renderTrustReportStatus(renderer, report.Status))
	fmt.Fprintf(w, "Gate:     %s\n", strings.ToUpper(report.Gate.Status))
	fmt.Fprintf(w, "Summary:  total=%d pass=%d warn=%d fail=%d\n", report.Summary.Total, report.Summary.Pass, report.Summary.Warn, report.Summary.Fail)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Categories:")
	for _, category := range report.Categories {
		fmt.Fprintf(w, "  %s: %s\n", category.Key, strings.ToUpper(category.Status))
	}
	if details := nonPassingTrustChecks(report.Categories); len(details) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Check Details:")
		for _, detail := range details {
			fmt.Fprintf(w, "  [%s] %s %s: %s\n", detail.status, detail.categoryKey, detail.title, detail.details)
		}
	}

	if report.CAS != nil {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "CAS:")
		fmt.Fprintf(w, "  Profile:   %s\n", report.CAS.ProfileID)
		fmt.Fprintf(w, "  Class:     %s\n", report.CAS.WorkflowClass)
		fmt.Fprintf(w, "  Grade:     %s  (%.2f)\n", report.CAS.Grade, report.CAS.Overall)
		fmt.Fprintf(w, "  Anchor:    %s  (XC=%.2f AC=%.2f)\n", report.CAS.AnchorQuality.Label, report.CAS.AnchorQuality.XC, report.CAS.AnchorQuality.AC)
		fmt.Fprintln(w, "  Sub-scores:")
		fmt.Fprintf(
			w,
			"    EC  %.2f   FC  %.2f   RC  %.2f   TC  %.2f\n",
			report.CAS.SubScores["EC"],
			report.CAS.SubScores["FC"],
			report.CAS.SubScores["RC"],
			report.CAS.SubScores["TC"],
		)
		fmt.Fprintf(
			w,
			"    SC  %.2f   XC  %.2f   AC  %.2f   GC  %.2f\n",
			report.CAS.SubScores["SC"],
			report.CAS.SubScores["XC"],
			report.CAS.SubScores["AC"],
			report.CAS.SubScores["GC"],
		)
	}
	if len(report.ReviewerIdentities) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Reviewer identity evidence (caller-provided; not independently verified by ATB):")
		for _, evidence := range report.ReviewerIdentities {
			fmt.Fprintf(w, "  seq=%d provider=%s subject=%s assertion=%s digest=%s\n",
				evidence.Sequence,
				evidence.IdentityProvider,
				evidence.Subject,
				evidence.AssertionType,
				evidence.AssertionDigest,
			)
		}
	}
}

type trustCheckDetail struct {
	status      string
	categoryKey string
	title       string
	details     string
}

func nonPassingTrustChecks(categories []trust.Category) []trustCheckDetail {
	details := []trustCheckDetail{}
	for _, category := range categories {
		for _, check := range category.Checks {
			if check.Status == trust.StatusPass {
				continue
			}
			details = append(details, trustCheckDetail{
				status:      strings.ToUpper(check.Status),
				categoryKey: category.Key,
				title:       check.Title,
				details:     check.Details,
			})
		}
	}
	return details
}

func renderTrustReportStatus(renderer verifyTextRenderer, status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case trust.StatusPass:
		return renderer.colourise("PASS", ansiGreen)
	case trust.StatusFail:
		return renderer.colourise("FAIL", ansiRed)
	case trust.StatusWarn:
		return renderer.colourise("WARN", ansiYellow)
	default:
		return strings.ToUpper(strings.TrimSpace(status))
	}
}

func relativeOrOriginal(bundlePath string, candidate string) string {
	if !filepath.IsAbs(candidate) {
		return candidate
	}
	base := filepath.Dir(bundlePath)
	rel, err := filepath.Rel(base, candidate)
	if err != nil {
		return candidate
	}
	return rel
}

func sortedFloatKeys(values map[string]float64) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func formatFloatMap(values map[string]float64, keys []string) string {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%.3f", key, values[key]))
	}
	return strings.Join(parts, ", ")
}
