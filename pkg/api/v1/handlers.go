// SPDX-License-Identifier: MIT
package apiv1

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	atbembed "github.com/pcguest/atb"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/hash"
	"github.com/pcguest/atb/internal/sessionindex"
	verifypkg "github.com/pcguest/atb/internal/verify"
	atbauth "github.com/pcguest/atb/pkg/auth"
)

const (
	defaultEventsLimit = 200
	maxEventsLimit     = 1000
	revealAuthCookie   = "atb_reveal_token"
	revealAuthHeader   = "X-ATB-Viewer-Token"
	revealLegacyHeader = "X-ATB-Reveal-Token"
	sessionAuthHeader  = "X-ATB-Session-Token"
	sessionAuthParam   = "session_token"
)

const (
	defaultRevealRateLimit  = 10
	defaultRevealRateWindow = time.Minute
	revealRetryAfterSeconds = 60
	revealSidecarSchemaV1   = "atb.reveal.sidecar.v1"
	piiFieldsPathEnv        = "ATB_PII_FIELDS_PATH"
	piiFieldsRelativePath   = "docs/compliance/pii-fields.json"
)

// APIConfig configures the local viewer JSON API.
type APIConfig struct {
	BundlePath       string
	Bundle           *bundle.Bundle
	VerifyErr        error
	SessionToken     string // optional; if non-empty all read endpoints require X-ATB-Session-Token
	RevealAuthToken  string
	RevealRateLimit  int
	RevealRateWindow time.Duration
	ProfileReport    *ProfileReportSummary // optional; pre-computed at startup via --profile
	VerifierReport   *verifypkg.VerifierReport
	ProfilePath      string // optional; re-used by POST /api/v1/bundle/verify
	Sessions         []sessionindex.SessionEntry
	SessionsByActor  map[string][]sessionindex.SessionEntry
	SessionIndexErr  error
	SessionIndexed   bool
	JWTValidator     *atbauth.JWTValidator
}

// APIServer serves dashboard data from a verified (or failed-verification) bundle snapshot.
type APIServer struct {
	bundlePath         string
	b                  *bundle.Bundle
	verifyErr          error
	sessionToken       string
	revealAuthToken    string
	revealRateLimit    int
	revealRateWindow   time.Duration
	revealRateCounters map[string]revealRateCounter
	profileReport      *ProfileReportSummary
	verifierReport     *verifypkg.VerifierReport
	profilePath        string
	sessions           []sessionindex.SessionEntry
	sessionsByActor    map[string][]sessionindex.SessionEntry
	sessionIndexErr    error
	sessionIndexed     bool
	jwtValidator       *atbauth.JWTValidator
	mu                 sync.Mutex
	rateMu             sync.Mutex
}

// authMiddleware enforces authentication and sets the role in the context.
func (s *APIServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var authenticatedRole atbauth.Role = atbauth.RoleViewer // Default to viewer if no auth or OIDC not configured
		var authenticated bool

		// 1. Try session token authentication (for local-only dev mode)
		if s.sessionToken != "" {
			token := strings.TrimSpace(r.Header.Get(sessionAuthHeader))
			if token == "" {
				token = strings.TrimSpace(r.URL.Query().Get(sessionAuthParam))
			}
			if subtle.ConstantTimeCompare([]byte(token), []byte(s.sessionToken)) == 1 {
				authenticated = true
				authenticatedRole = atbauth.RoleAdmin // Session token implies admin access for local viewer
			}
		}

		// 2. If not authenticated by session token, try JWT validation
		if !authenticated && s.jwtValidator != nil {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")
				claims, err := s.jwtValidator.Validate(r.Context(), tokenString)
				if err != nil {
					writeJSON(w, http.StatusUnauthorized, APIError{Error: "invalid JWT"})
					return
				}
				role, err := s.jwtValidator.ExtractRole(claims)
				if err != nil || !role.IsValid() {
					// If no valid role claim, default to viewer
					authenticatedRole = atbauth.RoleViewer
				} else {
					authenticatedRole = role
				}
				authenticated = true
			}
		}

		// If no authentication mechanism is configured, or if authentication failed
		if !authenticated && s.sessionToken == "" && s.jwtValidator == nil {
			// No authentication configured, allow all access (dev mode)
			authenticatedRole = atbauth.RoleAdmin // Assume admin for unauthenticated dev mode
		} else if !authenticated {
			// Authentication configured but failed
			writeJSON(w, http.StatusUnauthorized, APIError{Error: "unauthorized"})
			return
		}

		// Store the authenticated role in context.
		ctx := atbauth.WithRole(r.Context(), authenticatedRole)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireRole is a helper function to check if the authenticated user has the required role.
func (s *APIServer) requireRole(w http.ResponseWriter, r *http.Request, requiredRole atbauth.Role) bool {
	role, ok := atbauth.GetRoleFromContext(r.Context())
	if !ok || !role.HasPermission(requiredRole) {
		writeJSON(w, http.StatusForbidden, APIError{Error: "forbidden"})
		return false
	}
	return true
}

type revealRateCounter struct {
	windowStart time.Time
	count       int
}

// NewAPIServer constructs a local-only viewer API server.
func NewAPIServer(cfg APIConfig) *APIServer {
	revealRateLimit := cfg.RevealRateLimit
	if revealRateLimit <= 0 {
		revealRateLimit = defaultRevealRateLimit
	}

	revealRateWindow := cfg.RevealRateWindow
	if revealRateWindow <= 0 {
		revealRateWindow = defaultRevealRateWindow
	}

	return &APIServer{
		bundlePath:         cfg.BundlePath,
		b:                  cfg.Bundle,
		verifyErr:          cfg.VerifyErr,
		sessionToken:       strings.TrimSpace(cfg.SessionToken),
		revealAuthToken:    strings.TrimSpace(cfg.RevealAuthToken),
		revealRateLimit:    revealRateLimit,
		revealRateWindow:   revealRateWindow,
		revealRateCounters: make(map[string]revealRateCounter),
		profileReport:      cfg.ProfileReport,
		verifierReport:     cfg.VerifierReport,
		profilePath:        cfg.ProfilePath,
		sessions:           cfg.Sessions,
		sessionsByActor:    cfg.SessionsByActor,
		sessionIndexErr:    cfg.SessionIndexErr,
		sessionIndexed:     cfg.SessionIndexed,
		jwtValidator:       cfg.JWTValidator,
	}
}

