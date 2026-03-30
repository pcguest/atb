package verify

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

const (
	profileIDPrivilegedToolAction = "atb.profile.privileged_tool_action"
	profileIDRAGAnswer            = "atb.profile.rag_answer"
	profileIDDataExport           = "atb.profile.data_export"
	profileIDPolicyDecision       = "atb.profile.policy_decision"
	profileIDHumanOverride        = "atb.profile.human_override"
	profileIDBackgroundAutomation = "atb.profile.background_automation"
)

type eventRequirement struct {
	eventType      string
	requiredFields []string
}

var privilegedCriticalRequirements = []eventRequirement{
	{eventType: event.TypeBundleManifest},
	{eventType: event.TypeAIRequestReceived, requiredFields: []string{"request_id", "actor_id_hash", "purpose_tag"}},
	{eventType: event.TypeAIActionPrecommit, requiredFields: []string{"action_id", "action_type", "action_parameters_digest", "target_resource_id", "intended_effect"}},
	{eventType: event.TypeAIPolicyDecision, requiredFields: []string{"policy_id", "policy_version", "decision", "decision_reason_codes", "subject_id_hash", "action_id"}},
	{eventType: event.TypeAIActionExecuted, requiredFields: []string{"action_id", "execution_outcome", "tool_receipt_digest"}},
	{eventType: event.TypeAIActionCommitted, requiredFields: []string{"action_id", "commit_outcome", "sink_receipt_digest"}},
}

var ragCriticalRequirements = []eventRequirement{
	{eventType: event.TypeBundleManifest},
	{eventType: event.TypeAIRequestReceived, requiredFields: []string{"request_id", "actor_id_hash", "purpose_tag"}},
	{eventType: event.TypeAIModelInvoked, requiredFields: []string{"model_provider", "model_id", "model_parameters_digest", "prompt_digest"}},
	{eventType: event.TypeAIModelOutput, requiredFields: []string{"output_digest", "output_format"}},
}

var dataExportCriticalRequirements = []eventRequirement{
	{eventType: event.TypeBundleManifest},
	{eventType: event.TypeAIRequestReceived, requiredFields: []string{"request_id", "actor_id_hash", "purpose_tag"}},
	{eventType: event.TypeAIPolicyDecision, requiredFields: []string{"policy_id", "policy_version", "decision", "decision_reason_codes", "subject_id_hash"}},
	{eventType: event.TypeAIActionPrecommit, requiredFields: []string{"action_id", "action_type", "action_parameters_digest", "target_resource_id", "intended_effect"}},
	{eventType: event.TypeAIActionExecuted, requiredFields: []string{"action_id", "execution_outcome", "tool_receipt_digest"}},
	{eventType: event.TypeAIActionCommitted, requiredFields: []string{"action_id", "commit_outcome", "sink_receipt_digest"}},
}

var policyDecisionCriticalRequirements = []eventRequirement{
	{eventType: event.TypeBundleManifest},
	{eventType: event.TypeAIRequestReceived, requiredFields: []string{"request_id", "actor_id_hash", "purpose_tag"}},
	{eventType: event.TypeAIPolicyDecision, requiredFields: []string{"policy_id", "policy_version", "decision", "decision_reason_codes", "subject_id_hash", "action_id"}},
}

var humanOverrideCriticalRequirements = []eventRequirement{
	{eventType: event.TypeBundleManifest},
	{eventType: event.TypeAIRequestReceived, requiredFields: []string{"request_id", "actor_id_hash", "purpose_tag"}},
	{eventType: event.TypeAIHumanApproval, requiredFields: []string{"approval_id", "approver_id_hash", "approval_outcome", "justification_digest", "action_id"}},
	{eventType: event.TypeAIActionPrecommit, requiredFields: []string{"action_id", "action_type", "action_parameters_digest", "target_resource_id", "intended_effect"}},
	{eventType: event.TypeAIActionExecuted, requiredFields: []string{"action_id", "execution_outcome", "tool_receipt_digest"}},
}

var backgroundAutomationCriticalRequirements = []eventRequirement{
	{eventType: event.TypeBundleManifest},
	{eventType: event.TypeAIRequestReceived, requiredFields: []string{"request_id", "actor_id_hash", "purpose_tag"}},
	{eventType: event.TypeAIPolicyDecision, requiredFields: []string{"policy_id", "policy_version", "decision", "decision_reason_codes", "subject_id_hash", "action_id"}},
	{eventType: event.TypeAIActionPrecommit, requiredFields: []string{"action_id", "action_type", "action_parameters_digest", "target_resource_id", "intended_effect"}},
	{eventType: event.TypeAIActionExecuted, requiredFields: []string{"action_id", "execution_outcome", "tool_receipt_digest"}},
	{eventType: event.TypeAIActionCommitted, requiredFields: []string{"action_id", "commit_outcome", "sink_receipt_digest"}},
}

var highImpactActionTypes = map[string]struct{}{
	"delete_records": {},
	"deploy_change":  {},
	"export_data":    {},
	"transfer_funds": {},
}

var registeredProfiles []Profile

// Profile defines the interface for a pluggable obligation profile evaluator.
type Profile interface {
	// ID returns the profile identifier.
	ID() string
	// Version returns the profile schema version.
	Version() int
	// WorkflowClass returns the human-readable workflow class name.
	WorkflowClass() string
	// Evaluate evaluates the bundle records against this profile's obligations.
	Evaluate(records []bundle.Record) ProfileResult
	// BlindSpots returns the verbatim blind-spot declaration strings.
	BlindSpots() []string
	// DefaultWeights returns the CAS weight vector for this profile.
	DefaultWeights() map[string]float64
}

// PrivilegedToolActionProfile evaluates privileged tool action workflows.
type PrivilegedToolActionProfile struct{}

// RAGAnswerProfile evaluates retrieval-augmented generation answer workflows.
type RAGAnswerProfile struct{}

// DataExportProfile evaluates data export workflows.
type DataExportProfile struct{}

// PolicyDecisionProfile evaluates policy decision workflows.
type PolicyDecisionProfile struct{}

// HumanOverrideProfile evaluates human override workflows.
type HumanOverrideProfile struct{}

// BackgroundAutomationProfile evaluates background automation workflows.
type BackgroundAutomationProfile struct{}

// Register adds a profile to the global registry.
func Register(p Profile) {
	if p == nil {
		return
	}
	registeredProfiles = append(registeredProfiles, p)
}

// AllProfiles returns a copy of all registered profiles.
func AllProfiles() []Profile {
	profiles := make([]Profile, len(registeredProfiles))
	copy(profiles, registeredProfiles)
	return profiles
}

// ProfileByID returns the profile with the given ID, or nil.
func ProfileByID(id string) Profile {
	for _, profile := range registeredProfiles {
		if profile.ID() == id {
			return profile
		}
	}
	return nil
}

