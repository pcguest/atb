//go:build integration

// SPDX-License-Identifier: MIT
package integration_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
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

func TestPhase9GitHubCorroboratorParsesAuditLogResponse(t *testing.T) {
	client := &http.Client{Transport: phase9RoundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/orgs/pcguest/audit-log" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("phrase"); got != "doc-123" {
			t.Fatalf("phrase = %q, want doc-123", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"X-Ratelimit-Limit":     []string{"5000"},
				"X-Ratelimit-Remaining": []string{"4998"},
			},
			Body: io.NopCloser(strings.NewReader(`[
			{
				"_document_id": "doc-123",
				"action": "repo.destroy",
				"actor": "octocat",
				"repo": "pcguest/atb",
				"created_at": 1779930123000
			}
		]`)),
		}, nil
	})}

	c := corrobgithub.NewGitHubCorroborator(
		"token-ref",
		corrobgithub.WithOrganisation("pcguest"),
		corrobgithub.WithBaseURL("https://github.test"),
		corrobgithub.WithHTTPClient(client),
	)
	got, err := c.Verify(&event.Event{
		Type: event.TypeAIActionExecuted,
		Data: map[string]any{
			"document_id": "doc-123",
		},
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !got.Matched {
		t.Fatal("Matched = false, want true")
	}
	if got.DocumentID != "doc-123" || got.Action != "repo.destroy" {
		t.Fatalf("result = %+v", got)
	}
	if got.RateLimitRemaining != 4998 {
		t.Fatalf("RateLimitRemaining = %d, want 4998", got.RateLimitRemaining)
	}
}

type phase9RoundTripFunc func(*http.Request) (*http.Response, error)

func (f phase9RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
