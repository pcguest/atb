package verify

import (
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

const missingTrustFieldValue = "[missing]"

// TrustReport is the structured output of atb trust-report.
// It is distinct from VerifierReport: VerifierReport is for CI
// pipeline consumers; TrustReport is for auditors and compliance
// reviewers.
type TrustReport struct {
	BundlePath    string             `json:"bundle_path"`
	ProfileID     string             `json:"profile_id"`
	WorkflowClass string             `json:"workflow_class"`
	Pass          bool               `json:"pass"`
	CASScore      float64            `json:"cas_score,omitempty"`
	CASGrade      string             `json:"cas_grade,omitempty"`
	ResidualRisk  string             `json:"residual_risk"`
	Chain         TrustChainSection  `json:"chain"`
	Anchoring     TrustAnchorSection `json:"anchoring"`
	Sections      []TrustSection     `json:"sections"`
	Warnings      []string           `json:"warnings,omitempty"`
	BlindSpots    []string           `json:"blind_spots,omitempty"`
}

// TrustChainSection summarises hash-chain integrity.
type TrustChainSection struct {
	Valid            bool   `json:"valid"`
	RecordCount      int    `json:"record_count"`
	FirstSeq         int    `json:"first_seq"`
	LastSeq          int    `json:"last_seq"`
	Canonicalisation string `json:"canonicalisation"` // always "rfc8785"
	HashAlgo         string `json:"hash_algo"`        // always "sha256"
}

// TrustAnchorSection summarises TSA anchoring state.
type TrustAnchorSection struct {
	Present     bool   `json:"present"`
	TSAVerified bool   `json:"tsa_verified"`
	AnchorHash  string `json:"anchor_hash,omitempty"`
}

// TrustSection is one profile-specific evidence section.
type TrustSection struct {
	Title  string            `json:"title"`
	Pass   bool              `json:"pass"`
	Fields map[string]string `json:"fields,omitempty"`
	Notes  []string          `json:"notes,omitempty"`
}

func TrustReportFromVerify(r Report, b *bundle.Bundle) TrustReport {
	report := TrustReport{
		BundlePath:   r.BundlePath,
		ResidualRisk: r.ResidualRisk.Level,
		Chain: TrustChainSection{
			Valid:            r.Integrity.ChainValid,
			FirstSeq:         r.Integrity.FirstSeq,
			LastSeq:          r.Integrity.LastSeq,
			Canonicalisation: r.Integrity.Canonicalization,
			HashAlgo:         r.Integrity.HashAlgo,
		},
		Anchoring: TrustAnchorSection{
			Present:     r.Anchoring.AnchorPresent,
			TSAVerified: r.Anchoring.TSAVerified,
			AnchorHash:  r.Anchoring.AnchorHash,
		},
		Sections: buildTrustSections(r, b),
	}

	if b != nil {
		report.Chain.RecordCount = len(b.Records)
	}
	if len(r.Exclusions) > 0 {
		report.BlindSpots = append([]string(nil), r.Exclusions...)
	}
	if r.CAS != nil {
		report.CASScore = r.CAS.Overall
		report.CASGrade = r.CAS.Grade
	}
	if len(r.Profiles) == 0 {
		return report
	}

	profile := r.Profiles[0]
	report.ProfileID = profile.ProfileID
	report.WorkflowClass = profile.WorkflowClass
	report.Pass = profile.Pass
	if len(profile.RequiredWarnings) > 0 {
		report.Warnings = append([]string(nil), profile.RequiredWarnings...)
	}

	return report
}

func buildTrustSections(r Report, b *bundle.Bundle) []TrustSection {
	if len(r.Profiles) == 0 {
		return []TrustSection{}
	}

	switch r.Profiles[0].ProfileID {
	case profileIDRAGAnswer:
		requestID := firstEventField(b, event.TypeAIRequestReceived, "request_id")
		actorIDHash := firstEventField(b, event.TypeAIRequestReceived, "actor_id_hash")
		purposeTag := firstEventField(b, event.TypeAIRequestReceived, "purpose_tag")
		modelProvider := firstEventField(b, event.TypeAIModelInvoked, "model_provider")
		modelID := firstEventField(b, event.TypeAIModelInvoked, "model_id")
		modelParametersDigest := firstEventField(b, event.TypeAIModelInvoked, "model_parameters_digest")
		promptDigest := firstEventField(b, event.TypeAIModelInvoked, "prompt_digest")
		responseRequestID := firstEventField(b, event.TypeAIResponseSent, "request_id")

		responseNote := "ai.response.sent absent"
		if hasEventType(b, event.TypeAIResponseSent) {
			responseNote = "ai.response.sent present, request_id not bound to ai.request.received"
			if requestID != "" && requestID == responseRequestID {
				responseNote = "ai.response.sent present, request_id bound"
			}
		}

		return []TrustSection{
			{
				Title: "Request",
				Pass:  requestID != "" && actorIDHash != "" && purposeTag != "",
				Fields: map[string]string{
					"request_id":    trustFieldValue(requestID),
					"actor_id_hash": trustFieldValue(actorIDHash),
					"purpose_tag":   trustFieldValue(purposeTag),
				},
			},
			{
				Title: "Model invocation",
				Pass:  modelProvider != "" && modelID != "" && modelParametersDigest != "" && promptDigest != "",
				Fields: map[string]string{
					"model_provider": trustFieldValue(modelProvider),
					"model_id":       trustFieldValue(modelID),
				},
				Notes: []string{digestPresenceNote("model_parameters_digest", modelParametersDigest)},
			},
			{
				Title: "Retrieval",
				Pass:  true,
				Notes: []string{retrievalPresenceNote(hasEventType(b, event.TypeAIRetrievalExecuted))},
			},
			{
				Title: "Response",
				Pass:  true,
				Notes: []string{responseNote},
			},
		}
	case profileIDPrivilegedToolAction:
		return []TrustSection{
			buildRequestTrustSection(b),
			buildPolicyDecisionTrustSection(b),
			buildActionGateTrustSection(b),
		}
	case profileIDDataExport:
		return []TrustSection{
			buildRequestTrustSection(b),
			buildPolicyDecisionTrustSection(b),
			buildActionGateTrustSection(b),
		}
	case profileIDBackgroundAutomation:
		jobID := firstEventField(b, event.TypeAIJobScheduled, "job_id")
		jobType := firstEventField(b, event.TypeAIJobScheduled, "job_type")
		triggerSource := firstEventField(b, event.TypeAIJobScheduled, "trigger_source")
		scheduledByIDHash := firstEventField(b, event.TypeAIJobScheduled, "scheduled_by_id_hash")
		startedPresent := hasEventType(b, event.TypeAIJobStarted)
		completedPresent := hasEventType(b, event.TypeAIJobCompleted)
		startedJobID := firstEventField(b, event.TypeAIJobStarted, "job_id")
		completedJobID := firstEventField(b, event.TypeAIJobCompleted, "job_id")
		outcome := firstEventField(b, event.TypeAIJobCompleted, "outcome")

		jobExecutionPass := hasEventType(b, event.TypeAIJobScheduled) &&
			startedPresent &&
			completedPresent &&
			jobID != "" &&
			jobID == startedJobID &&
			jobID == completedJobID

		jobExecutionNotes := []string{"job_id consistent across scheduled/started/completed"}
		if !jobExecutionPass {
			jobExecutionNotes = missingEventNotes(
				map[string]bool{
					event.TypeAIJobScheduled: hasEventType(b, event.TypeAIJobScheduled),
					event.TypeAIJobStarted:   startedPresent,
					event.TypeAIJobCompleted: completedPresent,
				},
				"job_id mismatch across ai.job.scheduled, ai.job.started, ai.job.completed",
			)
		}

		return []TrustSection{
			{
				Title: "Job scheduling",
				Pass:  jobID != "" && jobType != "" && triggerSource != "" && scheduledByIDHash != "",
				Fields: map[string]string{
					"job_id":   trustFieldValue(jobID),
					"job_type": trustFieldValue(jobType),
				},
			},
			{
				Title: "Job execution",
				Pass:  jobExecutionPass,
				Fields: map[string]string{
					"outcome": trustFieldValue(outcome),
				},
				Notes: jobExecutionNotes,
			},
		}
	case profileIDPolicyDecision:
		return []TrustSection{
			buildRequestTrustSection(b),
			buildPolicyDecisionTrustSection(b),
		}
	case profileIDHumanOverride:
		approvalID := firstEventField(b, event.TypeAIHumanApproval, "approval_id")
		approverIDHash := firstEventField(b, event.TypeAIHumanApproval, "approver_id_hash")
		approvalOutcome := firstEventField(b, event.TypeAIHumanApproval, "approval_outcome")
		actionID := firstEventField(b, event.TypeAIHumanApproval, "action_id")

		return []TrustSection{
			buildRequestTrustSection(b),
			{
				Title: "Human approval",
				Pass:  approvalID != "" && approverIDHash != "" && approvalOutcome != "" && actionID != "",
				Fields: map[string]string{
					"approval_outcome": trustFieldValue(approvalOutcome),
				},
				Notes: []string{outcomeNote("approval_outcome", approvalOutcome)},
			},
			buildActionGateTrustSection(b),
		}
	default:
		notes := append([]string(nil), r.Profiles[0].InformationalNotes...)
		return []TrustSection{
			{
				Title: "Evidence summary",
				Pass:  true,
				Notes: notes,
			},
		}
	}
}

func buildRequestTrustSection(b *bundle.Bundle) TrustSection {
	requestID := firstEventField(b, event.TypeAIRequestReceived, "request_id")
	actorIDHash := firstEventField(b, event.TypeAIRequestReceived, "actor_id_hash")
	purposeTag := firstEventField(b, event.TypeAIRequestReceived, "purpose_tag")

	return TrustSection{
		Title: "Request",
		Pass:  requestID != "" && actorIDHash != "" && purposeTag != "",
		Fields: map[string]string{
			"request_id":    trustFieldValue(requestID),
			"actor_id_hash": trustFieldValue(actorIDHash),
			"purpose_tag":   trustFieldValue(purposeTag),
		},
	}
}

func buildPolicyDecisionTrustSection(b *bundle.Bundle) TrustSection {
	policyID := firstEventField(b, event.TypeAIPolicyDecision, "policy_id")
	policyVersion := firstEventField(b, event.TypeAIPolicyDecision, "policy_version")
	decision := firstEventField(b, event.TypeAIPolicyDecision, "decision")
	actionID := firstEventField(b, event.TypeAIPolicyDecision, "action_id")

	return TrustSection{
		Title: "Policy decision",
		Pass:  policyID != "" && policyVersion != "" && decision != "" && actionID != "",
		Fields: map[string]string{
			"policy_id": trustFieldValue(policyID),
			"decision":  trustFieldValue(decision),
		},
		Notes: []string{outcomeNote("decision", decision)},
	}
}

func buildActionGateTrustSection(b *bundle.Bundle) TrustSection {
	precommitPresent := hasEventType(b, event.TypeAIActionPrecommit)
	executedPresent := hasEventType(b, event.TypeAIActionExecuted)
	committedPresent := hasEventType(b, event.TypeAIActionCommitted)
	precommitActionID := firstEventField(b, event.TypeAIActionPrecommit, "action_id")
	executedActionID := firstEventField(b, event.TypeAIActionExecuted, "action_id")
	committedActionID := firstEventField(b, event.TypeAIActionCommitted, "action_id")
	actionID := firstNonEmpty(precommitActionID, executedActionID, committedActionID)

	actionGatePass := precommitPresent &&
		executedPresent &&
		committedPresent &&
		precommitActionID != "" &&
		precommitActionID == executedActionID &&
		precommitActionID == committedActionID

	notes := []string{"precommit → execute → commit chain intact"}
	if !actionGatePass {
		notes = missingEventNotes(
			map[string]bool{
				event.TypeAIActionPrecommit: precommitPresent,
				event.TypeAIActionExecuted:  executedPresent,
				event.TypeAIActionCommitted: committedPresent,
			},
			"action_id mismatch across ai.action.precommit, ai.action.executed, ai.action.committed",
		)
	}

	return TrustSection{
		Title: "Action gate",
		Pass:  actionGatePass,
		Fields: map[string]string{
			"action_id": trustFieldValue(actionID),
		},
		Notes: notes,
	}
}

func missingEventNotes(presence map[string]bool, mismatchNote string) []string {
	notes := []string{}
	for _, eventType := range []string{
		event.TypeAIJobScheduled,
		event.TypeAIJobStarted,
		event.TypeAIJobCompleted,
		event.TypeAIActionPrecommit,
		event.TypeAIActionExecuted,
		event.TypeAIActionCommitted,
	} {
		present, ok := presence[eventType]
		if ok && !present {
			notes = append(notes, eventType+" missing")
		}
	}
	if len(notes) == 0 {
		return []string{mismatchNote}
	}
	return notes
}

func trustFieldValue(value string) string {
	if value == "" {
		return missingTrustFieldValue
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func hasEventType(b *bundle.Bundle, eventType string) bool {
	if b == nil {
		return false
	}
	for _, record := range b.Records {
		if record.Event.Type == eventType {
			return true
		}
	}
	return false
}

func digestPresenceNote(fieldName string, value string) string {
	if value != "" {
		return fieldName + " present"
	}
	return fieldName + " absent"
}

func retrievalPresenceNote(present bool) string {
	if present {
		return "ai.retrieval.executed present"
	}
	return "ai.retrieval.executed absent — retrieval step not recorded"
}

func outcomeNote(label string, value string) string {
	trimmed := strings.TrimSpace(value)
	switch strings.ToLower(trimmed) {
	case "allow", "deny", "approved", "rejected":
		return label + ": " + strings.ToLower(trimmed)
	case "":
		return label + ": " + missingTrustFieldValue
	default:
		return label + ": " + trimmed
	}
}

func firstEventField(b *bundle.Bundle, eventType, fieldName string) string {
	if b == nil {
		return ""
	}
	for _, record := range b.Records {
		if record.Event.Type != eventType {
			continue
		}
		fields, ok := record.Event.Data.(map[string]any)
		if !ok {
			return ""
		}
		value, ok := fields[fieldName].(string)
		if !ok {
			return ""
		}
		return strings.TrimSpace(value)
	}
	return ""
}