// ComputeCAS computes the Completeness Assurance Score for the given
// sub-scores and weights.
func ComputeCAS(
	subScores map[string]float64,
	weights map[string]float64,
	integrityValid bool,
) CASResult {
	copiedScores := copyFloatMap(subScores)
	copiedWeights := copyFloatMap(weights)
	if !integrityValid {
		return CASResult{
			Overall:      0,
			Grade:        "Insufficient",
			SubScores:    copiedScores,
			WeightVector: copiedWeights,
		}
	}

	var total float64
	for key, weight := range copiedWeights {
		total += weight * copiedScores[key]
	}
	return CASResult{
		Overall:      total,
		Grade:        gradeFromScore(total),
		SubScores:    copiedScores,
		WeightVector: copiedWeights,
	}
}

func gradeFromScore(score float64) string {
	switch {
	case score >= 0.85:
		return "High"
	case score >= 0.60:
		return "Medium"
	case score >= 0.30:
		return "Low"
	default:
		return "Insufficient"
	}
}

// dataMap safely extracts Event.Data as map[string]any.
// Returns nil if Data is not a map type.
func dataMap(data interface{}) map[string]any {
	if m, ok := data.(map[string]any); ok {
		return m
	}
	return nil
}

// hasField reports whether the map contains a non-empty string or non-nil
// value for the given key.
func hasField(m map[string]any, key string) bool {
	value, ok := m[key]
	if !ok {
		return false
	}
	if s, ok := value.(string); ok {
		return s != ""
	}
	return value != nil
}

// ID returns the profile identifier.
func (p *PrivilegedToolActionProfile) ID() string { return profileIDPrivilegedToolAction }

// Version returns the profile schema version.
func (p *PrivilegedToolActionProfile) Version() int { return 1 }

// WorkflowClass returns the workflow class name.
func (p *PrivilegedToolActionProfile) WorkflowClass() string { return "privileged_tool_action" }

// BlindSpots returns the profile blind spots.
func (p *PrivilegedToolActionProfile) BlindSpots() []string {
	return []string{
		"If an operator bypasses the gate, this profile cannot detect that bypass without external reconciliation.",
		"Tool provider internal processing is not attested unless tool receipts are cryptographically verifiable.",
	}
}

// DefaultWeights returns the profile weight vector.
func (p *PrivilegedToolActionProfile) DefaultWeights() map[string]float64 {
	return map[string]float64{
		"EC": 0.15,
		"FC": 0.10,
		"RC": 0.15,
		"TC": 0.10,
		"SC": 0.10,
		"XC": 0.15,
		"AC": 0.10,
		"GC": 0.15,
	}
}

// Evaluate evaluates bundle records against the privileged tool action profile.
func (p *PrivilegedToolActionProfile) Evaluate(records []bundle.Record) ProfileResult {
	result := ProfileResult{
		ProfileID:          p.ID(),
		Version:            p.Version(),
		WorkflowClass:      p.WorkflowClass(),
		CriticalFailures:   []CriticalFailure{},
		RequiredWarnings:   []string{},
		InformationalNotes: []string{},
	}

	recordsByType := indexRecordsByType(records)
	for _, requirement := range privilegedCriticalRequirements {
		evaluateCriticalRequirement(&result, recordsByType[requirement.eventType], requirement)
	}

	if requiresHumanApproval(recordsByType[event.TypeAIActionPrecommit]) {
		checkOptionalRequirement(
			&result,
			recordsByType[event.TypeAIHumanApproval],
			eventRequirement{
				eventType:      event.TypeAIHumanApproval,
				requiredFields: []string{"approval_id", "approver_id_hash", "approval_outcome", "justification_digest", "action_id"},
			},
			fmt.Sprintf("%s required for high-impact action types", event.TypeAIHumanApproval),
		)
	}

	precommitByAction := indexByField(recordsByType[event.TypeAIActionPrecommit], "action_id")
	policyByAction := indexByField(recordsByType[event.TypeAIPolicyDecision], "action_id")
	executedByAction := indexByField(recordsByType[event.TypeAIActionExecuted], "action_id")

	if len(recordsByType[event.TypeAIActionCommitted]) > 0 && len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
		!allRecordsBound(recordsByType[event.TypeAIActionCommitted], "action_id", precommitByAction) {
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind: "relation_violation",
			Detail: fmt.Sprintf(
				"commit_requires_precommit: %s action_id does not match %s",
				event.TypeAIActionCommitted,
				event.TypeAIActionPrecommit,
			),
		})
	}

	if len(recordsByType[event.TypeAIPolicyDecision]) > 0 && len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
		!allRecordsBound(recordsByType[event.TypeAIPolicyDecision], "action_id", precommitByAction) {
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind: "relation_violation",
			Detail: fmt.Sprintf(
				"policy_binds_action: %s action_id does not match %s",
				event.TypeAIPolicyDecision,
				event.TypeAIActionPrecommit,
			),
		})
	}

	if len(recordsByType[event.TypeAIActionExecuted]) > 0 && !allExecutedAuthorised(recordsByType[event.TypeAIActionExecuted], policyByAction) {
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind: "relation_violation",
			Detail: fmt.Sprintf(
				"execution_after_authorization: %s decision is not allow for executed action_id",
				event.TypeAIPolicyDecision,
			),
		})
	}

	for _, warning := range boundedExecutionWindowWarnings(precommitByAction, executedByAction) {
		result.RequiredWarnings = append(result.RequiredWarnings, warning)
	}

	if len(recordsByType[event.TypeBundleAnchor]) == 0 {
		result.RequiredWarnings = append(result.RequiredWarnings,
			fmt.Sprintf("rfc3161_anchor_on_commit: no %s event present in bundle", event.TypeBundleAnchor))
	}

	result.InformationalNotes = append(result.InformationalNotes,
		"db_reconciliation: no database reconciliation evidence present (v1)",
		"source_signature_policy_engine: no policy engine signature present (v1)",
	)

	finaliseProfileResult(&result)
	return result
}

// ID returns the profile identifier.
func (p *DataExportProfile) ID() string { return profileIDDataExport }

// Version returns the profile schema version.
func (p *DataExportProfile) Version() int { return 1 }

// WorkflowClass returns the workflow class name.
func (p *DataExportProfile) WorkflowClass() string { return "data_export" }

// BlindSpots returns the profile blind spots.
func (p *DataExportProfile) BlindSpots() []string {
	return []string{
		"Does not verify downstream data handling or recipient controls after export is committed.",
		"Classification labels on exported records are presence-checked only; correctness of labelling is not attested.",
	}
}

// DefaultWeights returns the profile weight vector.
func (p *DataExportProfile) DefaultWeights() map[string]float64 {
	return map[string]float64{
		"EC": 0.20,
		"FC": 0.15,
		"RC": 0.20,
		"TC": 0.05,
		"SC": 0.10,
		"XC": 0.10,
		"AC": 0.10,
		"GC": 0.10,
	}
}

