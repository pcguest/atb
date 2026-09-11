// SPDX-License-Identifier: MIT

package incident

import (
	"sort"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/sessionindex"
)

// Finding explains one anomaly flag: what it means, how serious it is, and the
// sequence numbers of the events that triggered it. The session index remains
// the sole authority on whether a flag fires; a Finding is the human-readable
// explanation, located against the session's own events. An investigator should
// not have to re-derive "what happened and where" from a bare flag string — the
// report does that correlation for them.
type Finding struct {
	Flag               string `json:"flag"`
	Severity           string `json:"severity"`
	Title              string `json:"title"`
	Detail             string `json:"detail"`
	Basis              string `json:"basis"`
	SessionID          string `json:"session_id,omitempty"`
	ProfileID          string `json:"profile_id,omitempty"`
	RuleID             string `json:"rule_id,omitempty"`
	Boundedness        string `json:"boundedness"`
	WhatCanConclude    string `json:"what_atb_can_conclude"`
	WhatCannotConclude string `json:"what_atb_cannot_conclude"`
	EventSeqs          []int  `json:"event_seqs,omitempty"`
}

// scopedEvent is the minimal view of a session event needed to locate the
// trigger for an anomaly flag.
type scopedEvent struct {
	seq  int
	typ  string
	data map[string]any
}

type flagMeta struct {
	severity       string
	title          string
	detail         string
	canConclude    string
	cannotConclude string
}

// flagExplanations maps each anomaly flag the session index can raise to its
// severity and plain-English meaning. Keep this in step with
// sessionindex.anomalyFlags; an unmapped flag is still surfaced honestly rather
// than dropped (see buildFindings).
var flagExplanations = map[string]flagMeta{
	"tool_without_approval": {
		severity:       "high",
		title:          "Tool call with no preceding approval",
		detail:         "No matching earlier human approval exists in the captured session evidence for this tool call.",
		canConclude:    "The captured evidence contains no earlier matching approval for the tool call.",
		cannotConclude: "ATB cannot conclude that no approval existed outside the captured evidence.",
	},
	"policy_denied_executed": {
		severity:       "critical",
		title:          "Action executed after policy denial",
		detail:         "A captured policy decision denied an action with the same action_id before captured execution evidence was recorded.",
		canConclude:    "The presented records bind a denial and later execution evidence to the same action_id.",
		cannotConclude: "ATB cannot determine from these records alone why execution followed the denial.",
	},
	"action_failed": {
		severity:       "medium",
		title:          "Action error recorded",
		detail:         "An ai.action.error was recorded: an attempted action did not complete successfully. Review the error event for the failure class.",
		canConclude:    "The producer recorded an action error in the captured evidence.",
		cannotConclude: "ATB cannot independently establish the underlying system state from the producer assertion alone.",
	},
	"unresolved_identity": {
		severity:       "medium",
		title:          "Unresolved actor identity",
		detail:         "The session actor is an unattributed API key; the human or service principal behind it is not recorded, so actions cannot be attributed to a person.",
		canConclude:    "The captured actor metadata does not resolve the API key to a person or service principal.",
		cannotConclude: "ATB cannot establish real-world identity from caller-supplied identifiers alone.",
	},
	"session_not_closed": {
		severity:       "low",
		title:          "Session not closed",
		detail:         "No atb.session.close event was recorded. The capture may be truncated, interrupted, or the session may still be open — treat the event list as possibly incomplete.",
		canConclude:    "No session-close event exists in the captured evidence.",
		cannotConclude: "ATB cannot distinguish an open session from uncaptured or lost close evidence.",
	},
	"load_error": {
		severity:       "high",
		title:          "Bundle could not be verified",
		detail:         "The bundle failed to load or verify cleanly during indexing. Its contents may be unreliable.",
		canConclude:    "ATB could not establish integrity for the presented bundle.",
		cannotConclude: "ATB cannot use an integrity failure to infer what was or was not captured.",
	},
}

