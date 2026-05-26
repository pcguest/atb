// SPDX-License-Identifier: MIT
// Package github defines the Phase 9 scaffold for corroborating ATB bundle events
// against GitHub audit log or webhook payloads.
package github

import (
	"errors"
	"time"

	"github.com/pcguest/atb/internal/event"
)

// ErrNotImplemented indicates the GitHub corroborator is scaffold-only.
var ErrNotImplemented = errors.New("corroborate/github: verification not implemented")

// CorrobResult reports whether an external source matched a bundle event.
type CorrobResult struct {
	Matched   bool
	Source    string
	Timestamp time.Time
	Note      string
}

// Corroborator verifies an ATB event against an external GitHub evidence source.
type Corroborator interface {
	Verify(ev *event.Event) (CorrobResult, error)
}

// GitHubCorroborator checks events against GitHub audit log or webhook data.
// Phase 9 scaffold only: no HTTP calls to GitHub APIs.
type GitHubCorroborator struct {
	// Organisation is the GitHub org slug used when resolving audit log entries.
	Organisation string
	// WebhookSecretRef names a local secret reference for webhook signature checks (future).
	WebhookSecretRef string
}

// NewGitHubCorroborator returns a corroborator with the given organisation context.
func NewGitHubCorroborator(organisation string) *GitHubCorroborator {
	return &GitHubCorroborator{Organisation: organisation}
}

// Verify implements Corroborator.
func (c *GitHubCorroborator) Verify(ev *event.Event) (CorrobResult, error) {
	if ev == nil {
		return CorrobResult{}, errors.New("corroborate/github: event is nil")
	}
	_ = c.Organisation
	return CorrobResult{
		Matched:   false,
		Source:    "github",
		Timestamp: time.Time{},
		Note:      "scaffold: verification not implemented",
	}, ErrNotImplemented
}
