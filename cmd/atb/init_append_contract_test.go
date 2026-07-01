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
)

func TestInitManifestVersionParsing(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantVersion int
		wantArgs    []string
		wantErr     string
	}{
		{name: "default", args: []string{"--dry-run"}, wantVersion: 1, wantArgs: []string{"--dry-run"}},
		{name: "v1 separate", args: []string{"--manifest-version", "1"}, wantVersion: 1},
		{name: "v2 equals", args: []string{"--manifest-version=2", "--dry-run"}, wantVersion: 2, wantArgs: []string{"--dry-run"}},
		{name: "missing", args: []string{"--manifest-version"}, wantErr: "missing value"},
		{name: "empty", args: []string{"--manifest-version="}, wantErr: "expected 1|2"},
		{name: "unsupported", args: []string{"--manifest-version", "3"}, wantErr: "expected 1|2"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotArgs, gotVersion, err := parseInitManifestVersionFlag(tc.args)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || gotVersion != tc.wantVersion || strings.Join(gotArgs, ",") != strings.Join(tc.wantArgs, ",") {
				t.Fatalf("args=%v version=%d err=%v", gotArgs, gotVersion, err)
			}
		})
	}
}

func TestRunInitDryRunCreateAndNoopContracts(t *testing.T) {
	t.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	if code := runInit([]string{"--dry-run"}, &stdout, &stderr); code != exitSuccess {
		t.Fatalf("dry-run exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "would initialise") {
		t.Fatalf("dry-run stdout = %q", stdout.String())
	}
	if _, err := os.Stat(bundle.DefaultPath()); !os.IsNotExist(err) {
		t.Fatalf("dry run created bundle: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := runInit([]string{"--manifest-version=2", "--format", "json"}, &stdout, &stderr); code != exitSuccess {
		t.Fatalf("create exit = %d, stderr = %q", code, stderr.String())
	}
	var created mutationResult
	if err := json.Unmarshal(stdout.Bytes(), &created); err != nil {
		t.Fatalf("decode create output %q: %v", stdout.String(), err)
	}
	if created.Action != "init" || created.DryRun {
		t.Fatalf("created result = %+v", created)
	}
	if !strings.Contains(stderr.String(), "manifest version 2") {
		t.Fatalf("v2 warning = %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runInit(nil, &stdout, &stderr); code != exitSuccess {
		t.Fatalf("noop exit = %d", code)
	}
	if !strings.Contains(stdout.String(), "already exists") {
		t.Fatalf("noop stdout = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runInit([]string{"--dry-run", "--format=json"}, &stdout, &stderr); code != exitSuccess {
		t.Fatalf("JSON noop exit = %d", code)
	}
	var noop mutationResult
	if err := json.Unmarshal(stdout.Bytes(), &noop); err != nil {
		t.Fatalf("decode noop output %q: %v", stdout.String(), err)
	}
	if noop.Action != "noop" || !noop.DryRun {
		t.Fatalf("noop result = %+v", noop)
	}
}

func TestRunInitErrorsUseProvidedWriters(t *testing.T) {
	t.Chdir(t.TempDir())

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "manifest value", args: []string{"--manifest-version"}, want: "missing value"},
		{name: "mutation flag", args: []string{"--format", "yaml"}, want: "expected text|json"},
		{name: "positional", args: []string{"extra"}, want: "Usage: atb init"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := runInit(tc.args, &stdout, &stderr); code != exitUserError {
				t.Fatalf("exit = %d", code)
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tc.want)
			}
		})
	}

	var stdout, stderr bytes.Buffer
	code := runInit([]string{"extra", "--format=json"}, &stdout, &stderr)
	if code != exitUserError {
		t.Fatalf("JSON usage exit = %d", code)
	}
	var result mutationResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode JSON usage %q: %v", stdout.String(), err)
	}
	if result.Status != "error" || result.ExitCode != exitUserError {
		t.Fatalf("result = %+v", result)
	}
}

