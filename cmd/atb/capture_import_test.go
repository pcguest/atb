// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

var update = flag.Bool("update", false, "rewrite golden fixtures")

func TestParseImportChatlogArgs(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "custom.atb")
	inputPath := filepath.Join(tmp, "chatlog.jsonl")

	cfg, err := parseImportChatlogArgs([]string{
		"--from", "generic-jsonl",
		"--input", inputPath,
		"--bundle", bundlePath,
		"--snapshot", "imported",
	})
	if err != nil {
		t.Fatalf("parseImportChatlogArgs() error = %v", err)
	}
	if cfg.From != "generic-jsonl" {
		t.Fatalf("From = %q, want %q", cfg.From, "generic-jsonl")
	}
	if cfg.InputPath != inputPath {
		t.Fatalf("InputPath = %q, want %q", cfg.InputPath, inputPath)
	}
	if cfg.BundlePath != bundlePath {
		t.Fatalf("BundlePath = %q, want %q", cfg.BundlePath, bundlePath)
	}
	if cfg.SnapshotName != "imported" {
		t.Fatalf("SnapshotName = %q, want %q", cfg.SnapshotName, "imported")
	}
}

func TestRunImportChatlog(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	inputPath := filepath.Join(tmp, "chatlog.jsonl")
	raw := strings.Join([]string{
		`{"role":"system","content":"Use the handbook.","timestamp":"2026-04-24T09:00:00Z","session_id":"sess-1"}`,
		`{"role":"user","content":"Can I carry annual leave into next year?","timestamp":"2026-04-24T09:00:10Z","session_id":"sess-1","request_id":"req-1","actor_id_hash":"sha256:user-1","purpose_tag":"rag_answer"}`,
		`{"role":"assistant","content":"I will check the handbook.","timestamp":"2026-04-24T09:00:11Z","session_id":"sess-1","model":"gpt-4o-mini"}`,
		`{"role":"tool","content":"{\"policy\":\"up to five days\"}","timestamp":"2026-04-24T09:00:12Z","session_id":"sess-1","tool_name":"hr.policy.lookup","tool_args":{"query":"annual leave carry over"}}`,
		`{"role":"assistant","content":"Yes. Up to five days can be carried over.","timestamp":"2026-04-24T09:00:13Z","session_id":"sess-1","model":"gpt-4o-mini"}`,
	}, "\n")
	if err := os.WriteFile(inputPath, []byte(raw), 0600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", inputPath,
		"--bundle", bundlePath,
		"--snapshot", "imported_chatlog",
	}, nil, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runImportChatlog() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}
	if !strings.Contains(stdout.String(), "imported:") {
		t.Fatalf("expected import summary, got %q", stdout.String())
	}
	for _, b := range stdout.Bytes() {
		if b > 0x7f {
			t.Fatalf("text output contains non-ASCII byte %#x: %q", b, stdout.String())
		}
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("bundle.Load() error = %v", err)
	}
	if len(b.Records) != 10 {
		t.Fatalf("len(b.Records) = %d, want %d", len(b.Records), 10)
	}
	if got := b.Records[len(b.Records)-1].Event.Type; got != event.TypeSnapshot {
		t.Fatalf("last event type = %q, want %q", got, event.TypeSnapshot)
	}

	report, err := verifypkg.EvaluateBundle(verifypkg.EvaluateConfig{
		BundlePath: bundlePath,
		Records:    b.Records,
		Profiles:   []verifypkg.Profile{verifypkg.ProfileByID("atb.profile.rag_answer")},
	})
	if err != nil {
		t.Fatalf("EvaluateBundle() error = %v", err)
	}
	if !report.Profiles[0].Pass {
		t.Fatalf("expected imported bundle to pass rag profile, got %+v", report.Profiles[0])
	}
}

