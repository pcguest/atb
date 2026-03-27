package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

func TestParseAppendPayload(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{name: "single json", args: []string{"{\"ok\":true}"}, want: "{\"ok\":true}"},
		{name: "data flag", args: []string{"--data", "{\"ok\":true}"}, want: "{\"ok\":true}"},
		{name: "missing", args: nil, wantErr: true},
		{name: "invalid", args: []string{"--data"}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseAppendPayload(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected payload: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestParseAppendCommandArgs(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantJSON        string
		wantActorID     string
		wantOrgID       string
		wantWorkspaceID string
		wantErr         bool
	}{
		{
			name:            "json positional with metadata flags",
			args:            []string{`{"ok":true}`, "--actor-id", "paddy", "--org-id=pcguest", "--workspace-id", "local"},
			wantJSON:        `{"ok":true}`,
			wantActorID:     "paddy",
			wantOrgID:       "pcguest",
			wantWorkspaceID: "local",
		},
		{
			name:            "data flag",
			args:            []string{"--data", `{"x":1}`, "--actor-id=actor-1"},
			wantJSON:        `{"x":1}`,
			wantActorID:     "actor-1",
			wantOrgID:       "",
			wantWorkspaceID: "",
		},
		{
			name:    "missing actor id",
			args:    []string{"--data", `{"x":1}`, "--actor-id"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"--data", `{"x":1}`, "--wat", "x"},
			wantErr: true,
		},
		{
			name:    "missing payload",
			args:    []string{"--actor-id", "paddy"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseAppendCommandArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.RawJSON != tc.wantJSON {
				t.Fatalf("unexpected json payload: got %q want %q", got.RawJSON, tc.wantJSON)
			}

			gotActor := ""
			if got.Options.ActorID != nil {
				gotActor = *got.Options.ActorID
			}
			gotOrg := ""
			if got.Options.OrgID != nil {
				gotOrg = *got.Options.OrgID
			}
			gotWorkspace := ""
			if got.Options.WorkspaceID != nil {
				gotWorkspace = *got.Options.WorkspaceID
			}
			if gotActor != tc.wantActorID {
				t.Fatalf("unexpected actor_id: got %q want %q", gotActor, tc.wantActorID)
			}
			if gotOrg != tc.wantOrgID {
				t.Fatalf("unexpected org_id: got %q want %q", gotOrg, tc.wantOrgID)
			}
			if gotWorkspace != tc.wantWorkspaceID {
				t.Fatalf("unexpected workspace_id: got %q want %q", gotWorkspace, tc.wantWorkspaceID)
			}
		})
	}
}

func TestNormalizeBundlePath(t *testing.T) {
	tmp := t.TempDir()
	if got := normalizeBundlePath(tmp); got != filepath.Join(tmp, bundle.BundleFile) {
		t.Fatalf("expected directory path to resolve to bundle file, got %q", got)
	}

	custom := filepath.Join(tmp, "custom.atb")
	if got := normalizeBundlePath(custom); got != custom {
		t.Fatalf("expected file path to remain unchanged, got %q", got)
	}
}

func TestParseVerifyArgs(t *testing.T) {
	tmp := t.TempDir()
	custom := filepath.Join(tmp, "custom.atb")

	tests := []struct {
		name       string
		args       []string
		wantPath   string
		wantFormat string
		wantTrace  bool
		wantErr    bool
	}{
		{
			name:       "defaults",
			args:       nil,
			wantPath:   bundle.DefaultPath(),
			wantFormat: verifyFormatText,
			wantTrace:  false,
		},
		{
			name:       "path only",
			args:       []string{custom},
			wantPath:   custom,
			wantFormat: verifyFormatText,
			wantTrace:  false,
		},
		{
			name:       "json format flag",
			args:       []string{"--format", "json"},
			wantPath:   bundle.DefaultPath(),
			wantFormat: verifyFormatJSON,
			wantTrace:  false,
		},
		{
			name:       "json format equals syntax",
			args:       []string{"--format=json"},
			wantPath:   bundle.DefaultPath(),
			wantFormat: verifyFormatJSON,
			wantTrace:  false,
		},
		{
			name:       "path and format",
			args:       []string{custom, "--format", "json"},
			wantPath:   custom,
			wantFormat: verifyFormatJSON,
			wantTrace:  false,
		},
		{
			name:       "trace flag",
			args:       []string{"--trace"},
			wantPath:   bundle.DefaultPath(),
			wantFormat: verifyFormatText,
			wantTrace:  true,
		},
		{
			name:       "path format and trace",
			args:       []string{custom, "--format", "json", "--trace"},
			wantPath:   custom,
			wantFormat: verifyFormatJSON,
			wantTrace:  true,
		},
		{
			name:    "missing format value",
			args:    []string{"--format"},
			wantErr: true,
		},
		{
			name:    "unknown format",
			args:    []string{"--format", "yaml"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"--wat"},
			wantErr: true,
		},
		{
			name:    "too many paths",
			args:    []string{"one.atb", "two.atb"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, gotFormat, gotTrace, err := parseVerifyArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotPath != tc.wantPath {
				t.Fatalf("unexpected path: got %q want %q", gotPath, tc.wantPath)
			}
			if gotFormat != tc.wantFormat {
				t.Fatalf("unexpected format: got %q want %q", gotFormat, tc.wantFormat)
			}
			if gotTrace != tc.wantTrace {
				t.Fatalf("unexpected trace value: got %v want %v", gotTrace, tc.wantTrace)
			}
		})
	}
}

func TestParseMutationFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantArgs   []string
		wantFormat string
		wantDryRun bool
		wantErr    bool
	}{
		{
			name:       "no flag",
			args:       []string{"snapshot", "--gate", "pass"},
			wantArgs:   []string{"snapshot", "--gate", "pass"},
			wantFormat: verifyFormatText,
			wantDryRun: false,
		},
		{
			name:       "flag at end",
			args:       []string{"snapshot", "--gate", "pass", "--dry-run"},
			wantArgs:   []string{"snapshot", "--gate", "pass"},
			wantFormat: verifyFormatText,
			wantDryRun: true,
		},
		{
			name:       "flag at start",
			args:       []string{"--dry-run", "feature", "{\"ok\":true}"},
			wantArgs:   []string{"feature", "{\"ok\":true}"},
			wantFormat: verifyFormatText,
			wantDryRun: true,
		},
		{
			name:       "json format split",
			args:       []string{"feature", "{\"ok\":true}", "--format", "json"},
			wantArgs:   []string{"feature", "{\"ok\":true}"},
			wantFormat: verifyFormatJSON,
			wantDryRun: false,
		},
		{
			name:       "json format equals",
			args:       []string{"feature", "--format=json", "{\"ok\":true}"},
			wantArgs:   []string{"feature", "{\"ok\":true}"},
			wantFormat: verifyFormatJSON,
			wantDryRun: false,
		},
		{
			name:       "unknown flag passed through",
			args:       []string{"--wat"},
			wantArgs:   []string{"--wat"},
			wantFormat: verifyFormatText,
			wantDryRun: false,
		},
		{
			name:    "missing format value",
			args:    []string{"--format"},
			wantErr: true,
		},
		{
			name:    "invalid format",
			args:    []string{"--format", "yaml"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotArgs, gotFormat, gotDryRun, err := parseMutationFlags(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotDryRun != tc.wantDryRun {
				t.Fatalf("unexpected dry-run: got %v want %v", gotDryRun, tc.wantDryRun)
			}
			if gotFormat != tc.wantFormat {
				t.Fatalf("unexpected format: got %q want %q", gotFormat, tc.wantFormat)
			}
			if len(gotArgs) != len(tc.wantArgs) {
				t.Fatalf("unexpected args length: got %d want %d", len(gotArgs), len(tc.wantArgs))
			}
			for i := range gotArgs {
				if gotArgs[i] != tc.wantArgs[i] {
					t.Fatalf("unexpected arg at index %d: got %q want %q", i, gotArgs[i], tc.wantArgs[i])
				}
			}
		})
	}
}

func TestAppendToDefaultBundleDryRunDoesNotPersist(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	record, err := appendToDefaultBundle("dev.session", map[string]any{"ok": true}, true)
	if err != nil {
		t.Fatalf("appendToDefaultBundle dry-run error: %v", err)
	}
	if record.Event.Sequence != 1 {
		t.Fatalf("unexpected sequence: got %d want 1", record.Event.Sequence)
	}
	if record.Event.Timestamp == "" {
		t.Fatalf("expected timestamp to be populated on appended record")
	}
	if _, err := os.Stat(bundle.DefaultPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no bundle file written in dry-run mode, stat err=%v", err)
	}
}

func TestAppendToDefaultBundleRejectsCorruptExistingBundle(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	if err := os.MkdirAll(bundle.BundleDir, 0755); err != nil {
		t.Fatalf("mkdir bundle dir: %v", err)
	}
	if err := os.WriteFile(bundle.DefaultPath(), []byte("{not-json}\n"), 0644); err != nil {
		t.Fatalf("write corrupt bundle: %v", err)
	}

	_, err = appendToDefaultBundle("dev.session", map[string]any{"ok": true}, true)
	if err == nil {
		t.Fatalf("expected error for corrupt existing bundle")
	}

	var loadErr mutationLoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("expected mutationLoadError, got %T", err)
	}
}

