package verify

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	profiledsl "github.com/pcguest/atb/internal/profiles"
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
	{eventType: event.TypeAIJobScheduled, requiredFields: []string{"job_id", "job_type", "trigger_source", "scheduled_by_id_hash"}},
	{eventType: event.TypeAIJobStarted, requiredFields: []string{"job_id", "worker_id_hash", "started_at"}},
	{eventType: event.TypeAIJobCompleted, requiredFields: []string{"job_id", "outcome", "completion_reason"}},
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

type subScoreBuilder func([]bundle.Record, AnchorVerifyResult) map[string]float64

var profileSubScoreBuilders = map[string]subScoreBuilder{
	profileIDPrivilegedToolAction: privilegedToolActionSubScores,
	profileIDRAGAnswer:            ragAnswerSubScores,
	profileIDDataExport:           dataExportSubScores,
	profileIDPolicyDecision:       policyDecisionSubScores,
	profileIDHumanOverride:        humanOverrideSubScores,
	profileIDBackgroundAutomation: backgroundAutomationSubScores,
}

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
func (p *PrivilegedToolActionProfile) Version() int { return loadSchema(p.ID()).Version }

// WorkflowClass returns the workflow class name.
func (p *PrivilegedToolActionProfile) WorkflowClass() string { return loadSchema(p.ID()).WorkflowClass }

// BlindSpots returns the profile blind spots.
func (p *PrivilegedToolActionProfile) BlindSpots() []string {
	return copyStringSlice(loadSchema(p.ID()).BlindSpots)
}

// DefaultWeights returns the profile weight vector.
func (p *PrivilegedToolActionProfile) DefaultWeights() map[string]float64 {
	return copyFloatMap(loadSchema(p.ID()).Weights)
}

// Evaluate evaluates bundle records against the privileged tool action profile.
func (p *PrivilegedToolActionProfile) Evaluate(records []bundle.Record) ProfileResult {
	return evaluateSchemaProfile(loadSchema(p.ID()), records)
}

// ID returns the profile identifier.
func (p *DataExportProfile) ID() string { return profileIDDataExport }

// Version returns the profile schema version.
func (p *DataExportProfile) Version() int { return loadSchema(p.ID()).Version }

// WorkflowClass returns the workflow class name.
func (p *DataExportProfile) WorkflowClass() string { return loadSchema(p.ID()).WorkflowClass }

// BlindSpots returns the profile blind spots.
func (p *DataExportProfile) BlindSpots() []string {
	return copyStringSlice(loadSchema(p.ID()).BlindSpots)
}

// DefaultWeights returns the profile weight vector.
func (p *DataExportProfile) DefaultWeights() map[string]float64 {
	return copyFloatMap(loadSchema(p.ID()).Weights)
}

// Evaluate evaluates bundle records against the data export profile.
func (p *DataExportProfile) Evaluate(records []bundle.Record) ProfileResult {
	return evaluateSchemaProfile(loadSchema(p.ID()), records)
}

// ID returns the profile identifier.
func (p *PolicyDecisionProfile) ID() string { return profileIDPolicyDecision }

// Version returns the profile schema version.
func (p *PolicyDecisionProfile) Version() int { return loadSchema(p.ID()).Version }

// WorkflowClass returns the workflow class name.
func (p *PolicyDecisionProfile) WorkflowClass() string { return loadSchema(p.ID()).WorkflowClass }

// BlindSpots returns the profile blind spots.
func (p *PolicyDecisionProfile) BlindSpots() []string {
	return copyStringSlice(loadSchema(p.ID()).BlindSpots)
}

// DefaultWeights returns the profile weight vector.
func (p *PolicyDecisionProfile) DefaultWeights() map[string]float64 {
	return copyFloatMap(loadSchema(p.ID()).Weights)
}

// Evaluate evaluates bundle records against the policy decision profile.
func (p *PolicyDecisionProfile) Evaluate(records []bundle.Record) ProfileResult {
	return evaluateSchemaProfile(loadSchema(p.ID()), records)
}

// ID returns the profile identifier.
func (p *HumanOverrideProfile) ID() string { return profileIDHumanOverride }

// Version returns the profile schema version.
func (p *HumanOverrideProfile) Version() int { return loadSchema(p.ID()).Version }

// WorkflowClass returns the workflow class name.
func (p *HumanOverrideProfile) WorkflowClass() string { return loadSchema(p.ID()).WorkflowClass }

// BlindSpots returns the profile blind spots.
func (p *HumanOverrideProfile) BlindSpots() []string {
	return copyStringSlice(loadSchema(p.ID()).BlindSpots)
}

// DefaultWeights returns the profile weight vector.
func (p *HumanOverrideProfile) DefaultWeights() map[string]float64 {
	return copyFloatMap(loadSchema(p.ID()).Weights)
}