// Register mounts dashboard API endpoints on the provided mux.
func (s *APIServer) Register(mux *http.ServeMux) {
	mux.Handle("/api/v1/verification", s.authMiddleware(http.HandlerFunc(s.handleVerification)))
	mux.Handle("/api/v1/bundle/meta", s.authMiddleware(http.HandlerFunc(s.handleBundleMeta)))
	mux.Handle("/api/v1/bundle/events", s.authMiddleware(http.HandlerFunc(s.handleBundleEvents)))
	mux.Handle("/api/v1/bundle/graph", s.authMiddleware(http.HandlerFunc(s.handleBundleGraph)))
	mux.Handle("/api/v1/bundle/profile", s.authMiddleware(http.HandlerFunc(s.handleBundleProfile)))
	mux.Handle("/api/v1/bundle/verify", s.authMiddleware(http.HandlerFunc(s.handleBundleVerify)))
	mux.Handle("/api/v1/bundle/verify/report", s.authMiddleware(http.HandlerFunc(s.handleBundleVerifyReport)))
	mux.Handle("/api/v1/privacy/reveal", s.authMiddleware(http.HandlerFunc(s.handlePrivacyReveal)))
	mux.Handle("/api/v1/sessions", s.authMiddleware(http.HandlerFunc(s.handleSessions)))
	mux.Handle("/api/v1/sessions/by-actor", s.authMiddleware(http.HandlerFunc(s.handleSessionsByActor)))
	mux.Handle("/api/v1/schema/status", s.authMiddleware(http.HandlerFunc(s.handleSchemaStatus)))
}

// SessionsResponse is the API response for GET /api/v1/sessions.
type SessionsResponse struct {
	Sessions []sessionindex.SessionEntry `json:"sessions"`
}

// ActorsResponse is the API response for GET /api/v1/sessions/by-actor.
type ActorsResponse struct {
	Actors map[string][]sessionindex.SessionEntry `json:"actors"`
}

type SessionIndexErrorResponse struct {
	Error   string                   `json:"error"`
	Bundles []sessionindex.LoadError `json:"bundles"`
}

// @OpenAPI
// @Summary      Get all sessions
// @Description  Returns a list of all sessions from the indexed bundles.
// @Tags         viewer
// @Produce      json
// @Success      200  {object}  SessionsResponse
// @Failure      403  {object}  APIError
// @Failure      405  {object}  APIError
// @Router       /api/v1/sessions [get]
func (s *APIServer) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}
	if !s.requireRole(w, r, atbauth.RoleViewer) {
		return
	}
	if !s.requireVerified(w) {
		return
	}

	sessions, err := s.sessionsForRequest(r)
	if err != nil {
		writeSessionIndexError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, SessionsResponse{Sessions: sessions})
}

// @OpenAPI
// @Summary      Get sessions grouped by actor
// @Description  Returns a map of sessions grouped by actor display name.
// @Tags         viewer
// @Produce      json
// @Success      200  {object}  ActorsResponse
// @Failure      403  {object}  APIError
// @Failure      405  {object}  APIError
// @Router       /api/v1/sessions/by-actor [get]
func (s *APIServer) handleSessionsByActor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}
	if !s.requireRole(w, r, atbauth.RoleViewer) {
		return
	}
	if !s.requireVerified(w) {
		return
	}

	actors, err := s.actorsForRequest(r)
	if err != nil {
		writeSessionIndexError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, ActorsResponse{Actors: actors})
}

func (s *APIServer) actorsForRequest(r *http.Request) (map[string][]sessionindex.SessionEntry, error) {
	if strings.TrimSpace(r.URL.Query().Get("bundle_dir")) == "" && s.sessionIndexed {
		if s.sessionIndexErr != nil {
			return cloneActorSessions(s.sessionsByActor), s.sessionIndexErr
		}
		return cloneActorSessions(s.sessionsByActor), nil
	}
	sessions, err := s.sessionsForRequest(r)
	if err != nil {
		return nil, err
	}
	return sessionindex.GroupByActor(sessions), nil
}

func (s *APIServer) sessionsForRequest(r *http.Request) ([]sessionindex.SessionEntry, error) {
	if bundleDir := strings.TrimSpace(r.URL.Query().Get("bundle_dir")); bundleDir != "" {
		return buildSessionIndexForDir(r.Context(), bundleDir)
	}
	if s.sessionIndexed {
		if s.sessionIndexErr != nil {
			return append([]sessionindex.SessionEntry(nil), s.sessions...), s.sessionIndexErr
		}
		return append([]sessionindex.SessionEntry(nil), s.sessions...), nil
	}
	return buildSessionIndexForDir(r.Context(), filepath.Dir(s.bundlePath))
}

func buildSessionIndexForDir(ctx context.Context, dir string) ([]sessionindex.SessionEntry, error) {
	paths, err := sessionBundlePaths(dir)
	if err != nil {
		return nil, err
	}
	return sessionindex.BuildIndex(ctx, paths)
}

