// SPDX-License-Identifier: MIT
package agent

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const workspaceReadmeName = "README"

const workspaceReadmeContent = `# ATB Agent workspace

This directory holds closed session bundles written by the local ATB Agent.
Each session stores a tamper-evident, hash-chained .atb file under sessions/<id>/.

Configuration (highest priority first):
  ATB_AGENT_DATA_DIR, ATB_AGENT_LISTEN_ADDR environment variables
  agent.listen_addr and agent.data_dir in ~/.atb/config.json (or ./.atb/config.json)
  defaults: 127.0.0.1:6180 and ~/.atb/agent

To capture your first bundle:
  1. Keep this Agent running: atb agent run
  2. Instrument a workflow with TypeScript AutomationSession (ATB_AGENT_URL / ATB_AGENT_AUTO)
     or run: atb capture run

See docs/capture/overview.md in the ATB repository for details.
`

// PrepareWorkspace ensures the agent data directory exists and logs first-run guidance
// when the workspace has no session bundles yet.
func PrepareWorkspace(dataDir string, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	empty, err := isWorkspaceEmpty(dataDir)
	if err != nil {
		return fmt.Errorf("agent: inspect workspace: %w", err)
	}

	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return fmt.Errorf("agent: mkdir data dir: %w", err)
	}

	if !empty {
		return nil
	}

	readmePath := filepath.Join(dataDir, workspaceReadmeName)
	if _, err := os.Stat(readmePath); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(readmePath, []byte(workspaceReadmeContent), 0600); err != nil {
			return fmt.Errorf("agent: write workspace readme: %w", err)
		}
	}

	logger.Warn(
		"ATB agent first run: workspace is empty",
		"data_dir", dataDir,
		"next_steps", "instrument with AutomationSession or `atb capture run` while the agent is running",
	)
	return nil
}

func isWorkspaceEmpty(dataDir string) (bool, error) {
	entries, err := os.ReadDir(dataDir)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		switch entry.Name() {
		case workspaceReadmeName:
			continue
		default:
			return false, nil
		}
	}
	return true, nil
}