// Evaluate evaluates bundle records against the data export profile.
func (p *DataExportProfile) Evaluate(records []bundle.Record) ProfileResult {
	result := ProfileResult{
		ProfileID:          p.ID(),
		Version:            p.Version(),
		WorkflowClass:      p.WorkflowClass(),
		CriticalFailures:   []CriticalFailure{},
		RequiredWarnings:   []string{},
		InformationalNotes: []string{},
	}

	recordsByType := indexRecordsByType(records)
	for _, requirement := range dataExportCriticalRequirements {
		evaluateCriticalRequirement(&result, recordsByType[requirement.eventType], requirement)
	}

	checkOptionalRequirement(
		&result,
		recordsByType[event.TypeAIHumanApproval],
		eventRequirement{
			eventType:      event.TypeAIHumanApproval,
			requiredFields: []string{"approval_id", "approver_id_hash", "approval_outcome", "justification_digest", "action_id"},
		},
		fmt.Sprintf("%s required for data export workflows", event.TypeAIHumanApproval),
	)

	precommitByAction := indexByField(recordsByType[event.TypeAIActionPrecommit], "action_id")
	policyByAction := indexByField(recordsByType[event.TypeAIPolicyDecision], "action_id")
	executedByAction := indexByField(recordsByType[event.TypeAIActionExecuted], "action_id")

	if len(recordsByType[event.TypeAIActionCommitted]) > 0 && len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
		!allRecordsBound(recordsByType[event.TypeAIActionCommitted], "action_id", precommitByAction) {
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind: "relation_violation",
			Detail: fmt.Sprintf(
				"commit_requires_precommit: %s action_id does not match %s",
				event.TypeAIActionCommitted,
				event.TypeAIActionPrecommit,
			),
		})
	}

	if len(recordsByType[event.TypeAIPolicyDecision]) > 0 && len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
		!allRecordsBound(recordsByType[event.TypeAIPolicyDecision], "action_id", precommitByAction) {
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind: "relation_violation",
			Detail: fmt.Sprintf(
				"policy_binds_action: %s action_id does not match %s",
				event.TypeAIPolicyDecision,
				event.TypeAIActionPrecommit,
			),
		})
	}

	if len(recordsByType[event.TypeAIActionExecuted]) > 0 && !allExecutedAuthorised(recordsByType[event.TypeAIActionExecuted], policyByAction) {
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind: "relation_violation",
			Detail: fmt.Sprintf(
				"execution_after_authorization: %s decision is not allow for executed action_id",
				event.TypeAIPolicyDecision,
			),
		})
	}

	for _, warning := range boundedExecutionWindowWarnings(precommitByAction, executedByAction) {
		result.RequiredWarnings = append(result.RequiredWarnings, warning)
	}

	if len(recordsByType[event.TypeBundleAnchor]) == 0 {
		result.RequiredWarnings = append(result.RequiredWarnings,
			fmt.Sprintf("rfc3161_anchor_on_commit: no %s event present in bundle", event.TypeBundleAnchor))
	}

	result.InformationalNotes = append(result.InformationalNotes,
		"classification_label_check: label presence-only in v1",
		"recipient_controls: downstream recipient controls not attested (v1)",
	)

	finaliseProfileResult(&result)
	return result
}

// ID returns the profile identifier.
func (p *PolicyDecisionProfile) ID() string { return profileIDPolicyDecision }

// Version returns the profile schema version.
func (p *PolicyDecisionProfile) Version() int { return 1 }

// WorkflowClass returns the workflow class name.
func (p *PolicyDecisionProfile) WorkflowClass() string { return "policy_decision" }

// BlindSpots returns the profile blind spots.
func (p *PolicyDecisionProfile) BlindSpots() []string {
	return []string{
		"Does not verify the correctness or completeness of the policy engine's rule set; only that a decision event was recorded with required fields.",
		"Policy engine internal state and rule evaluation logic are not attested.",
	}
}

// DefaultWeights returns the profile weight vector.
func (p *PolicyDecisionProfile) DefaultWeights() map[string]float64 {
	return map[string]float64{
		"EC": 0.20,
		"FC": 0.20,
		"RC": 0.15,
		"TC": 0.10,
		"SC": 0.10,
		"XC": 0.10,
		"AC": 0.05,
		"GC": 0.10,
	}
}

// Evaluate evaluates bundle records against the policy decision profile.
func (p *PolicyDecisionProfile) Evaluate(records []bundle.Record) ProfileResult {
	result := ProfileResult{
		ProfileID:          p.ID(),
		Version:            p.Version(),
		WorkflowClass:      p.WorkflowClass(),
		CriticalFailures:   []CriticalFailure{},
		RequiredWarnings:   []string{},
		InformationalNotes: []string{},
	}

	recordsByType := indexRecordsByType(records)
	for _, requirement := range policyDecisionCriticalRequirements {
		evaluateCriticalRequirement(&result, recordsByType[requirement.eventType], requirement)
	}

	checkOptionalRequirement(
		&result,
		recordsByType[event.TypeAIActionPrecommit],
		eventRequirement{
			eventType:      event.TypeAIActionPrecommit,
			requiredFields: []string{"action_id", "action_type", "action_parameters_digest"},
		},
		fmt.Sprintf("%s recommended to bind policy to a pending action", event.TypeAIActionPrecommit),
	)

	precommitByAction := indexByField(recordsByType[event.TypeAIActionPrecommit], "action_id")
	if len(recordsByType[event.TypeAIPolicyDecision]) > 0 && len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
		!allRecordsBound(recordsByType[event.TypeAIPolicyDecision], "action_id", precommitByAction) {
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind: "relation_violation",
			Detail: fmt.Sprintf(
				"policy_binds_action: %s action_id does not match %s",
				event.TypeAIPolicyDecision,
				event.TypeAIActionPrecommit,
			),
		})
	}

	if len(recordsByType[event.TypeBundleAnchor]) == 0 {
		result.RequiredWarnings = append(result.RequiredWarnings,
			fmt.Sprintf("rfc3161_anchor_on_commit: no %s event present in bundle", event.TypeBundleAnchor))
	}

	result.InformationalNotes = append(result.InformationalNotes,
		"policy_engine_state: policy engine internal state not attested (v1)",
		"rule_set_completeness: rule set correctness not verified (v1)",
	)

	finaliseProfileResult(&result)
	return result
}

// ID returns the profile identifier.
func (p *HumanOverrideProfile) ID() string { return profileIDHumanOverride }

// Version returns the profile schema version.
func (p *HumanOverrideProfile) Version() int { return 1 }

// WorkflowClass returns the workflow class name.
func (p *HumanOverrideProfile) WorkflowClass() string { return "human_override" }

// BlindSpots returns the profile blind spots.
func (p *HumanOverrideProfile) BlindSpots() []string {
	return []string{
		"Cannot verify that the recorded approver identity matches the individual who physically performed the approval action.",
		"Justification digest binds the recorded justification text only; content quality is not assessed.",
	}
}

