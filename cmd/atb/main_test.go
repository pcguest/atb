// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	signpkg "github.com/pcguest/atb/internal/sign"
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
		wantSignPolicy  string
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
			name:            "sign policy flag",
			args:            []string{"--data", `{"x":1}`, "--sign-policy", "./keys/atb-key.pem"},
			wantJSON:        `{"x":1}`,
			wantSignPolicy:  filepath.Clean("./keys/atb-key.pem"),
			wantActorID:     "",
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
			if got.SignPolicyKeyPath != tc.wantSignPolicy {
				t.Fatalf("unexpected sign-policy path: got %q want %q", got.SignPolicyKeyPath, tc.wantSignPolicy)
			}
		})
	}
}

func TestRunAppendSignPolicyAddsSignatureFields(t *testing.T) {
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

	keyPath := writeSignTestPrivateKey(t, t.TempDir())

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runAppend([]string{
		event.TypeAIPolicyDecision,
		"--data",
		`{"policy_id":"pol-1","policy_version":"2026-04","decision":"allow","decision_reason_codes":["ticket_present"],"subject_id_hash":"subject-hash","action_id":"act-1"}`,
		"--sign-policy",
		keyPath,
	}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	b, err := bundle.Load(bundle.DefaultPath())
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	last := b.Records[len(b.Records)-1]
	if last.Event.Type != event.TypeAIPolicyDecision {
		t.Fatalf("unexpected last event type: got %q want %q", last.Event.Type, event.TypeAIPolicyDecision)
	}

	fields, ok := last.Event.Data.(map[string]any)
	if !ok {
		t.Fatalf("policy decision data type = %T, want map[string]any", last.Event.Data)
	}
	if fields[event.FieldPolicySignature] == "" {
		t.Fatalf("expected non-empty %s", event.FieldPolicySignature)
	}
	if fields[event.FieldPolicySignerPubKey] == "" {
		t.Fatalf("expected non-empty %s", event.FieldPolicySignerPubKey)
	}
	if err := signpkg.VerifyPolicyDecision(fields); err != nil {
		t.Fatalf("verify signed policy decision: %v", err)
	}
}

func TestRunAppendSignPolicyMissingKeypairFailsClearly(t *testing.T) {
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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runAppend([]string{
		event.TypeAIPolicyDecision,
		"--data",
		`{"policy_id":"pol-1","policy_version":"2026-04","decision":"allow","decision_reason_codes":["ticket_present"],"subject_id_hash":"subject-hash","action_id":"act-1"}`,
		"--sign-policy",
		filepath.Join(tmp, "keys", "atb-key.pem"),
	}, &stdout, &stderr)
	if exitCode != exitUserError {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitUserError, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if got := strings.TrimSpace(stderr.String()); got != "atb append: no signing key found; run 'atb keygen' before using --sign-policy" {
		t.Fatalf("unexpected stderr: got %q", got)
	}
}

func TestRunAppendSignPolicyIgnoredForNonPolicyDecision(t *testing.T) {
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

	keyPath := writeSignTestPrivateKey(t, t.TempDir())

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runAppend([]string{
		"dev.session",
		`{"ok":true}`,
		"--sign-policy",
		keyPath,
	}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--sign-policy ignored") {
		t.Fatalf("expected ignored warning, got %q", stderr.String())
	}
}

func TestRunAppendPolicyDocEmbedsPolicyDocHash(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	docPath := filepath.Join(tmp, "policy.txt")
	if err := os.WriteFile(docPath, []byte("Policy v1: allow if ticket present."), 0600); err != nil {
		t.Fatalf("write policy doc: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := runAppend([]string{
		event.TypeAIPolicyDecision,
		"--data",
		`{"policy_id":"pol-1","policy_version":"2026-04","decision":"allow","decision_reason_codes":["ticket_present"],"subject_id_hash":"subject-hash","action_id":"act-1"}`,
		"--policy-doc", docPath,
	}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("exit code %d, stderr=%q", exitCode, stderr.String())
	}

	b, err := bundle.Load(bundle.DefaultPath())
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	fields, ok := b.Records[len(b.Records)-1].Event.Data.(map[string]any)
	if !ok {
		t.Fatal("expected map data")
	}
	docHash, ok := fields[event.FieldPolicyDocHash].(string)
	if !ok || docHash == "" {
		t.Fatalf("expected non-empty %s", event.FieldPolicyDocHash)
	}
	if _, ok := fields[event.FieldPolicyDocSignature]; ok {
		t.Fatal("policy_doc_signature should be absent when --sign-policy not set")
	}
}

func TestRunAppendPolicyDocWithSignPolicyAddsCompoundSignature(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	keyPath := writeSignTestPrivateKey(t, t.TempDir())
	docPath := filepath.Join(tmp, "policy.txt")
	if err := os.WriteFile(docPath, []byte("Policy v1: allow if ticket present."), 0600); err != nil {
		t.Fatalf("write policy doc: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := runAppend([]string{
		event.TypeAIPolicyDecision,
		"--data",
		`{"policy_id":"pol-1","policy_version":"2026-04","decision":"allow","decision_reason_codes":["ticket_present"],"subject_id_hash":"subject-hash","action_id":"act-1"}`,
		"--policy-doc", docPath,
		"--sign-policy", keyPath,
	}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("exit code %d, stderr=%q", exitCode, stderr.String())
	}

	b, err := bundle.Load(bundle.DefaultPath())
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	fields, ok := b.Records[len(b.Records)-1].Event.Data.(map[string]any)
	if !ok {
		t.Fatal("expected map data")
	}
	if fields[event.FieldPolicyDocHash] == "" {
		t.Fatalf("expected non-empty %s", event.FieldPolicyDocHash)
	}
	if fields[event.FieldPolicyDocSignature] == "" {
		t.Fatalf("expected non-empty %s", event.FieldPolicyDocSignature)
	}
	if err := signpkg.VerifyPolicyDocSignature(fields); err != nil {
		t.Fatalf("VerifyPolicyDocSignature: %v", err)
	}
}

func TestRunAppendPolicyDocIgnoredForNonPolicyDecision(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	docPath := filepath.Join(tmp, "policy.txt")
	if err := os.WriteFile(docPath, []byte("irrelevant"), 0600); err != nil {
		t.Fatalf("write doc: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := runAppend([]string{
		"dev.session",
		`{"ok":true}`,
		"--policy-doc", docPath,
	}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("exit code %d, stderr=%q", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--policy-doc ignored") {
		t.Fatalf("expected ignored warning, got %q", stderr.String())
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
		name                  string
		args                  []string
		wantPath              string
		wantFormat            string
		wantTrace             bool
		wantWithAnchor        bool
		wantWithSnapshotCheck bool
		wantRoots             string
		wantErr               bool
	}{
		{
			name:                  "defaults",
			args:                  nil,
			wantPath:              bundle.DefaultPath(),
			wantFormat:            formatText,
			wantTrace:             false,
			wantWithAnchor:        false,
			wantWithSnapshotCheck: false,
			wantRoots:             "",
		},
		{
			name:                  "path only",
			args:                  []string{custom},
			wantPath:              custom,
			wantFormat:            formatText,
			wantTrace:             false,
			wantWithAnchor:        false,
			wantWithSnapshotCheck: false,
		},
		{
			name:                  "json format flag",
			args:                  []string{"--format", "json"},
			wantPath:              bundle.DefaultPath(),
			wantFormat:            formatJSON,
			wantTrace:             false,
			wantWithAnchor:        false,
			wantWithSnapshotCheck: false,
			wantRoots:             "",
		},
		{
			name:                  "json format equals syntax",
			args:                  []string{"--format=json"},
			wantPath:              bundle.DefaultPath(),
			wantFormat:            formatJSON,
			wantTrace:             false,
			wantWithAnchor:        false,
			wantWithSnapshotCheck: false,
			wantRoots:             "",
		},
		{
			name:                  "path and format",
			args:                  []string{custom, "--format", "json"},
			wantPath:              custom,
			wantFormat:            formatJSON,
			wantTrace:             false,
			wantWithAnchor:        false,
			wantWithSnapshotCheck: false,
		},
		{
			name:                  "trace flag",
			args:                  []string{"--trace"},
			wantPath:              bundle.DefaultPath(),
			wantFormat:            formatText,
			wantTrace:             true,
			wantWithAnchor:        false,
			wantWithSnapshotCheck: false,
			wantRoots:             "",
		},
		{
			name:                  "path format and trace",
			args:                  []string{custom, "--format", "json", "--trace"},
			wantPath:              custom,
			wantFormat:            formatJSON,
			wantTrace:             true,
			wantWithAnchor:        false,
			wantWithSnapshotCheck: false,
			wantRoots:             "",
		},
		{
			name:                  "with-anchor flag",
			args:                  []string{"--with-anchor"},
			wantPath:              bundle.DefaultPath(),
			wantFormat:            formatText,
			wantTrace:             false,
			wantWithAnchor:        true,
			wantWithSnapshotCheck: false,
			wantRoots:             "",
		},
		{
			name:                  "with-snapshot-check flag",
			args:                  []string{"--with-snapshot-check"},
			wantPath:              bundle.DefaultPath(),
			wantFormat:            formatText,
			wantTrace:             false,
			wantWithAnchor:        false,
			wantWithSnapshotCheck: true,
			wantRoots:             "",
		},
		{
			name:                  "path format trace with-anchor and roots",
			args:                  []string{custom, "--format", "json", "--trace", "--with-anchor", "--roots", "tsa-roots.pem"},
			wantPath:              custom,
			wantFormat:            formatJSON,
			wantTrace:             true,
			wantWithAnchor:        true,
			wantWithSnapshotCheck: false,
			wantRoots:             "tsa-roots.pem",
		},
		{
			name:                  "roots equals syntax",
			args:                  []string{"--roots=tsa-roots.pem"},
			wantPath:              bundle.DefaultPath(),
			wantFormat:            formatText,
			wantTrace:             false,
			wantWithAnchor:        false,
			wantWithSnapshotCheck: false,
			wantRoots:             "tsa-roots.pem",
		},
		{
			name:    "missing format value",
			args:    []string{"--format"},
			wantErr: true,
		},
		{
			name:    "missing roots value",
			args:    []string{"--roots"},
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
			gotPath, gotFormat, gotTrace, gotWithAnchor, gotWithSnapshotCheck, gotRoots, err := parseVerifyArgs(tc.args)
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
			if gotWithAnchor != tc.wantWithAnchor {
				t.Fatalf("unexpected with-anchor value: got %v want %v", gotWithAnchor, tc.wantWithAnchor)
			}
			if gotWithSnapshotCheck != tc.wantWithSnapshotCheck {
				t.Fatalf("unexpected with-snapshot-check value: got %v want %v", gotWithSnapshotCheck, tc.wantWithSnapshotCheck)
			}
			if gotRoots != tc.wantRoots {
				t.Fatalf("unexpected roots value: got %q want %q", gotRoots, tc.wantRoots)
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
			wantFormat: formatText,
			wantDryRun: false,
		},
		{
			name:       "flag at end",
			args:       []string{"snapshot", "--gate", "pass", "--dry-run"},
			wantArgs:   []string{"snapshot", "--gate", "pass"},
			wantFormat: formatText,
			wantDryRun: true,
		},
		{
			name:       "flag at start",
			args:       []string{"--dry-run", "feature", "{\"ok\":true}"},
			wantArgs:   []string{"feature", "{\"ok\":true}"},
			wantFormat: formatText,
			wantDryRun: true,
		},
		{
			name:       "json format split",
			args:       []string{"feature", "{\"ok\":true}", "--format", "json"},
			wantArgs:   []string{"feature", "{\"ok\":true}"},
			wantFormat: formatJSON,
			wantDryRun: false,
		},
		{
			name:       "json format equals",
			args:       []string{"feature", "--format=json", "{\"ok\":true}"},
			wantArgs:   []string{"feature", "{\"ok\":true}"},
			wantFormat: formatJSON,
			wantDryRun: false,
		},
		{
			name:       "unknown flag passed through",
			args:       []string{"--wat"},
			wantArgs:   []string{"--wat"},
			wantFormat: formatText,
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

func TestRunBundleNew(t *testing.T) {
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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runBundle([]string{"new"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Initialised ATB bundle") {
		t.Fatalf("expected init output, got %q", stdout.String())
	}
	if _, err := os.Stat(bundle.DefaultPath()); err != nil {
		t.Fatalf("expected bundle file to be created: %v", err)
	}
}

func TestRunBundleNewDryRunDoesNotCreateFiles(t *testing.T) {
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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runBundle([]string{"new", "--dry-run"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}
	if !strings.Contains(stdout.String(), "would initialise ATB bundle") {
		t.Fatalf("expected dry-run output, got %q", stdout.String())
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
	b := newTestBundle(t)
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

func TestUsageJSONIncludesVerifyFlagsAndExitCodes(t *testing.T) {
	got := usageJSON()
	if got.Name != "atb" {
		t.Fatalf("unexpected name: got %q", got.Name)
	}
	if got.ExitCodes["2"] != "integrity verification failure" {
		t.Fatalf("missing exit code mapping for 2")
	}
	foundVerify := false
	foundBundle := false
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
			if !strings.Contains(cmd.Usage, "--quiet") {
				t.Fatalf("verify usage missing --quiet flag: %q", cmd.Usage)
			}
			if !strings.Contains(cmd.Usage, "--with-anchor") {
				t.Fatalf("verify usage missing --with-anchor flag: %q", cmd.Usage)
			}
			if !strings.Contains(cmd.Usage, "--with-snapshot-check") {
				t.Fatalf("verify usage missing --with-snapshot-check flag: %q", cmd.Usage)
			}
			if !strings.Contains(cmd.Usage, "--roots") {
				t.Fatalf("verify usage missing --roots flag: %q", cmd.Usage)
			}
		}
		if cmd.Name == "bundle" {
			foundBundle = true
			if !strings.Contains(cmd.Usage, "bundle new") {
				t.Fatalf("bundle usage missing bundle new alias: %q", cmd.Usage)
			}
		}
		if _, ok := foundMutatingFormat[cmd.Name]; ok {
			if !strings.Contains(cmd.Usage, "--format") {
				t.Fatalf("%s usage missing --format: %q", cmd.Name, cmd.Usage)
			}
			if cmd.Name == "append" && !strings.Contains(cmd.Usage, "--sign-policy") {
				t.Fatalf("append usage missing --sign-policy: %q", cmd.Usage)
			}
			foundMutatingFormat[cmd.Name] = true
		}
	}
	if !foundVerify {
		t.Fatalf("verify command missing from usage JSON")
	}
	if !foundBundle {
		t.Fatalf("bundle command missing from usage JSON")
	}
	for name, found := range foundMutatingFormat {
		if !found {
			t.Fatalf("%s command missing from usage JSON", name)
		}
	}
}

func TestRunVersionJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runVersion([]string{"--json"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}

	var out struct {
		Version   string `json:"version"`
		Algorithm string `json:"algorithm"`
		Anchor    string `json:"anchor"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal version JSON: %v\noutput=%s", err, stdout.String())
	}
	if out.Version != version {
		t.Errorf("version field: got %q, want %q", out.Version, version)
	}
	if out.Algorithm == "" {
		t.Error("algorithm field must not be empty")
	}
	if out.Anchor == "" {
		t.Error("anchor field must not be empty")
	}
}
