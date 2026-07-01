// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

type verifyErrorWriter struct{ err error }

func (w verifyErrorWriter) Write([]byte) (int, error) { return 0, w.err }

func TestParseVerifyCommandArgsContracts(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "bundle value", args: []string{"--bundle"}, want: "missing value"},
		{name: "duplicate bundle", args: []string{"--bundle=a", "--bundle=b"}, want: "already set"},
		{name: "profile value", args: []string{"--profile"}, want: "missing value"},
		{name: "format value", args: []string{"--format"}, want: "missing value"},
		{name: "short format value", args: []string{"-f"}, want: "missing value"},
		{name: "invalid format", args: []string{"--format=yaml"}, want: "expected text|json"},
		{name: "roots value", args: []string{"--roots"}, want: "missing value"},
		{name: "policy value", args: []string{"--corroboration-policy"}, want: "missing value"},
		{name: "remote value", args: []string{"--remote"}, want: "missing value"},
		{name: "schema output value", args: []string{"--schema-out"}, want: "missing value"},
		{name: "unknown flag", args: []string{"--wat"}, want: "unknown flag"},
		{name: "extra path", args: []string{"one.atb", "two.atb"}, want: "at most one"},
		{name: "remote conflict", args: []string{"--remote=s3://bucket/key", "--bundle=bundle.atb"}, want: "mutually exclusive"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseVerifyCommandArgs(tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v want=%q", err, tc.want)
			}
		})
	}

	cfg, dryRun, err := parseVerifyArgsWithDryRun([]string{
		"--bundle=bundle.atb",
		"--profile=policy_decision",
		"--json",
		"--quiet",
		"-f=json",
		"--trace",
		"--with-anchor",
		"--with-snapshot-check",
		"--strict-source-signatures",
		"--roots=roots.pem",
		"--corroboration-policy=policy.json",
		"--schema",
		"--schema-out=schema.json",
		"--dry-run",
	})
	if err != nil || !dryRun || !cfg.JSON || !cfg.Quiet || !cfg.Trace || !cfg.WithAnchor ||
		!cfg.WithSnapshotCheck || !cfg.StrictSourceSignatures || !cfg.Schema {
		t.Fatalf("cfg=%+v dry=%v err=%v", cfg, dryRun, err)
	}
	if _, err := parseVerifyCommandArgs([]string{"--help"}); !errors.Is(err, errVerifyHelp) {
		t.Fatalf("help error=%v", err)
	}
}