// DefaultWeights returns the profile weight vector.
func (p *HumanOverrideProfile) DefaultWeights() map[string]float64 {
	return map[string]float64{
		"EC": 0.15,
		"FC": 0.15,
		"RC": 0.20,
		"TC": 0.10,
		"SC": 0.10,
		"XC": 0.15,
		"AC": 0.10,
		"GC": 0.05,
	}
}

// Evaluate evaluates bundle records against the human override profile.
func (p *HumanOverrideProfile) Evaluate(records []bundle.Record) ProfileResult {
	result := ProfileResult{
		ProfileID:          p.ID(),
		Version:            p.Version(),
		WorkflowClass:      p.WorkflowClass(),
		CriticalFailures:   []CriticalFailure{},
		RequiredWarnings:   []string{},
		InformationalNotes: []string{},
	}

	recordsByType := indexRecordsByType(records)
	for _, requirement := range humanOverrideCriticalRequirements {
		evaluateCriticalRequirement(&result, recordsByType[requirement.eventType], requirement)
	}

	precommitByAction := indexByField(recordsByType[event.TypeAIActionPrecommit], "action_id")
	approvalByAction := indexByField(recordsByType[event.TypeAIHumanApproval], "action_id")
	executedByAction := indexByField(recordsByType[event.TypeAIActionExecuted], "action_id")

	if len(recordsByType[event.TypeAIHumanApproval]) > 0 && len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
		!allRecordsBound(recordsByType[event.TypeAIHumanApproval], "action_id", precommitByAction) {
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind: "relation_violation",
			Detail: fmt.Sprintf(
				"approval_binds_action: %s action_id does not match %s",
				event.TypeAIHumanApproval,
				event.TypeAIActionPrecommit,
			),
		})
	}

	if len(recordsByType[event.TypeAIActionExecuted]) > 0 && !allExecutedApproved(recordsByType[event.TypeAIActionExecuted], approvalByAction) {
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind:   "relation_violation",
			Detail: "execution_after_approval: executed action_id has no approved ai.human.approval record",
		})
	}

	for _, warning := range boundedExecutionWindowWarnings(precommitByAction, executedByAction) {
		result.RequiredWarnings = append(result.RequiredWarnings, warning)
	}

	if len(recordsByType[event.TypeBundleAnchor]) == 0 {
		result.RequiredWarnings = append(result.RequiredWarnings,
			fmt.Sprintf("rfc3161_anchor_on_commit: no %s event present in bundle", event.TypeBundleAnchor))
	}

	checkOptionalRequirement(
		&result,
		recordsByType[event.TypeAIActionCommitted],
		eventRequirement{
			eventType:      event.TypeAIActionCommitted,
			requiredFields: []string{"action_id", "commit_outcome", "sink_receipt_digest"},
		},
		fmt.Sprintf("%s recommended to confirm committed outcome", event.TypeAIActionCommitted),
	)

	result.InformationalNotes = append(result.InformationalNotes,
		"approver_identity: physical approver identity not attested (v1)",
		"justification_quality: justification content quality not assessed (v1)",
	)

	finaliseProfileResult(&result)
	return result
}

// ID returns the profile identifier.
func (p *BackgroundAutomationProfile) ID() string { return profileIDBackgroundAutomation }

// Version returns the profile schema version.
func (p *BackgroundAutomationProfile) Version() int { return 1 }

// WorkflowClass returns the workflow class name.
func (p *BackgroundAutomationProfile) WorkflowClass() string { return "background_automation" }

// BlindSpots returns the profile blind spots.
func (p *BackgroundAutomationProfile) BlindSpots() []string {
	return []string{
		"Scheduling system integrity is not attested; a compromised scheduler could trigger executions without a corresponding schedule record.",
		"Does not verify that the automated action's actual effect matches the declared intended_effect beyond what tool receipts provide.",
	}
}

// DefaultWeights returns the profile weight vector.
func (p *BackgroundAutomationProfile) DefaultWeights() map[string]float64 {
	return map[string]float64{
		"EC": 0.20,
		"FC": 0.15,
		"RC": 0.15,
		"TC": 0.10,
		"SC": 0.10,
		"XC": 0.10,
		"AC": 0.10,
		"GC": 0.10,
	}
}

// Evaluate evaluates bundle records against the background automation profile.
func (p *BackgroundAutomationProfile) Evaluate(records []bundle.Record) ProfileResult {
	result := ProfileResult{
		ProfileID:          p.ID(),
		Version:            p.Version(),
		WorkflowClass:      p.WorkflowClass(),
		CriticalFailures:   []CriticalFailure{},
		RequiredWarnings:   []string{},
		InformationalNotes: []string{},
	}

	recordsByType := indexRecordsByType(records)
	for _, requirement := range backgroundAutomationCriticalRequirements {
		evaluateCriticalRequirement(&result, recordsByType[requirement.eventType], requirement)
	}

	precommitByAction := indexByField(recordsByType[event.TypeAIActionPrecommit], "action_id")
	policyByAction := indexByField(recordsByType[event.TypeAIPolicyDecision], "action_id")
	executedByAction := indexByField(recordsByType[event.TypeAIActionExecuted], "action_id")

	if len(recordsByType[event.TypeAIActionCommitted]) > 0 && len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
		!allRecordsBound(recordsByType[event.TypeAIActionCommitted], "action_id", precommitByAction) {
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind: "relation_violation",
			Detail: fmt.Sprintf(
				"commit_requires_precommit: %s action_id does not match %s",
				event.TypeAIActionCommitted,
				event.TypeAIActionPrecommit,
			),
		})
	}

	if len(recordsByType[event.TypeAIPolicyDecision]) > 0 && len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
		!allRecordsBound(recordsByType[event.TypeAIPolicyDecision], "action_id", precommitByAction) {
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind: "relation_violation",
			Detail: fmt.Sprintf(
				"policy_binds_action: %s action_id does not match %s",
				event.TypeAIPolicyDecision,
				event.TypeAIActionPrecommit,
			),
		})
	}

	if len(recordsByType[event.TypeAIActionExecuted]) > 0 && !allExecutedAuthorised(recordsByType[event.TypeAIActionExecuted], policyByAction) {
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind: "relation_violation",
			Detail: fmt.Sprintf(
				"execution_after_authorization: %s decision is not allow for executed action_id",
				event.TypeAIPolicyDecision,
			),
		})
	}

	for _, warning := range boundedExecutionWindowWarnings(precommitByAction, executedByAction) {
		result.RequiredWarnings = append(result.RequiredWarnings, warning)
	}

	if len(recordsByType[event.TypeBundleAnchor]) == 0 {
		result.RequiredWarnings = append(result.RequiredWarnings,
			fmt.Sprintf("rfc3161_anchor_on_commit: no %s event present in bundle", event.TypeBundleAnchor))
	}

	result.InformationalNotes = append(result.InformationalNotes,
		"scheduler_integrity: scheduling system integrity not attested (v1)",
		"intended_effect_verification: actual effect vs declared intended_effect not verified beyond tool receipts (v1)",
	)

	finaliseProfileResult(&result)
	return result
}