// buildFindings turns the session index's anomaly flags into explained,
// located findings. The flags argument is authoritative: a finding is produced
// only for a flag the index actually raised. The triggering sequence numbers
// are located from the session's own events; if none can be pinned (e.g. a
// session-level flag), the finding still stands on the flag alone.
func buildFindings(flags []string, events []scopedEvent) []Finding {
	if len(flags) == 0 {
		return nil
	}
	triggers := locateTriggers(events)
	out := make([]Finding, 0, len(flags))
	for _, flag := range flags {
		meta, ok := flagExplanations[flag]
		if !ok {
			out = append(out, Finding{
				Flag:               flag,
				Severity:           "unknown",
				Title:              flag,
				Detail:             "Flagged by the session index; no detailed explanation is registered for this flag.",
				Basis:              "session index anomaly flag",
				Boundedness:        "inconclusive",
				WhatCanConclude:    "The session index emitted this anomaly flag.",
				WhatCannotConclude: "No stronger conclusion is registered for this flag.",
			})
			continue
		}
		out = append(out, Finding{
			Flag:               flag,
			Severity:           meta.severity,
			Title:              meta.title,
			Detail:             meta.detail,
			Basis:              "captured event sequence and session-index rule",
			RuleID:             "session_index:" + flag,
			Boundedness:        "bounded",
			WhatCanConclude:    meta.canConclude,
			WhatCannotConclude: meta.cannotConclude,
			EventSeqs:          triggers[flag],
		})
	}
	return out
}

// BuildBundleFindings returns explained findings for every indexed session in
// a bundle. Session index flags remain authoritative; this function only binds
// them to their supporting captured records and bounded language.
func BuildBundleFindings(b *bundle.Bundle, sessions []sessionindex.SessionEntry) []Finding {
	if b == nil || len(sessions) == 0 {
		return []Finding{}
	}
	findings := []Finding{}
	for _, session := range sessions {
		scoped := []scopedEvent{}
		for _, record := range b.Records {
			data, _ := record.Event.Data.(map[string]any)
			if sessionID(data) != session.SessionID {
				continue
			}
			scoped = append(scoped, scopedEvent{
				seq:  record.Event.Sequence,
				typ:  record.Event.Type,
				data: data,
			})
		}
		for _, finding := range buildFindings(session.AnomalyFlags, scoped) {
			finding.SessionID = session.SessionID
			finding.ProfileID = session.InferredProfile
			findings = append(findings, finding)
		}
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].SessionID == findings[j].SessionID {
			return findings[i].Flag < findings[j].Flag
		}
		return findings[i].SessionID < findings[j].SessionID
	})
	return findings
}

func sessionID(data map[string]any) string {
	for _, key := range []string{"session_id", "sessionId"} {
		if value, ok := data[key].(string); ok {
			if value = strings.TrimSpace(value); value != "" {
				return value
			}
		}
	}
	return ""
}

// locateTriggers walks the session's events once and returns, per flag, the
// sequence numbers of the events that triggered it. The detection conditions
// here mirror sessionindex.applyEvent so the located events match the flags the
// index raised; the index remains authoritative on whether a flag fires at all.
func locateTriggers(events []scopedEvent) map[string][]int {
	triggers := map[string][]int{}
	approvalSeen := false
	denied := map[string]bool{}

	add := func(flag string, seq int) { triggers[flag] = append(triggers[flag], seq) }

	for _, e := range events {
		switch e.typ {
		case "atb.human.approval":
			approvalSeen = true
		case "atb.tool.call":
			if !approvalSeen {
				add("tool_without_approval", e.seq)
			}
		case "ai.policy.decision":
			if str(e.data, "decision") == "deny" {
				if id := str(e.data, "action_id"); id != "" {
					denied[id] = true
				}
			}
		case "ai.action.executed", "ai.action.committed":
			if id := str(e.data, "action_id"); id != "" && denied[id] {
				add("policy_denied_executed", e.seq)
			}
		case "ai.action.error":
			add("action_failed", e.seq)
		}
	}
	for flag := range triggers {
		sort.Ints(triggers[flag])
	}
	return triggers
}

// triggeredFlagsBySeq inverts located triggers into a per-sequence map, so an
// NDJSON line can carry the specific flags that the event triggered (useful for
// SIEM alerting on the exact offending record rather than the whole session).
func triggeredFlagsBySeq(findings []Finding) map[int][]string {
	bySeq := map[int][]string{}
	for _, f := range findings {
		for _, seq := range f.EventSeqs {
			bySeq[seq] = append(bySeq[seq], f.Flag)
		}
	}
	for seq := range bySeq {
		sort.Strings(bySeq[seq])
	}
	return bySeq
}

func str(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	if s, ok := data[key].(string); ok {
		return s
	}
	return ""
}