func sessionBundlePaths(dir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.atb"))
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func cloneActorSessions(in map[string][]sessionindex.SessionEntry) map[string][]sessionindex.SessionEntry {
	out := make(map[string][]sessionindex.SessionEntry, len(in))
	for key, entries := range in {
		out[key] = append([]sessionindex.SessionEntry(nil), entries...)
	}
	return out
}

func writeSessionIndexError(w http.ResponseWriter, err error) {
	if loadErrors := sessionindex.ExtractLoadErrors(err); len(loadErrors) > 0 {
		writeJSON(w, http.StatusForbidden, SessionIndexErrorResponse{
			Error:   "one or more session bundles failed verification",
			Bundles: loadErrors,
		})
		return
	}
	writeJSON(w, http.StatusInternalServerError, APIError{Error: "session index unavailable"})
}

// EventTypeStatusDTO describes the contract health of a single event type in the
// loaded bundle: whether the schema declares it, how many records were observed,
// and how many of those observed records are missing a schema-required field.
type EventTypeStatusDTO struct {
	Type           string   `json:"type"`
	Criticality    string   `json:"criticality"`
	Declared       bool     `json:"declared"`
	Observed       int      `json:"observed"`
	RequiredFields []string `json:"required_fields"`
	Incomplete     int      `json:"incomplete"`
	MissingFields  []string `json:"missing_fields,omitempty"`
}

// SchemaStatusResponse is the API response for GET /api/v1/schema/status. It
// gives operators an ecosystem-level view of contract health for the loaded
// bundle rather than only per-session detail: declared-vs-observed event types,
// required-field completeness, and any undeclared (unknown) types present.
type SchemaStatusResponse struct {
	SchemaSource     string               `json:"schema_source"`
	DeclaredTypes    int                  `json:"declared_types"`
	ObservedTypes    int                  `json:"observed_types"`
	TotalEvents      int                  `json:"total_events"`
	IncompleteEvents int                  `json:"incomplete_events"`
	UndeclaredTypes  []string             `json:"undeclared_types"`
	Types            []EventTypeStatusDTO `json:"types"`
}

// @OpenAPI
// @Summary      Schema contract status
// @Description  Returns ecosystem-level contract health for the loaded bundle: declared vs observed event types, required-field completeness, and any undeclared types.
// @Tags         viewer
// @Produce      json
// @Success      200  {object}  SchemaStatusResponse
// @Failure      403  {object}  APIError
// @Failure      405  {object}  APIError
// @Router       /api/v1/schema/status [get]
func (s *APIServer) handleSchemaStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}
	if !s.requireRole(w, r, atbauth.RoleViewer) {
		return
	}
	if !s.requireVerified(w) {
		return
	}

	writeJSON(w, http.StatusOK, s.computeSchemaStatus())
}

// computeSchemaStatus walks the loaded bundle and scores every event type
// against the schema-generated contract (event.RegistryGenerated and
// event.RequiredFieldsGenerated). It is pure given the bundle snapshot so it can
// be unit-tested without a live server.
func (s *APIServer) computeSchemaStatus() SchemaStatusResponse {
	declaredOrder := event.RegistryGenerated
	requiredByType := event.RequiredFieldsGenerated

	observed := make(map[string]int)
	missingByType := make(map[string]map[string]int) // type -> required field -> records missing it
	incompleteByType := make(map[string]int)
	totalEvents := 0
	incompleteEvents := 0

	if s.b != nil {
		for _, record := range s.b.Records {
			eventType := record.Event.Type
			observed[eventType]++
			totalEvents++

			required := requiredByType[eventType]
			if len(required) == 0 {
				continue
			}
			data, ok := record.Event.Data.(map[string]any)
			// A non-map payload (e.g. the legacy v1 manifest string) cannot be
			// field-checked here; do not flag it as incomplete on shape alone.
			if !ok {
				continue
			}
			recordMissing := false
			for _, field := range required {
				if _, present := data[field]; !present {
					if missingByType[eventType] == nil {
						missingByType[eventType] = make(map[string]int)
					}
					missingByType[eventType][field]++
					recordMissing = true
				}
			}
			if recordMissing {
				incompleteByType[eventType]++
				incompleteEvents++
			}
		}
	}

	types := make([]EventTypeStatusDTO, 0, len(declaredOrder))
	declaredSet := make(map[string]struct{}, len(declaredOrder))
	for _, info := range declaredOrder {
		declaredSet[info.Type] = struct{}{}
		types = append(types, EventTypeStatusDTO{
			Type:           info.Type,
			Criticality:    info.Criticality,
			Declared:       true,
			Observed:       observed[info.Type],
			RequiredFields: normaliseStringSlice(requiredByType[info.Type]),
			Incomplete:     incompleteByType[info.Type],
			MissingFields:  sortedKeys(missingByType[info.Type]),
		})
	}

	// Observed types the schema does not declare are surfaced explicitly so a
	// drifted or rogue producer is visible rather than silently absorbed.
	var undeclared []string
	for eventType := range observed {
		if _, ok := declaredSet[eventType]; ok {
			continue
		}
		undeclared = append(undeclared, eventType)
		types = append(types, EventTypeStatusDTO{
			Type:           eventType,
			Criticality:    "",
			Declared:       false,
			Observed:       observed[eventType],
			RequiredFields: []string{},
			Incomplete:     0,
		})
	}
	sort.Strings(undeclared)

	observedTypes := 0
	for _, count := range observed {
		if count > 0 {
			observedTypes++
		}
	}

	return SchemaStatusResponse{
		SchemaSource:     "schemas/event.v1.json",
		DeclaredTypes:    len(declaredOrder),
		ObservedTypes:    observedTypes,
		TotalEvents:      totalEvents,
		IncompleteEvents: incompleteEvents,
		UndeclaredTypes:  undeclared,
		Types:            types,
	}
}

func normaliseStringSlice(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	return in
}