// ID returns the profile identifier.
func (p *RAGAnswerProfile) ID() string { return profileIDRAGAnswer }

// Version returns the profile schema version.
func (p *RAGAnswerProfile) Version() int { return 1 }

// WorkflowClass returns the workflow class name.
func (p *RAGAnswerProfile) WorkflowClass() string { return "rag_answer" }

// BlindSpots returns the profile blind spots.
func (p *RAGAnswerProfile) BlindSpots() []string {
	return []string{
		"Does not prove retrieval completeness beyond recorded corpus/version.",
		"Does not prove the model produced output exactly as provider executed internally; only that recorded invocation/output digests match the bundle.",
	}
}

// DefaultWeights returns the profile weight vector.
func (p *RAGAnswerProfile) DefaultWeights() map[string]float64 {
	return map[string]float64{
		"EC": 0.20,
		"FC": 0.15,
		"RC": 0.20,
		"TC": 0.10,
		"SC": 0.10,
		"XC": 0.10,
		"AC": 0.05,
		"GC": 0.10,
	}
}

// Evaluate evaluates bundle records against the RAG answer profile.
func (p *RAGAnswerProfile) Evaluate(records []bundle.Record) ProfileResult {
	result := ProfileResult{
		ProfileID:          p.ID(),
		Version:            p.Version(),
		WorkflowClass:      p.WorkflowClass(),
		CriticalFailures:   []CriticalFailure{},
		RequiredWarnings:   []string{},
		InformationalNotes: []string{},
	}

	recordsByType := indexRecordsByType(records)
	for _, requirement := range ragCriticalRequirements {
		evaluateCriticalRequirement(&result, recordsByType[requirement.eventType], requirement)
	}

	checkOptionalRequirement(
		&result,
		recordsByType[event.TypeAIPolicyDecision],
		eventRequirement{
			eventType:      event.TypeAIPolicyDecision,
			requiredFields: []string{"policy_id", "policy_version", "decision", "decision_reason_codes"},
		},
		fmt.Sprintf("%s recommended for recorded authorisation context", event.TypeAIPolicyDecision),
	)
	checkOptionalRequirement(
		&result,
		recordsByType[event.TypeAIRetrievalExecuted],
		eventRequirement{
			eventType:      event.TypeAIRetrievalExecuted,
			requiredFields: []string{"retrieval_query_hash", "retrieval_corpus_id", "retrieval_corpus_version", "top_k", "result_set_digest"},
		},
		fmt.Sprintf("%s recommended for RAG answer provenance", event.TypeAIRetrievalExecuted),
	)
	checkOptionalRequirement(
		&result,
		recordsByType[event.TypeAIResponseSent],
		eventRequirement{
			eventType:      event.TypeAIResponseSent,
			requiredFields: []string{"request_id", "output_digest"},
		},
		fmt.Sprintf("%s recommended for response delivery evidence", event.TypeAIResponseSent),
	)

	requests := recordsByType[event.TypeAIRequestReceived]
	responses := recordsByType[event.TypeAIResponseSent]
	if len(requests) > 0 && len(responses) > 0 {
		requestID := fieldString(requests[0], "request_id")
		responseID := fieldString(responses[0], "request_id")
		if requestID == "" || responseID == "" || requestID != responseID {
			result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
				Kind: "relation_violation",
				Detail: fmt.Sprintf(
					"request_to_response: %s request_id does not match %s",
					event.TypeAIResponseSent,
					event.TypeAIRequestReceived,
				),
			})
		}
	}

	retrievals := recordsByType[event.TypeAIRetrievalExecuted]
	modelInvocations := recordsByType[event.TypeAIModelInvoked]
	if len(retrievals) == 0 || len(modelInvocations) == 0 || fieldString(retrievals[0], "result_set_digest") == "" {
		result.RequiredWarnings = append(result.RequiredWarnings,
			"retrieval_bound_to_prompt: retrieval evidence missing or result_set_digest empty")
	}

	if warning := beforeModelWarning("policy_before_model", recordsByType[event.TypeAIPolicyDecision], modelInvocations); warning != "" {
		result.RequiredWarnings = append(result.RequiredWarnings, warning)
	}
	if warning := beforeModelWarning("retrieval_before_model", retrievals, modelInvocations); warning != "" {
		result.RequiredWarnings = append(result.RequiredWarnings, warning)
	}

	result.InformationalNotes = append(result.InformationalNotes,
		"gating_evidence: not applicable to rag_answer; partial credit applied",
		"retrieval_bound_to_prompt: digest binding is presence-only in v1",
		"source_signatures: no source signature evidence present (v1)",
	)

	finaliseProfileResult(&result)
	return result
}

func subScoresForProfile(profile Profile, records []bundle.Record, anchorPresent bool) map[string]float64 {
	switch profile.ID() {
	case profileIDPrivilegedToolAction:
		return privilegedToolActionSubScores(records, anchorPresent)
	case profileIDRAGAnswer:
		return ragAnswerSubScores(records, anchorPresent)
	case profileIDDataExport:
		return dataExportSubScores(records, anchorPresent)
	case profileIDPolicyDecision:
		return policyDecisionSubScores(records, anchorPresent)
	case profileIDHumanOverride:
		return humanOverrideSubScores(records, anchorPresent)
	case profileIDBackgroundAutomation:
		return backgroundAutomationSubScores(records, anchorPresent)
	default:
		return map[string]float64{
			"EC": 0,
			"FC": 0,
			"RC": 0,
			"TC": 0,
			"SC": 0,
			"XC": 0,
			"AC": 0,
			"GC": 0,
		}
	}
}

func policyDecisionSubScores(records []bundle.Record, anchorPresent bool) map[string]float64 {
	recordsByType := indexRecordsByType(records)
	precommitByAction := indexByField(recordsByType[event.TypeAIActionPrecommit], "action_id")
	policyByAction := indexByField(recordsByType[event.TypeAIPolicyDecision], "action_id")

	rc := boolScore(
		len(recordsByType[event.TypeAIPolicyDecision]) > 0 &&
			len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
			allRecordsBound(recordsByType[event.TypeAIPolicyDecision], "action_id", precommitByAction),
	)

	tc := temporalBeforeScore(
		recordsByType[event.TypeAIPolicyDecision],
		recordsByType[event.TypeAIActionExecuted],
	)

	gc := boolScore(
		len(recordsByType[event.TypeAIPolicyDecision]) > 0 &&
			len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
			allRecordsBound(recordsByType[event.TypeAIPolicyDecision], "action_id", precommitByAction) &&
			allRecordsBound(recordsByType[event.TypeAIPolicyDecision], "action_id", policyByAction),
	)

	external := boolScore(anchorPresent)

	return map[string]float64{
		"EC": eventCoverage(recordsByType, policyDecisionCriticalRequirements),
		"FC": fieldCompleteness(recordsByType, policyDecisionCriticalRequirements),
		"RC": rc,
		"TC": tc,
		"SC": computeSC(&bundle.Bundle{Records: records}, profileIDPolicyDecision),
		"XC": external,
		"AC": external,
		"GC": gc,
	}
}

