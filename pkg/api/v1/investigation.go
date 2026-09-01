// SPDX-License-Identifier: MIT
package apiv1

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/pcguest/atb/internal/capabilities"
	"github.com/pcguest/atb/internal/contextlineage"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/incident"
	atbauth "github.com/pcguest/atb/pkg/auth"
)

var trustLimitations = []string{
	"ATB does not prove complete capture.",
	"ATB does not prove pre-capture truthfulness.",
	"ATB does not prove model correctness.",
	"ATB does not prove real-world identity solely from caller data.",
	"ATB does not prove universal absence.",
	"ATB does not certify legal or regulatory compliance.",
	"ATB does not prove external immutable custody without external custody evidence.",
}

func (s *APIServer) handleInvestigationOverview(w http.ResponseWriter, r *http.Request) {
	if !s.allowInvestigationRead(w, r, false) {
		return
	}
	integrityValid := s.verifyErr == nil
	var findings []incident.Finding
	if integrityValid {
		var ok bool
		findings, ok = s.investigationFindings(w, r)
		if !ok {
			return
		}
	}
	critical := 0
	for _, finding := range findings {
		if finding.Severity == "critical" {
			critical++
		}
	}
	status := "FAILED"
	summary := "The presented bundle failed integrity verification. Coverage does not repair invalid integrity."
	if integrityValid {
		status = "VERIFIED"
		summary = investigationSummary(findings, s.profileReport)
	}
	writeJSON(w, http.StatusOK, InvestigationOverviewResponse{
		BundlePath:       s.bundlePath,
		EventCount:       recordCount(s),
		IntegrityValid:   integrityValid,
		IntegrityStatus:  status,
		Profile:          cloneProfileSummary(s.profileReport, integrityValid),
		FindingCount:     len(findings),
		CriticalFindings: critical,
		CustodyState:     custodyState(s),
		Summary:          summary,
	})
}

func (s *APIServer) handleInvestigationFindings(w http.ResponseWriter, r *http.Request) {
	if !s.allowInvestigationRead(w, r, true) {
		return
	}
	findings, ok := s.investigationFindings(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, InvestigationFindingsResponse{
		Findings: findings,
	})
}

func (s *APIServer) investigationFindings(w http.ResponseWriter, r *http.Request) ([]incident.Finding, bool) {
	sessions, err := s.sessionsForRequest(r)
	if err != nil {
		writeSessionIndexError(w, err)
		return nil, false
	}
	return incident.BuildBundleFindings(s.b, sessions), true
}

func (s *APIServer) handleInvestigationTimeline(w http.ResponseWriter, r *http.Request) {
	if !s.allowInvestigationRead(w, r, true) {
		return
	}
	events := make([]TimelineEventDTO, 0, len(s.b.Records))
	for _, record := range s.b.Records {
		events = append(events, TimelineEventDTO{
			Sequence:   record.Event.Sequence,
			Type:       record.Event.Type,
			Label:      humanEventLabel(record.Event.Type),
			Timestamp:  record.Event.Timestamp,
			Hash:       record.Hash,
			Family:     eventFamily(record.Event.Type),
			CausalEdge: false,
		})
	}
	writeJSON(w, http.StatusOK, InvestigationTimelineResponse{Events: events})
}

func (s *APIServer) handleInvestigationContext(w http.ResponseWriter, r *http.Request) {
	if !s.allowInvestigationRead(w, r, true) {
		return
	}
	writeJSON(w, http.StatusOK, InvestigationContextResponse{
		Lineage:      contextlineage.Build(s.b.Records),
		Capabilities: capabilities.Derive(s.b.Records),
	})
}

func (s *APIServer) handleInvestigationRelationships(w http.ResponseWriter, r *http.Request) {
	if !s.allowInvestigationRead(w, r, true) {
		return
	}
	writeJSON(w, http.StatusOK, InvestigationRelationshipsResponse{
		Relationships: supportedRelationships(s),
	})
}