func TestVerifySchemaAndLegacyJSONWriterErrors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("block"), 0o600); err != nil {
		t.Fatal(err)
	}
	code := writeVerifySchema(verifyCLIConfig{SchemaOut: filepath.Join(blocker, "schema.json")}, &stdout, &stderr)
	if code != exitSystemError || !strings.Contains(stderr.String(), "write schema") {
		t.Fatalf("schema path exit=%d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = writeVerifySchema(verifyCLIConfig{}, verifyErrorWriter{err: errors.New("write failed")}, &stderr)
	if code != exitSystemError || !strings.Contains(stderr.String(), "write failed") {
		t.Fatalf("schema writer exit=%d stderr=%q", code, stderr.String())
	}

	result := verifyResult{Status: "fail"}
	var encoded bytes.Buffer
	if err := writeLegacyVerifyJSON(&encoded, result, errors.New("integrity failed")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(encoded.String(), "integrity failed") {
		t.Fatalf("legacy JSON=%q", encoded.String())
	}
	if err := writeLegacyVerifyJSON(verifyErrorWriter{err: errors.New("encode failed")}, result, nil); err == nil {
		t.Fatal("expected legacy JSON writer error")
	}
}

func TestVerifyErrorClassificationAndBundleValidation(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{err: os.ErrNotExist, want: exitUserError},
		{err: os.ErrPermission, want: exitSystemError},
		{err: bundle.ErrMalformed, want: exitUserError},
		{err: errors.New("unmarshal event"), want: exitUserError},
		{err: errors.New("parse manifest"), want: exitUserError},
		{err: errors.New("unsupported manifest version"), want: exitUserError},
		{err: errors.New("scan bundle"), want: exitSystemError},
		{err: errors.New("transport"), want: exitSystemError},
	}
	for _, tc := range tests {
		if got := classifyVerifyBundleLoadError(tc.err); got != tc.want {
			t.Fatalf("classify(%v)=%d want=%d", tc.err, got, tc.want)
		}
	}
	if got := classifyVerifyRootsError(os.ErrNotExist); got != exitSystemError {
		t.Fatalf("missing roots=%d", got)
	}
	if got := classifyVerifyRootsError(errors.New("bad PEM")); got != exitUserError {
		t.Fatalf("bad roots=%d", got)
	}

	if err := validateVerifyBundle(nil); err == nil {
		t.Fatal("nil bundle validated")
	}
	if err := validateVerifyBundle(&bundle.Bundle{}); err == nil {
		t.Fatal("empty bundle validated")
	}
	notManifest := &bundle.Bundle{Records: []bundle.Record{{}}}
	if err := validateVerifyBundle(notManifest); err == nil {
		t.Fatal("non-manifest bundle validated")
	}
	if err := validateVerifyBundle(newTestBundle(t)); err != nil {
		t.Fatalf("valid bundle: %v", err)
	}
}

func TestVerifyProfilePathAndCorroborationPolicyContracts(t *testing.T) {
	for _, path := range []string{"profiles/custom.yaml", `profiles\custom.yml`, "custom.YAML"} {
		if !isVerifyProfilePath(path) {
			t.Fatalf("%q not detected as profile path", path)
		}
	}
	if isVerifyProfilePath("atb.profile.policy_decision") {
		t.Fatal("profile ID detected as path")
	}

	if option, err := buildCorroborationOption(verifyCLIConfig{}); err != nil || option != nil {
		t.Fatalf("disabled option=%v err=%v", option, err)
	}
	if option, err := buildCorroborationOption(verifyCLIConfig{WithAnchor: true}); err != nil || option == nil {
		t.Fatalf("default option=%v err=%v", option, err)
	}

	dir := t.TempDir()
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "invalid JSON", content: "{", want: "parse"},
		{name: "invalid policy", content: `{"anchor_bonus":0.1,"signature_bonus":0.1,"max_bonus":0.1}`, want: "component bonuses"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, strings.ReplaceAll(tc.name, " ", "-")+".json")
			if err := os.WriteFile(path, []byte(tc.content), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := buildCorroborationOption(verifyCLIConfig{WithAnchor: true, CorroborationPolicyPath: path})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v want=%q", err, tc.want)
			}
		})
	}
	validPath := filepath.Join(dir, "valid.json")
	if err := os.WriteFile(validPath, []byte(`{"anchor_bonus":0.02,"signature_bonus":0.01,"snapshot_bonus":0.01,"max_bonus":0.1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if option, err := buildCorroborationOption(verifyCLIConfig{WithAnchor: true, CorroborationPolicyPath: validPath}); err != nil || option == nil {
		t.Fatalf("valid option=%v err=%v", option, err)
	}
	_, err := buildCorroborationOption(verifyCLIConfig{
		WithAnchor: true, CorroborationPolicyPath: filepath.Join(dir, "missing.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "read") {
		t.Fatalf("missing policy error=%v", err)
	}
}

func TestLoadVerifyRootsContracts(t *testing.T) {
	if pool, err := loadVerifyRoots(""); err != nil || pool != nil {
		t.Fatalf("empty roots pool=%v err=%v", pool, err)
	}
	if _, err := loadVerifyRoots(filepath.Join(t.TempDir(), "missing.pem")); err == nil {
		t.Fatal("missing roots succeeded")
	}
	invalid := filepath.Join(t.TempDir(), "invalid.pem")
	if err := os.WriteFile(invalid, []byte("not PEM"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadVerifyRoots(invalid); err == nil || !strings.Contains(err.Error(), "no certificates") {
		t.Fatalf("invalid roots error=%v", err)
	}
	valid := filepath.Join(t.TempDir(), "roots.pem")
	if err := os.WriteFile(valid, buildUnrelatedRootPEM(t), 0o600); err != nil {
		t.Fatal(err)
	}
	pool, err := loadVerifyRoots(valid)
	if err != nil || pool == nil {
		t.Fatalf("valid roots pool=%v err=%v", pool, err)
	}
}

func TestWriteVerifierReportJSONFailure(t *testing.T) {
	report := verifypkg.VerifierReport{}
	err := writeVerifierReportJSON(verifyErrorWriter{err: fmt.Errorf("report write failed")}, report)
	if err == nil || !strings.Contains(err.Error(), "report write failed") {
		t.Fatalf("error=%v", err)
	}
}

func TestWriteVerifySelectionErrorContracts(t *testing.T) {
	var stderr bytes.Buffer
	writeVerifySelectionError(&stderr, "missing", fmt.Errorf("%w: missing", verifypkg.ErrProfileUnknown))
	if !strings.Contains(stderr.String(), "not found in config") {
		t.Fatalf("unknown profile stderr=%q", stderr.String())
	}
	stderr.Reset()
	writeVerifySelectionError(&stderr, "bad.yaml", errors.New("parse failed"))
	if !strings.Contains(stderr.String(), "parse failed") || !strings.Contains(stderr.String(), "Usage: atb verify") {
		t.Fatalf("parse stderr=%q", stderr.String())
	}
}

func TestWriteLegacyVerifyJSONShape(t *testing.T) {
	var output bytes.Buffer
	result := verifyResult{Status: "ok"}
	if err := writeLegacyVerifyJSON(&output, result, nil); err != nil {
		t.Fatal(err)
	}
	var decoded verifyResult
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil || decoded.Status != "ok" {
		t.Fatalf("decoded=%+v err=%v", decoded, err)
	}
}
