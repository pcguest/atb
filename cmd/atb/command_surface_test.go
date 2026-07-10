// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/identity"
)

func TestPlainUsageListsStructuredCommands(t *testing.T) {
	text := captureStdout(t, printUsage)
	for _, cmd := range usageJSON().Commands {
		if !strings.Contains(text, cmd.Name) {
			t.Fatalf("plain help missing structured command %q", cmd.Name)
		}
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	defer func() { os.Stdout = old }()
	os.Stdout = w
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return string(out)
}

func TestBundleAndCaptureCommandRouting(t *testing.T) {
	t.Run("bundle", func(t *testing.T) {
		for _, tc := range []struct {
			args     []string
			wantCode int
			want     string
		}{
			{wantCode: exitUserError, want: "missing sub-command"},
			{args: []string{"help"}, wantCode: exitSuccess, want: "bundle new"},
			{args: []string{"unknown"}, wantCode: exitUserError, want: "unknown sub-command"},
			{args: []string{"new", "--help"}, wantCode: exitSuccess, want: "Usage: atb bundle new"},
		} {
			var stdout, stderr bytes.Buffer
			code := runBundle(tc.args, &stdout, &stderr)
			if code != tc.wantCode {
				t.Fatalf("runBundle(%q) code = %d, want %d", tc.args, code, tc.wantCode)
			}
			if output := stdout.String() + stderr.String(); !strings.Contains(output, tc.want) {
				t.Fatalf("runBundle(%q) output %q does not contain %q", tc.args, output, tc.want)
			}
		}
	})

	t.Run("capture", func(t *testing.T) {
		for _, tc := range []struct {
			args     []string
			wantCode int
			want     string
		}{
			{wantCode: exitUserError, want: "missing sub-command"},
			{args: []string{"help"}, wantCode: exitSuccess, want: "capture run"},
			{args: []string{"unknown"}, wantCode: exitUserError, want: "unknown sub-command"},
			{args: []string{"run", "--help"}, wantCode: exitSuccess, want: "capture run"},
			{args: []string{"run"}, wantCode: exitUserError, want: "missing -- before"},
		} {
			var stdout, stderr bytes.Buffer
			code := runCapture(tc.args, strings.NewReader(""), &stdout, &stderr)
			if code != tc.wantCode {
				t.Fatalf("runCapture(%q) code = %d, want %d", tc.args, code, tc.wantCode)
			}
			if output := stdout.String() + stderr.String(); !strings.Contains(output, tc.want) {
				t.Fatalf("runCapture(%q) output %q does not contain %q", tc.args, output, tc.want)
			}
		}
	})
}

func TestAnchorArgumentParsing(t *testing.T) {
	cfg, err := parseAnchorArgs(nil)
	if err != nil {
		t.Fatalf("parse defaults: %v", err)
	}
	if cfg.BundlePath == "" || cfg.TSAURL == "" {
		t.Fatalf("defaults are incomplete: %+v", cfg)
	}

	cfg, err = parseAnchorArgs([]string{"bundle.atb", "--tsa-url=https://tsa.example"})
	if err != nil {
		t.Fatalf("parse explicit values: %v", err)
	}
	if cfg.BundlePath != "bundle.atb" || cfg.TSAURL != "https://tsa.example" {
		t.Fatalf("explicit config = %+v", cfg)
	}

	for _, args := range [][]string{
		{"--tsa-url"},
		{"--tsa-url="},
		{"--unknown"},
		{"one.atb", "two.atb"},
	} {
		if _, err := parseAnchorArgs(args); err == nil {
			t.Errorf("parseAnchorArgs(%q) unexpectedly succeeded", args)
		}
	}
}

func TestInterceptArgumentParsing(t *testing.T) {
	t.Setenv("ATB_MORTISE_TOKEN", "test-token")
	t.Setenv("ATB_CUSTOS_TOKEN", "legacy-token")
	cfg, err := parseInterceptArgs([]string{
		"--port", "8443",
		"--bundle", "run.atb/bundle.atb",
		"--target", "openai, api.example.com",
		"--identity-map", "key-1=Patrick",
		"--mortise", "https://mortise.example",
		"--capture-bodies",
	})
	if err != nil {
		t.Fatalf("parse full config: %v", err)
	}
	if cfg.ListenAddr != "127.0.0.1:8443" ||
		cfg.BundlePath != "run.atb/bundle.atb" ||
		cfg.MortiseEndpoint != "https://mortise.example" ||
		cfg.MortiseToken != "test-token" ||
		!cfg.CaptureBodies ||
		cfg.IdentityMap["key-1"] != "Patrick" ||
		len(cfg.TargetHosts) == 0 {
		t.Fatalf("full config = %+v", cfg)
	}

	cfg, err = parseInterceptArgs([]string{
		"--port=9443",
		"--bundle=other.atb",
		"--target=anthropic",
		"--identity-map=key-2=Operator",
		"--custos=https://custody.example",
	})
	if err != nil {
		t.Fatalf("parse equals config: %v", err)
	}
	if extractPort(cfg.ListenAddr) != 9443 || cfg.IdentityMap["key-2"] != "Operator" {
		t.Fatalf("equals config = %+v", cfg)
	}
	if cfg.MortiseEndpoint != "https://custody.example" {
		t.Fatalf("legacy alias endpoint = %q", cfg.MortiseEndpoint)
	}

	errorCases := [][]string{
		nil,
		{"--help"},
		{"--port"},
		{"--port", "0", "--bundle", "x"},
		{"--port=65536", "--bundle", "x"},
		{"--bundle"},
		{"--target"},
		{"--identity-map"},
		{"--identity-map", "invalid", "--bundle", "x"},
		{"--mortise"},
		{"--custos"},
		{"--bundle", "x", "--mortise", "https://one.example", "--custos", "https://two.example"},
		{"--unknown"},
	}
	for _, args := range errorCases {
		_, err := parseInterceptArgs(args)
		if err == nil {
			t.Errorf("parseInterceptArgs(%q) unexpectedly succeeded", args)
		}
		if len(args) == 1 && args[0] == "--help" && !errors.Is(err, errInterceptHelp) {
			t.Errorf("help error = %v, want errInterceptHelp", err)
		}
	}

	if key, name, err := parseIdentityPair(" api-key = Pat "); err != nil || key != "api-key" || name != "Pat" {
		t.Fatalf("parseIdentityPair = %q, %q, %v", key, name, err)
	}
	for _, value := range []string{"", "key", "=name", "key="} {
		if _, _, err := parseIdentityPair(value); err == nil {
			t.Errorf("parseIdentityPair(%q) unexpectedly succeeded", value)
		}
	}
	if got := splitCSV(" openai, ,anthropic "); len(got) != 2 || got[0] != "openai" || got[1] != "anthropic" {
		t.Fatalf("splitCSV = %v", got)
	}
	if got := extractPort("invalid"); got != 8080 {
		t.Fatalf("extractPort invalid = %d", got)
	}
	if got := extractPort("127.0.0.1:not-a-port"); got != 8080 {
		t.Fatalf("extractPort non-numeric = %d", got)
	}
}

func TestSignArgumentAndBackendValidation(t *testing.T) {
	cfg, err := parseSignArgs([]string{
		"--bundle", "bundle.atb",
		"--key", "./keys/private.pem",
		"--out", "./signed.atb",
		"--backend", "https-http",
		"--key-id", "key-id",
		"--sign-endpoint", "https://sign.example",
		"--sign-api-key", " token with spaces ",
		"--fallback-local",
		"--lock-wait", "2s",
	})
	if err != nil {
		t.Fatalf("parse full sign config: %v", err)
	}
	if cfg.BundlePath != "bundle.atb" ||
		cfg.KeyPath != filepath.Clean("./keys/private.pem") ||
		cfg.OutputPath != filepath.Clean("./signed.atb") ||
		cfg.Backend != signBackendHTTPHTTP ||
		cfg.KeyID != "key-id" ||
		cfg.Endpoint != "https://sign.example" ||
		cfg.APIKey != " token with spaces " ||
		!cfg.FallbackLocal ||
		cfg.LockWait != 2*time.Second ||
		!cfg.LockWaitSet {
		t.Fatalf("full sign config = %+v", cfg)
	}

	cfg, err = parseSignArgs([]string{
		"--bundle=other.atb",
		"--key=key.pem",
		"--out=out.atb",
		"--backend=aws-kms",
		"--key-id=key",
		"--sign-endpoint=https://unused.example",
		"--sign-api-key=secret",
		"--lock-wait=0",
	})
	if err != nil {
		t.Fatalf("parse equals sign config: %v", err)
	}
	if cfg.OutputPath != "out.atb" || cfg.LockWait != 0 || !cfg.LockWaitSet {
		t.Fatalf("equals sign config = %+v", cfg)
	}

	for _, args := range [][]string{
		nil,
		{"--help"},
		{"--bundle"},
		{"--bundle", "x", "--key"},
		{"--bundle", "x", "--out"},
		{"--bundle", "x", "--backend"},
		{"--bundle", "x", "--key-id"},
		{"--bundle", "x", "--sign-endpoint"},
		{"--bundle", "x", "--sign-api-key"},
		{"--bundle", "x", "--lock-wait"},
		{"--bundle", "x", "--lock-wait=bad"},
		{"--bundle", "x", "--unknown"},
		{"--bundle", "x", "unexpected"},
	} {
		_, err := parseSignArgs(args)
		if err == nil {
			t.Errorf("parseSignArgs(%q) unexpectedly succeeded", args)
		}
		if len(args) == 1 && args[0] == "--help" && !errors.Is(err, errSignHelp) {
			t.Errorf("help error = %v, want errSignHelp", err)
		}
	}

	envConfig := applySignBackendEnv(signConfig{}, []string{
		"ATB_SIGN_BACKEND=vault",
		"ATB_SIGN_KEY_ID=transit/key",
		"ATB_SIGN_ENDPOINT=https://vault.example",
		"ATB_SIGN_API_KEY= secret ",
		"ATB_SIGN_FALLBACK_LOCAL=yes",
	})
	if envConfig.Backend != signBackendVault ||
		envConfig.KeyID != "transit/key" ||
		envConfig.Endpoint != "https://vault.example" ||
		envConfig.APIKey != " secret " ||
		!envConfig.FallbackLocal {
		t.Fatalf("environment sign config = %+v", envConfig)
	}
	if got := applySignBackendEnv(signConfig{}, nil); got.Backend != signBackendLocal {
		t.Fatalf("default backend = %q", got.Backend)
	}

	validations := []struct {
		name    string
		cfg     signConfig
		wantErr bool
	}{
		{name: "local", cfg: signConfig{Backend: signBackendLocal, KeyPath: "key.pem"}},
		{name: "local missing key", cfg: signConfig{Backend: signBackendLocal}, wantErr: true},
		{name: "http", cfg: signConfig{Backend: signBackendHTTPHTTP, Endpoint: "https://sign.example"}},
		{name: "http missing endpoint", cfg: signConfig{Backend: signBackendHTTPHTTP}, wantErr: true},
		{name: "http fallback missing key", cfg: signConfig{Backend: signBackendHTTPHTTP, Endpoint: "https://sign.example", FallbackLocal: true}, wantErr: true},
		{name: "aws", cfg: signConfig{Backend: signBackendAWSKMS, KeyID: "key"}},
		{name: "gcp", cfg: signConfig{Backend: signBackendGCPKMS, KeyID: "key"}},
		{name: "vault", cfg: signConfig{Backend: signBackendVault, KeyID: "key"}},
		{name: "remote missing key id", cfg: signConfig{Backend: signBackendVault}, wantErr: true},
		{name: "remote fallback missing local key", cfg: signConfig{Backend: signBackendAWSKMS, KeyID: "key", FallbackLocal: true}, wantErr: true},
		{name: "unknown", cfg: signConfig{Backend: "unknown"}, wantErr: true},
	}
	for _, tc := range validations {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.cfg
			err := validateSignConfig(&cfg, &bytes.Buffer{})
			if tc.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestImportOTelArgumentParsing(t *testing.T) {
	cfg, err := parseImportOTelArgs([]string{
		"--input", "trace.json",
		"--bundle", "bundle.atb",
		"--snapshot", "after-import",
		"--format", "JSON",
		"--max-input-size", "4096",
	})
	if err != nil {
		t.Fatalf("parse full OTLP config: %v", err)
	}
	if cfg.InputPath != "trace.json" ||
		cfg.BundlePath != "bundle.atb" ||
		cfg.SnapshotName != "after-import" ||
		cfg.Format != formatJSON ||
		cfg.MaxInputBytes != 4096 {
		t.Fatalf("full OTLP config = %+v", cfg)
	}

	cfg, err = parseImportOTelArgs([]string{
		"--input=-",
		"--bundle=other.atb",
		"--snapshot=done",
		"--format=text",
		"--max-input-size=8192",
	})
	if err != nil {
		t.Fatalf("parse equals OTLP config: %v", err)
	}
	if cfg.InputPath != "-" || cfg.MaxInputBytes != 8192 {
		t.Fatalf("equals OTLP config = %+v", cfg)
	}

	for _, args := range [][]string{
		nil,
		{"--help"},
		{"--input"},
		{"--input", "-", "--format"},
		{"--input", "-", "--format=xml"},
		{"--input", "-", "--max-input-size"},
		{"--input", "-", "--max-input-size", "0"},
		{"--input", "-", "--max-input-size=bad"},
		{"--input", "-", "--bundle"},
		{"--input", "-", "--bundle", "one", "--bundle=two"},
		{"--input", "-", "--snapshot"},
		{"--input", "-", "--unknown"},
	} {
		_, err := parseImportOTelArgs(args)
		if err == nil {
			t.Errorf("parseImportOTelArgs(%q) unexpectedly succeeded", args)
		}
		if len(args) == 1 && args[0] == "--help" && !errors.Is(err, errImportHelp) {
			t.Errorf("help error = %v, want errImportHelp", err)
		}
	}
}

func TestInspectArgumentParsing(t *testing.T) {
	cfg, err := parseInspectCommandArgs([]string{"bundle.atb", "--json", "--seq", "4"})
	if err != nil {
		t.Fatalf("parse inspect config: %v", err)
	}
	if cfg.BundlePath != "bundle.atb" || !cfg.JSON || !cfg.SeqSet || cfg.Seq != 4 {
		t.Fatalf("inspect config = %+v", cfg)
	}

	cfg, err = parseInspectCommandArgs([]string{"--bundle=other.atb", "--seq=7"})
	if err != nil {
		t.Fatalf("parse inspect equals config: %v", err)
	}
	if cfg.BundlePath != "other.atb" || cfg.Seq != 7 {
		t.Fatalf("inspect equals config = %+v", cfg)
	}

	for _, args := range [][]string{
		{"--help"},
		{"--bundle"},
		{"--bundle", "one", "--bundle=two"},
		{"--seq"},
		{"--seq=bad"},
		{"--unknown"},
		{"one.atb", "two.atb"},
	} {
		_, err := parseInspectCommandArgs(args)
		if err == nil {
			t.Errorf("parseInspectCommandArgs(%q) unexpectedly succeeded", args)
		}
		if len(args) == 1 && args[0] == "--help" && !errors.Is(err, errInspectHelp) {
			t.Errorf("help error = %v, want errInspectHelp", err)
		}
	}
}

func TestProfilesArgumentParsingAndHelpers(t *testing.T) {
	cfg, err := parseProfilesValidateArgs([]string{
		"--file", "./one.yaml",
		"--file=./two.yml",
		"--dir", "./profiles",
		"--dir=./more",
		"--format", "JSON",
	})
	if err != nil {
		t.Fatalf("parse profiles config: %v", err)
	}
	if len(cfg.Files) != 2 || len(cfg.Dirs) != 2 || cfg.Format != formatJSON {
		t.Fatalf("profiles config = %+v", cfg)
	}

	for _, args := range [][]string{
		{"--help"},
		{"--file"},
		{"--dir"},
		{"--format"},
		{"--format=xml"},
		{"--unknown"},
		{"unexpected"},
	} {
		_, err := parseProfilesValidateArgs(args)
		if err == nil {
			t.Errorf("parseProfilesValidateArgs(%q) unexpectedly succeeded", args)
		}
		if len(args) == 1 && args[0] == "--help" && !errors.Is(err, errProfilesHelp) {
			t.Errorf("help error = %v, want errProfilesHelp", err)
		}
	}

	if got := weightSum(map[string]float64{"a": 0.25, "b": 0.75}); got != 1 {
		t.Fatalf("weightSum = %v, want 1", got)
	}
	if got := detectProfileValidationKind([]byte("workflow_class: example\n")); got != profileValidationKindSchema {
		t.Fatalf("schema kind = %v", got)
	}
	if got := detectProfileValidationKind([]byte("display_name: Example\n")); got != profileValidationKindLegacy {
		t.Fatalf("legacy kind = %v", got)
	}
	if got := detectProfileValidationKind([]byte("required_events: []\n")); got != profileValidationKindDSL {
		t.Fatalf("DSL kind = %v", got)
	}
	if got := detectProfileValidationKind([]byte("not: [valid")); got != profileValidationKindUnknown {
		t.Fatalf("unknown kind = %v", got)
	}
}

func TestAgentCommandSurface(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantOutput string
	}{
		{name: "missing subcommand", wantCode: exitUserError, wantOutput: "atb agent run"},
		{name: "help", args: []string{"--help"}, wantCode: exitSuccess, wantOutput: "atb agent run"},
		{name: "unknown", args: []string{"unknown"}, wantCode: exitUserError, wantOutput: "unknown sub-command"},
		{name: "run help", args: []string{"run", "--help"}, wantCode: exitSuccess, wantOutput: "ATB_AGENT_LISTEN_ADDR"},
		{name: "run unknown argument", args: []string{"run", "--unknown"}, wantCode: exitUserError, wantOutput: "unknown argument"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runAgentCommand(tc.args, &stdout, &stderr)
			if code != tc.wantCode {
				t.Fatalf("exit code = %d, want %d", code, tc.wantCode)
			}
			if output := stdout.String() + stderr.String(); !strings.Contains(output, tc.wantOutput) {
				t.Fatalf("output %q does not contain %q", output, tc.wantOutput)
			}
		})
	}
}

func TestMCPCommandSurface(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantOutput string
	}{
		{name: "missing subcommand", wantCode: exitUserError, wantOutput: "atb mcp serve"},
		{name: "help", args: []string{"help"}, wantCode: exitSuccess, wantOutput: "atb mcp serve"},
		{name: "unknown", args: []string{"unknown"}, wantCode: exitUserError, wantOutput: "unknown sub-command"},
		{name: "serve help", args: []string{"serve", "--help"}, wantCode: exitSuccess, wantOutput: "JSON-RPC 2.0"},
		{name: "serve unknown argument", args: []string{"serve", "--unknown"}, wantCode: exitUserError, wantOutput: "unknown argument"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runMCPCommand(tc.args, &stdout, &stderr)
			if code != tc.wantCode {
				t.Fatalf("exit code = %d, want %d", code, tc.wantCode)
			}
			if output := stdout.String() + stderr.String(); !strings.Contains(output, tc.wantOutput) {
				t.Fatalf("output %q does not contain %q", output, tc.wantOutput)
			}
		})
	}
}

