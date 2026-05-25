// SPDX-License-Identifier: MIT
package agent

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSessionOpenHandler(t *testing.T) {
	srv := mustTestServer(t)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantErr    string
		validate   func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:       "happy path",
			body:       `{"actor_id":"actor-1","purpose_tag":"demo","profile_id":"atb.profile.test"}`,
			wantStatus: http.StatusCreated,
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp openSessionResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if !strings.HasPrefix(resp.SessionID, "sess_") {
					t.Fatalf("session_id = %q, want sess_ prefix", resp.SessionID)
				}
				if resp.BundlePath == "" {
					t.Fatal("expected bundle_path")
				}
				if resp.ActorID != "actor-1" {
					t.Fatalf("actor_id = %q, want actor-1", resp.ActorID)
				}
				if resp.ProfileID != "atb.profile.test" {
					t.Fatalf("profile_id = %q", resp.ProfileID)
				}
				if resp.PurposeTag != "demo" {
					t.Fatalf("purpose_tag = %q", resp.PurposeTag)
				}
			},
		},
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid JSON body",
		},
		{
			name:       "unknown field",
			body:       `{"actor_id":"a","extra":true}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid JSON body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/session/open", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantErr != "" {
				var errBody ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
					t.Fatalf("unmarshal error: %v", err)
				}
				if !strings.Contains(errBody.Error, tt.wantErr) {
					t.Fatalf("error = %q, want substring %q", errBody.Error, tt.wantErr)
				}
				return
			}
			if tt.validate != nil {
				tt.validate(t, rec)
			}
		})
	}
}

func TestSessionEventHandler(t *testing.T) {
	srv := mustTestServer(t)
	sessionID := openTestSession(t, srv, `{"actor_id":"actor-1"}`)

	tests := []struct {
		name       string
		sessionID  string
		body       string
		wantStatus int
		wantErr    string
	}{
		{
			name:       "happy path",
			sessionID:  sessionID,
			body:       `{"event_type":"ai.request.received","payload":{"request_id":"req-1"}}`,
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "missing event_type",
			sessionID:  sessionID,
			body:       `{"payload":{"x":1}}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "event_type is required",
		},
		{
			name:       "unknown session",
			sessionID:  "sess_deadbeefdeadbeefdeadbeefdeadbeef",
			body:       `{"event_type":"test"}`,
			wantStatus: http.StatusNotFound,
			wantErr:    "session not found",
		},
		{
			name:       "invalid json",
			sessionID:  sessionID,
			body:       `{`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid JSON body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/v1/session/" + tt.sessionID + "/event"
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus == http.StatusAccepted {
				var resp appendEventResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if resp.Status != "queued" {
					t.Fatalf("status = %q, want queued", resp.Status)
				}
				return
			}
			if tt.wantErr != "" {
				var errBody ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
					t.Fatalf("unmarshal error: %v", err)
				}
				if !strings.Contains(errBody.Error, tt.wantErr) {
					t.Fatalf("error = %q, want substring %q", errBody.Error, tt.wantErr)
				}
			}
		})
	}
}

func TestSessionCloseHandler(t *testing.T) {
	srv := mustTestServer(t)

	t.Run("happy path", func(t *testing.T) {
		sessionID := openTestSession(t, srv, `{"actor_id":"actor-1","profile_id":"atb.profile.test"}`)

		for _, body := range []string{
			`{"event_type":"e1","payload":{"n":1}}`,
			`{"event_type":"e2","payload":{"n":2}}`,
		} {
			postCapture(t, srv, "/v1/session/"+sessionID+"/event", body, http.StatusAccepted)
		}

		rec := postCapture(t, srv, "/v1/session/"+sessionID+"/close", `{}`, http.StatusOK)
		var resp closeSessionResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.SessionID != sessionID {
			t.Fatalf("session_id = %q, want %q", resp.SessionID, sessionID)
		}
		if resp.EventCount != 2 {
			t.Fatalf("event_count = %d, want 2", resp.EventCount)
		}
		if resp.BundlePath == "" {
			t.Fatal("expected bundle_path")
		}
		if resp.HeadHash == "" {
			t.Fatal("expected head_hash")
		}
		if resp.OpenedAt == "" || resp.ClosedAt == "" {
			t.Fatalf("opened_at=%q closed_at=%q", resp.OpenedAt, resp.ClosedAt)
		}
	})

	t.Run("unknown session", func(t *testing.T) {
		rec := postCapture(t, srv, "/v1/session/sess_unknownunknownunknownunknown/close", `{}`, http.StatusNotFound)
		var errBody ErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if errBody.Error != "session not found" {
			t.Fatalf("error = %q", errBody.Error)
		}
	})

	t.Run("closed session", func(t *testing.T) {
		sessionID := openTestSession(t, srv, `{}`)
		postCapture(t, srv, "/v1/session/"+sessionID+"/close", `{}`, http.StatusOK)
		rec := postCapture(t, srv, "/v1/session/"+sessionID+"/close", `{}`, http.StatusConflict)
		var errBody ErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if errBody.Error != "session closed" {
			t.Fatalf("error = %q, want session closed", errBody.Error)
		}
	})

	t.Run("append after close", func(t *testing.T) {
		sessionID := openTestSession(t, srv, `{}`)
		postCapture(t, srv, "/v1/session/"+sessionID+"/close", `{}`, http.StatusOK)
		rec := postCapture(t, srv, "/v1/session/"+sessionID+"/event", `{"event_type":"late"}`, http.StatusConflict)
		var errBody ErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if errBody.Error != "session closed" {
			t.Fatalf("error = %q, want session closed", errBody.Error)
		}
	})
}

func openTestSession(t *testing.T, srv *Server, body string) string {
	t.Helper()
	rec := postCapture(t, srv, "/v1/session/open", body, http.StatusCreated)
	var resp openSessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal open response: %v", err)
	}
	return resp.SessionID
}

func postCapture(t *testing.T, srv *Server, path, body string, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("POST %s: status = %d, want %d, body = %s", path, rec.Code, wantStatus, rec.Body.String())
	}
	return rec
}