// Evaluate evaluates bundle records against the human override profile.
func (p *HumanOverrideProfile) Evaluate(records []bundle.Record) ProfileResult {
	return evaluateSchemaProfile(loadSchema(p.ID()), records)
}

// ID returns the profile identifier.
func (p *BackgroundAutomationProfile) ID() string { return profileIDBackgroundAutomation }

// Version returns the profile schema version.
func (p *BackgroundAutomationProfile) Version() int { return loadSchema(p.ID()).Version }

// WorkflowClass returns the workflow class name.
func (p *BackgroundAutomationProfile) WorkflowClass() string { return loadSchema(p.ID()).WorkflowClass }

// BlindSpots returns the profile blind spots.
func (p *BackgroundAutomationProfile) BlindSpots() []string {
	return copyStringSlice(loadSchema(p.ID()).BlindSpots)
}

// DefaultWeights returns the profile weight vector.
func (p *BackgroundAutomationProfile) DefaultWeights() map[string]float64 {
	return copyFloatMap(loadSchema(p.ID()).Weights)
}

// Evaluate evaluates bundle records against the background automation profile.
func (p *BackgroundAutomationProfile) Evaluate(records []bundle.Record) ProfileResult {
	return evaluateSchemaProfile(loadSchema(p.ID()), records)
}

// ID returns the profile identifier.
func (p *RAGAnswerProfile) ID() string { return profileIDRAGAnswer }

// Version returns the profile schema version.
func (p *RAGAnswerProfile) Version() int { return loadSchema(p.ID()).Version }

// WorkflowClass returns the workflow class name.
func (p *RAGAnswerProfile) WorkflowClass() string { return loadSchema(p.ID()).WorkflowClass }

// BlindSpots returns the profile blind spots.
func (p *RAGAnswerProfile) BlindSpots() []string {
	return copyStringSlice(loadSchema(p.ID()).BlindSpots)
}

// DefaultWeights returns the profile weight vector.
func (p *RAGAnswerProfile) DefaultWeights() map[string]float64 {
	return copyFloatMap(loadSchema(p.ID()).Weights)
}

// Evaluate evaluates bundle records against the RAG answer profile.
func (p *RAGAnswerProfile) Evaluate(records []bundle.Record) ProfileResult {
	return evaluateSchemaProfile(loadSchema(p.ID()), records)
}

// profileWithSchema is implemented by externalSchemaProfile (profile_loader.go).
// It exposes the underlying ProfileSchema so that CAS support and generic
// sub-score computation work for user-defined profiles without a separate code path.
type profileWithSchema interface {
	profileSchema() profiledsl.ProfileSchema
}

func subScoresForProfile(profile Profile, records []bundle.Record, anchorResult AnchorVerifyResult) map[string]float64 {
	if profile == nil {
		return zeroSubScores()
	}

	builder, ok := profileSubScoreBuilders[profile.ID()]
	if ok {
		return builder(records, anchorResult)
	}

	// Generic fallback for schema-backed custom profiles (DSL and schema-format files).
	if sp, ok := profile.(profileWithSchema); ok {
		return genericSchemaSubScores(sp.profileSchema(), records, anchorResult)
	}

	return zeroSubScores()
}

// genericSchemaSubScores computes CAS sub-scores for a custom ProfileSchema.
// EC and FC are derived from the schema's required event rules. RC, TC, SC, and
// GC cannot be computed generically and are 0.0. XC and AC are always derived
// from the anchor classification result, as they are bundle-level properties.
func genericSchemaSubScores(schema profiledsl.ProfileSchema, records []bundle.Record, anchorResult AnchorVerifyResult) map[string]float64 {
	reqs := make([]eventRequirement, 0, len(schema.Required))
	for _, rule := range schema.Required {
		reqs = append(reqs, eventRequirement{
			eventType:      rule.Type,
			requiredFields: rule.Fields,
		})
	}
	recordsByType := indexRecordsByType(records)
	return map[string]float64{
		"EC": eventCoverage(recordsByType, reqs),
		"FC": fieldCompleteness(recordsByType, reqs),
		"RC": 0.0,
		"TC": 0.0,
		"SC": 0.0,
		"XC": xcScore(anchorResult),
		"AC": acScore(anchorResult),
		"GC": 0.0,
	}
}

func policyDecisionSubScores(records []bundle.Record, anchorResult AnchorVerifyResult) map[string]float64 {
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

	return map[string]float64{
		"EC": eventCoverage(recordsByType, policyDecisionCriticalRequirements),
		"FC": fieldCompleteness(recordsByType, policyDecisionCriticalRequirements),
		"RC": rc,
		"TC": tc,
		"SC": computeSC(&bundle.Bundle{Records: records}, profileIDPolicyDecision),
		"XC": xcScore(anchorResult),
		"AC": acScore(anchorResult),
		"GC": gc,
	}
}