func humanOverrideSubScores(records []bundle.Record, anchorPresent bool) map[string]float64 {
	recordsByType := indexRecordsByType(records)
	precommitByAction := indexByField(recordsByType[event.TypeAIActionPrecommit], "action_id")
	approvalByAction := indexByField(recordsByType[event.TypeAIHumanApproval], "action_id")
	executedByAction := indexByField(recordsByType[event.TypeAIActionExecuted], "action_id")

	approvalBound := len(recordsByType[event.TypeAIHumanApproval]) > 0 &&
		len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
		allRecordsBound(recordsByType[event.TypeAIHumanApproval], "action_id", precommitByAction)

	executionApproved := allExecutedApproved(recordsByType[event.TypeAIActionExecuted], approvalByAction)
	rc := averageScores(boolScore(approvalBound), boolScore(executionApproved))

	tc := 1.0
	if len(boundedExecutionWindowWarnings(precommitByAction, executedByAction)) > 0 {
		tc = 0.7
	}

	gc := boolScore(
		len(recordsByType[event.TypeAIHumanApproval]) > 0 &&
			len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
			len(recordsByType[event.TypeAIActionExecuted]) > 0 &&
			allRecordsBound(recordsByType[event.TypeAIHumanApproval], "action_id", precommitByAction) &&
			allRecordsBound(recordsByType[event.TypeAIActionExecuted], "action_id", precommitByAction),
	)

	external := boolScore(anchorPresent)

	return map[string]float64{
		"EC": eventCoverage(recordsByType, humanOverrideCriticalRequirements),
		"FC": fieldCompleteness(recordsByType, humanOverrideCriticalRequirements),
		"RC": rc,
		"TC": tc,
		"SC": computeSC(&bundle.Bundle{Records: records}, profileIDHumanOverride),
		"XC": external,
		"AC": external,
		"GC": gc,
	}
}

func backgroundAutomationSubScores(records []bundle.Record, anchorPresent bool) map[string]float64 {
	recordsByType := indexRecordsByType(records)
	precommitByAction := indexByField(recordsByType[event.TypeAIActionPrecommit], "action_id")
	policyByAction := indexByField(recordsByType[event.TypeAIPolicyDecision], "action_id")
	executedByAction := indexByField(recordsByType[event.TypeAIActionExecuted], "action_id")
	committedByAction := indexByField(recordsByType[event.TypeAIActionCommitted], "action_id")

	rc := averageScores(
		boolScore(len(recordsByType[event.TypeAIActionCommitted]) > 0 &&
			len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
			allRecordsBound(recordsByType[event.TypeAIActionCommitted], "action_id", precommitByAction)),
		boolScore(len(recordsByType[event.TypeAIPolicyDecision]) > 0 &&
			len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
			allRecordsBound(recordsByType[event.TypeAIPolicyDecision], "action_id", precommitByAction)),
		boolScore(len(recordsByType[event.TypeAIActionExecuted]) > 0 &&
			len(recordsByType[event.TypeAIPolicyDecision]) > 0 &&
			allExecutedAuthorised(recordsByType[event.TypeAIActionExecuted], policyByAction)),
	)

	tc := 1.0
	if len(boundedExecutionWindowWarnings(precommitByAction, executedByAction)) > 0 {
		tc = 0.7
	}

	gc := boolScore(
		len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
			len(recordsByType[event.TypeAIActionExecuted]) > 0 &&
			len(recordsByType[event.TypeAIActionCommitted]) > 0 &&
			allRecordsBound(recordsByType[event.TypeAIActionExecuted], "action_id", precommitByAction) &&
			allRecordsBound(recordsByType[event.TypeAIActionCommitted], "action_id", precommitByAction) &&
			allRecordsBound(recordsByType[event.TypeAIActionCommitted], "action_id", executedByAction) &&
			allRecordsBound(recordsByType[event.TypeAIActionExecuted], "action_id", committedByAction),
	)

	external := boolScore(anchorPresent)

	return map[string]float64{
		"EC": eventCoverage(recordsByType, backgroundAutomationCriticalRequirements),
		"FC": fieldCompleteness(recordsByType, backgroundAutomationCriticalRequirements),
		"RC": rc,
		"TC": tc,
		"SC": computeSC(&bundle.Bundle{Records: records}, profileIDBackgroundAutomation),
		"XC": external,
		"AC": external,
		"GC": gc,
	}
}

func dataExportSubScores(records []bundle.Record, anchorPresent bool) map[string]float64 {
	recordsByType := indexRecordsByType(records)
	precommitByAction := indexByField(recordsByType[event.TypeAIActionPrecommit], "action_id")
	policyByAction := indexByField(recordsByType[event.TypeAIPolicyDecision], "action_id")
	executedByAction := indexByField(recordsByType[event.TypeAIActionExecuted], "action_id")
	committedByAction := indexByField(recordsByType[event.TypeAIActionCommitted], "action_id")

	rc := averageScores(
		boolScore(len(recordsByType[event.TypeAIActionCommitted]) > 0 &&
			len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
			allRecordsBound(recordsByType[event.TypeAIActionCommitted], "action_id", precommitByAction)),
		boolScore(len(recordsByType[event.TypeAIPolicyDecision]) > 0 &&
			len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
			allRecordsBound(recordsByType[event.TypeAIPolicyDecision], "action_id", precommitByAction)),
		boolScore(len(recordsByType[event.TypeAIActionExecuted]) > 0 &&
			len(recordsByType[event.TypeAIPolicyDecision]) > 0 &&
			allExecutedAuthorised(recordsByType[event.TypeAIActionExecuted], policyByAction)),
	)

	tc := 1.0
	if len(boundedExecutionWindowWarnings(precommitByAction, executedByAction)) > 0 {
		tc = 0.7
	}

	gc := boolScore(
		len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
			len(recordsByType[event.TypeAIActionExecuted]) > 0 &&
			len(recordsByType[event.TypeAIActionCommitted]) > 0 &&
			allRecordsBound(recordsByType[event.TypeAIActionExecuted], "action_id", precommitByAction) &&
			allRecordsBound(recordsByType[event.TypeAIActionCommitted], "action_id", precommitByAction) &&
			allRecordsBound(recordsByType[event.TypeAIActionCommitted], "action_id", executedByAction) &&
			allRecordsBound(recordsByType[event.TypeAIActionExecuted], "action_id", committedByAction),
	)

	external := boolScore(anchorPresent)

	return map[string]float64{
		"EC": eventCoverage(recordsByType, dataExportCriticalRequirements),
		"FC": fieldCompleteness(recordsByType, dataExportCriticalRequirements),
		"RC": rc,
		"TC": tc,
		"SC": computeSC(&bundle.Bundle{Records: records}, profileIDDataExport),
		"XC": external,
		"AC": external,
		"GC": gc,
	}
}