func sortedKeys(m map[string]int) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// NewTamperHandler returns a static red warning page for invalid hash chains.
func NewTamperHandler(bundlePath string, verifyErr error) http.Handler {
	message := html.EscapeString(verifyErr.Error())
	path := html.EscapeString(bundlePath)
	body := fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>ATB Viewer — Tamper Detected</title>
  <style>
    body { margin: 0; font-family: ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #0f0a0a; color: #fee2e2; }
    main { max-width: 900px; margin: 48px auto; padding: 24px; }
    .banner { border: 2px solid #ef4444; background: #7f1d1d; border-radius: 12px; padding: 18px; font-weight: 700; font-size: 1.2rem; }
    .meta { margin-top: 12px; color: #fecaca; }
    code { background: #1f2937; color: #e5e7eb; padding: 2px 6px; border-radius: 6px; }
  </style>
</head>
<body>
  <main>
    <div class="banner">TAMPER DETECTED — HASH CHAIN VERIFICATION FAILED</div>
    <p class="meta">Bundle: <code>%s</code></p>
    <p class="meta">Details: <code>%s</code></p>
    <p class="meta">Event data is blocked. Review <code>/api/v1/verification</code> for machine-readable status.</p>
  </main>
</body>
</html>`, path, message)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})
}

// @OpenAPI
// @Summary      Verify bundle hash chain status
// @Description  Reports whether the loaded bundle chain is valid or invalid.
// @Tags         viewer
// @Produce      json
// @Success      200  {object}  VerificationResponse
// @Failure      405  {object}  APIError
// @Router       /api/v1/verification [get]
func (s *APIServer) handleVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}
	if !s.requireRole(w, r, atbauth.RoleViewer) {
		return
	}

	out := VerificationResponse{
		Status:      "valid",
		Message:     "hash chain verified",
		BundlePath:  s.bundlePath,
		ChainLength: len(s.b.Records),
	}
	if len(s.b.Records) > 0 {
		out.HeadHash = s.b.Records[len(s.b.Records)-1].Hash
	}
	if s.verifyErr != nil {
		out.Status = "invalid"
		out.Message = s.verifyErr.Error()
	}
	writeJSON(w, http.StatusOK, out)
}

// @OpenAPI
// @Summary      Get bundle metadata
// @Description  Returns aggregate counts and timestamp bounds for the loaded bundle.
// @Tags         viewer
// @Produce      json
// @Success      200  {object}  BundleMetaResponse
// @Failure      403  {object}  APIError
// @Failure      405  {object}  APIError
// @Router       /api/v1/bundle/meta [get]
func (s *APIServer) handleBundleMeta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}
	if !s.requireRole(w, r, atbauth.RoleViewer) {
		return
	}
	if !s.requireVerified(w) {
		return
	}

	typeCounts := make(map[string]int)
	firstTS := ""
	lastTS := ""
	for _, record := range s.b.Records {
		typeCounts[record.Event.Type]++
		if record.Event.Type == bundle.ManifestEventType {
			continue
		}
		ts := record.Event.Timestamp
		if ts == "" {
			ts = extractTimestamp(record.Event.Data)
		}
		if ts == "" {
			continue
		}
		if firstTS == "" || ts < firstTS {
			firstTS = ts
		}
		if lastTS == "" || ts > lastTS {
			lastTS = ts
		}
	}

	verifiedAt := ""
	if len(s.b.Records) > 0 && s.verifyErr == nil {
		verifiedAt = time.Now().UTC().Format(time.RFC3339)
	}

	out := BundleMetaResponse{
		BundlePath:      s.bundlePath,
		EventCount:      len(s.b.Records),
		TypeCounts:      typeCounts,
		FirstTimestamp:  firstTS,
		LastTimestamp:   lastTS,
		GenesisHash:     hash.GenesisHash,
		VerifiedAt:      verifiedAt,
		Verified:        true,
		VerificationMsg: "hash chain verified",
	}
	writeJSON(w, http.StatusOK, out)
}

// @OpenAPI
// @Summary      Get paginated bundle events
// @Description  Returns masked event payloads with trace/span context.
// @Tags         viewer
// @Produce      json
// @Param        offset  query     int  false  "Zero-based offset"
// @Param        limit   query     int  false  "Page size (max 1000)"
// @Success      200     {object}  BundleEventsResponse
// @Failure      400     {object}  APIError
// @Failure      403     {object}  APIError
// @Failure      405     {object}  APIError
// @Router       /api/v1/bundle/events [get]
func (s *APIServer) handleBundleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}
	if !s.requireRole(w, r, atbauth.RoleViewer) {
		return
	}
	if !s.requireVerified(w) {
		return
	}

	offset, limit, err := parsePagination(r.URL.Query().Get("offset"), r.URL.Query().Get("limit"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		return
	}

	// Exclude the manifest record: it is bundle metadata (seq=0), not a pageable workflow event.
	// handleBundleMeta applies the same filter for consistency.
	workflowRecords := make([]bundle.Record, 0, len(s.b.Records))
	for _, r := range s.b.Records {
		if r.Event.Type != bundle.ManifestEventType {
			workflowRecords = append(workflowRecords, r)
		}
	}

	total := len(workflowRecords)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	events := make([]EventRecordDTO, 0, end-offset)
	for _, record := range workflowRecords[offset:end] {
		events = append(events, EventRecordDTO{
			Sequence:     record.Event.Sequence,
			Type:         record.Event.Type,
			Hash:         record.Hash,
			PrevHash:     record.Event.PrevHash,
			Timestamp:    record.Event.Timestamp,
			TraceID:      record.Event.TraceID,
			SpanID:       record.Event.SpanID,
			ParentSpanID: record.Event.ParentSpanID,
			Data:         maskSensitive(record.Event.Data),
		})
	}

	writeJSON(w, http.StatusOK, BundleEventsResponse{
		Offset: offset,
		Limit:  limit,
		Total:  total,
		Events: events,
	})
}

// @OpenAPI
// @Summary      Get trace/span graph view
// @Description  Returns graph nodes and edges derived from event sequence and span links.
// @Tags         viewer
// @Produce      json
// @Success      200  {object}  BundleGraphResponse
// @Failure      403  {object}  APIError
// @Failure      405  {object}  APIError
// @Router       /api/v1/bundle/graph [get]
func (s *APIServer) handleBundleGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}
	if !s.requireRole(w, r, atbauth.RoleViewer) {
		return
	}
	if !s.requireVerified(w) {
		return
	}

	nodes := make([]GraphNodeDTO, 0, len(s.b.Records))
	edges := make([]GraphEdgeDTO, 0, len(s.b.Records)*2)
	spanNodeMap := make(map[string]string)

	for _, record := range s.b.Records {
		nodeID := fmt.Sprintf("evt-%d", record.Event.Sequence)
		nodeType := graphNodeType(record.Event.Type)
		label := fmt.Sprintf("%s (#%d)", record.Event.Type, record.Event.Sequence)
		nodes = append(nodes, GraphNodeDTO{
			ID:    nodeID,
			Label: label,
			Type:  nodeType,
		})
		if record.Event.Sequence > 0 {
			edges = append(edges, GraphEdgeDTO{
				ID:     fmt.Sprintf("seq-%d-%d", record.Event.Sequence-1, record.Event.Sequence),
				Source: fmt.Sprintf("evt-%d", record.Event.Sequence-1),
				Target: nodeID,
				Label:  "sequence",
			})
		}

		_, spanID, _ := extractSpanContext(record.Event.Data)
		if spanID != "" {
			if _, exists := spanNodeMap[spanID]; !exists {
				spanNodeMap[spanID] = nodeID
			}
		}
	}

	seenEdges := make(map[string]struct{})
	for _, record := range s.b.Records {
		_, spanID, parentSpanID := extractSpanContext(record.Event.Data)
		if spanID == "" || parentSpanID == "" {
			continue
		}
		childNode, okChild := spanNodeMap[spanID]
		parentNode, okParent := spanNodeMap[parentSpanID]
		if !okChild || !okParent || childNode == parentNode {
			continue
		}
		edgeID := fmt.Sprintf("span-%s-%s", parentNode, childNode)
		if _, exists := seenEdges[edgeID]; exists {
			continue
		}
		seenEdges[edgeID] = struct{}{}
		edges = append(edges, GraphEdgeDTO{
			ID:     edgeID,
			Source: parentNode,
			Target: childNode,
			Label:  "parent",
		})
	}

	writeJSON(w, http.StatusOK, BundleGraphResponse{Nodes: nodes, Edges: edges})
}

// @OpenAPI
// @Summary      Get profile and CAS summary
// @Description  Returns the stored profile evaluation and CAS outcome, or 204 if not yet computed.
// @Tags         viewer
// @Produce      json
// @Success      200  {object}  ProfileReportSummary
// @Success      204  "No profile report available; run POST /api/v1/bundle/verify first"
// @Failure      403  {object}  APIError
// @Failure      405  {object}  APIError
// @Router       /api/v1/bundle/profile [get]
func (s *APIServer) handleBundleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}
	if !s.requireRole(w, r, atbauth.RoleViewer) {
		return
	}
	if !s.requireVerified(w) {
		return
	}

	s.mu.Lock()
	report := s.profileReport
	s.mu.Unlock()

	if report == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// @OpenAPI
// @Summary      Run (or re-run) profile verification
// @Description  Evaluates the bundle against the configured profile and stores the result.
// @Tags         viewer
// @Produce      json
// @Success      200  {object}  ProfileReportSummary
// @Failure      403  {object}  APIError
// @Failure      405  {object}  APIError
// @Failure      422  {object}  APIError
// @Router       /api/v1/bundle/verify [post]
func (s *APIServer) handleBundleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}
	if !s.requireRole(w, r, atbauth.RoleOperator) {
		return
	}
	if !s.requireVerified(w) {
		return
	}

	// Snapshot the bundle pointer under the lock so we don't race with reveal appends.
	s.mu.Lock()
	bSnap := s.b
	s.mu.Unlock()

	cfg := verifypkg.EvaluateConfig{
		BundlePath:    s.bundlePath,
		Records:       bSnap.Records,
		AllApplicable: s.profilePath == "",
	}
	if s.profilePath != "" {
		profile, err := verifypkg.ResolveProfile(s.profilePath)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, APIError{Error: fmt.Sprintf("load profile: %v", err)})
			return
		}
		cfg.Profiles = []verifypkg.Profile{profile}
	}
	report, err := verifypkg.EvaluateBundle(cfg)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, APIError{Error: err.Error()})
		return
	}

	summary := verifyReportToSummary(*report)
	fullReport := verifypkg.ReportFromVerifyWithBundle(*report, bSnap)
	s.mu.Lock()
	s.profileReport = &summary
	s.verifierReport = &fullReport
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, summary)
}

// @OpenAPI
// @Summary      Get full verifier report
// @Description  Returns the stable VerifierReport (verify.report.v1) from the last profile evaluation.
// @Tags         viewer
// @Produce      json
// @Success      200  {object}  verifypkg.VerifierReport
// @Success      204  "No verifier report available; run POST /api/v1/bundle/verify first"
// @Failure      403  {object}  APIError
// @Failure      405  {object}  APIError
// @Router       /api/v1/bundle/verify/report [get]
func (s *APIServer) handleBundleVerifyReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}
	if !s.requireRole(w, r, atbauth.RoleViewer) {
		return
	}
	if !s.requireVerified(w) {
		return
	}

	s.mu.Lock()
	report := s.verifierReport
	s.mu.Unlock()

	if report == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// verifyReportToSummary converts an internal verify.Report to the API summary shape.
func verifyReportToSummary(r verifypkg.Report) ProfileReportSummary {
	summary := ProfileReportSummary{
		ChainValid:       r.Integrity.ChainValid,
		AnchorStatus:     r.Anchoring.Status,
		CriticalFailures: []FailureDTO{},
		Warnings:         []string{},
	}

	if r.CAS != nil {
		summary.CASScore = r.CAS.Overall
		summary.CASGrade = r.CAS.Grade
		if r.CAS.CorroborationBonus != 0 {
			summary.CorroborationBonus = r.CAS.CorroborationBonus
			summary.EffectiveScore = r.CAS.EffectiveScore
		}
		if len(r.CAS.SubScores) > 0 {
			summary.SubScores = make(map[string]float64, len(r.CAS.SubScores))
			for k, v := range r.CAS.SubScores {
				summary.SubScores[k] = v
			}
		}
	}

	if len(r.Profiles) > 0 {
		p := r.Profiles[0]
		summary.ProfileID = p.ProfileID
		summary.ProfileVersion = p.Version
		summary.Pass = r.Integrity.ChainValid && p.Pass
		for _, f := range p.CriticalFailures {
			summary.CriticalFailures = append(summary.CriticalFailures, FailureDTO{Kind: f.Kind, Detail: f.Detail})
		}
		summary.Warnings = append([]string{}, p.RequiredWarnings...)
	}

	if len(r.ProvabilityGaps) > 0 {
		summary.ProvabilityGaps = make([]ProvabilityGapDTO, len(r.ProvabilityGaps))
		for i, gap := range r.ProvabilityGaps {
			summary.ProvabilityGaps[i] = ProvabilityGapDTO{
				Gap:        gap.Gap,
				Layer:      gap.Layer,
				Mitigation: gap.Mitigation,
				ClosedWhen: gap.ClosedWhen,
			}
		}
	}

	if len(r.Exclusions) > 0 {
		summary.Exclusions = append([]string(nil), r.Exclusions...)
	}
	if r.ResidualRisk.Level != "" {
		summary.ResidualRiskLevel = r.ResidualRisk.Level
	}

	return summary
}

// @OpenAPI
// @Summary      Reveal one masked field
// @Description  Reveals a specific field path from one event and always hash-chain audit-logs the reveal.
// @Tags         viewer
// @Accept       json
// @Produce      json
// @Security     RevealCookieAuth
// @Security     RevealBearerAuth
// @Param        request  body      PrivacyRevealRequest  true  "Reveal request"
// @Success      200      {object}  PrivacyRevealResponse
// @Failure      400      {object}  APIError
// @Failure      401      {object}  APIError
// @Failure      403      {object}  APIError
// @Failure      404      {object}  APIError
// @Failure      429      {object}  APIError
// @Failure      405      {object}  APIError
// @Failure      500      {object}  APIError
// @Router       /api/v1/privacy/reveal [post]
func (s *APIServer) handlePrivacyReveal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}
	if !s.requireRole(w, r, atbauth.RoleOperator) {
		return
	}
	if !s.allowPrivacyReveal(r) {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", revealRetryAfterSeconds))
		writeJSON(w, http.StatusTooManyRequests, APIError{
			Error:      "privacy reveal rate limit exceeded",
			RetryAfter: revealRetryAfterSeconds,
		})
		return
	}
	if !s.requireRevealAuth(w, r) {
		return
	}
	if !s.requireVerified(w) {
		return
	}

	var req PrivacyRevealRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "invalid JSON body"})
		return
	}
	if req.Sequence <= 0 {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "seq must be >= 1"})
		return
	}
	if strings.TrimSpace(req.FieldPath) == "" {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "field_path is required"})
		return
	}

	record, ok := findRecordBySequence(s.b.Records, req.Sequence)
	if !ok {
		writeJSON(w, http.StatusNotFound, APIError{Error: "event sequence not found"})
		return
	}

	resolvedPath := normalizeRevealFieldPath(req.FieldPath)
	value, ok := resolveField(record.Event.Data, resolvedPath)
	if !ok {
		writeJSON(w, http.StatusNotFound, APIError{Error: "field_path not found"})
		return
	}

	if err := s.appendRevealAuditEvent(r, req, resolvedPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "audit append failed"})
		return
	}

	writeJSON(w, http.StatusOK, PrivacyRevealResponse{
		Sequence:  req.Sequence,
		FieldPath: req.FieldPath,
		Value:     value,
	})
}

func (s *APIServer) requireVerified(w http.ResponseWriter) bool {
	if s.verifyErr == nil {
		return true
	}
	writeJSON(w, http.StatusForbidden, APIError{Error: "bundle verification failed; event data is blocked"})
	return false
}

func (s *APIServer) requireRevealAuth(w http.ResponseWriter, r *http.Request) bool {
	if s.revealAuthToken == "" {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "privacy reveal auth token is not configured"})
		return false
	}

	for _, candidate := range authTokensForRequest(r) {
		if subtle.ConstantTimeCompare([]byte(candidate), []byte(s.revealAuthToken)) == 1 {
			return true
		}
	}

	writeJSON(w, http.StatusUnauthorized, APIError{Error: "unauthorized"})
	return false
}

func authTokensForRequest(r *http.Request) []string {
	candidates := make([]string, 0, 4)

	if token := strings.TrimSpace(r.Header.Get(revealAuthHeader)); token != "" {
		candidates = append(candidates, token)
	}
	if token := strings.TrimSpace(r.Header.Get(revealLegacyHeader)); token != "" {
		candidates = append(candidates, token)
	}
	if authHeader := strings.TrimSpace(r.Header.Get("Authorization")); authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			candidates = append(candidates, strings.TrimSpace(parts[1]))
		}
	}
	if cookie, err := r.Cookie(revealAuthCookie); err == nil {
		candidates = append(candidates, strings.TrimSpace(cookie.Value))
	}

	return candidates
}

func primaryRateLimitToken(r *http.Request) string {
	candidates := authTokensForRequest(r)
	if len(candidates) == 0 {
		return "anonymous"
	}
	return candidates[0]
}

func (s *APIServer) allowPrivacyReveal(r *http.Request) bool {
	ipAddr, _ := resolveRemoteIdentity(r.RemoteAddr)
	key := ipAddr + "|" + primaryRateLimitToken(r)
	now := time.Now().UTC()

	s.rateMu.Lock()
	defer s.rateMu.Unlock()

	counter := s.revealRateCounters[key]
	if counter.windowStart.IsZero() || now.Sub(counter.windowStart) >= s.revealRateWindow {
		s.revealRateCounters[key] = revealRateCounter{
			windowStart: now,
			count:       1,
		}
		return true
	}
	if counter.count >= s.revealRateLimit {
		return false
	}
	counter.count++
	s.revealRateCounters[key] = counter
	return true
}

func (s *APIServer) appendRevealAuditEvent(r *http.Request, req PrivacyRevealRequest, resolvedPath string) error {
	ipAddr, user := resolveRemoteIdentity(r.RemoteAddr)
	revealedAt := time.Now().UTC().Format(time.RFC3339)

	data := map[string]interface{}{
		"schema_version": revealSidecarSchemaV1,
		"seq":            req.Sequence,
		"field_path":     req.FieldPath,
		"revealed_at":    revealedAt,
		"user":           user,
		"ip":             ipAddr,
	}
	if resolvedPath != req.FieldPath {
		data["field_path_resolved"] = resolvedPath
	}
	if reason := strings.TrimSpace(req.Reason); reason != "" {
		data["reason"] = reason
	}

	// Bind the reveal to the authoritative bundle without mutating it, so the
	// sidecar entry proves which bundle and which chain head it annotates.
	if m := s.b.Manifest(); m != nil {
		data["source_bundle_id"] = m.BundleID
	}
	if n := len(s.b.Records); n > 0 {
		data["source_head_hash"] = s.b.Records[n-1].Hash
	}

	actorID := getActorFromContext(r)
	orgID := getOrgFromContext(r)
	workspaceID := getWorkspaceFromContext(r)
	opts := &bundle.AppendOptions{
		ActorID:     actorID,
		OrgID:       orgID,
		WorkspaceID: workspaceID,
		Timestamp:   revealedAt,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.appendRevealToSidecar(data, opts)
}

// revealSidecarPath is the path of the reveal sidecar log. Reveal events are
// recorded here, in a separate hash chain, so that inspecting a field in the
// viewer never mutates the authoritative bundle on disk.
func (s *APIServer) revealSidecarPath() string {
	return s.bundlePath + ".reveals"
}

// appendRevealToSidecar appends a privacy.reveal event to a separate sidecar
// bundle with its own genesis-rooted hash chain. The authoritative bundle is
// never written. The sidecar is itself a valid ATB bundle and verifies
// independently with the same machinery (for example, atb verify).
func (s *APIServer) appendRevealToSidecar(data map[string]interface{}, opts *bundle.AppendOptions) error {
	path := s.revealSidecarPath()

	var sidecar *bundle.Bundle
	switch _, statErr := os.Stat(path); {
	case statErr == nil:
		loaded, err := bundle.LoadVerified(path)
		if err != nil {
			return fmt.Errorf("reveal sidecar load: %w", err)
		}
		sidecar = loaded
	case errors.Is(statErr, os.ErrNotExist):
		created, err := bundle.New()
		if err != nil {
			return fmt.Errorf("reveal sidecar init: %w", err)
		}
		sidecar = created
	default:
		return fmt.Errorf("reveal sidecar stat: %w", statErr)
	}

	if err := sidecar.AppendWithOptions("privacy.reveal", data, opts); err != nil {
		return err
	}
	return sidecar.Save(path)
}

func optionalRequestID(r *http.Request, headerKeys ...string) *string {
	for _, key := range headerKeys {
		if value := strings.TrimSpace(r.Header.Get(key)); value != "" {
			v := value
			return &v
		}
	}
	return nil
}

func getActorFromContext(r *http.Request) *string {
	return optionalRequestID(r, "X-ATB-Actor-ID", "X-Actor-ID")
}

func getOrgFromContext(r *http.Request) *string {
	return optionalRequestID(r, "X-ATB-Org-ID", "X-Org-ID")
}

func getWorkspaceFromContext(r *http.Request) *string {
	return optionalRequestID(r, "X-ATB-Workspace-ID", "X-Workspace-ID")
}

func normalizeRevealFieldPath(fieldPath string) string {
	trimmed := strings.TrimSpace(fieldPath)
	return strings.TrimPrefix(trimmed, "data.")
}

func resolveRemoteIdentity(remoteAddr string) (string, string) {
	ipAddr := strings.TrimSpace(remoteAddr)
	if host, _, err := net.SplitHostPort(ipAddr); err == nil {
		ipAddr = host
	}
	user := ipAddr
	if ipAddr == "127.0.0.1" || ipAddr == "::1" || ipAddr == "localhost" {
		user = "localhost"
	}
	return ipAddr, user
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func parsePagination(offsetRaw, limitRaw string) (int, int, error) {
	offset := 0
	limit := defaultEventsLimit

	if strings.TrimSpace(offsetRaw) != "" {
		v, err := strconv.Atoi(offsetRaw)
		if err != nil || v < 0 {
			return 0, 0, fmt.Errorf("offset must be >= 0")
		}
		offset = v
	}
	if strings.TrimSpace(limitRaw) != "" {
		v, err := strconv.Atoi(limitRaw)
		if err != nil || v <= 0 {
			return 0, 0, fmt.Errorf("limit must be > 0")
		}
		limit = v
	}
	if limit > maxEventsLimit {
		limit = maxEventsLimit
	}
	return offset, limit, nil
}

func graphNodeType(eventType string) string {
	switch {
	case strings.HasPrefix(eventType, "ai.llm"):
		return "llm"
	case strings.HasPrefix(eventType, "ai.tool"):
		return "tool"
	case strings.HasPrefix(eventType, "ai.chain"):
		return "chain"
	default:
		return "event"
	}
}

func findRecordBySequence(records []bundle.Record, seq int) (bundle.Record, bool) {
	for _, record := range records {
		if record.Event.Sequence == seq {
			return record, true
		}
	}
	return bundle.Record{}, false
}

func extractSpanContext(data interface{}) (traceID string, spanID string, parentSpanID string) {
	m, ok := data.(map[string]interface{})
	if !ok {
		return "", "", ""
	}
	return stringAtKey(m, "trace_id"), stringAtKey(m, "span_id"), stringAtKey(m, "parent_span_id")
}

func extractTimestamp(data interface{}) string {
	m, ok := data.(map[string]interface{})
	if !ok {
		return ""
	}
	for _, key := range []string{"timestamp", "ts", "created_at", "started_at"} {
		if value := stringAtKey(m, key); value != "" {
			return value
		}
	}
	if timing, ok := m["timing"].(map[string]interface{}); ok {
		if value := stringAtKey(timing, "started_at"); value != "" {
			return value
		}
	}
	return ""
}

func stringAtKey(m map[string]interface{}, key string) string {
	raw, ok := m[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func maskSensitive(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			if isSensitiveFieldKey(key) {
				out[key] = "[REDACTED]"
				continue
			}
			out[key] = maskSensitive(item)
		}
		return out
	case []interface{}:
		out := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			out = append(out, maskSensitive(item))
		}
		return out
	default:
		return typed
	}
}

type piiFieldRulesFile struct {
	AllowFields         []string `json:"allow_fields"`
	SensitiveFields     []string `json:"sensitive_fields"`
	SensitiveSubstrings []string `json:"sensitive_substrings"`
	SensitiveSuffixes   []string `json:"sensitive_suffixes"`
}

type piiMaskRules struct {
	allowFields         map[string]struct{}
	sensitiveFields     map[string]struct{}
	sensitiveSubstrings []string
	sensitiveSuffixes   []string
}

var (
	piiMaskRulesOnce sync.Once
	piiRules         piiMaskRules
)

func activePIIMaskRules() piiMaskRules {
	piiMaskRulesOnce.Do(func() {
		piiRules = loadPIIMaskRules()
	})
	return piiRules
}

func loadPIIMaskRules() piiMaskRules {
	defaults := defaultPIIMaskRules()

	raw, err := loadPIIRulesFile()
	if err != nil {
		return defaults
	}

	var rulesFile piiFieldRulesFile
	if err := json.Unmarshal(raw, &rulesFile); err != nil {
		return defaults
	}

	rules := piiMaskRules{
		allowFields:         normalizeFieldSet(rulesFile.AllowFields),
		sensitiveFields:     normalizeFieldSet(rulesFile.SensitiveFields),
		sensitiveSubstrings: normalizeFieldList(rulesFile.SensitiveSubstrings),
		sensitiveSuffixes:   normalizeFieldList(rulesFile.SensitiveSuffixes),
	}
	if len(rules.sensitiveFields) == 0 {
		return defaults
	}
	return rules
}

func loadPIIRulesFile() ([]byte, error) {
	if path := strings.TrimSpace(os.Getenv(piiFieldsPathEnv)); path != "" {
		return os.ReadFile(filepath.Clean(path))
	}

	path, err := findPIIFieldsPath()
	if err == nil {
		return os.ReadFile(filepath.Clean(path))
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return atbembed.ReadEmbeddedPIIFields()
}

func findPIIFieldsPath() (string, error) {
	if path := strings.TrimSpace(os.Getenv(piiFieldsPathEnv)); path != "" {
		return path, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := wd
	for {
		candidate := filepath.Join(dir, piiFieldsRelativePath)
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	return "", os.ErrNotExist
}

func defaultPIIMaskRules() piiMaskRules {
	return piiMaskRules{
		allowFields: normalizeFieldSet([]string{
			"trace_id",
			"span_id",
			"parent_span_id",
			"run_id",
			"event_id",
			"hash",
			"prev_hash",
			"timestamp",
			"ts",
			"created_at",
			"updated_at",
		}),
		sensitiveFields: normalizeFieldSet([]string{
			"email",
			"ip",
			"user_id",
			"actor_id",
			"org_id",
			"workspace_id",
			"id",
		}),
		sensitiveSubstrings: normalizeFieldList([]string{"payment", "health", "bio"}),
		sensitiveSuffixes:   normalizeFieldList([]string{"_id"}),
	}
}

func normalizeFieldSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := normalizeFieldKey(value)
		if normalized == "" {
			continue
		}
		out[normalized] = struct{}{}
	}
	return out
}

func normalizeFieldList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := normalizeFieldKey(value)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func normalizeFieldKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isSensitiveFieldKey(key string) bool {
	lower := normalizeFieldKey(key)
	if lower == "" {
		return false
	}

	rules := activePIIMaskRules()
	if _, ok := rules.allowFields[lower]; ok {
		return false
	}
	if _, ok := rules.sensitiveFields[lower]; ok {
		return true
	}

	for _, token := range rules.sensitiveSubstrings {
		if strings.Contains(lower, token) {
			return true
		}
	}

	for _, suffix := range rules.sensitiveSuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func resolveField(root interface{}, fieldPath string) (interface{}, bool) {
	parts := strings.Split(strings.TrimSpace(fieldPath), ".")
	if len(parts) == 0 {
		return nil, false
	}
	current := root
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return nil, false
		}

		switch typed := current.(type) {
		case map[string]interface{}:
			next, ok := typed[part]
			if !ok {
				return nil, false
			}
			current = next
		case []interface{}:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(typed) {
				return nil, false
			}
			current = typed[idx]
		default:
			return nil, false
		}
	}
	return current, true
}
