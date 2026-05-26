// SPDX-License-Identifier: MIT
// Package langchain defines the Phase 9 scaffold for corroborating ATB bundle events
// against LangSmith run logs or LangGraph execution traces.
package langchain

import (
	"errors"
	"time"

	"github.com/pcguest/atb/internal/event"
)

// ErrNotImplemented indicates the LangChain corroborator is scaffold-only.
var ErrNotImplemented = errors.New("corroborate/langchain: verification not implemented")

// CorrobResult reports whether an external source matched a bundle event.
type CorrobResult struct {
	Matched   bool
	Source    string
	Timestamp time.Time
	Note      string
}

// Corroborator verifies an ATB event against an external LangChain or LangGraph evidence source.
type Corroborator interface {
	Verify(ev *event.Event) (CorrobResult, error)
}

// LangChainCorroborator checks events against LangSmith or LangGraph trace data.
// Phase 9 scaffold only: no HTTP calls to LangSmith APIs.
type LangChainCorroborator struct {
	// ProjectName is the LangSmith project or chain name used when resolving runs.
	ProjectName string
	// EndpointRef names a future LangSmith API endpoint reference (local config key).
	EndpointRef string
}

// NewLangChainCorroborator returns a corroborator with the given project context.
func NewLangChainCorroborator(projectName string) *LangChainCorroborator {
	return &LangChainCorroborator{ProjectName: projectName}
}

// Verify implements Corroborator.
func (c *LangChainCorroborator) Verify(ev *event.Event) (CorrobResult, error) {
	if ev == nil {
		return CorrobResult{}, errors.New("corroborate/langchain: event is nil")
	}
	_ = c.ProjectName
	return CorrobResult{
		Matched:   false,
		Source:    "langchain",
		Timestamp: time.Time{},
		Note:      "scaffold: LangChain/LangGraph verification not implemented",
	}, ErrNotImplemented
}
