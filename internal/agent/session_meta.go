// SPDX-License-Identifier: MIT
package agent

import (
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