func TestVerifyWithTraceIncludesPerEventLogs(t *testing.T) {
	b := bundle.New()
	appendTestBundleEvent(t, b, "dev.session", map[string]any{"ok": true})
	b.Records[0].Hash = strings.Repeat("0", 64)

	var trace bytes.Buffer
	err := verifyWithTrace(b, &trace)
	if err == nil {
		t.Fatalf("expected verification error")
	}
	out := trace.String()
	if !strings.Contains(out, "trace: event_index=0") {
		t.Fatalf("expected trace output to include event_index, got %q", out)
	}
	if !strings.Contains(out, "match=false") {
		t.Fatalf("expected trace output to include mismatch result, got %q", out)
	}
}

func TestParseHelpArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{
			name: "defaults to text",
			args: nil,
			want: "text",
		},
		{
			name: "json format split",
			args: []string{"--format", "json"},
			want: "json",
		},
		{
			name: "json format equals",
			args: []string{"--format=json"},
			want: "json",
		},
		{
			name:    "missing format value",
			args:    []string{"--format"},
			wantErr: true,
		},
		{
			name:    "invalid format",
			args:    []string{"--format", "yaml"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"--wat"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseHelpArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected format: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestUsageJSONIncludesVerifyTraceAndExitCodes(t *testing.T) {
	got := usageJSON()
	if got.Name != "atb" {
		t.Fatalf("unexpected name: got %q", got.Name)
	}
	if got.ExitCodes["2"] != "integrity verification failure" {
		t.Fatalf("missing exit code mapping for 2")
	}
	foundVerify := false
	foundMutatingFormat := map[string]bool{
		"init":     false,
		"append":   false,
		"snapshot": false,
	}
	for _, cmd := range got.Commands {
		if cmd.Name == "verify" {
			foundVerify = true
			if !strings.Contains(cmd.Usage, "--trace") {
				t.Fatalf("verify usage missing --trace flag: %q", cmd.Usage)
			}
		}
		if _, ok := foundMutatingFormat[cmd.Name]; ok {
			if !strings.Contains(cmd.Usage, "--format") {
				t.Fatalf("%s usage missing --format: %q", cmd.Name, cmd.Usage)
			}
			foundMutatingFormat[cmd.Name] = true
		}
	}
	if !foundVerify {
		t.Fatalf("verify command missing from usage JSON")
	}
	for name, found := range foundMutatingFormat {
		if !found {
			t.Fatalf("%s command missing from usage JSON", name)
		}
	}
}
