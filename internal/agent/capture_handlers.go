// SPDX-License-Identifier: MIT
package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

const maxCaptureBodyBytes = 1 << 20 // 1 MiB

// ErrorResponse is the standard JSON error envelope for capture endpoints.
type ErrorResponse struct {
	Error string `json:"error"`
}

type openSessionRequest struct {
	ActorID    string `json:"actor_id,omitempty"`
	PurposeTag string `json:"purpose_tag,omitempty"`
	ProfileID  string `json:"profile_id,omitempty"`
	BundlePath string `json:"bundle_path,omitempty"`
}

type openSessionResponse struct {
	SessionID  string `json:"session_id"`
	BundlePath string `json:"bundle_path"`
	ActorID    string `json:"actor_id,omitempty"`
	ProfileID  string `json:"profile_id,omitempty"`
	PurposeTag string `json:"purpose_tag,omitempty"`
}

type appendEventRequest struct {
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type appendEventResponse struct {
	Status string `json:"status"`
}

type closeSessionRequest struct {
	SnapshotName string `json:"snapshot_name,omitempty"`
}

type closeSessionResponse struct {
	SessionID  string `json:"session_id"`
	BundlePath string `json:"bundle_path"`
	ProfileID  string `json:"profile_id,omitempty"`
	HeadHash   string `json:"head_hash"`
	EventCount int    `json:"event_count"`
	OpenedAt   string `json:"opened_at"`
	ClosedAt   string `json:"closed_at"`
}

func (s *Server) handleSessionOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeCaptureError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req openSessionRequest
	if err := decodeJSONBody(r, &req); err != nil {
		s.logger.Warn("session open rejected", "reason", err.Error())
		writeCaptureError(w, http.StatusBadRequest, err.Error())
		return
	}

	params := OpenParams{
		ActorID:    strings.TrimSpace(req.ActorID),
		PurposeTag: strings.TrimSpace(req.PurposeTag),
		ProfileID:  strings.TrimSpace(req.ProfileID),
		BundlePath: strings.TrimSpace(req.BundlePath),
	}

	sessionID, err := s.bundleManager.OpenSession(r.Context(), params)
	if err != nil {
		s.logger.Error("session open failed", "error", err)
		writeCaptureError(w, http.StatusInternalServerError, "failed to open session")
		return
	}

	bundlePath := resolvedBundlePath(s.cfg.DataDir, sessionID, params.BundlePath)
	s.logger.Info("session opened",
		"session_id", sessionID,
		"actor_id", params.ActorID,
		"profile_id", params.ProfileID,
		"purpose_tag", params.PurposeTag,
	)

	writeJSON(w, http.StatusCreated, openSessionResponse{
		SessionID:  sessionID.String(),
		BundlePath: bundlePath,
		ActorID:    params.ActorID,
		ProfileID:  params.ProfileID,
		PurposeTag: params.PurposeTag,
	})
}

func (s *Server) handleSessionEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeCaptureError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sessionID := SessionID(r.PathValue("id"))
	if sessionID == "" {
		writeCaptureError(w, http.StatusBadRequest, "missing session id")
		return
	}

	var req appendEventRequest
	if err := decodeJSONBody(r, &req); err != nil {
		s.logger.Warn("session event rejected", "session_id", sessionID, "reason", err.Error())
		writeCaptureError(w, http.StatusBadRequest, err.Error())
		return
	}
	eventType := strings.TrimSpace(req.EventType)
	if eventType == "" {
		writeCaptureError(w, http.StatusBadRequest, "event_type is required")
		return
	}

	event := PendingEvent{
		EventType: eventType,
		Payload:   string(req.Payload),
	}
	if err := s.bundleManager.AppendEvent(r.Context(), sessionID, event); err != nil {
		status, msg := agentErrorHTTP(err)
		if status >= http.StatusInternalServerError {
			s.logger.Error("session event failed", "session_id", sessionID, "error", err)
		} else {
			s.logger.Warn("session event rejected", "session_id", sessionID, "status", status, "error", err)
		}
		writeCaptureError(w, status, msg)
		return
	}

	writeJSON(w, http.StatusAccepted, appendEventResponse{Status: "queued"})
}

func (s *Server) handleSessionClose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeCaptureError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sessionID := SessionID(r.PathValue("id"))
	if sessionID == "" {
		writeCaptureError(w, http.StatusBadRequest, "missing session id")
		return
	}

	var req closeSessionRequest
	if err := decodeJSONBody(r, &req); err != nil {
		s.logger.Warn("session close rejected", "session_id", sessionID, "reason", err.Error())
		writeCaptureError(w, http.StatusBadRequest, err.Error())
		return
	}

	meta, err := s.bundleManager.CloseSession(r.Context(), sessionID, CloseSessionOpts{
		SnapshotName: strings.TrimSpace(req.SnapshotName),
	})
	if err != nil {
		status, msg := agentErrorHTTP(err)
		if status >= http.StatusInternalServerError {
			s.logger.Error("session close failed", "session_id", sessionID, "error", err)
		} else {
			s.logger.Warn("session close rejected", "session_id", sessionID, "status", status, "error", err)
		}
		writeCaptureError(w, status, msg)
		return
	}

	s.logger.Info("session closed",
		"session_id", meta.SessionID,
		"event_count", meta.EventCount,
		"head_hash", meta.HeadHash,
	)

	writeJSON(w, http.StatusOK, closeSessionResponse{
		SessionID:  meta.SessionID.String(),
		BundlePath: meta.Path,
		ProfileID:  meta.ProfileID,
		HeadHash:   meta.HeadHash,
		EventCount: meta.EventCount,
		OpenedAt:   meta.CreatedAt.UTC().Format(timeRFC3339Nano),
		ClosedAt:   meta.ClosedAt.UTC().Format(timeRFC3339Nano),
	})
}

const timeRFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"

func resolvedBundlePath(dataDir string, sessionID SessionID, override string) string {
	if path := strings.TrimSpace(override); path != "" {
		return path
	}
	return filepath.Join(dataDir, "sessions", sessionID.String(), "bundle.atb")
}

func decodeJSONBody(r *http.Request, dst any) error {
	if r.Body == nil {
		return fmt.Errorf("request body is required")
	}
	defer r.Body.Close()

	limited := io.LimitReader(r.Body, maxCaptureBodyBytes+1)
	dec := json.NewDecoder(limited)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("invalid JSON body: multiple JSON values")
		}
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	return nil
}

func agentErrorHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, ErrSessionNotFound):
		return http.StatusNotFound, "session not found"
	case errors.Is(err, ErrSessionClosed):
		return http.StatusConflict, "session closed"
	default:
		return http.StatusInternalServerError, "internal error"
	}
}

func writeCaptureError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}
