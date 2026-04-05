package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

func TestRunInspect(t *testing.T) {
	tests := []struct {
		name       string
		build      func(testing.TB) *bundle.Bundle
		args       func(path string) []string
		path       func(t *testing.T) string
		wantExit   int
		wantStderr []string
		validate   func(t *testing.T, stdout string, records []bundle.Record)
	}{
		{
			name:     "default output contains rows and truncated data",
			build:    buildInspectTestBundle,
			wantExit: exitSuccess,
			args: func(path string) []string {
				return []string{"--bundle", path}
			},
			validate: func(t *testing.T, stdout string, records []bundle.Record) {
				t.Helper()

				if !strings.Contains(stdout, "SEQ  TYPE") {
					t.Fatalf("expected table header, got %q", stdout)
				}
				for _, record := range records {
					if !strings.Contains(stdout, fmt.Sprintf("%d", record.Event.Sequence)) {
						t.Fatalf("expected sequence %d in output %q", record.Event.Sequence, stdout)
					}
					if !strings.Contains(stdout, record.Event.Type) {
						t.Fatalf("expected event type %q in output %q", record.Event.Type, stdout)
					}
				}
				if !strings.Contains(stdout, "...") {
					t.Fatalf("expected truncated preview in output %q", stdout)
				}
				if strings.Contains(stdout, strings.Repeat("b", 96)) {
					t.Fatalf("expected signature preview to be truncated, got %q", stdout)
				}
			},
		},
		{
			name:     "json output emits record array",
			build:    buildInspectTestBundle,
			wantExit: exitSuccess,
			args: func(path string) []string {
				return []string{"--bundle", path, "--json"}
			},
			validate: func(t *testing.T, stdout string, records []bundle.Record) {
				t.Helper()

				var got []bundle.Record
				if err := json.Unmarshal([]byte(stdout), &got); err != nil {
					t.Fatalf("expected valid JSON array: %v", err)
				}
				if len(got) != len(records) {
					t.Fatalf("unexpected record count: got %d want %d", len(got), len(records))
				}
			},
		},
		{
			name:     "seq emits full data for record",
			build:    buildInspectTestBundle,
			wantExit: exitSuccess,
			args: func(path string) []string {
				return []string{"--bundle", path, "--seq", "0"}
			},
			validate: func(t *testing.T, stdout string, _ []bundle.Record) {
				t.Helper()

				if !strings.Contains(stdout, "\"bundle_id\"") {
					t.Fatalf("expected manifest data in output %q", stdout)
				}
				if strings.Contains(stdout, "SEQ  TYPE") {
					t.Fatalf("expected record data output, got table %q", stdout)
				}
			},
		},
		{
			name:     "seq out of range",
			build:    buildInspectTestBundle,
			wantExit: exitUserError,
			args: func(path string) []string {
				return []string{"--bundle", path, "--seq", "99"}
			},
			wantStderr: []string{"out of range"},
		},
		{
			name:     "bundle not found",
			wantExit: exitUserError,
			path: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "missing.atb")
			},
			args: func(path string) []string {
				return []string{"--bundle", path}
			},
			wantStderr: []string{"atb inspect:", "bundle: load: open:"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var records []bundle.Record
			path := ""
			if tc.path != nil {
				path = tc.path(t)
			} else {
				b := tc.build(t)
				records = append(records, b.Records...)
				path = writeVerifyTestBundle(t, b)
			}

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := runInspect(tc.args(path), &stdout, &stderr)
			if exitCode != tc.wantExit {
				t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, tc.wantExit, stderr.String())
			}

			for _, want := range tc.wantStderr {
				if !strings.Contains(stderr.String(), want) {
					t.Fatalf("expected stderr to contain %q, got %q", want, stderr.String())
				}
			}

			if tc.validate != nil {
				tc.validate(t, stdout.String(), records)
			}
		})
	}
}

func buildInspectTestBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b := newTestBundle(t)
	appendTestBundleEvent(t, b, event.TypeDevSession, map[string]any{
		"note": strings.Repeat("test-note-", 12),
	})
	appendTestBundleEvent(t, b, event.TypeBundleSignature, map[string]string{
		"algorithm":   "ed25519",
		"pubkey":      strings.Repeat("a", 44),
		"signature":   strings.Repeat("b", 96),
		"bundle_hash": strings.Repeat("c", 64),
	})
	return b
}
