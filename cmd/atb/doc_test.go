// SPDX-License-Identifier: MIT
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDocGenOpenAPIWritesCanonicalSpec(t *testing.T) {
	output := filepath.Join(t.TempDir(), "nested", "openapi.yaml")
	if err := runDocGenOpenAPI([]string{"--output", output}); err != nil {
		t.Fatalf("runDocGenOpenAPI: %v", err)
	}
	body, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read generated spec: %v", err)
	}
	if !strings.HasPrefix(string(body), "openapi:") || body[len(body)-1] != '\n' {
		t.Fatalf("unexpected generated spec framing: %q", body)
	}

	equalsOutput := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := runDocGenOpenAPI([]string{"--output=" + equalsOutput}); err != nil {
		t.Fatalf("runDocGenOpenAPI equals form: %v", err)
	}
}

func TestRunDocGenOpenAPIRejectsInvalidArgumentsAndPaths(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing output", args: []string{"--output"}, want: "missing value"},
		{name: "empty output", args: []string{"--output="}, want: "cannot be empty"},
		{name: "unknown flag", args: []string{"--wat"}, want: "unknown flag"},
		{name: "positional argument", args: []string{"openapi.yaml"}, want: "unexpected argument"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := runDocGenOpenAPI(tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}

	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("block"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := runDocGenOpenAPI([]string{"--output", filepath.Join(blocker, "openapi.yaml")})
	if err == nil || !strings.Contains(err.Error(), "create output directory") {
		t.Fatalf("blocked path error = %v", err)
	}
}