func (s *APIServer) handleInvestigationTrust(w http.ResponseWriter, r *http.Request) {
	if !s.allowInvestigationRead(w, r, false) {
		return
	}
	response := InvestigationTrustResponse{
		ProofStatement:   "ATB proves the integrity and order of the records presented in a bundle.",
		IntegrityValid:   s.verifyErr == nil,
		Canonicalisation: "rfc8785",
		SignatureStatus:  signatureStatus(s),
		AnchorStatus:     "absent",
		CustodyState:     custodyState(s),
		Limitations:      append([]string{}, trustLimitations...),
	}
	if s.profileReport != nil {
		response.ProfileID = s.profileReport.ProfileID
		response.ProfilePass = s.profileReport.Pass
		response.AssuranceValid = s.profileReport.AssuranceValid
		response.AnchorStatus = s.profileReport.AnchorStatus
		if response.IntegrityValid && s.profileReport.IntegrityValid {
			response.CoverageScore = s.profileReport.CoverageScore
			response.CoverageGrade = s.profileReport.CoverageGrade
			response.AssessmentCoverage = s.profileReport.AssessmentCoverage
		}
	}
	if s.b != nil {
		for _, record := range s.b.Records {
			if record.Event.Type == event.TypeCorroborationExternal {
				response.ExternalCorroboration = true
				break
			}
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *APIServer) handleInvestigationReport(w http.ResponseWriter, r *http.Request) {
	if !s.allowInvestigationRead(w, r, true) {
		return
	}
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "session_id is required"})
		return
	}
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "markdown"
	}
	if format != "markdown" && format != "json" {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "format must be markdown or json"})
		return
	}
	report, err := incident.Build(r.Context(), s.bundlePath, sessionID)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, APIError{Error: "incident report could not be built"})
		return
	}
	if !report.Found {
		writeJSON(w, http.StatusNotFound, APIError{Error: "session not found in captured evidence"})
		return
	}
	if format == "json" {
		payload, err := report.JSON()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIError{Error: "incident report could not be rendered"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="atb-incident-report.json"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="atb-incident-report.md"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(report.Markdown()))
}

func (s *APIServer) allowInvestigationRead(w http.ResponseWriter, r *http.Request, requireIntegrity bool) bool {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return false
	}
	if !s.requireRole(w, r, atbauth.RoleViewer) {
		return false
	}
	if requireIntegrity && !s.requireVerified(w) {
		return false
	}
	return true
}

func investigationSummary(findings []incident.Finding, profile *ProfileReportSummary) string {
	if len(findings) > 0 {
		return fmt.Sprintf("The captured evidence contains %d investigation finding(s). Open Findings for their bounded basis.", len(findings))
	}
	if profile != nil && !profile.Pass {
		return "The applicable profile has unsatisfied critical evidence obligations."
	}
	return "No investigation findings were identified in the captured evidence."
}

func supportedRelationships(s *APIServer) []RelationshipDTO {
	if s.b == nil {
		return []RelationshipDTO{}
	}
	keys := []string{"action_id", "request_id", "approval_id", "policy_id", "retrieval_id", "invocation_id"}
	lastByBinding := map[string]int{}
	relationships := []RelationshipDTO{}
	for _, record := range s.b.Records {
		data, _ := record.Event.Data.(map[string]any)
		for _, key := range keys {
			value, _ := data[key].(string)
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			binding := key + "\x00" + value
			if previous, exists := lastByBinding[binding]; exists {
				relationships = append(relationships, RelationshipDTO{
					ID:             fmt.Sprintf("%s:%d:%d", key, previous, record.Event.Sequence),
					SourceSequence: previous,
					TargetSequence: record.Event.Sequence,
					Kind:           key,
					EvidenceValue:  value,
					Strength:       "semantic",
				})
			}
			lastByBinding[binding] = record.Event.Sequence
		}
		if record.Event.TraceID != "" {
			binding := "trace_id\x00" + record.Event.TraceID
			if previous, exists := lastByBinding[binding]; exists {
				relationships = append(relationships, RelationshipDTO{
					ID:             fmt.Sprintf("trace_id:%d:%d", previous, record.Event.Sequence),
					SourceSequence: previous,
					TargetSequence: record.Event.Sequence,
					Kind:           "trace_context",
					EvidenceValue:  record.Event.TraceID,
					Strength:       "semantic",
				})
			}
			lastByBinding[binding] = record.Event.Sequence
		}
	}
	sort.SliceStable(relationships, func(i, j int) bool {
		return relationships[i].ID < relationships[j].ID
	})
	return relationships
}