func TestRunImportChatlogEventsOnlyNoSnapshot(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	inputPath := filepath.Join(tmp, "chatlog.jsonl")
	raw := strings.Join([]string{
		`{"role":"user","content":"hi","timestamp":"2026-04-24T09:00:10Z","session_id":"sess-1","request_id":"req-1","actor_id_hash":"sha256:user-1","purpose_tag":"rag_answer"}`,
		`{"role":"assistant","content":"hello","timestamp":"2026-04-24T09:00:11Z","session_id":"sess-1","model":"gpt-4o-mini"}`,
	}, "\n")
	if err := os.WriteFile(inputPath, []byte(raw), 0600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exit := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", inputPath,
		"--bundle", bundlePath,
	}, nil, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit = %d, want %d (stderr=%q)", exit, exitSuccess, stderr.String())
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, r := range b.Records {
		if r.Event.Type == event.TypeSnapshot {
			t.Fatalf("did not expect snapshot record, got %v", r.Event.Type)
		}
	}
}

func TestRunImportChatlogSkipsUnknownTurns(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	inputPath := filepath.Join(tmp, "chatlog.jsonl")
	raw := strings.Join([]string{
		`{"role":"user","content":"hi","timestamp":"2026-04-24T09:00:10Z","session_id":"s","request_id":"r1","actor_id_hash":"sha256:u","purpose_tag":"rag_answer"}`,
		`{"role":"critic","content":"unrecognised","timestamp":"2026-04-24T09:00:11Z","session_id":"s"}`,
		`{"content":"missing role","timestamp":"2026-04-24T09:00:12Z","session_id":"s"}`,
		`{"role":"assistant","content":"hello","timestamp":"2026-04-24T09:00:13Z","session_id":"s","model":"gpt-4o-mini"}`,
	}, "\n")
	if err := os.WriteFile(inputPath, []byte(raw), 0600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exit := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", inputPath,
		"--bundle", bundlePath,
	}, nil, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit = %d, want %d (stderr=%q)", exit, exitSuccess, stderr.String())
	}
	if !strings.Contains(stderr.String(), "skipping unrecognised turn 1") {
		t.Fatalf("stderr = %q, want unknown turn 1 log", stderr.String())
	}
	if !strings.Contains(stderr.String(), "skipping unrecognised turn 2") {
		t.Fatalf("stderr = %q, want unknown turn 2 log", stderr.String())
	}
	if !strings.Contains(stdout.String(), "(2 source records skipped)") {
		t.Fatalf("stdout = %q, want skipped count", stdout.String())
	}
}

func TestRunImportChatlogUnknownTurnsFixture(t *testing.T) {
	if *update {
		t.Log("import fixture tests have no generated golden files to rewrite")
	}

	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")

	var stdout, stderr bytes.Buffer
	exit := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", filepath.Join("testdata", "import_unknown_turns.jsonl"),
		"--bundle", bundlePath,
	}, nil, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit = %d, want %d (stderr=%q)", exit, exitSuccess, stderr.String())
	}
	if !strings.Contains(stderr.String(), "skipping unrecognised turn 2") {
		t.Fatalf("stderr = %q, want unknown turn 2 log", stderr.String())
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := countNonManifestRecords(b); got != 2 {
		t.Fatalf("non-manifest records = %d, want 2", got)
	}
}

func TestRunImportChatlogSystemOnlyFixtureCreatesManifestOnlyBundle(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")

	var stdout, stderr bytes.Buffer
	exit := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", filepath.Join("testdata", "import_system_only.jsonl"),
		"--bundle", bundlePath,
	}, nil, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit = %d, want %d (stderr=%q)", exit, exitSuccess, stderr.String())
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := countNonManifestRecords(b); got != 0 {
		t.Fatalf("non-manifest records = %d, want 0", got)
	}
}

func TestRunImportChatlogEmptyArrayFixtureCreatesManifestOnlyBundle(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")

	var stdout, stderr bytes.Buffer
	exit := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", filepath.Join("testdata", "import_empty.jsonl"),
		"--bundle", bundlePath,
	}, nil, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit = %d, want %d (stderr=%q)", exit, exitSuccess, stderr.String())
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := countNonManifestRecords(b); got != 0 {
		t.Fatalf("non-manifest records = %d, want 0", got)
	}
}

