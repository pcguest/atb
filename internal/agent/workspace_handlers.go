// SPDX-License-Identifier: MIT
package agent

import (
	"net/http"
)

type listBundlesResponse struct {
	Bundles []bundleSummaryResponse `json:"bundles"`
}

type bundleSummaryResponse struct {
	ID         string `json:"id"`
	SessionID  string `json:"session_id"`
	BundlePath string `json:"bundle_path"`
	ProfileID  string `json:"profile_id,omitempty"`
	HeadHash   string `json:"head_hash,omitempty"`
	EventCount int    `json:"event_count"`
	OpenedAt   string `json:"opened_at"`
	ClosedAt   string `json:"closed_at"`
}

func (s *Server) handleWorkspaceBundles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeCaptureError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	bundles, err := s.workspaceIndex.ListBundles(r.Context())
	if err != nil {
		s.logger.Error("workspace bundle list failed", "error", err)
		writeCaptureError(w, http.StatusInternalServerError, "failed to list bundles")
		return
	}

	resp := listBundlesResponse{
		Bundles: make([]bundleSummaryResponse, 0, len(bundles)),
	}
	for _, summary := range bundles {
		resp.Bundles = append(resp.Bundles, bundleSummaryResponse{
			ID:         summary.ID,
			SessionID:  summary.SessionID,
			BundlePath: summary.BundlePath,
			ProfileID:  summary.ProfileID,
			HeadHash:   summary.HeadHash,
			EventCount: summary.EventCount,
			OpenedAt:   summary.OpenedAt.UTC().Format(timeRFC3339Nano),
			ClosedAt:   summary.ClosedAt.UTC().Format(timeRFC3339Nano),
		})
	}

	writeJSON(w, http.StatusOK, resp)
}