func TestRunInitWithOptionsSystemErrors(t *testing.T) {
	t.Run("invalid manifest", func(t *testing.T) {
		t.Chdir(t.TempDir())
		var stdout, stderr bytes.Buffer
		code := runInitWithOptions(initRunOptions{ManifestVersion: 99}, &stdout, &stderr)
		if code != exitSystemError || !strings.Contains(stderr.String(), "manifest") {
			t.Fatalf("exit=%d stderr=%q", code, stderr.String())
		}
	})

	t.Run("stat failure JSON", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if err := os.WriteFile(filepath.Dir(bundle.DefaultPath()), []byte("block"), 0o600); err != nil {
			t.Fatal(err)
		}
		var stdout, stderr bytes.Buffer
		code := runInitWithOptions(initRunOptions{OutputFormat: formatJSON}, &stdout, &stderr)
		if code != exitSystemError {
			t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
		}
		var result mutationResult
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("decode system error %q: %v", stdout.String(), err)
		}
		if result.Status != "error" || !strings.Contains(result.Error, "stat") {
			t.Fatalf("result = %+v", result)
		}
	})
}

func TestRunAppendValidationDryRunAndJSONContracts(t *testing.T) {
	t.Chdir(t.TempDir())

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "mutation flag", args: []string{"--format", "yaml"}, want: "expected text|json"},
		{name: "usage", args: nil, want: "Usage: atb append"},
		{name: "append flag", args: []string{"dev.session", "--wat"}, want: "unknown flag"},
		{name: "invalid JSON", args: []string{"dev.session", "{"}, want: "invalid JSON"},
		{name: "missing policy doc", args: []string{event.TypeAIPolicyDecision, `{"decision":"deny"}`, "--policy-doc", "missing.txt"}, want: "policy-doc: read"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := runAppend(tc.args, &stdout, &stderr); code != exitUserError {
				t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tc.want)
			}
		})
	}

	var stdout, stderr bytes.Buffer
	code := runAppend([]string{"dev.session", `{"ok":true}`, "--dry-run", "--format=json"}, &stdout, &stderr)
	if code != exitSuccess {
		t.Fatalf("JSON dry-run exit = %d, stderr = %q", code, stderr.String())
	}
	var result mutationResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode dry-run %q: %v", stdout.String(), err)
	}
	if result.Action != "preview_append" || !result.DryRun {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(bundle.DefaultPath()); !os.IsNotExist(err) {
		t.Fatalf("dry run created bundle: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(bundle.DefaultPath()), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bundle.DefaultPath(), []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = runAppend([]string{"dev.session", `{"ok":true}`}, &stdout, &stderr)
	if code == exitSuccess || !strings.Contains(stderr.String(), "load bundle") {
		t.Fatalf("invalid bundle exit=%d stderr=%q", code, stderr.String())
	}
}

func TestMutationFlagValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing format", args: []string{"--format"}, want: "missing value"},
		{name: "empty format", args: []string{"--format="}, want: "expected text|json"},
		{name: "missing lock wait", args: []string{"--lock-wait"}, want: "missing value"},
		{name: "invalid lock wait", args: []string{"--lock-wait=eventually"}, want: "invalid"},
		{name: "negative lock wait", args: []string{"--lock-wait=-1s"}, want: "non-negative"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, _, err := parseMutationFlagsWithLockWait(tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}

	filtered, format, dryRun, lockWait, err := parseMutationFlagsWithLockWait(
		[]string{"event", "--dry-run", "--format=json", "--lock-wait=250ms"},
	)
	if err != nil || format != formatJSON || !dryRun || lockWait.String() != "250ms" ||
		len(filtered) != 1 || filtered[0] != "event" {
		t.Fatalf("filtered=%v format=%q dry=%v lock=%v err=%v", filtered, format, dryRun, lockWait, err)
	}

	_, format, _, lockWait, err = parseMutationFlagsWithLockWait(
		[]string{"--format=json", "--format=text", "--lock-wait=1s", "--lock-wait=2s"},
	)
	if err != nil || format != formatText || lockWait != 2_000_000_000 {
		t.Fatalf("last-value-wins format=%q lock=%v err=%v", format, lockWait, err)
	}
}

func TestParseAppendCommandArgumentBoundaries(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "data missing", args: []string{"--data"}, want: "missing JSON"},
		{name: "data equals empty", args: []string{"--data="}, want: "missing JSON"},
		{name: "duplicate data", args: []string{`{"a":1}`, "--data", `{"b":2}`}, want: "expected <json>"},
		{name: "duplicate positional", args: []string{`{"a":1}`, `{"b":2}`}, want: "expected <json>"},
		{name: "actor missing", args: []string{`{}`, "--actor-id"}, want: "missing value"},
		{name: "actor empty", args: []string{`{}`, "--actor-id="}, want: "cannot be empty"},
		{name: "org missing", args: []string{`{}`, "--org-id"}, want: "missing value"},
		{name: "org empty", args: []string{`{}`, "--org-id="}, want: "cannot be empty"},
		{name: "workspace missing", args: []string{`{}`, "--workspace-id"}, want: "missing value"},
		{name: "workspace empty", args: []string{`{}`, "--workspace-id="}, want: "cannot be empty"},
		{name: "sign missing", args: []string{`{}`, "--sign-policy"}, want: "missing value"},
		{name: "sign empty", args: []string{`{}`, "--sign-policy="}, want: "cannot be empty"},
		{name: "policy doc missing", args: []string{`{}`, "--policy-doc"}, want: "missing value"},
		{name: "policy doc empty", args: []string{`{}`, "--policy-doc="}, want: "cannot be empty"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseAppendCommandArgs(tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v want=%q", err, tc.want)
			}
		})
	}

	got, err := parseAppendCommandArgs([]string{
		"--data={}",
		"--actor-id=actor",
		"--org-id=org",
		"--workspace-id=workspace",
		"--sign-policy=keys/key.pem",
		"--policy-doc=policy.md",
	})
	if err != nil || got.RawJSON != `{}` || got.SignPolicyKeyPath == "" || got.PolicyDocPath == "" ||
		got.Options.ActorID == nil || got.Options.OrgID == nil || got.Options.WorkspaceID == nil {
		t.Fatalf("parsed=%+v err=%v", got, err)
	}
}

