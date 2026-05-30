// SPDX-License-Identifier: MIT
// Package langchain constructs LangSmith corroboration lookup URLs for ATB events.
package langchain

import (
	"net/url"
	"strings"
	"time"

	"github.com/pcguest/atb/pkg/corroborate"
)

// Event is the shared adapter-facing ATB event type.
type Event = corroborate.Event

// CorrobResult is the shared corroboration result type.
type CorrobResult = corroborate.CorrobResult

// LangChainCorroborator maps ATB events to LangSmith run queries.
type LangChainCorroborator struct {
	// LangSmithAPIURL is the base URL for the LangSmith API.
	LangSmithAPIURL string
	// ProjectName is the LangSmith project used when resolving runs.
	ProjectName string
}

// NewLangChainCorroborator returns a LangSmith query URL corroborator.
func NewLangChainCorroborator(
	langsmithAPIURL string,
	projectName string,
) *LangChainCorroborator {
	// Added: The constructor stores only local query context and performs no network work.
	return &LangChainCorroborator{
		LangSmithAPIURL: langsmithAPIURL,
		ProjectName:     projectName,
	}
}

// Verify constructs a LangSmith run lookup URL without making an HTTP request.
func (c *LangChainCorroborator) Verify(event *Event) (CorrobResult, error) {
	// Added: Nil inputs cannot be mapped to a supported ATB event type.
	if c == nil || event == nil {
		return CorrobResult{}, corroborate.ErrEventTypeNotSupported
	}
	// Added: Unsupported event types are rejected before query construction.
	if !supportsEventType(event.Type) {
		return CorrobResult{}, corroborate.ErrEventTypeNotSupported
	}
	// Added: Tool names provide the closest LangSmith run-name lookup when present.
	runName := event.Type
	if event.Metadata != nil {
		if toolName := event.Metadata["tool_name"]; toolName != "" {
			runName = toolName
		}
	}

	// Added: url.Values safely escapes every query parameter value.
	values := url.Values{}
	values.Set("project_name", c.ProjectName)
	values.Set("filter", `eq(name,"`+runName+`")`)
	values.Set("start_time", event.OccurredAt.UTC().Format(time.RFC3339))

	// Added: Only the base path is joined directly; query parameters come from url.Values.Encode.
	baseURL := strings.TrimRight(c.LangSmithAPIURL, "/") + "/api/v1/runs"
	return CorrobResult{
		Matched:   false,
		Source:    "langchain",
		Timestamp: event.OccurredAt,
		Note:      "query URL constructed; live HTTP lookup deferred",
		QueryURL:  baseURL + "?" + values.Encode(),
	}, nil
}

// supportsEventType reports whether LangChain can map the ATB event type.
func supportsEventType(eventType string) bool {
	// Added: Retrieval events are a namespace and therefore use a prefix match.
	if strings.HasPrefix(eventType, "atb.retrieval.") {
		return true
	}
	switch eventType {
	case "atb.tool.call", "atb.data.export":
		return true
	default:
		return false
	}
}
