// SPDX-License-Identifier: MIT
package github_test

import (
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/pkg/corroborate"
	"github.com/pcguest/atb/pkg/corroborate/github"
)

func TestVerifyConstructsAuditLogURL(t *testing.T) {
	t.Parallel()
	// A fixed timestamp keeps the created date deterministic.
	occurredAt := time.Date(2024, 3, 20, 14, 30, 0, 0, time.UTC)
	// The configured API URL must be preserved as the query URL prefix.
	auditLogURL := "https://api.github.com"
	// The corroborator only constructs URLs and performs no live lookup.
	c := github.NewGitHubCorroborator(auditLogURL, "pcguest")

	// Email takes precedence over display name for actor lookup.
	result, err := c.Verify(&corroborate.Event{
		Type:       "atb.tool.call",
		OccurredAt: occurredAt,
		Actor: corroborate.Actor{
			Email:       "dev@example.com",
			DisplayName: "Dev User",
		},
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.Source != "github" {
		t.Fatalf("Source = %q, want github", result.Source)
	}
	if !strings.HasPrefix(result.QueryURL, auditLogURL) {
		t.Fatalf("QueryURL = %q, want prefix %q", result.QueryURL, auditLogURL)
	}
	parsed, err := url.Parse(result.QueryURL)
	if err != nil {
		t.Fatalf("parse QueryURL: %v", err)
	}
	if phrase := parsed.Query().Get("phrase"); !strings.Contains(phrase, "dev@example.com") {
		t.Fatalf("phrase = %q, want dev@example.com", phrase)
	}
	if !strings.Contains(result.QueryURL, "2024-03-20") {
		t.Fatalf("QueryURL = %q, want date 2024-03-20", result.QueryURL)
	}
}

func TestVerifyActorFallbackToDisplayName(t *testing.T) {
	t.Parallel()
	// Display name is the fallback when no email is recorded.
	c := github.NewGitHubCorroborator("https://api.github.com", "pcguest")

	// Empty email should still produce a usable actor phrase from display name.
	result, err := c.Verify(&corroborate.Event{
		Type:       "atb.tool.call",
		OccurredAt: time.Date(2024, 3, 20, 14, 30, 0, 0, time.UTC),
		Actor: corroborate.Actor{
			DisplayName: "bot-agent",
		},
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	parsed, err := url.Parse(result.QueryURL)
	if err != nil {
		t.Fatalf("parse QueryURL: %v", err)
	}
	if phrase := parsed.Query().Get("phrase"); !strings.Contains(phrase, "bot-agent") {
		t.Fatalf("phrase = %q, want bot-agent", phrase)
	}
}

func TestVerifyEmptyActor(t *testing.T) {
	t.Parallel()
	// GitHub audit-log lookup needs at least one actor identifier.
	c := github.NewGitHubCorroborator("https://api.github.com", "pcguest")

	// Empty actor fields must be rejected rather than producing a broad query.
	_, err := c.Verify(&corroborate.Event{
		Type:       "atb.tool.call",
		OccurredAt: time.Date(2024, 3, 20, 14, 30, 0, 0, time.UTC),
		Actor:      corroborate.Actor{},
	})
	if err == nil {
		t.Fatal("expected empty actor error")
	}
}

func TestVerifyUnsupportedTypeGitHub(t *testing.T) {
	t.Parallel()
	// Session lifecycle events are not GitHub audit-log corroboration targets.
	c := github.NewGitHubCorroborator("https://api.github.com", "pcguest")

	// Unsupported types must return the shared sentinel for errors.Is checks.
	_, err := c.Verify(&corroborate.Event{
		Type:       "atb.session.open",
		OccurredAt: time.Date(2024, 3, 20, 14, 30, 0, 0, time.UTC),
		Actor: corroborate.Actor{
			Email: "dev@example.com",
		},
	})
	if !errors.Is(err, corroborate.ErrEventTypeNotSupported) {
		t.Fatalf("Verify error = %v, want ErrEventTypeNotSupported", err)
	}
}

func TestVerifyPolicyDecision(t *testing.T) {
	t.Parallel()
	// Policy decision events are supported GitHub corroboration targets.
	c := github.NewGitHubCorroborator("https://api.github.com", "pcguest")

	// A supported policy decision event should construct a query URL without error.
	_, err := c.Verify(&corroborate.Event{
		Type:       "atb.policy.decision",
		OccurredAt: time.Date(2024, 3, 20, 14, 30, 0, 0, time.UTC),
		Actor: corroborate.Actor{
			Email: "dev@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
}
