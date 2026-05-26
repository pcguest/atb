// SPDX-License-Identifier: MIT
package github_test

import (
	"testing"

	"github.com/pcguest/atb/pkg/corroborate/github"
)

func TestNewGitHubCorroborator(t *testing.T) {
	t.Parallel()
	c := github.NewGitHubCorroborator("pcguest")
	if c == nil {
		t.Fatal("NewGitHubCorroborator() returned nil")
	}
	if c.Organisation != "pcguest" {
		t.Fatalf("Organisation = %q, want pcguest", c.Organisation)
	}
}