func humanEventLabel(eventType string) string {
	labels := map[string]string{
		event.TypeAIRequestReceived:   "Request received",
		event.TypeAIRetrievalExecuted: "Context retrieved",
		event.TypeRAGRetrieval:        "PageIndex context retrieved",
		event.TypeAIContextUnit:       "Context unit recorded",
		event.TypeAIContextOperation:  "Context transformed or assembled",
		event.TypeMcpOperation:        "MCP operation recorded",
		event.TypeAIModelInvoked:      "Model invoked",
		event.TypeAIModelOutput:       "Model output recorded",
		event.TypeAIPolicyDecision:    "Policy decision recorded",
		event.TypeAIHumanApproval:     "Human approval recorded",
		event.TypeAIActionExecuted:    "Action executed",
		event.TypeAIResponseSent:      "Response sent",
	}
	if label := labels[eventType]; label != "" {
		return label
	}
	return strings.ReplaceAll(eventType, ".", " ")
}

// eventFamily is a display overlay only. Dual wire families stay distinct types.
func eventFamily(eventType string) string {
	switch {
	case strings.HasPrefix(eventType, "ai.llm"), strings.HasPrefix(eventType, "atb.llm"), strings.HasPrefix(eventType, "ai.model"):
		return "llm"
	case strings.HasPrefix(eventType, "ai.tool"), eventType == event.TypeToolCall:
		return "tool"
	case strings.HasPrefix(eventType, "ai.chain"):
		return "chain"
	case strings.HasPrefix(eventType, "ai.policy"):
		return "policy"
	case strings.HasPrefix(eventType, "ai.action"), strings.HasPrefix(eventType, "atb.mcp"):
		return "action"
	case strings.HasPrefix(eventType, "ai.human"), strings.HasPrefix(eventType, "atb.human"):
		return "human"
	case strings.HasPrefix(eventType, "ai.job"):
		return "job"
	case strings.HasPrefix(eventType, "atb.corroboration"):
		return "corroboration"
	case strings.HasPrefix(eventType, "ai.export"), strings.HasPrefix(eventType, "data.export"), strings.HasPrefix(eventType, "atb.data.export"):
		return "export"
	case strings.HasPrefix(eventType, "data.retention"):
		return "retention"
	case strings.Contains(eventType, "context"), strings.Contains(eventType, "retrieval"), strings.Contains(eventType, "rag_"):
		return "context"
	default:
		return "other"
	}
}

func recordCount(s *APIServer) int {
	if s.b == nil {
		return 0
	}
	return len(s.b.Records)
}

func signatureStatus(s *APIServer) string {
	if s.verifierReport == nil || len(s.verifierReport.Signatures) == 0 {
		return "absent"
	}
	for _, signature := range s.verifierReport.Signatures {
		if signature.Valid {
			return "verified"
		}
	}
	return "failed"
}

func custodyState(_ *APIServer) string {
	return "Local only"
}

func cloneProfileSummary(profile *ProfileReportSummary, integrityValid bool) *ProfileReportSummary {
	if profile == nil {
		return nil
	}
	copy := *profile
	if !integrityValid || !copy.IntegrityValid {
		copy.CoverageScore = 0
		copy.CoverageGrade = ""
		copy.AssessmentCoverage = 0
		copy.DimensionAssessments = nil
	}
	return &copy
}
