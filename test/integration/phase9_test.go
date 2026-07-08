//go:build integration

// SPDX-License-Identifier: MIT
package integration_test

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/pkg/corroborate"
	corrobgithub "github.com/pcguest/atb/pkg/corroborate/github"
	"github.com/pcguest/atb/pkg/otel"
)

func TestPhase9OTelTranslationAppendsVerifiableBundleEvent(t *testing.T) {
	start := time.Date(2026, 3, 9, 9, 15, 2, 0, time.UTC)
	translated, err := otel.Translate(otel.OTelSpan{
		TraceID:   "0102030405060708090a0b0c0d0e0f10",
		SpanID:    "0102030405060708",
		Name:      "gen_ai.chat",
		StartTime: start,
		Attributes: map[string]any{
			"gen_ai.system":        "openai",
			"gen_ai.request.model": "gpt-4.1-mini",
			"atb.phase":            "start",
		},
	})
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	b, err := bundle.New()
	if err != nil {
		t.Fatalf("bundle.New: %v", err)
	}
	if err := b.AppendWithOptions(translated.Type, translated.Data, &bundle.AppendOptions{
		Timestamp:    translated.Timestamp,
		TraceID:      translated.TraceID,
		SpanID:       translated.SpanID,
		ParentSpanID: translated.ParentSpanID,
	}); err != nil {
		t.Fatalf("append translated event: %v", err)
	}
	if err := b.Verify(); err != nil {
		t.Fatalf("verify translated bundle: %v", err)
	}

	got := b.Records[len(b.Records)-1].Event
	if got.Type != event.TypeAILLMCall {
		t.Fatalf("event type = %q, want %q", got.Type, event.TypeAILLMCall)
	}
	if got.TraceID != translated.TraceID || got.SpanID != translated.SpanID {
		t.Fatalf("trace fields = %q/%q, want %q/%q", got.TraceID, got.SpanID, translated.TraceID, translated.SpanID)
	}
}

func TestPhase9GitHubCorroboratorConstructsAuditLogQuery(t *testing.T) {
	c := corrobgithub.NewGitHubCorroborator("https://github.test", "pcguest")
	occurredAt := time.Date(2026, 5, 28, 9, 2, 3, 0, time.UTC)

	got, err := c.Verify(&corroborate.Event{
		Type:       event.TypeToolCall,
		OccurredAt: occurredAt,
		Actor: corroborate.Actor{
			Email:       "octocat@example.com",
			DisplayName: "octocat",
		},
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if got.Matched {
		t.Fatal("Matched = true, want false for deferred live lookup")
	}
	if got.Source != "github" {
		t.Fatalf("Source = %q, want github", got.Source)
	}
	parsed, err := url.Parse(got.QueryURL)
	if err != nil {
		t.Fatalf("parse query URL: %v", err)
	}
	if parsed.Path != "/enterprises/pcguest/audit-log" {
		t.Fatalf("path = %q, want /enterprises/pcguest/audit-log", parsed.Path)
	}
	if phrase := parsed.Query().Get("phrase"); !strings.Contains(phrase, "actor:octocat@example.com") {
		t.Fatalf("phrase = %q, want actor email", phrase)
	}
	if created := parsed.Query().Get("created"); created != "2026-05-28" {
		t.Fatalf("created = %q, want 2026-05-28", created)
	}
}