func privilegedToolActionSubScores(records []bundle.Record, anchorPresent bool) map[string]float64 {
	recordsByType := indexRecordsByType(records)
	precommitByAction := indexByField(recordsByType[event.TypeAIActionPrecommit], "action_id")
	policyByAction := indexByField(recordsByType[event.TypeAIPolicyDecision], "action_id")
	executedByAction := indexByField(recordsByType[event.TypeAIActionExecuted], "action_id")
	committedByAction := indexByField(recordsByType[event.TypeAIActionCommitted], "action_id")

	rc := averageScores(
		boolScore(len(recordsByType[event.TypeAIActionCommitted]) > 0 &&
			len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
			allRecordsBound(recordsByType[event.TypeAIActionCommitted], "action_id", precommitByAction)),
		boolScore(len(recordsByType[event.TypeAIPolicyDecision]) > 0 &&
			len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
			allRecordsBound(recordsByType[event.TypeAIPolicyDecision], "action_id", precommitByAction)),
		boolScore(len(recordsByType[event.TypeAIActionExecuted]) > 0 &&
			len(recordsByType[event.TypeAIPolicyDecision]) > 0 &&
			allExecutedAuthorised(recordsByType[event.TypeAIActionExecuted], policyByAction)),
	)

	tc := 1.0
	if len(boundedExecutionWindowWarnings(precommitByAction, executedByAction)) > 0 {
		tc = 0.7
	}

	gc := boolScore(
		len(recordsByType[event.TypeAIActionPrecommit]) > 0 &&
			len(recordsByType[event.TypeAIActionExecuted]) > 0 &&
			len(recordsByType[event.TypeAIActionCommitted]) > 0 &&
			allRecordsBound(recordsByType[event.TypeAIActionExecuted], "action_id", precommitByAction) &&
			allRecordsBound(recordsByType[event.TypeAIActionCommitted], "action_id", precommitByAction) &&
			allRecordsBound(recordsByType[event.TypeAIActionCommitted], "action_id", executedByAction) &&
			allRecordsBound(recordsByType[event.TypeAIActionExecuted], "action_id", committedByAction),
	)

	external := 0.0
	if anchorPresent {
		external = 1.0
	}

	return map[string]float64{
		"EC": eventCoverage(recordsByType, privilegedCriticalRequirements),
		"FC": fieldCompleteness(recordsByType, privilegedCriticalRequirements),
		"RC": rc,
		"TC": tc,
		"SC": computeSC(&bundle.Bundle{Records: records}, profileIDPrivilegedToolAction),
		"XC": external,
		"AC": external,
		"GC": gc,
	}
}

func ragAnswerSubScores(records []bundle.Record, anchorPresent bool) map[string]float64 {
	recordsByType := indexRecordsByType(records)

	requestToResponse := 0.0
	requests := recordsByType[event.TypeAIRequestReceived]
	responses := recordsByType[event.TypeAIResponseSent]
	if len(requests) > 0 && len(responses) > 0 {
		if requestID := fieldString(requests[0], "request_id"); requestID != "" && requestID == fieldString(responses[0], "request_id") {
			requestToResponse = 1.0
		}
	}

	temporalScores := []float64{
		temporalBeforeScore(recordsByType[event.TypeAIPolicyDecision], recordsByType[event.TypeAIModelInvoked]),
		temporalBeforeScore(recordsByType[event.TypeAIRetrievalExecuted], recordsByType[event.TypeAIModelInvoked]),
	}

	external := 0.0
	if anchorPresent {
		external = 1.0
	}

	return map[string]float64{
		"EC": eventCoverage(recordsByType, ragCriticalRequirements),
		"FC": fieldCompleteness(recordsByType, ragCriticalRequirements),
		"RC": requestToResponse,
		"TC": averageScores(temporalScores...),
		"SC": computeSC(&bundle.Bundle{Records: records}, profileIDRAGAnswer),
		"XC": external,
		"AC": external,
		"GC": 0.3,
	}
}

func evaluateCriticalRequirement(result *ProfileResult, records []bundle.Record, requirement eventRequirement) {
	if len(records) == 0 {
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind:   "missing_event",
			Detail: fmt.Sprintf("%s missing", requirement.eventType),
		})
		return
	}
	if len(requirement.requiredFields) == 0 {
		return
	}

	for _, record := range records {
		data := dataMap(record.Event.Data)
		if data == nil {
			result.InformationalNotes = append(result.InformationalNotes,
				fmt.Sprintf("%s: event data is not a map[string]any; field checks skipped", requirement.eventType))
			result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
				Kind:   "missing_field",
				Detail: fmt.Sprintf("%s missing required fields: %s", requirement.eventType, strings.Join(requirement.requiredFields, ", ")),
			})
			continue
		}

		missing := missingFields(data, requirement.requiredFields)
		if len(missing) == 0 {
			continue
		}
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind:   "missing_field",
			Detail: fmt.Sprintf("%s missing required fields: %s", requirement.eventType, strings.Join(missing, ", ")),
		})
	}
}

func checkOptionalRequirement(result *ProfileResult, records []bundle.Record, requirement eventRequirement, missingMessage string) {
	if len(records) == 0 {
		result.RequiredWarnings = append(result.RequiredWarnings,
			fmt.Sprintf("missing_event: %s", missingMessage))
		return
	}
	if len(requirement.requiredFields) == 0 {
		return
	}

	for _, record := range records {
		data := dataMap(record.Event.Data)
		if data == nil {
			result.InformationalNotes = append(result.InformationalNotes,
				fmt.Sprintf("%s: event data is not a map[string]any; field checks skipped", requirement.eventType))
			result.RequiredWarnings = append(result.RequiredWarnings,
				fmt.Sprintf("missing_field: %s missing required fields: %s", requirement.eventType, strings.Join(requirement.requiredFields, ", ")))
			continue
		}

		missing := missingFields(data, requirement.requiredFields)
		if len(missing) == 0 {
			continue
		}
		result.RequiredWarnings = append(result.RequiredWarnings,
			fmt.Sprintf("missing_field: %s missing required fields: %s", requirement.eventType, strings.Join(missing, ", ")))
	}
}

func requiresHumanApproval(precommits []bundle.Record) bool {
	for _, record := range precommits {
		actionType := fieldString(record, "action_type")
		if _, ok := highImpactActionTypes[actionType]; ok {
			return true
		}
	}
	return false
}