func TestRunImportChatlogFullToolTurnFixture(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")

	var stdout, stderr bytes.Buffer
	exit := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", filepath.Join("testdata", "import_full_tool_turn.jsonl"),
		"--bundle", bundlePath,
	}, nil, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit = %d, want %d (stderr=%q)", exit, exitSuccess, stderr.String())
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var toolData map[string]any
	for _, record := range b.Records {
		if record.Event.Type == event.TypeAIToolExec {
			data, ok := record.Event.Data.(map[string]any)
			if !ok {
				t.Fatalf("tool event data = %#v, want map", record.Event.Data)
			}
			toolData = data
			break
		}
	}
	if toolData == nil {
		t.Fatalf("missing %s event", event.TypeAIToolExec)
	}
	if toolData["tool_name"] != "legal.policy.lookup" {
		t.Fatalf("tool_name = %#v", toolData["tool_name"])
	}
	if toolData["tool_call_id"] != "call-nda-001" {
		t.Fatalf("tool_call_id = %#v", toolData["tool_call_id"])
	}
	if toolData["tool_output"] != "NDA required before access provisioning." {
		t.Fatalf("tool_output = %#v", toolData["tool_output"])
	}
	args, ok := toolData["tool_args"].(map[string]any)
	if !ok {
		t.Fatalf("tool_args = %#v, want map", toolData["tool_args"])
	}
	if args["query"] != "contractor NDA onboarding" || args["jurisdiction"] != "AU" || args["include_archived"] != false {
		t.Fatalf("tool_args = %#v", args)
	}
}

func TestRunImportChatlogEmptyInputCreatesManifestOnlyBundle(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	inputPath := filepath.Join(tmp, "chatlog.jsonl")
	if err := os.WriteFile(inputPath, nil, 0600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exit := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", inputPath,
		"--bundle", bundlePath,
	}, nil, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit = %d, want %d (stderr=%q)", exit, exitSuccess, stderr.String())
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(b.Records) != 1 {
		t.Fatalf("len(b.Records) = %d, want manifest only", len(b.Records))
	}
	if got := b.Records[0].Event.Type; got != event.TypeBundleManifest {
		t.Fatalf("manifest event type = %q, want %q", got, event.TypeBundleManifest)
	}
}

func countNonManifestRecords(b *bundle.Bundle) int {
	count := 0
	for _, record := range b.Records {
		if record.Event.Type != event.TypeBundleManifest {
			count++
		}
	}
	return count
}

func TestValidateSnapshotNameRejectsEmpty(t *testing.T) {
	if err := validateSnapshotName(""); err == nil {
		t.Fatalf("expected error for empty name")
	}
	if err := validateSnapshotName("   "); err == nil {
		t.Fatalf("expected error for whitespace-only name")
	}
	if err := validateSnapshotName("ok"); err != nil {
		t.Fatalf("unexpected error for valid name: %v", err)
	}
}

func TestRunImportChatlogSaveFailureLeavesBundleUntouched(t *testing.T) {
	tmp := t.TempDir()

	// Bundle path's parent will be a regular file, so Save's mkdir/rename fails.
	parentFile := filepath.Join(tmp, "blocker")
	if err := os.WriteFile(parentFile, []byte("x"), 0600); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	bundlePath := filepath.Join(parentFile, "bundle.atb")

	inputPath := filepath.Join(tmp, "chatlog.jsonl")
	if err := os.WriteFile(inputPath, []byte(`{"role":"user","content":"hi","timestamp":"2026-04-24T09:00:10Z","request_id":"req-3","actor_id_hash":"sha256:user-3","purpose_tag":"rag_answer"}`+"\n"), 0600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exit := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", inputPath,
		"--bundle", bundlePath,
	}, nil, &stdout, &stderr)
	if exit == exitSuccess {
		t.Fatalf("expected non-success exit, got %d", exit)
	}
	// Save uses temp+rename; on failure the target path must not have a
	// successfully-written bundle. Because the parent is a regular file,
	// no readable bundle file should exist at the target path.
	if data, err := os.ReadFile(bundlePath); err == nil {
		t.Fatalf("expected no readable bundle at %s, got %d bytes", bundlePath, len(data))
	}
}

