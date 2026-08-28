// SPDX-License-Identifier: MIT

// Package incident builds a session-scoped forensic report over an ATB bundle.
//
// A bundle's integrity is verified across the whole hash chain, so a single
// session cannot be carved into an independently verifiable sub-bundle. The
// incident report therefore scopes one session for review while the full
// signed bundle remains the authoritative, tamper-evident evidence: every event
// row carries its sequence and record hash, so each is checkable against that
// bundle. The report is the lens; the bundle is the proof.
package incident

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/identityevidence"
	"github.com/pcguest/atb/internal/sessionindex"
	"github.com/pcguest/atb/internal/verify"
)

// EventRow is one event scoped to the incident session.
type EventRow struct {
	Seq       int    `json:"seq"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp,omitempty"`
	Hash      string `json:"hash"`
	Summary   string `json:"summary,omitempty"`
}

// CaptureScope is the recorder's coverage attestation (atb.capture.scope).
type CaptureScope struct {
	Targets     []string `json:"targets,omitempty"`
	CaptureMode string   `json:"capture_mode,omitempty"`
	OutOfScope  string   `json:"out_of_scope,omitempty"`
}

// Report is a session-scoped incident report over a single bundle.
type Report struct {
	BundlePath         string                       `json:"bundle_path"`
	SessionID          string                       `json:"session_id"`
	Found              bool                         `json:"found"`
	IntegrityValid     bool                         `json:"integrity_valid"`
	ChainHeadHash      string                       `json:"chain_head_hash"`
	Signatures         []verify.SignatureProvenance `json:"signatures,omitempty"`
	CaptureScope       *CaptureScope                `json:"capture_scope,omitempty"`
	Session            *sessionindex.SessionEntry   `json:"session,omitempty"`
	Findings           []Finding                    `json:"findings,omitempty"`
	ReviewerIdentities []identityevidence.Evidence  `json:"reviewer_identities,omitempty"`
	Events             []EventRow                   `json:"events"`
}

// Build loads bundlePath, verifies its integrity, and assembles the report for
// the events belonging to sessionID. Integrity status is reported (not fatal)
// so a tampered bundle still yields a report that records the failure.
func Build(ctx context.Context, bundlePath, sessionID string) (Report, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return Report{}, fmt.Errorf("incident: session id is required")
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		return Report{}, fmt.Errorf("incident: load bundle %q: %w", bundlePath, err)
	}

	rep := Report{
		BundlePath: bundlePath,
		SessionID:  sessionID,
		Events:     []EventRow{},
	}
	if n := len(b.Records); n > 0 {
		rep.ChainHeadHash = b.Records[n-1].Hash
	}

	// Integrity (hash chain) and signature provenance come from the canonical
	// verifier so the incident report makes no independent trust claims.
	if report, evalErr := verify.EvaluateBundle(verify.EvaluateConfig{
		BundlePath:    bundlePath,
		Records:       b.Records,
		AllApplicable: true,
	}); evalErr == nil && report != nil {
		rep.IntegrityValid = report.Integrity.ChainValid
		rep.Signatures = report.Signatures
	} else {
		rep.IntegrityValid = b.Verify() == nil
	}

	// Session summary + anomaly flags from the session index (best effort).
	if entries, err := sessionindex.BuildIndex(ctx, []string{bundlePath}); err == nil {
		for i := range entries {
			if entries[i].SessionID == sessionID {
				e := entries[i]
				rep.Session = &e
				break
			}
		}
	}

	// Capture-coverage attestation is bundle-level (no session_id); take the latest.
	for _, rec := range b.Records {
		if rec.Event.Type == event.TypeCaptureScope {
			rep.CaptureScope = captureScopeFrom(rec.Event)
		}
	}
	rep.ReviewerIdentities = identityEvidenceForSession(b, sessionID)

	var scoped []scopedEvent
	for _, rec := range b.Records {
		if eventSessionID(rec.Event) != sessionID {
			continue
		}
		rep.Found = true
		rep.Events = append(rep.Events, EventRow{
			Seq:       rec.Event.Sequence,
			Type:      rec.Event.Type,
			Timestamp: rec.Event.Timestamp,
			Hash:      rec.Hash,
			Summary:   summarise(rec.Event),
		})
		data, _ := rec.Event.Data.(map[string]any)
		scoped = append(scoped, scopedEvent{seq: rec.Event.Sequence, typ: rec.Event.Type, data: data})
	}
	sort.SliceStable(rep.Events, func(i, j int) bool { return rep.Events[i].Seq < rep.Events[j].Seq })
	sort.SliceStable(scoped, func(i, j int) bool { return scoped[i].seq < scoped[j].seq })

	// Explain the session index's anomaly flags as located findings. The index
	// stays authoritative on which flags fire; here we attach severity, meaning,
	// and the triggering event sequence numbers.
	if rep.Session != nil {
		rep.Findings = buildFindings(rep.Session.AnomalyFlags, scoped)
	}
	return rep, nil
}

