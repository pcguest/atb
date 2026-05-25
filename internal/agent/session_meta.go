// SPDX-License-Identifier: MIT
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const sessionMetaFilename = "meta.json"

// sessionMetaFile is persisted beside bundle.atb when a session closes.
// It provides lightweight metadata for the workspace index without
// re-parsing bundle event streams on every list call.
type sessionMetaFile struct {
	SessionID  string `json:"session_id"`
	BundlePath string `json:"bundle_path"`
	ProfileID  string `json:"profile_id,omitempty"`
	HeadHash   string `json:"head_hash"`
	EventCount int    `json:"event_count"`
	OpenedAt   string `json:"opened_at"`
	ClosedAt   string `json:"closed_at"`
}

func sessionMetaPath(dataDir string, sessionID SessionID) string {
	return filepath.Join(dataDir, "sessions", sessionID.String(), sessionMetaFilename)
}

func sessionMetaFromBundleMetadata(meta BundleMetadata) sessionMetaFile {
	return sessionMetaFile{
		SessionID:  meta.SessionID.String(),
		BundlePath: meta.Path,
		ProfileID:  meta.ProfileID,
		HeadHash:   meta.HeadHash,
		EventCount: meta.EventCount,
		OpenedAt:   meta.CreatedAt.UTC().Format(timeRFC3339Nano),
		ClosedAt:   meta.ClosedAt.UTC().Format(timeRFC3339Nano),
	}
}

func writeSessionMeta(dataDir string, meta BundleMetadata) error {
	payload, err := json.MarshalIndent(sessionMetaFromBundleMetadata(meta), "", "  ")
	if err != nil {
		return fmt.Errorf("agent: marshal session meta: %w", err)
	}
	path := sessionMetaPath(dataDir, meta.SessionID)
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("agent: mkdir session meta: %w", err)
	}
	if err := os.WriteFile(path, payload, 0640); err != nil {
		return fmt.Errorf("agent: write session meta: %w", err)
	}
	return nil
}

func readSessionMeta(path string) (sessionMetaFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return sessionMetaFile{}, fmt.Errorf("agent: read session meta: %w", err)
	}
	var meta sessionMetaFile
	if err := json.Unmarshal(raw, &meta); err != nil {
		return sessionMetaFile{}, fmt.Errorf("agent: parse session meta: %w", err)
	}
	return meta, nil
}