func TestRunImportChatlogFormatJSONSuccess(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	inputPath := filepath.Join(tmp, "chatlog.jsonl")
	raw := strings.Join([]string{
		`{"role":"user","content":"hi","timestamp":"2026-04-24T09:00:10Z","session_id":"s","request_id":"r1","actor_id_hash":"sha256:u","purpose_tag":"rag_answer"}`,
		`{"role":"assistant","content":"hello","timestamp":"2026-04-24T09:00:11Z","session_id":"s","model":"gpt-4o-mini"}`,
	}, "\n")
	if err := os.WriteFile(inputPath, []byte(raw), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exit := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", inputPath,
		"--bundle", bundlePath,
		"--snapshot", "snap1",
		"--format", "json",
	}, nil, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit = %d, want %d (stderr=%q)", exit, exitSuccess, stderr.String())
	}

	var got importChatlogResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode json: %v\nstdout=%q", err, stdout.String())
	}
	if got.EventsWritten <= 0 {
		t.Fatalf("EventsWritten = %d, want >0", got.EventsWritten)
	}
	if got.BundlePath != bundlePath {
		t.Fatalf("BundlePath = %q, want %q", got.BundlePath, bundlePath)
	}
	if !got.SnapshotAppended || got.SnapshotName != "snap1" {
		t.Fatalf("snapshot fields: appended=%v name=%q", got.SnapshotAppended, got.SnapshotName)
	}
}

func TestRunImportChatlogFormatJSONFailure(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	inputPath := filepath.Join(tmp, "missing.jsonl") // does not exist

	var stdout, stderr bytes.Buffer
	exit := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", inputPath,
		"--bundle", bundlePath,
		"--format", "json",
	}, nil, &stdout, &stderr)
	if exit == exitSuccess {
		t.Fatalf("expected failure, got exitSuccess")
	}

	var errOut importChatlogError
	if err := json.Unmarshal(stdout.Bytes(), &errOut); err != nil {
		t.Fatalf("decode json: %v\nstdout=%q", err, stdout.String())
	}
	if errOut.Error == "" {
		t.Fatalf("expected error field, got %+v", errOut)
	}
	if errOut.EventsWritten != 0 {
		t.Fatalf("EventsWritten = %d, want 0", errOut.EventsWritten)
	}
}

func TestRunImportChatlogFormatUnknownExitsUserError(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	inputPath := filepath.Join(tmp, "chatlog.jsonl")
	if err := os.WriteFile(inputPath, []byte(`{"role":"user","content":"hi","timestamp":"2026-04-24T09:00:10Z"}`+"\n"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exit := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", inputPath,
		"--bundle", bundlePath,
		"--format", "yaml",
	}, nil, &stdout, &stderr)
	if exit != exitUserError {
		t.Fatalf("exit = %d, want %d", exit, exitUserError)
	}
	if _, err := os.Stat(bundlePath); !os.IsNotExist(err) {
		t.Fatalf("expected no bundle at %s, got err=%v", bundlePath, err)
	}
}

func TestRunImportChatlogReadsStdin(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	raw := strings.Join([]string{
		`{"role":"user","content":"hi","timestamp":"2026-04-24T09:00:10Z","session_id":"s","request_id":"r1","actor_id_hash":"sha256:u","purpose_tag":"rag_answer"}`,
		`{"role":"assistant","content":"hello","timestamp":"2026-04-24T09:00:11Z","session_id":"s","model":"gpt-4o-mini"}`,
	}, "\n")
	stdin := bytes.NewBufferString(raw)

	var stdout, stderr bytes.Buffer
	exit := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", "-",
		"--bundle", bundlePath,
	}, stdin, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit = %d, want %d (stderr=%q)", exit, exitSuccess, stderr.String())
	}
	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(b.Records) == 0 {
		t.Fatalf("expected records in bundle, got 0")
	}
}

