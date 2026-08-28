// SPDX-License-Identifier: MIT
package sessionindex

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/hash"
	"github.com/pcguest/atb/internal/verify"
)

const (
	profilePrivilegedToolAction = "atb.profile.privileged_tool_action"
	profileDataExport           = "atb.profile.data_export"
	profilePolicyDecision       = "atb.profile.policy_decision"
	profileHumanOverride        = "atb.profile.human_override"
	profileRAG                  = "atb.profile.rag_answer"
	profileBackgroundJob        = "atb.profile.background_automation"
)

// SessionEntry is the aggregated metadata exposed by the local viewer session index.
type SessionEntry struct {
	SessionID       string       `json:"session_id"`
	Actor           ActorSummary `json:"actor"`
	StartedAt       time.Time    `json:"started_at"`
	ClosedAt        *time.Time   `json:"closed_at,omitempty"`
	ExchangeCount   int          `json:"exchange_count"`
	InferredProfile string       `json:"inferred_profile"`
	CASGrade        string       `json:"cas_grade"`
	AnomalyFlags    []string     `json:"anomaly_flags"`
	BundlePath      string       `json:"bundle_path"`
}

// ActorSummary is the limited actor metadata exposed by the session index.
type ActorSummary struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

// LoadError describes one bundle that could not be loaded through LoadVerified.
type LoadError struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

func (e LoadError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Reason)
}

// LoadErrors reports all bundle verification failures encountered while indexing.
type LoadErrors []LoadError

func (e LoadErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	return fmt.Sprintf("%d bundle load errors", len(e))
}

// BuildIndex loads verified bundles and returns session metadata for the viewer.
func BuildIndex(ctx context.Context, bundles []string) ([]SessionEntry, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	entries := make([]SessionEntry, 0, len(bundles))
	var loadErrors LoadErrors
	for _, path := range bundles {
		if err := ctx.Err(); err != nil {
			return entries, err
		}

		absPath := absolutePath(path)
		loaded, err := bundle.LoadVerified(path)
		if err != nil {
			reason := safeErrorReason(err)
			fmt.Fprintf(os.Stderr, "atb session index: load verified %s: %s\n", path, reason)
			loadErrors = append(loadErrors, LoadError{Path: absPath, Reason: reason})
			entries = append(entries, SessionEntry{
				SessionID:       sessionIDFromPath(path),
				Actor:           ActorSummary{},
				StartedAt:       time.Time{},
				ClosedAt:        nil,
				ExchangeCount:   0,
				InferredProfile: "",
				CASGrade:        "",
				AnomalyFlags:    []string{"load_error"},
				BundlePath:      absPath,
			})
			continue
		}

		if err := ctx.Err(); err != nil {
			return entries, err
		}

		entries = append(entries, entriesForBundle(absPath, loaded)...)
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].StartedAt.After(entries[j].StartedAt)
	})

	if len(loadErrors) > 0 {
		return entries, loadErrors
	}
	return entries, nil
}

// GroupByActor groups session entries by actor display name.
func GroupByActor(entries []SessionEntry) map[string][]SessionEntry {
	grouped := make(map[string][]SessionEntry)
	for _, entry := range entries {
		key := strings.TrimSpace(entry.Actor.DisplayName)
		if key == "" {
			key = "unknown"
		}
		grouped[key] = append(grouped[key], entry)
	}
	for key := range grouped {
		sort.SliceStable(grouped[key], func(i, j int) bool {
			return grouped[key][i].StartedAt.After(grouped[key][j].StartedAt)
		})
	}
	return grouped
}

// ExtractLoadErrors returns structured load errors from a BuildIndex error.
func ExtractLoadErrors(err error) []LoadError {
	var loadErrors LoadErrors
	if errors.As(err, &loadErrors) {
		return append([]LoadError(nil), loadErrors...)
	}
	return nil
}