func TestIdentityCommandSurface(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantOutput string
	}{
		{name: "missing subcommand", wantCode: exitUserError, wantOutput: "Usage:"},
		{name: "help", args: []string{"help"}, wantCode: exitSuccess, wantOutput: "identity set"},
		{name: "unknown", args: []string{"unknown"}, wantCode: exitUserError, wantOutput: "unknown sub-command"},
		{name: "missing key value", args: []string{"set", "--key"}, wantCode: exitUserError, wantOutput: "missing value for --key"},
		{name: "missing name value", args: []string{"set", "--name"}, wantCode: exitUserError, wantOutput: "missing value for --name"},
		{name: "missing email value", args: []string{"set", "--email"}, wantCode: exitUserError, wantOutput: "missing value for --email"},
		{name: "unknown set argument", args: []string{"set", "--unknown"}, wantCode: exitUserError, wantOutput: "unknown argument"},
		{name: "missing required values", args: []string{"set"}, wantCode: exitSystemError, wantOutput: "key and display name are required"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempHome := t.TempDir()
			t.Setenv("HOME", tempHome)
			t.Setenv("USERPROFILE", tempHome) // os.UserHomeDir reads USERPROFILE on Windows
			var stdout, stderr bytes.Buffer
			code := runIdentityCommand(tc.args, &stdout, &stderr)
			if code != tc.wantCode {
				t.Fatalf("exit code = %d, want %d", code, tc.wantCode)
			}
			if output := stdout.String() + stderr.String(); !strings.Contains(output, tc.wantOutput) {
				t.Fatalf("output %q does not contain %q", output, tc.wantOutput)
			}
		})
	}
}

func TestIdentitySetPersistsResolvableMapping(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir reads USERPROFILE on Windows

	var stdout, stderr bytes.Buffer
	code := runIdentityCommand(
		[]string{"set", "--key", "secret-key", "--name", "Pat", "--email", "pat@example.com"},
		&stdout,
		&stderr,
	)
	if code != exitSuccess {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}

	path := filepath.Join(home, ".atb", "identity-map.yaml")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat identity map: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 { // Unix permission bits are not enforced on Windows file modes
		t.Fatalf("identity map mode = %o, want 600", info.Mode().Perm())
	}
	got, err := (identity.FileResolver{Path: path}).Resolve("secret-key")
	if err != nil {
		t.Fatalf("resolve identity mapping: %v", err)
	}
	if got.DisplayName != "Pat" || got.Email != "pat@example.com" {
		t.Fatalf("resolved identity = %+v", got)
	}
	if !strings.Contains(stdout.String(), path) {
		t.Fatalf("stdout %q does not identify written mapping %q", stdout.String(), path)
	}
}