func TestRunImportChatlogMissingFileExitsUserError(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	missing := filepath.Join(tmp, "does-not-exist.jsonl")

	var stdout, stderr bytes.Buffer
	exit := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", missing,
		"--bundle", bundlePath,
	}, nil, &stdout, &stderr)
	if exit != exitUserError {
		t.Fatalf("exit = %d, want %d", exit, exitUserError)
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Fatalf("stderr = %q, want contains 'not found'", stderr.String())
	}
}

func TestRunImportChatlogPermissionDeniedExitsSystemError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root")
	}
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	dir := filepath.Join(tmp, "locked")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	inputPath := filepath.Join(dir, "chatlog.jsonl")
	if err := os.WriteFile(inputPath, []byte(`{"role":"user","content":"hi","timestamp":"2026-04-24T09:00:10Z"}`+"\n"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Chmod(dir, 0000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) })

	var stdout, stderr bytes.Buffer
	exit := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", inputPath,
		"--bundle", bundlePath,
	}, nil, &stdout, &stderr)
	if exit != exitSystemError {
		t.Fatalf("exit = %d, want %d (stderr=%q)", exit, exitSystemError, stderr.String())
	}
	if !strings.Contains(stderr.String(), "cannot open input file") {
		t.Fatalf("stderr = %q, want contains 'cannot open input file'", stderr.String())
	}
}

func TestRunImportChatlogExceedsMaxInputSize(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	inputPath := filepath.Join(tmp, "chatlog.jsonl")
	// Write ~4 KiB of valid lines so we comfortably exceed a 256-byte cap.
	line := `{"role":"user","content":"hi","timestamp":"2026-04-24T09:00:10Z","session_id":"s","request_id":"r","actor_id_hash":"sha256:u","purpose_tag":"rag_answer"}` + "\n"
	if err := os.WriteFile(inputPath, []byte(strings.Repeat(line, 32)), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exit := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", inputPath,
		"--bundle", bundlePath,
		"--max-input-size", "256",
	}, nil, &stdout, &stderr)
	if exit != exitUserError {
		t.Fatalf("exit = %d, want %d (stderr=%q)", exit, exitUserError, stderr.String())
	}
	if !strings.Contains(stderr.String(), "exceeds maximum size") {
		t.Fatalf("stderr = %q, want contains 'exceeds maximum size'", stderr.String())
	}
}

func TestRunImportChatlogMalformedExitsUserError(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	inputPath := filepath.Join(tmp, "chatlog.jsonl")
	// Invalid JSON on a line.
	if err := os.WriteFile(inputPath, []byte("{not json}\n"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exit := runImportChatlog([]string{
		"--from", "generic-jsonl",
		"--input", inputPath,
		"--bundle", bundlePath,
	}, nil, &stdout, &stderr)
	if exit != exitUserError {
		t.Fatalf("exit = %d, want %d (stderr=%q)", exit, exitUserError, stderr.String())
	}
}

func TestParseCaptureRunArgs(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")

	cfg, err := parseCaptureRunArgs([]string{
		"--bundle", bundlePath,
		"--snapshot", "after_run",
		"--env-prefix", "myapp",
		"--profile", "atb.profile.rag_answer",
		"--",
		"sh", "-c", "true",
	})
	if err != nil {
		t.Fatalf("parseCaptureRunArgs() error = %v", err)
	}
	if cfg.BundlePath != bundlePath {
		t.Fatalf("BundlePath = %q, want %q", cfg.BundlePath, bundlePath)
	}
	if cfg.EnvPrefix != "MYAPP" {
		t.Fatalf("EnvPrefix = %q, want %q", cfg.EnvPrefix, "MYAPP")
	}
	if !cfg.EnvPrefixSet {
		t.Fatalf("EnvPrefixSet = false, want true")
	}
	if cfg.ProfileID != "atb.profile.rag_answer" {
		t.Fatalf("ProfileID = %q, want %q", cfg.ProfileID, "atb.profile.rag_answer")
	}
	if len(cfg.Command) != 3 {
		t.Fatalf("len(Command) = %d, want %d", len(cfg.Command), 3)
	}
}

func TestParseCaptureRunArgsEnvPrefixEmptyForms(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "equals",
			args: []string{"--env-prefix=", "--", "sh", "-c", "true"},
		},
		{
			name: "space",
			args: []string{"--env-prefix", "", "--", "sh", "-c", "true"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := parseCaptureRunArgs(tc.args)
			if err != nil {
				t.Fatalf("parseCaptureRunArgs() error = %v", err)
			}
			if cfg.EnvPrefix != "" {
				t.Fatalf("EnvPrefix = %q, want empty", cfg.EnvPrefix)
			}
			if !cfg.EnvPrefixSet {
				t.Fatalf("EnvPrefixSet = false, want true")
			}
		})
	}
}

