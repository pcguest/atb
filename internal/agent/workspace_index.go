// SPDX-License-Identifier: MIT
package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BundleSummary describes a closed agent session bundle for workspace listing.
type BundleSummary struct {
	ID         string
	SessionID  string
	BundlePath string
	ProfileID  string
	HeadHash   string
	EventCount int
	OpenedAt   time.Time
	ClosedAt   time.Time
}

// WorkspaceIndex scans the agent data directory for closed session bundles.
type WorkspaceIndex struct {
	dataDir string
}

// NewWorkspaceIndex constructs a read-only workspace index rooted at dataDir.
func NewWorkspaceIndex(dataDir string) *WorkspaceIndex {
	return &WorkspaceIndex{dataDir: dataDir}
}

// ListBundles returns closed session bundles sorted by closed_at descending.
// Only sessions with a persisted meta.json (written on close) are included.
func (idx *WorkspaceIndex) ListBundles(ctx context.Context) ([]BundleSummary, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	pattern := filepath.Join(idx.dataDir, "sessions", "*", sessionMetaFilename)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("agent: glob session meta: %w", err)
	}

	summaries := make([]BundleSummary, 0, len(matches))
	for _, metaPath := range matches {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		meta, err := readSessionMeta(metaPath)
		if err != nil {
			return nil, err
		}

		summary, err := bundleSummaryFromMeta(meta)
		if err != nil {
			return nil, fmt.Errorf("agent: session meta %s: %w", metaPath, err)
		}
		summaries = append(summaries, summary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].ClosedAt.After(summaries[j].ClosedAt)
	})

	return summaries, nil
}

func bundleSummaryFromMeta(meta sessionMetaFile) (BundleSummary, error) {
	sessionID := strings.TrimSpace(meta.SessionID)
	if sessionID == "" {
		return BundleSummary{}, fmt.Errorf("missing session_id")
	}

	openedAt, err := time.Parse(timeRFC3339Nano, meta.OpenedAt)
	if err != nil {
		return BundleSummary{}, fmt.Errorf("parse opened_at: %w", err)
	}
	closedAt, err := time.Parse(timeRFC3339Nano, meta.ClosedAt)
	if err != nil {
		return BundleSummary{}, fmt.Errorf("parse closed_at: %w", err)
	}

	return BundleSummary{
		ID:         sessionID,
		SessionID:  sessionID,
		BundlePath: meta.BundlePath,
		ProfileID:  meta.ProfileID,
		HeadHash:   meta.HeadHash,
		EventCount: meta.EventCount,
		OpenedAt:   openedAt.UTC(),
		ClosedAt:   closedAt.UTC(),
	}, nil
}
