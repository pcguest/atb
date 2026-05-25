// SPDX-License-Identifier: MIT
package agent

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareWorkspaceFirstRun(t *testing.T) {
	root := filepath.Join(t.TempDir(), "agent")
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := PrepareWorkspace(root, logger); err != nil {
		t.Fatalf("PrepareWorkspace: %v", err)
	}

	readmePath := filepath.Join(root, workspaceReadmeName)
	raw, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read workspace README: %v", err)
	}
	if !strings.Contains(string(raw), "AutomationSession") {
		t.Fatalf("README missing AutomationSession guidance: %q", raw)
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "first run") {
		t.Fatalf("expected first-run log message, got %q", logOutput)
	}
}

func TestPrepareWorkspaceExistingSessionsNotFirstRun(t *testing.T) {
	root := t.TempDir()
	sessionsDir := filepath.Join(root, "sessions", "sess_test")
	if err := os.MkdirAll(sessionsDir, 0750); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := PrepareWorkspace(root, logger); err != nil {
		t.Fatalf("PrepareWorkspace: %v", err)
	}
	if strings.Contains(logBuf.String(), "first run") {
		t.Fatalf("unexpected first-run log for non-empty workspace: %q", logBuf.String())
	}
}

func TestIsWorkspaceEmpty(t *testing.T) {
	root := t.TempDir()
	empty, err := isWorkspaceEmpty(root)
	if err != nil {
		t.Fatalf("isWorkspaceEmpty missing dir: %v", err)
	}
	if !empty {
		t.Fatal("expected missing dir to be empty")
	}

	if err := os.MkdirAll(root, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, workspaceReadmeName), []byte("x"), 0640); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	empty, err = isWorkspaceEmpty(root)
	if err != nil {
		t.Fatalf("isWorkspaceEmpty readme-only: %v", err)
	}
	if !empty {
		t.Fatal("expected readme-only dir to be empty")
	}
}
