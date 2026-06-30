// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/identity"
)

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
			t.Setenv("HOME", t.TempDir())
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
	if info.Mode().Perm() != 0o600 {
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