func TestRunCaptureRunSetsEnvironmentAndVerifies(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	seenPath := filepath.Join(tmp, "seen.txt")
	writeRAGFixtureBundle(t, bundlePath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runCaptureRun([]string{
		"--bundle", bundlePath,
		"--env-prefix", "MYAPP",
		"--profile", "atb.profile.rag_answer",
		"--",
		"sh", "-c", "printf '%s|%s|%s' \"$ATB_BUNDLE_PATH\" \"$MYAPP_BUNDLE_PATH\" \"$MYAPP_CAPTURE_MODE\" > \"$0\"", seenPath,
	}, bytes.NewBuffer(nil), &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runCaptureRun() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	seen, err := os.ReadFile(seenPath)
	if err != nil {
		t.Fatalf("read seen file: %v", err)
	}
	wantPrefix := bundlePath + "|" + bundlePath + "|run"
	if strings.TrimSpace(string(seen)) != wantPrefix {
		t.Fatalf("seen env = %q, want %q", strings.TrimSpace(string(seen)), wantPrefix)
	}

	var verifierReport verifypkg.VerifierReport
	if err := json.Unmarshal(stdout.Bytes(), &verifierReport); err != nil {
		t.Fatalf("decode verifier report: %v\nstdout=%s", err, stdout.String())
	}
	if verifierReport.ProfileID != "atb.profile.rag_answer" || !verifierReport.Pass {
		t.Fatalf("unexpected verifier report: %+v", verifierReport)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunCaptureRunEnvironmentIsolation(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	t.Setenv("ATB_TEST_PARENT_ENV", "preserved")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runCaptureRun([]string{
		"--bundle", bundlePath,
		"--env-prefix=TEST",
		"--",
		"sh", "-c", "env",
	}, bytes.NewBuffer(nil), &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runCaptureRun() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	env := parseEnvOutput(stdout.String())
	resolvedBundlePath, err := filepath.Abs(bundlePath)
	if err != nil {
		t.Fatalf("Abs(%q): %v", bundlePath, err)
	}
	want := map[string]string{
		"ATB_BUNDLE_PATH":     resolvedBundlePath,
		"ATB_CAPTURE_MODE":    "run",
		"TEST_BUNDLE_PATH":    resolvedBundlePath,
		"TEST_CAPTURE_MODE":   "run",
		"ATB_TEST_PARENT_ENV": "preserved",
	}
	for key, value := range want {
		if env[key] != value {
			t.Fatalf("%s = %q, want %q", key, env[key], value)
		}
	}
	if env["ATB_CAPTURE_RUN_ID"] == "" {
		t.Fatalf("ATB_CAPTURE_RUN_ID missing")
	}
	if env["TEST_CAPTURE_RUN_ID"] != env["ATB_CAPTURE_RUN_ID"] {
		t.Fatalf("TEST_CAPTURE_RUN_ID = %q, want %q", env["TEST_CAPTURE_RUN_ID"], env["ATB_CAPTURE_RUN_ID"])
	}
}

func TestRunCaptureRunEmptyEnvPrefixEmitsNoPrefixedVars(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	t.Setenv("MY_APP_BUNDLE_PATH", "")
	t.Setenv("MY_APP_CAPTURE_RUN_ID", "")
	t.Setenv("MY_APP_CAPTURE_MODE", "")
	if err := os.Unsetenv("MY_APP_BUNDLE_PATH"); err != nil {
		t.Fatalf("Unsetenv: %v", err)
	}
	if err := os.Unsetenv("MY_APP_CAPTURE_RUN_ID"); err != nil {
		t.Fatalf("Unsetenv: %v", err)
	}
	if err := os.Unsetenv("MY_APP_CAPTURE_MODE"); err != nil {
		t.Fatalf("Unsetenv: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runCaptureRun([]string{
		"--bundle", bundlePath,
		"--env-prefix=",
		"--",
		"sh", "-c", "env",
	}, bytes.NewBuffer(nil), &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runCaptureRun() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}

	env := parseEnvOutput(stdout.String())
	for _, key := range []string{"MY_APP_BUNDLE_PATH", "MY_APP_CAPTURE_RUN_ID", "MY_APP_CAPTURE_MODE"} {
		if _, ok := env[key]; ok {
			t.Fatalf("%s unexpectedly present in child environment", key)
		}
	}
	if env["ATB_BUNDLE_PATH"] == "" || env["ATB_CAPTURE_RUN_ID"] == "" || env["ATB_CAPTURE_MODE"] != "run" {
		t.Fatalf("canonical ATB env vars missing or incomplete: %+v", env)
	}
}

func TestRunCaptureRunWarnsWhenEnvPrefixDuplicatesATB(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("bundle.New() error = %v", err)
	}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("bundle.Save() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runCaptureRun([]string{
		"--bundle", bundlePath,
		"--env-prefix=ATB",
		"--",
		"sh", "-c", "true",
	}, bytes.NewBuffer(nil), &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runCaptureRun() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}
	const want = "atb capture run: warning: --env-prefix=ATB duplicates the canonical prefix\n"
	if stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestRunCaptureRunReturnsChildExitCode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runCaptureRun([]string{
		"--",
		"sh", "-c", "exit 7",
	}, bytes.NewBuffer(nil), &stdout, &stderr)
	if exitCode != 7 {
		t.Fatalf("runCaptureRun() exit code = %d, want %d", exitCode, 7)
	}
}

func parseEnvOutput(raw string) map[string]string {
	env := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		env[key] = value
	}
	return env
}

func writeRAGFixtureBundle(t *testing.T, path string) {
	t.Helper()

	b, err := bundle.New()
	if err != nil {
		t.Fatalf("bundle.New() error = %v", err)
	}
	appendAt := func(eventType string, data map[string]any, timestamp string) {
		t.Helper()
		if err := b.AppendWithOptions(eventType, data, &bundle.AppendOptions{Timestamp: timestamp}); err != nil {
			t.Fatalf("AppendWithOptions(%s) error = %v", eventType, err)
		}
	}

	appendAt(event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-1",
		"actor_id_hash": "sha256:user-1",
		"purpose_tag":   "rag_answer",
	}, "2026-04-24T09:00:10Z")
	appendAt(event.TypeAIModelInvoked, map[string]any{
		"request_id":              "req-1",
		"model_provider":          "openai",
		"model_id":                "gpt-4o-mini",
		"model_parameters_digest": "sha256:params-1",
		"prompt_digest":           "sha256:prompt-1",
	}, "2026-04-24T09:00:11Z")
	appendAt(event.TypeAIModelOutput, map[string]any{
		"request_id":    "req-1",
		"output_digest": "sha256:output-1",
		"output_format": "text/plain",
	}, "2026-04-24T09:00:12Z")
	appendAt(event.TypeAIResponseSent, map[string]any{
		"request_id":    "req-1",
		"output_digest": "sha256:output-1",
	}, "2026-04-24T09:00:13Z")

	if err := b.Save(path); err != nil {
		t.Fatalf("bundle.Save() error = %v", err)
	}
}
