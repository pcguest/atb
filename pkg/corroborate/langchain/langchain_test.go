// SPDX-License-Identifier: MIT
package langchain_test

import (
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/pkg/corroborate"
	"github.com/pcguest/atb/pkg/corroborate/langchain"
)

func TestVerifyToolCall(t *testing.T) {
	t.Parallel()
	// A fixed timestamp keeps the expected query URL deterministic.
	occurredAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	// The configured API URL and project name are surfaced in the query URL.
	apiURL := "https://api.smith.langchain.com"
	// The project name locks the LangSmith project query parameter.
	projectName := "atb-demo"
	// The corroborator only constructs URLs and performs no live lookup.
	c := langchain.NewLangChainCorroborator(apiURL, projectName)

	// Tool calls use the tool_name metadata value as the LangSmith run name.
	result, err := c.Verify(&corroborate.Event{
		Type:       "atb.tool.call",
		OccurredAt: occurredAt,
		Metadata:   map[string]string{"tool_name": "read_file"},
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.Source != "langchain" {
		t.Fatalf("Source = %q, want langchain", result.Source)
	}
	if result.Matched {
		t.Fatal("Matched = true, want false")
	}
	if !strings.Contains(result.QueryURL, "read_file") {
		t.Fatalf("QueryURL = %q, want read_file", result.QueryURL)
	}
	if !strings.Contains(result.QueryURL, projectName) {
		t.Fatalf("QueryURL = %q, want project name %q", result.QueryURL, projectName)
	}
	if !strings.HasPrefix(result.QueryURL, apiURL) {
		t.Fatalf("QueryURL = %q, want prefix %q", result.QueryURL, apiURL)
	}
}

func TestVerifyToolCallFallbackName(t *testing.T) {
	t.Parallel()
	// Empty metadata exercises the event-type fallback run name.
	c := langchain.NewLangChainCorroborator("https://api.smith.langchain.com", "atb-demo")

	// The event type becomes the run name when tool_name is absent.
	result, err := c.Verify(&corroborate.Event{
		Type:       "atb.tool.call",
		OccurredAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Metadata:   map[string]string{},
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !strings.Contains(result.QueryURL, "atb.tool.call") {
		t.Fatalf("QueryURL = %q, want event type fallback", result.QueryURL)
	}
}

func TestVerifyRetrievalPrefix(t *testing.T) {
	t.Parallel()
	// Retrieval events are supported by namespace prefix.
	c := langchain.NewLangChainCorroborator("https://api.smith.langchain.com", "atb-demo")

	// Vector-search retrieval proves prefix matching rather than exact matching.
	result, err := c.Verify(&corroborate.Event{
		Type:       "atb.retrieval.vector_search",
		OccurredAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	parsed, err := url.Parse(result.QueryURL)
	if err != nil {
		t.Fatalf("parse QueryURL: %v", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		t.Fatalf("QueryURL is not well-formed: %q", result.QueryURL)
	}
}

func TestVerifyDataExport(t *testing.T) {
	t.Parallel()
	// Data export events are one of the supported LangChain lookup classes.
	c := langchain.NewLangChainCorroborator("https://api.smith.langchain.com", "atb-demo")

	// A supported data export event should construct a query URL without error.
	_, err := c.Verify(&corroborate.Event{
		Type:       "atb.data.export",
		OccurredAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestVerifyUnsupportedType(t *testing.T) {
	t.Parallel()
	// Session lifecycle events are not LangSmith run lookup events.
	c := langchain.NewLangChainCorroborator("https://api.smith.langchain.com", "atb-demo")

	// Unsupported types must return the shared sentinel for errors.Is checks.
	_, err := c.Verify(&corroborate.Event{
		Type:       "atb.session.open",
		OccurredAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, corroborate.ErrEventTypeNotSupported) {
		t.Fatalf("Verify error = %v, want ErrEventTypeNotSupported", err)
	}
}

func TestQueryURLEncoding(t *testing.T) {
	t.Parallel()
	// Special characters in run names must be query-escaped by url.Values.
	c := langchain.NewLangChainCorroborator("https://api.smith.langchain.com", "project with spaces")

	// The tool name contains characters that would break manual query strings.
	result, err := c.Verify(&corroborate.Event{
		Type:       "atb.tool.call",
		OccurredAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Metadata:   map[string]string{"tool_name": "tool with spaces & special=chars"},
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if strings.Contains(result.QueryURL, "tool with spaces") {
		t.Fatalf("QueryURL contains raw spaces: %q", result.QueryURL)
	}
	if strings.Contains(result.QueryURL, "spaces &") {
		t.Fatalf("QueryURL contains raw ampersand in a value: %q", result.QueryURL)
	}
	if strings.Contains(result.QueryURL, "special=chars") {
		t.Fatalf("QueryURL contains raw equals sign in a value: %q", result.QueryURL)
	}
	parsed, err := url.Parse(result.QueryURL)
	if err != nil {
		t.Fatalf("parse QueryURL: %v", err)
	}
	if got := parsed.Query().Get("filter"); got != `eq(name,"tool with spaces & special=chars")` {
		t.Fatalf("filter = %q", got)
	}
}
