package apiv1

import (
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

const (
	defaultEventsLimit = 200
	maxEventsLimit     = 1000
)

// APIConfig configures the local viewer JSON API.
type APIConfig struct {
	BundlePath string
	Bundle     *bundle.Bundle
	VerifyErr  error
	LogReveals bool
}

// APIServer serves dashboard data from a verified (or failed-verification) bundle snapshot.
type APIServer struct {
	bundlePath      string
	b               *bundle.Bundle
	verifyErr       error
	logReveals      bool
	revealAuditPath string
	mu              sync.Mutex
}

// NewAPIServer constructs a local-only viewer API server.
func NewAPIServer(cfg APIConfig) *APIServer {
	return &APIServer{
		bundlePath:      cfg.BundlePath,
		b:               cfg.Bundle,
		verifyErr:       cfg.VerifyErr,
		logReveals:      cfg.LogReveals,
		revealAuditPath: filepath.Join(filepath.Dir(cfg.BundlePath), "viewer-audit.log"),
	}
}

// Register mounts dashboard API endpoints on the provided mux.
func (s *APIServer) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/verification", s.handleVerification)
	mux.HandleFunc("/api/v1/bundle/meta", s.handleBundleMeta)
	mux.HandleFunc("/api/v1/bundle/events", s.handleBundleEvents)
	mux.HandleFunc("/api/v1/bundle/graph", s.handleBundleGraph)
	mux.HandleFunc("/api/v1/privacy/reveal", s.handlePrivacyReveal)
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
	if !s.requireVerified(w) {
		return
	}

	typeCounts := make(map[string]int)
	firstTS := ""
	lastTS := ""
	for _, record := range s.b.Records {
		typeCounts[record.Event.Type]++
		ts := extractTimestamp(record.Event.Data)
		if ts == "" {
			continue
		}
		if firstTS == "" {
			firstTS = ts
		}
		lastTS = ts
	}

	out := BundleMetaResponse{
		BundlePath:      s.bundlePath,
		EventCount:      len(s.b.Records),
		TypeCounts:      typeCounts,
		FirstTimestamp:  firstTS,
		LastTimestamp:   lastTS,
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
	if !s.requireVerified(w) {
		return
	}

	offset, limit, err := parsePagination(r.URL.Query().Get("offset"), r.URL.Query().Get("limit"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		return
	}

	total := len(s.b.Records)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	events := make([]EventRecordDTO, 0, end-offset)
	for _, record := range s.b.Records[offset:end] {
		traceID, spanID, parentSpanID := extractSpanContext(record.Event.Data)
		events = append(events, EventRecordDTO{
			Sequence:     record.Event.Sequence,
			Type:         record.Event.Type,
			Hash:         record.Hash,
			PrevHash:     record.Event.PrevHash,
			Timestamp:    extractTimestamp(record.Event.Data),
			TraceID:      traceID,
			SpanID:       spanID,
			ParentSpanID: parentSpanID,
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
		if record.Event.Sequence > 1 {
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
// @Summary      Reveal one masked field
// @Description  Reveals a specific field path from one event and optionally audit-logs the reveal.
// @Tags         viewer
// @Accept       json
// @Produce      json
// @Param        request  body      PrivacyRevealRequest  true  "Reveal request"
// @Success      200      {object}  PrivacyRevealResponse
// @Failure      400      {object}  APIError
// @Failure      403      {object}  APIError
// @Failure      404      {object}  APIError
// @Failure      405      {object}  APIError
// @Failure      500      {object}  APIError
// @Router       /api/v1/privacy/reveal [post]
func (s *APIServer) handlePrivacyReveal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
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
	value, ok := resolveField(record.Event.Data, req.FieldPath)
	if !ok {
		writeJSON(w, http.StatusNotFound, APIError{Error: "field_path not found"})
		return
	}

	if err := s.logReveal(r.RemoteAddr, req); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: fmt.Sprintf("log reveal event: %v", err)})
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

func (s *APIServer) logReveal(remoteAddr string, req PrivacyRevealRequest) error {
	if !s.logReveals {
		return nil
	}

	ipAddr := remoteAddr
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		ipAddr = host
	}
	user := ipAddr
	if ipAddr == "127.0.0.1" || ipAddr == "::1" || ipAddr == "localhost" {
		user = "localhost"
	}

	entry := map[string]interface{}{
		"event_type":  "ui.privacy.reveal",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"bundle_path": s.bundlePath,
		"event_seq":   req.Sequence,
		"field_path":  req.FieldPath,
		"reason":      req.Reason,
		"user":        user,
		"ip":          ipAddr,
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.revealAuditPath), 0750); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Clean(s.revealAuditPath), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(raw, '\n'))
	return err
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

func isSensitiveFieldKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	switch lower {
	case "", "trace_id", "span_id", "parent_span_id", "run_id", "event_id", "hash", "prev_hash", "timestamp", "ts", "created_at", "updated_at":
		return false
	case "email", "ip", "user_id", "actor_id", "org_id", "workspace_id", "id":
		return true
	}
	if strings.Contains(lower, "payment") || strings.Contains(lower, "health") || strings.Contains(lower, "bio") {
		return true
	}
	if strings.HasSuffix(lower, "_id") {
		return true
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
