// SPDX-License-Identifier: MIT
package langchain_test

import (
	"testing"

	"github.com/pcguest/atb/pkg/corroborate/langchain"
)

func TestNewLangChainCorroborator(t *testing.T) {
	t.Parallel()
	c := langchain.NewLangChainCorroborator("atb-demo")
	if c == nil {
		t.Fatal("NewLangChainCorroborator() returned nil")
	}
	if c.ProjectName != "atb-demo" {
		t.Fatalf("ProjectName = %q, want atb-demo", c.ProjectName)
	}
}
