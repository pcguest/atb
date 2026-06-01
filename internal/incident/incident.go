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
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
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

// Report is a session-scoped incident report over a single bundle.
type Report struct {
	BundlePath     string                       `json:"bundle_path"`
	SessionID      string                       `json:"session_id"`
	Found          bool                         `json:"found"`
	IntegrityValid bool                         `json:"integrity_valid"`
	ChainHeadHash  string                       `json:"chain_head_hash"`
	Signatures     []verify.SignatureProvenance `json:"signatures,omitempty"`
	Session        *sessionindex.SessionEntry   `json:"session,omitempty"`
	Events         []EventRow                   `json:"events"`
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
	}
	sort.SliceStable(rep.Events, func(i, j int) bool { return rep.Events[i].Seq < rep.Events[j].Seq })
	return rep, nil
}

// JSON renders the report as indented JSON.
func (r Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
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

	b.WriteString("\n## Events\n\n")
	b.WriteString("| Seq | Type | Time | Summary | Record hash |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, e := range r.Events {
		fmt.Fprintf(&b, "| %d | `%s` | %s | %s | `%s` |\n",
			e.Seq, e.Type, e.Timestamp, e.Summary, shortHash(e.Hash))
	}
	b.WriteString("\n> Integrity proves these records are unaltered. This report scopes them to the named session; the full signed bundle is the authoritative evidence, and each row's record hash is verifiable against it.\n")
	return b.String()
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
	case event.TypeDataExportExecuted:
		if outcome := get("execution_outcome"); outcome != "" {
			return "export outcome=" + outcome
		}
		return "export"
	case event.TypeDataExport:
		if target := get("export_target"); target != "" {
			return "export=" + target
		}
		return "export"
	default:
		return ""
	}
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

func shortHash(h string) string {
	if len(h) > 16 {
		return h[:16] + "…"
	}
	return h
}