func TestVersionMutationWriterAndVerifyResultBoundaries(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runVersion(nil, &stdout, &stderr); code != exitSuccess || !strings.Contains(stdout.String(), "atb ") {
		t.Fatalf("text version exit=%d stdout=%q", code, stdout.String())
	}
	stdout.Reset()
	if code := runVersion([]string{"--wat"}, &stdout, &stderr); code != exitUserError {
		t.Fatalf("unknown version flag exit=%d", code)
	}
	stderr.Reset()
	if code := runVersion([]string{"--json"}, verifyErrorWriter{err: errors.New("encode failed")}, &stderr); code != exitSystemError {
		t.Fatalf("version writer exit=%d", code)
	}
	if code := writeMutationJSON(verifyErrorWriter{err: errors.New("encode failed")}, mutationResult{}, &stderr, "test"); code != exitSystemError {
		t.Fatalf("mutation writer exit=%d", code)
	}

	nilResult := newVerifyResult("bundle.atb", nil, "fail")
	if nilResult.ChainLength != 0 || nilResult.HeadHash != "" {
		t.Fatalf("nil verify result=%+v", nilResult)
	}
	b := newTestBundle(t)
	result := newVerifyResult("bundle.atb", b, "ok")
	if result.ChainLength != len(b.Records) || result.HeadHash == "" || result.Status != "ok" {
		t.Fatalf("verify result=%+v", result)
	}
}
