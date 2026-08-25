// SPDX-License-Identifier: MIT
// Package github constructs GitHub Audit Log corroboration lookup URLs for ATB events.
package github

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/pcguest/atb/pkg/corroborate"
)

// Event is the shared adapter-facing ATB event type.
type Event = corroborate.Event

// CorrobResult is the shared corroboration result type.
type CorrobResult = corroborate.CorrobResult

// GitHubCorroborator maps ATB events to GitHub Audit Log queries.
type GitHubCorroborator struct {
	// GitHubAuditLogURL is the base URL of the GitHub Audit Log API
	// (e.g. https://api.github.com).
	GitHubAuditLogURL string
	// Org is the GitHub enterprise or organisation slug used in audit-log paths.
	Org string
}

// NewGitHubCorroborator returns a GitHub Audit Log query URL corroborator.
func NewGitHubCorroborator(
	auditLogURL string,
	org string,
) *GitHubCorroborator {
	// The constructor stores only local query context and performs no network work.
	return &GitHubCorroborator{
		GitHubAuditLogURL: auditLogURL,
		Org:               org,
	}
}

// Verify constructs a GitHub Audit Log lookup URL without making an HTTP request.
func (c *GitHubCorroborator) Verify(event *Event) (CorrobResult, error) {
	// Nil inputs cannot be mapped to a supported ATB event type.
	if c == nil || event == nil {
		return CorrobResult{}, corroborate.ErrEventTypeNotSupported
	}
	// Unsupported event types are rejected before query construction.
	if !supportsEventType(event.Type) {
		return CorrobResult{}, corroborate.ErrEventTypeNotSupported
	}
	// Email is the strongest actor lookup value when the event captured it.
	actorIdent := event.Actor.Email
	if actorIdent == "" {
		actorIdent = event.Actor.DisplayName
	}
	if actorIdent == "" {
		return CorrobResult{}, errors.New("event actor has no identifiable field")
	}

	// url.Parse keeps the configured API base URL structurally valid.
	parsedURL, err := url.Parse(c.GitHubAuditLogURL)
	if err != nil {
		return CorrobResult{}, fmt.Errorf("parse github audit log url: %w", err)
	}
	// PathEscape prevents organisation names from altering the audit-log path structure.
	parsedURL.Path = fmt.Sprintf(
		"%s/enterprises/%s/audit-log",
		parsedURL.Path,
		url.PathEscape(c.Org),
	)
	// url.Values safely escapes every query parameter value.
	values := url.Values{}
	values.Set("phrase", "actor:"+actorIdent)
	// UTC date formatting is centralised so audit-log queries stay timezone-stable.
	values.Set("created", utcDate(event.OccurredAt))
	values.Set("include", "all")
	parsedURL.RawQuery = values.Encode()

	return CorrobResult{
		Matched:   false,
		Source:    "github",
		Timestamp: event.OccurredAt,
		Note:      "query URL constructed; live HTTP lookup deferred",
		QueryURL:  parsedURL.String(),
	}, nil
}

// supportsEventType reports whether GitHub can map the ATB event type.
func supportsEventType(eventType string) bool {
	switch eventType {
	case "atb.tool.call", "atb.data.export", "atb.policy.decision", "atb.human.override":
		return true
	default:
		return false
	}
}

// utcDate formats timestamps for the GitHub Audit Log created filter.
func utcDate(timestamp time.Time) string {
	// UTC date matching avoids local timezone drift in audit-log queries.
	return timestamp.UTC().Format("2006-01-02")
}