func boundedExecutionWindowWarnings(precommitByAction, executedByAction map[string][]bundle.Record) []string {
	warnings := []string{}
	for actionID, executed := range executedByAction {
		precommits := precommitByAction[actionID]
		if len(precommits) == 0 || len(executed) == 0 {
			continue
		}

		precommitTime, precommitPresent, precommitErr := parseEventTimestamp(precommits[0])
		executedTime, executedPresent, executedErr := parseEventTimestamp(executed[0])
		if !precommitPresent || !executedPresent {
			continue
		}
		if precommitErr != nil || executedErr != nil {
			warnings = append(warnings,
				fmt.Sprintf("bounded_execution_window: action_id %s has invalid RFC 3339 timestamp", actionID))
			continue
		}

		window := executedTime.Sub(precommitTime)
		if window > 10*time.Minute {
			warnings = append(warnings,
				fmt.Sprintf("bounded_execution_window: action_id %s execution window %s exceeded 10m limit", actionID, formatDuration(window)))
		}
	}
	sort.Strings(warnings)
	return warnings
}

func beforeModelWarning(label string, beforeRecords, modelRecords []bundle.Record) string {
	if len(beforeRecords) == 0 || len(modelRecords) == 0 {
		return ""
	}

	beforeTime, beforePresent, beforeErr := parseEventTimestamp(beforeRecords[0])
	modelTime, modelPresent, modelErr := parseEventTimestamp(modelRecords[0])
	if !beforePresent || !modelPresent {
		return ""
	}
	if beforeErr != nil || modelErr != nil {
		return fmt.Sprintf("%s: invalid RFC 3339 timestamp detected", label)
	}
	if beforeTime.After(modelTime) {
		return fmt.Sprintf("%s: event timestamp occurs after ai.model.invoked", label)
	}
	return ""
}

func temporalBeforeScore(beforeRecords, modelRecords []bundle.Record) float64 {
	if len(beforeRecords) == 0 || len(modelRecords) == 0 {
		return 1.0
	}

	beforeTime, beforePresent, beforeErr := parseEventTimestamp(beforeRecords[0])
	modelTime, modelPresent, modelErr := parseEventTimestamp(modelRecords[0])
	if !beforePresent || !modelPresent {
		return 1.0
	}
	if beforeErr != nil || modelErr != nil || beforeTime.After(modelTime) {
		return 0.7
	}
	return 1.0
}

func parseEventTimestamp(record bundle.Record) (time.Time, bool, error) {
	raw := strings.TrimSpace(record.Event.Timestamp)
	if raw == "" {
		return time.Time{}, false, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, true, err
	}
	return parsed, true, nil
}

func formatDuration(d time.Duration) string {
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	return d.Round(time.Second).String()
}

func eventCoverage(recordsByType map[string][]bundle.Record, requirements []eventRequirement) float64 {
	if len(requirements) == 0 {
		return 0
	}
	present := 0
	for _, requirement := range requirements {
		if len(recordsByType[requirement.eventType]) > 0 {
			present++
		}
	}
	return float64(present) / float64(len(requirements))
}

func fieldCompleteness(recordsByType map[string][]bundle.Record, requirements []eventRequirement) float64 {
	var total float64
	var count int
	for _, requirement := range requirements {
		records := recordsByType[requirement.eventType]
		for _, record := range records {
			count++
			if len(requirement.requiredFields) == 0 {
				total += 1.0
				continue
			}

			data := dataMap(record.Event.Data)
			if data == nil {
				continue
			}

			present := 0
			for _, field := range requirement.requiredFields {
				if hasField(data, field) {
					present++
				}
			}
			total += float64(present) / float64(len(requirement.requiredFields))
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func indexRecordsByType(records []bundle.Record) map[string][]bundle.Record {
	index := make(map[string][]bundle.Record, len(records))
	for _, record := range records {
		index[record.Event.Type] = append(index[record.Event.Type], record)
	}
	return index
}

func indexByField(records []bundle.Record, field string) map[string][]bundle.Record {
	index := make(map[string][]bundle.Record, len(records))
	for _, record := range records {
		value := fieldString(record, field)
		if value == "" {
			continue
		}
		index[value] = append(index[value], record)
	}
	return index
}

func fieldString(record bundle.Record, field string) string {
	data := dataMap(record.Event.Data)
	if data == nil {
		return ""
	}
	value, ok := data[field].(string)
	if !ok {
		return ""
	}
	return value
}

func missingFields(data map[string]any, requiredFields []string) []string {
	missing := make([]string, 0, len(requiredFields))
	for _, field := range requiredFields {
		if !hasField(data, field) {
			missing = append(missing, field)
		}
	}
	return missing
}

func allRecordsBound(records []bundle.Record, field string, targetIndex map[string][]bundle.Record) bool {
	for _, record := range records {
		value := fieldString(record, field)
		if value == "" || len(targetIndex[value]) == 0 {
			return false
		}
	}
	return len(records) > 0
}

func allExecutedAuthorised(executedRecords []bundle.Record, policyByAction map[string][]bundle.Record) bool {
	if len(executedRecords) == 0 {
		return false
	}
	for _, record := range executedRecords {
		actionID := fieldString(record, "action_id")
		if actionID == "" {
			return false
		}
		policies := policyByAction[actionID]
		if len(policies) == 0 {
			return false
		}
		allowed := false
		for _, policy := range policies {
			if strings.EqualFold(fieldString(policy, "decision"), "allow") {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}

func allExecutedApproved(executedRecords []bundle.Record, approvalByAction map[string][]bundle.Record) bool {
	for _, record := range executedRecords {
		actionID := fieldString(record, "action_id")
		if actionID == "" {
			return false
		}
		approvals := approvalByAction[actionID]
		approved := false
		for _, approval := range approvals {
			if strings.EqualFold(fieldString(approval, "approval_outcome"), "approved") {
				approved = true
				break
			}
		}
		if !approved {
			return false
		}
	}
	return true
}

func averageScores(scores ...float64) float64 {
	if len(scores) == 0 {
		return 0
	}
	var total float64
	for _, score := range scores {
		total += score
	}
	return total / float64(len(scores))
}

func boolScore(ok bool) float64 {
	if ok {
		return 1.0
	}
	return 0.0
}

func copyFloatMap(src map[string]float64) map[string]float64 {
	dst := make(map[string]float64, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func finaliseProfileResult(result *ProfileResult) {
	result.RequiredWarnings = appendUniqueStrings(nil, result.RequiredWarnings...)
	result.InformationalNotes = appendUniqueStrings(nil, result.InformationalNotes...)
	sort.Strings(result.RequiredWarnings)
	sort.Strings(result.InformationalNotes)
	sort.Slice(result.CriticalFailures, func(i, j int) bool {
		if result.CriticalFailures[i].Kind == result.CriticalFailures[j].Kind {
			return result.CriticalFailures[i].Detail < result.CriticalFailures[j].Detail
		}
		return result.CriticalFailures[i].Kind < result.CriticalFailures[j].Kind
	})
	result.Pass = len(result.CriticalFailures) == 0
}

func init() {
	Register(&PrivilegedToolActionProfile{})
	Register(&RAGAnswerProfile{})
	Register(&DataExportProfile{})
	Register(&PolicyDecisionProfile{})
	Register(&HumanOverrideProfile{})
	Register(&BackgroundAutomationProfile{})
}
