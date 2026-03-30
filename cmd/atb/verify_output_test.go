package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

func TestRunVerify_OutputModes(t *testing.T) {
	tests := []struct {
		name        string
		args        func(path string) []string
		build       func(testing.TB) *bundle.Bundle
		setNoColour bool
		wantExit    int
		wantOutput  []string
		wantQuiet   bool
		wantJSON    bool
		wantNoANSI  bool
	}{
		{
			name:     "pass bundle",
			build:    buildCLIPrivilegedToolActionBundle,
			wantExit: exitSuccess,
			args: func(path string) []string {
				return []string{"--bundle", path}
			},
			wantOutput: []string{"ATB Verification Report", "Verification: PASS"},
			wantNoANSI: true,
		},
		{
			name:     "fail bundle",
			wantExit: exitVerifyFailure,
			build: func(t testing.TB) *bundle.Bundle {
				t.Helper()

				b := bundle.New()
				appendTestBundleEvent(t, b, "ai.request.received", map[string]any{
					"request_id":    "req-1",
					"actor_id_hash": "actor-hash",
					"purpose_tag":   "approve-change",
				})
				appendTestBundleEvent(t, b, "ai.action.precommit", map[string]any{
					"action_id":                "act-1",
					"action_type":              "deploy_change",
					"action_parameters_digest": "params-digest",
					"target_resource_id":       "svc-1",
					"intended_effect":          "deploy build 42",
				})
				return b
			},
			args: func(path string) []string {
				return []string{"--bundle", path}
			},
			wantOutput: []string{"Verification: FAIL", "ai.policy.decision"},
			wantNoANSI: true,
		},
		{
			name:     "json output unchanged",
			build:    buildCLIPrivilegedToolActionBundle,
			wantExit: exitSuccess,
			args: func(path string) []string {
				return []string{"--bundle", path, "--json"}
			},
			wantJSON:   true,
			wantNoANSI: true,
		},
		{
			name:     "quiet output suppressed",
			build:    buildCLIPrivilegedToolActionBundle,
			wantExit: exitSuccess,
			args: func(path string) []string {
				return []string{"--bundle", path, "--quiet"}
			},
			wantQuiet: true,
		},
		{
			name:        "no colour env",
			build:       buildCLIPrivilegedToolActionBundle,
			wantExit:    exitSuccess,
			setNoColour: true,
			args: func(path string) []string {
				return []string{"--bundle", path}
			},
			wantOutput: []string{"Verification: PASS"},
			wantNoANSI: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setNoColour {
				t.Setenv("NO_COLOR", "1")
			}

			path := writeVerifyTestBundle(t, tc.build(t))
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := runVerify(tc.args(path), &stdout, &stderr)
			if exitCode != tc.wantExit {
				t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, tc.wantExit, stderr.String())
			}

			if tc.wantQuiet {
				if stdout.Len() != 0 || stderr.Len() != 0 {
					t.Fatalf("expected no output in quiet mode, got stdout=%q stderr=%q", stdout.String(), stderr.String())
				}
				return
			}

			if tc.wantJSON {
				var report verifypkg.Report
				if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
					t.Fatalf("expected valid JSON output: %v", err)
				}
			}

			out := stdout.String()
			for _, want := range tc.wantOutput {
				if !strings.Contains(out, want) {
					t.Fatalf("expected output to contain %q, got %q", want, out)
				}
			}
			if tc.wantNoANSI && strings.Contains(out, "\x1b[") {
				t.Fatalf("expected no ANSI escape sequences, got %q", out)
			}
		})
	}
}