type sessionAccumulator struct {
	entry        SessionEntry
	events       []hash.Event
	hasClose     bool
	approvalSeen bool
	// toolWithoutApproval records whether any atb.tool.call was observed in
	// this session before a preceding atb.human.approval. Ordering matters:
	// per docs/specification/events.md an approval only closes the
	// tool_without_approval anomaly when it is recorded in sequence before
	// the tool call.
	toolWithoutApproval bool
	seenSession         bool
	// deniedActions records action_ids a policy decision denied; if one later
	// executes, deniedThenExecuted is raised (policy said no, the action ran).
	deniedActions      map[string]bool
	deniedThenExecuted bool
	// actionFailed records that a privileged action did not succeed
	// (an ai.action.error was observed).
	actionFailed bool
}

// bundleLevelEvent reports whether the event type is a bundle-scoped system
// record (manifest, signature, anchor, push marker, snapshot) rather than
// session activity. These carry no session_id, so without this skip they would
// seed a spurious path-derived pseudo-session — visible, for example, once a
// captured bundle is signed.
func bundleLevelEvent(eventType string) bool {
	switch eventType {
	case event.TypeBundleManifest,
		event.TypeBundleSignature,
		event.TypeBundleAnchor,
		event.TypeSnapshot,
		event.TypeCaptureScope:
		return true
	default:
		return false
	}
}

func entriesForBundle(bundlePath string, b *bundle.Bundle) []SessionEntry {
	sessions := make(map[string]*sessionAccumulator)
	order := []string{}

	for _, record := range b.Records {
		if bundleLevelEvent(record.Event.Type) {
			continue
		}
		sessionID := sessionIDForEvent(record.Event, bundlePath)
		acc, ok := sessions[sessionID]
		if !ok {
			acc = &sessionAccumulator{
				entry: SessionEntry{
					SessionID:    sessionID,
					AnomalyFlags: []string{},
					BundlePath:   bundlePath,
				},
			}
			sessions[sessionID] = acc
			order = append(order, sessionID)
		}
		acc.seenSession = true
		acc.events = append(acc.events, record.Event)
		applyEvent(acc, record.Event)
	}

	entries := make([]SessionEntry, 0, len(order))
	for _, sessionID := range order {
		acc := sessions[sessionID]
		if !acc.seenSession {
			continue
		}
		acc.entry.InferredProfile = inferProfile(acc.events)
		acc.entry.CASGrade = casGradeForProfile(bundlePath, b, acc.entry.InferredProfile)
		acc.entry.AnomalyFlags = anomalyFlags(acc)
		entries = append(entries, acc.entry)
	}
	return entries
}

func applyEvent(acc *sessionAccumulator, event hash.Event) {
	if ts, ok := eventTime(event); ok {
		if acc.entry.StartedAt.IsZero() || ts.Before(acc.entry.StartedAt) {
			acc.entry.StartedAt = ts
		}
		if event.Type == "atb.session.close" {
			closedAt := ts
			acc.entry.ClosedAt = &closedAt
			acc.hasClose = true
		}
	}

	if actor := actorFromEvent(event); acc.entry.Actor.DisplayName == "" || acc.entry.Actor.Email == "" {
		if acc.entry.Actor.DisplayName == "" {
			acc.entry.Actor.DisplayName = actor.DisplayName
		}
		if acc.entry.Actor.Email == "" {
			acc.entry.Actor.Email = actor.Email
		}
	}

	switch event.Type {
	case "atb.exchange.complete":
		acc.entry.ExchangeCount++
	case "atb.human.approval":
		// Record approvals in arrival order. Records are applied in bundle
		// append order, so a tool call observed after this point is treated
		// as approved.
		acc.approvalSeen = true
	case "atb.tool.call":
		// A tool call with no preceding approval in the same session raises
		// the anomaly. A later approval does not retroactively close it.
		if !acc.approvalSeen {
			acc.toolWithoutApproval = true
		}
	case "atb.session.close":
		if count, ok := intField(eventDataMap(event), "exchange_count"); ok {
			acc.entry.ExchangeCount = count
		}
	case "ai.policy.decision":
		data := eventDataMap(event)
		if stringField(data, "decision") == "deny" {
			if id := stringField(data, "action_id"); id != "" {
				if acc.deniedActions == nil {
					acc.deniedActions = map[string]bool{}
				}
				acc.deniedActions[id] = true
			}
		}
	case "ai.action.executed", "ai.action.committed":
		if id := stringField(eventDataMap(event), "action_id"); id != "" && acc.deniedActions[id] {
			acc.deniedThenExecuted = true
		}
	case "ai.action.error":
		acc.actionFailed = true
	}
}