// JSON renders the report as indented JSON.
func (r Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// NDJSON renders one JSON object per session event, newline-delimited, for
// direct ingestion into a SIEM (Splunk/Elastic). Each line is denormalised with
// the session's integrity status and anomaly flags so it stands alone.
func (r Report) NDJSON() ([]byte, error) {
	var flags []string
	if r.Session != nil {
		flags = r.Session.AnomalyFlags
	}
	if flags == nil {
		flags = []string{}
	}
	bySeq := triggeredFlagsBySeq(r.Findings)
	var buf bytes.Buffer
	for _, e := range r.Events {
		triggered := bySeq[e.Seq]
		if triggered == nil {
			triggered = []string{}
		}
		line, err := json.Marshal(map[string]any{
			"session_id":      r.SessionID,
			"seq":             e.Seq,
			"type":            e.Type,
			"timestamp":       e.Timestamp,
			"summary":         e.Summary,
			"record_hash":     e.Hash,
			"integrity_valid": r.IntegrityValid,
			"anomaly_flags":   flags,
			// triggered_flags is the subset of anomaly_flags this specific event
			// triggered — so a SIEM can alert on the offending record, not just
			// the session.
			"triggered_flags": triggered,
		})
		if err != nil {
			return nil, err
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

func identityEvidenceForSession(b *bundle.Bundle, sessionID string) []identityevidence.Evidence {
	all := identityevidence.Extract(b)
	if len(all) == 0 {
		return nil
	}
	seqs := make(map[int]struct{})
	for _, record := range b.Records {
		if eventSessionID(record.Event) == sessionID {
			seqs[record.Event.Sequence] = struct{}{}
		}
	}
	out := make([]identityevidence.Evidence, 0, len(all))
	for _, evidence := range all {
		if _, ok := seqs[evidence.Sequence]; ok {
			out = append(out, evidence)
		}
	}
	return out
}

// ListSessions returns the sessions found in a bundle, so a reviewer can
// discover which session to report on. It is a thin wrapper over the session
// index, scoped to a single bundle.
func ListSessions(ctx context.Context, bundlePath string) ([]sessionindex.SessionEntry, error) {
	return sessionindex.BuildIndex(ctx, []string{bundlePath})
}

// SessionListMarkdown renders a session list as a reviewer-facing table.
func SessionListMarkdown(bundlePath string, entries []sessionindex.SessionEntry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Sessions in `%s`\n\n", bundlePath)
	if len(entries) == 0 {
		b.WriteString("No sessions found.\n")
		return b.String()
	}
	b.WriteString("| Session | Actor | Exchanges | Profile | CAS | Anomalies |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, e := range entries {
		anomalies := "none"
		if len(e.AnomalyFlags) > 0 {
			anomalies = strings.Join(e.AnomalyFlags, ", ")
		}
		fmt.Fprintf(&b, "| `%s` | %s | %d | %s | %s | %s |\n",
			e.SessionID, actorLabel(e.Actor), e.ExchangeCount,
			orDash(e.InferredProfile), orDash(e.CASGrade), anomalies)
	}
	return b.String()
}

// Markdown renders the report as a reviewer-facing markdown document.
func (r Report) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Incident report — session `%s`\n\n", r.SessionID)
	if !r.Found {
		fmt.Fprintf(&b, "No events found for session `%s` in `%s`.\n", r.SessionID, r.BundlePath)
		return b.String()
	}

	integrity := "FAIL"
	if r.IntegrityValid {
		integrity = "PASS"
	}
	fmt.Fprintf(&b, "- Bundle: `%s`\n", r.BundlePath)
	fmt.Fprintf(&b, "- Integrity (hash chain): **%s**\n", integrity)
	fmt.Fprintf(&b, "- Signature: %s\n", signatureSummary(r.Signatures))
	if r.CaptureScope != nil {
		fmt.Fprintf(&b, "- Capture coverage: targets=[%s] mode=%s\n",
			strings.Join(r.CaptureScope.Targets, ", "), orDash(r.CaptureScope.CaptureMode))
		if r.CaptureScope.OutOfScope != "" {
			fmt.Fprintf(&b, "  - Out of scope: %s\n", r.CaptureScope.OutOfScope)
		}
	}
	fmt.Fprintf(&b, "- Chain head hash: `%s`\n", r.ChainHeadHash)
	if r.Session != nil {
		fmt.Fprintf(&b, "- Actor: %s\n", actorLabel(r.Session.Actor))
		fmt.Fprintf(&b, "- Exchanges: %d\n", r.Session.ExchangeCount)
		fmt.Fprintf(&b, "- Inferred profile: %s\n", orDash(r.Session.InferredProfile))
		fmt.Fprintf(&b, "- CAS grade: %s\n", orDash(r.Session.CASGrade))
		flags := "none"
		if len(r.Session.AnomalyFlags) > 0 {
			flags = strings.Join(r.Session.AnomalyFlags, ", ")
		}
		fmt.Fprintf(&b, "- Anomalies: **%s**\n", flags)
	}

	if len(r.Findings) > 0 {
		b.WriteString("\n## Findings\n\n")
		b.WriteString("Each anomaly flag, what it means, and the event(s) that triggered it.\n\n")
		b.WriteString("| Severity | Finding | Triggered at | Detail |\n")
		b.WriteString("| --- | --- | --- | --- |\n")
		for _, f := range r.Findings {
			at := "—"
			if len(f.EventSeqs) > 0 {
				at = "seq " + joinSeqs(f.EventSeqs)
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
				strings.ToUpper(f.Severity), f.Title, at, f.Detail)
		}
	}
	if len(r.ReviewerIdentities) > 0 {
		b.WriteString("\n## Reviewer identity evidence\n\n")
		b.WriteString("These values were supplied by the caller and preserved by the ATB hash chain; ATB did not independently verify the identity assertion.\n\n")
		for _, evidence := range r.ReviewerIdentities {
			fmt.Fprintf(&b, "- Seq %d: %s\n", evidence.Sequence, identityevidence.Summary(evidence))
		}
	}

	b.WriteString("\n## Events\n\n")
	b.WriteString("| Seq | Type | Time | Summary | Record hash |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, e := range r.Events {
		fmt.Fprintf(&b, "| %d | `%s` | %s | %s | `%s` |\n",
			e.Seq, e.Type, e.Timestamp, e.Summary, shortHash(e.Hash))
	}
	b.WriteString("\n> Integrity PASS means the presented session records agree with the bundle's hash chain. It does not prove capture completeness or prevent whole-file replacement without external custody. This report scopes the bundle to one session, and each row's record hash is independently verifiable against it.\n")
	return b.String()
}

func captureScopeFrom(ev event.Event) *CaptureScope {
	data, ok := ev.Data.(map[string]any)
	if !ok {
		return nil
	}
	scope := &CaptureScope{}
	if mode, ok := data["capture_mode"].(string); ok {
		scope.CaptureMode = mode
	}
	if oos, ok := data["out_of_scope"].(string); ok {
		scope.OutOfScope = oos
	}
	if targets, ok := data["targets"].([]any); ok {
		for _, t := range targets {
			if s, ok := t.(string); ok {
				scope.Targets = append(scope.Targets, s)
			}
		}
	}
	return scope
}

func eventSessionID(ev event.Event) string {
	data, ok := ev.Data.(map[string]any)
	if !ok {
		return ""
	}
	for _, key := range []string{"session_id", "sessionId"} {
		if s, ok := data[key].(string); ok {
			if s = strings.TrimSpace(s); s != "" {
				return s
			}
		}
	}
	return ""
}

func summarise(ev event.Event) string {
	data, _ := ev.Data.(map[string]any)
	get := func(k string) string { s, _ := data[k].(string); return strings.TrimSpace(s) }
	switch ev.Type {
	case event.TypeAIActionPrecommit:
		base := "action=" + get("action_type")
		if p := principalSummary(data); p != "" {
			return base + " " + p
		}
		return base
	case event.TypeAIActionExecuted:
		base := "executed outcome=" + get("execution_outcome")
		if scope := get("effective_scope"); scope != "" {
			return base + " scope=" + scope
		}
		return base
	case event.TypeToolCall:
		return "tool=" + get("tool_name")
	case event.TypeAIActionError:
		return fmt.Sprintf("action=%s error_class=%s", get("action_id"), get("error_class"))
	case event.TypeLLMRequest:
		return strings.TrimSpace(get("method") + " " + get("path"))
	case event.TypeLLMResponse:
		s := strings.TrimSpace(get("method") + " " + get("path"))
		if code, ok := intField(data, "status_code"); ok {
			s = fmt.Sprintf("%s → %d", s, code)
		}
		return s
	case event.TypeAIPolicyDecision:
		return "decision=" + get("decision")
	case event.TypeAIHumanApproval:
		return "approval=" + get("approval_outcome")
	case event.TypeHumanApproval:
		return "approved=" + get("approved_action_id")
	case event.TypeHumanOverride:
		if actionID := get("overridden_action_id"); actionID != "" {
			return "override=" + actionID
		}
		return "human override"
	case event.TypeCaptureScope:
		return "capture=" + get("capture_mode")
	case event.TypeDataExportExecuted:
		if outcome := get("execution_outcome"); outcome != "" {
			return "export outcome=" + outcome
		}
		return "export"
	case event.TypeDataExportError:
		if ec := get("error_class"); ec != "" {
			return fmt.Sprintf("export action=%s error_class=%s", get("action_id"), ec)
		}
		return "export error"
	case event.TypeDataExport:
		if target := get("export_target"); target != "" {
			return "export=" + target
		}
		return "export"
	case event.TypeDataRetentionPolicySet, event.TypeDataRetentionPolicyChanged:
		if days, ok := intField(data, "days"); ok {
			return fmt.Sprintf("retention=%dd", days)
		}
		return "retention policy"
	case event.TypeDataRetentionEnforced:
		return strings.TrimSpace(get("operation") + " outcome=" + get("outcome"))
	default:
		return ""
	}
}

// principalSummary renders the acting-principal attribution from an event's
// optional `principal` object, e.g. "by agent:sha256-… on_behalf_of sha256-…".
func principalSummary(data map[string]any) string {
	p, ok := data["principal"].(map[string]any)
	if !ok {
		return ""
	}
	typ, _ := p["type"].(string)
	id, _ := p["id_hash"].(string)
	if strings.TrimSpace(typ) == "" || strings.TrimSpace(id) == "" {
		return ""
	}
	label := fmt.Sprintf("by %s:%s", typ, id)
	if obo, _ := p["on_behalf_of"].(string); strings.TrimSpace(obo) != "" {
		label += " on_behalf_of " + obo
	}
	return label
}

func intField(data map[string]any, key string) (int, bool) {
	switch v := data[key].(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return int(n), true
		}
	}
	return 0, false
}

// signatureSummary renders the bundle's signing provenance for the report
// header. An unsigned bundle is itself a chain-of-custody finding, so it is
// stated plainly rather than omitted.
func signatureSummary(sigs []verify.SignatureProvenance) string {
	if len(sigs) == 0 {
		return "none (unsigned bundle)"
	}
	latest := sigs[len(sigs)-1]
	verdict := "INVALID"
	if latest.Valid {
		verdict = "valid"
	}
	parts := []string{verdict}
	if latest.PubKey != "" {
		parts = append(parts, "pubkey "+shortHash(latest.PubKey))
	}
	if latest.SignedAt != "" {
		parts = append(parts, "signed "+latest.SignedAt)
	}
	if latest.Backend != "" {
		parts = append(parts, "backend "+latest.Backend)
	}
	summary := strings.Join(parts, ", ")
	if len(sigs) > 1 {
		summary = fmt.Sprintf("%d signatures; latest: %s", len(sigs), summary)
	}
	if !latest.Valid && latest.Error != "" {
		summary += " (" + latest.Error + ")"
	}
	return summary
}

func actorLabel(a sessionindex.ActorSummary) string {
	switch {
	case a.DisplayName != "" && a.Email != "":
		return fmt.Sprintf("%s <%s>", a.DisplayName, a.Email)
	case a.DisplayName != "":
		return a.DisplayName
	default:
		return "—"
	}
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func joinSeqs(seqs []int) string {
	parts := make([]string, len(seqs))
	for i, s := range seqs {
		parts[i] = fmt.Sprintf("%d", s)
	}
	return strings.Join(parts, ", ")
}

func shortHash(h string) string {
	if len(h) > 16 {
		return h[:16] + "…"
	}
	return h
}