func humanOverrideSubScores(records []bundle.Record, anchorResult AnchorVerifyResult) map[string]float64 {
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

	return map[string]float64{
		"EC": eventCoverage(recordsByType, humanOverrideCriticalRequirements),
		"FC": fieldCompleteness(recordsByType, humanOverrideCriticalRequirements),
		"RC": rc,
		"TC": tc,
		"SC": computeSC(&bundle.Bundle{Records: records}, profileIDHumanOverride),
		"XC": xcScore(anchorResult),
		"AC": acScore(anchorResult),
		"GC": gc,
	}
}

func backgroundAutomationSubScores(records []bundle.Record, anchorResult AnchorVerifyResult) map[string]float64 {
	recordsByType := indexRecordsByType(records)
	scheduledByJob := indexByField(recordsByType[event.TypeAIJobScheduled], "job_id")
	startedByJob := indexByField(recordsByType[event.TypeAIJobStarted], "job_id")
	completedByJob := indexByField(recordsByType[event.TypeAIJobCompleted], "job_id")

	jobLifecycleConsistent := len(recordsByType[event.TypeAIJobScheduled]) > 0 &&
		len(recordsByType[event.TypeAIJobStarted]) > 0 &&
		len(recordsByType[event.TypeAIJobCompleted]) > 0 &&
		allRecordsBound(recordsByType[event.TypeAIJobStarted], "job_id", scheduledByJob) &&
		allRecordsBound(recordsByType[event.TypeAIJobCompleted], "job_id", startedByJob) &&
		allRecordsBound(recordsByType[event.TypeAIJobStarted], "job_id", completedByJob)

	rc := boolScore(jobLifecycleConsistent)
	tc := 1.0
	gc := boolScore(jobLifecycleConsistent)

	return map[string]float64{
		"EC": eventCoverage(recordsByType, backgroundAutomationCriticalRequirements),
		"FC": fieldCompleteness(recordsByType, backgroundAutomationCriticalRequirements),
		"RC": rc,
		"TC": tc,
		"SC": computeSC(&bundle.Bundle{Records: records}, profileIDBackgroundAutomation),
		"XC": xcScore(anchorResult),
		"AC": acScore(anchorResult),
		"GC": gc,
	}
}

func dataExportSubScores(records []bundle.Record, anchorResult AnchorVerifyResult) map[string]float64 {
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

	return map[string]float64{
		"EC": eventCoverage(recordsByType, dataExportCriticalRequirements),
		"FC": fieldCompleteness(recordsByType, dataExportCriticalRequirements),
		"RC": rc,
		"TC": tc,
		"SC": computeSC(&bundle.Bundle{Records: records}, profileIDDataExport),
		"XC": xcScore(anchorResult),
		"AC": acScore(anchorResult),
		"GC": gc,
	}
}

func privilegedToolActionSubScores(records []bundle.Record, anchorResult AnchorVerifyResult) map[string]float64 {
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

	return map[string]float64{
		"EC": eventCoverage(recordsByType, privilegedCriticalRequirements),
		"FC": fieldCompleteness(recordsByType, privilegedCriticalRequirements),
		"RC": rc,
		"TC": tc,
		"SC": computeSC(&bundle.Bundle{Records: records}, profileIDPrivilegedToolAction),
		"XC": xcScore(anchorResult),
		"AC": acScore(anchorResult),
		"GC": gc,
	}
}

func ragAnswerSubScores(records []bundle.Record, anchorResult AnchorVerifyResult) map[string]float64 {
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

	return map[string]float64{
		"EC": eventCoverage(recordsByType, ragCriticalRequirements),
		"FC": fieldCompleteness(recordsByType, ragCriticalRequirements),
		"RC": requestToResponse,
		"TC": averageScores(temporalScores...),
		"SC": computeSC(&bundle.Bundle{Records: records}, profileIDRAGAnswer),
		"XC": xcScore(anchorResult),
		"AC": acScore(anchorResult),
		"GC": 0.3,
	}
}

func loadSchema(id string) profiledsl.ProfileSchema {
	return profiledsl.MustLoadSchema(id)
}

func evaluateSchemaProfile(schema profiledsl.ProfileSchema, records []bundle.Record) ProfileResult {
	evaluation := profiledsl.Evaluate(schema, records)
	result := ProfileResult{
		ProfileID:          schema.ID,
		Version:            schema.Version,
		WorkflowClass:      schema.WorkflowClass,
		Pass:               evaluation.Pass,
		CriticalFailures:   make([]CriticalFailure, 0, len(evaluation.CriticalFailures)),
		RequiredWarnings:   append([]string(nil), evaluation.RequiredWarnings...),
		InformationalNotes: append([]string(nil), evaluation.InformationalNotes...),
	}

	for _, failure := range evaluation.CriticalFailures {
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind:   failure.Kind,
			Detail: failure.Detail,
		})
	}

	return result
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

func copyStringSlice(src []string) []string {
	return append([]string(nil), src...)
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