func anomalyFlags(acc *sessionAccumulator) []string {
	flags := []string{}
	if acc.toolWithoutApproval {
		flags = append(flags, "tool_without_approval")
	}
	if acc.deniedThenExecuted {
		flags = append(flags, "policy_denied_executed")
	}
	if acc.actionFailed {
		flags = append(flags, "action_failed")
	}
	if strings.HasPrefix(acc.entry.Actor.DisplayName, "api-key:") {
		flags = append(flags, "unresolved_identity")
	}
	if !acc.hasClose {
		flags = append(flags, "session_not_closed")
	}
	return flags
}

func inferProfile(events []hash.Event) string {
	hasDataExport := false
	hasPolicyDecision := false
	hasHumanOverride := false
	hasRetrieval := false
	hasBackgroundJob := false
	for _, event := range events {
		switch {
		case event.Type == "atb.tool.call":
			return profilePrivilegedToolAction
		case event.Type == "atb.data.export":
			hasDataExport = true
		case event.Type == "ai.policy.decision":
			hasPolicyDecision = true
		case event.Type == "atb.human.override":
			hasHumanOverride = true
		case strings.HasPrefix(event.Type, "ai.retrieval."):
			hasRetrieval = true
		case strings.HasPrefix(event.Type, "ai.job."):
			hasBackgroundJob = true
		}
	}
	switch {
	case hasDataExport:
		return profileDataExport
	case hasPolicyDecision:
		return profilePolicyDecision
	case hasHumanOverride:
		return profileHumanOverride
	case hasRetrieval:
		return profileRAG
	case hasBackgroundJob:
		return profileBackgroundJob
	default:
		return ""
	}
}

func casGradeForProfile(bundlePath string, b *bundle.Bundle, profileID string) string {
	profile := verify.ProfileByID(profileID)
	if profile == nil {
		return ""
	}
	report, err := verify.EvaluateBundle(verify.EvaluateConfig{
		BundlePath: bundlePath,
		Records:    b.Records,
		Profiles:   []verify.Profile{profile},
	})
	if err != nil || report == nil {
		return ""
	}
	verifierReport := verify.ReportFromVerify(*report)
	return verifierReport.CASGrade
}

func sessionIDForEvent(event hash.Event, bundlePath string) string {
	if sessionID := eventSessionID(event); sessionID != "" {
		return sessionID
	}
	return sessionIDFromPath(bundlePath)
}

func eventSessionID(event hash.Event) string {
	data := eventDataMap(event)
	for _, key := range []string{"session_id", "sessionId"} {
		if value := stringField(data, key); value != "" {
			return value
		}
	}
	return ""
}

func actorFromEvent(event hash.Event) ActorSummary {
	data := eventDataMap(event)
	actorData, _ := data["actor"].(map[string]any)
	actor := ActorSummary{
		DisplayName: stringField(actorData, "display_name"),
		Email:       stringField(actorData, "email"),
	}
	if actor.DisplayName == "" {
		actor.DisplayName = stringField(data, "actor.display_name")
	}
	if actor.Email == "" {
		actor.Email = stringField(data, "actor.email")
	}
	if actor.DisplayName == "" && event.ActorID != nil {
		actor.DisplayName = strings.TrimSpace(*event.ActorID)
	}
	return actor
}

func eventTime(event hash.Event) (time.Time, bool) {
	for _, value := range []string{
		event.Timestamp,
		stringField(eventDataMap(event), "timestamp"),
		stringField(eventDataMap(event), "started_at"),
		stringField(eventDataMap(event), "closed_at"),
	} {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, value)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func eventDataMap(event hash.Event) map[string]any {
	if event.Data == nil {
		return nil
	}
	if data, ok := event.Data.(map[string]any); ok {
		return data
	}
	return nil
}

func stringField(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	value, ok := data[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func intField(data map[string]any, key string) (int, bool) {
	if data == nil {
		return 0, false
	}
	switch value := data[key].(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case float64:
		return int(value), true
	default:
		return 0, false
	}
}

func absolutePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func sessionIDFromPath(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if strings.TrimSpace(base) == "" {
		return "unknown"
	}
	return base
}

func safeErrorReason(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}
