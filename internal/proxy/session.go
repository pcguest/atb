// SPDX-License-Identifier: MIT
package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// TypeSessionClose is emitted when a capture session ends.
	TypeSessionClose = "atb.session.close"
	// TypeExchangeComplete is emitted after each proxied request/response pair.
	TypeExchangeComplete = "atb.exchange.complete"
	// defaultSessionIdleTimeout closes idle sessions after 30 seconds.
	defaultSessionIdleTimeout = 30 * time.Second
)

// Session tracks one conversational capture thread inside the proxy.
type Session struct {
	ID                   string
	ThreadKey            string
	Model                string
	ExchangeCount        int
	TotalTokens          int
	ActorID              string
	ExchangeCounter      int
	lastRequestEventHash string
	exchangeStartedAt    time.Time
	lastActivity         time.Time
	idleTimer            *time.Timer
	// Fixed: closed prevents duplicate close events from timer and shutdown races.
	closed bool
	mu     sync.Mutex
}

// UpdateActorID updates the session's actor ID if the provided ID is not empty.
// It is thread-safe.
func (s *Session) UpdateActorID(actorID string) {
	if actorID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ActorID = actorID
}

// NextExchangeID increments the exchange counter and returns a zero-padded 4-digit string.
// It is thread-safe.
func (s *Session) NextExchangeID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ExchangeCounter++
	return fmt.Sprintf("%04d", s.ExchangeCounter)
}

// noteExchangeStarted records when the current exchange request was captured.
func (s *Session) noteExchangeStarted(t time.Time) {
	if s == nil || t.IsZero() {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exchangeStartedAt = t.UTC()
}

// setLastRequestEventHash stores the bundle record hash of the latest request event.
func (s *Session) setLastRequestEventHash(hash string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastRequestEventHash = hash
}

func (s *Session) lastRequestEventHashLocked() string {
	return s.lastRequestEventHash
}

// SessionManager owns session lifecycle and idle expiry.
type SessionManager struct {
	idleTimeout time.Duration
	onClose     func(*Session) error // Updated to accept func(*Session) error

	mu       sync.Mutex
	sessions map[string]*Session
}

// NewSessionManager returns a session manager with the default idle timeout.
func NewSessionManager(onClose func(*Session) error) *SessionManager {
	return &SessionManager{
		idleTimeout: defaultSessionIdleTimeout,
		onClose:     onClose,
		sessions:    map[string]*Session{},
	}
}

// Resolve returns an existing session or creates one keyed by threadKey.
func (m *SessionManager) Resolve(threadKey string) *Session {
	if m == nil {
		return &Session{ID: newSessionID()}
	}
	m.mu.Lock()
	if threadKey == "" {
		threadKey = newSessionID()
	}
	if sess, ok := m.sessions[threadKey]; ok {
		m.mu.Unlock()
		sess.touch(m.idleTimeout, m.closeSession)
		return sess
	}
	sess := &Session{
		ID:           newSessionID(),
		ThreadKey:    threadKey,
		lastActivity: time.Now().UTC(),
	}
	m.sessions[threadKey] = sess
	m.mu.Unlock()
	sess.touch(m.idleTimeout, m.closeSession)
	return sess
}

// NoteExchange records model and token usage for the session summary.
func (m *SessionManager) NoteExchange(sess *Session, model string, promptTokens, outputTokens int) {
	if m == nil || sess == nil {
		return
	}
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if model != "" {
		sess.Model = model
	}
	sess.ExchangeCount++
	sess.TotalTokens += promptTokens + outputTokens
	sess.touchLocked(m.idleTimeout, m.closeSession)
}

func (s *Session) touch(idle time.Duration, closer func(*Session)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.touchLocked(idle, closer)
}

func (s *Session) touchLocked(idle time.Duration, closer func(*Session)) {
	// Fixed: Closed sessions must not be revived by late response accounting.
	if s.closed {
		return
	}
	s.lastActivity = time.Now().UTC()
	if s.idleTimer != nil {
		// Fixed: Replacing the idle timer uses the Stop-and-drain pattern for timer safety.
		stopTimer(s.idleTimer)
	}
	s.idleTimer = time.AfterFunc(idle, func() {
		closer(s)
	})
}

func (m *SessionManager) closeSession(sess *Session) {
	if m == nil || sess == nil {
		return
	}
	// Fixed: markClosed guarantees only one close callback per session.
	if !sess.markClosed() {
		return
	}
	m.mu.Lock()
	delete(m.sessions, sess.ThreadKey)
	m.mu.Unlock()
	if m.onClose != nil {
		_ = m.onClose(sess)
	}
}

// CloseAll flushes every active session, for example during proxy shutdown.
func (m *SessionManager) CloseAll() {
	if m == nil {
		return
	}
	m.mu.Lock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, sess := range m.sessions {
		sessions = append(sessions, sess)
	}
	m.sessions = map[string]*Session{}
	m.mu.Unlock()
	for _, sess := range sessions {
		// Fixed: markClosed stops the timer and suppresses duplicate shutdown callbacks.
		if !sess.markClosed() {
			continue
		}
		if m.onClose != nil {
			_ = m.onClose(sess)
		}
	}
}

// Fixed: markClosed atomically closes a session and stops its idle timer.
func (s *Session) markClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	s.closed = true
	stopTimer(s.idleTimer)
	s.idleTimer = nil
	return true
}

// Fixed: stopTimer follows the Stop-and-drain pattern to avoid late timer delivery.
func stopTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

// SessionCloseRecord builds the payload for an atb.session.close event.
func SessionCloseRecord(sess *Session) map[string]any {
	if sess == nil {
		return map[string]any{}
	}
	sess.mu.Lock()
	defer sess.mu.Unlock()
	// actor_id is a required field on atb.session.close per
	// docs/spec-ai-traces.md, so it is always emitted. When the proxy could
	// not resolve an actor for the session the value is the empty string,
	// which the session index surfaces as an unresolved identity rather than
	// silently dropping the field.
	rec := map[string]any{
		"session_id":     sess.ID,
		"actor_id":       sess.ActorID,
		"model":          sess.Model,
		"exchange_count": sess.ExchangeCount,
		"total_tokens":   sess.TotalTokens,
		"closed_at":      time.Now().UTC().Format(time.RFC3339Nano),
	}
	return rec
}

// ExchangeCompleteRecord builds the payload for an atb.exchange.complete event.
// exchangeID must come from NextExchangeID on the same session. requestEventID is
// the hash of the appended request record in the bundle.
func ExchangeCompleteRecord(
	sess *Session,
	exchangeID string,
	requestEventID string,
	model string,
	promptTokens int,
	outputTokens int,
	toolCallsCount int,
	completedAt time.Time,
) map[string]any {
	if sess == nil {
		return map[string]any{}
	}
	sess.mu.Lock()
	defer sess.mu.Unlock()

	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	} else {
		completedAt = completedAt.UTC()
	}

	latencyMs := 0
	if started := sess.exchangeStartedAt; !started.IsZero() {
		latencyMs = int(completedAt.Sub(started).Milliseconds())
		if latencyMs < 0 {
			latencyMs = 0
		}
	}

	rec := map[string]any{
		"session_id":       sess.ID,
		"exchange_id":      exchangeID,
		"request_event_id": requestEventID,
		"actor_id":         sess.ActorID,
		"completed_at":     completedAt.Format(time.RFC3339Nano),
		"tool_calls_count": toolCallsCount,
	}
	if model != "" {
		rec["model"] = model
	}
	if promptTokens > 0 {
		rec["input_tokens"] = promptTokens
	}
	if outputTokens > 0 {
		rec["output_tokens"] = outputTokens
	}
	if latencyMs > 0 {
		rec["latency_ms"] = latencyMs
	}
	return rec
}

// ExtractThreadKey derives a stable conversation key from provider headers or body.
func ExtractThreadKey(host string, req *http.Request, body []byte) string {
	if req == nil {
		return ""
	}
	if sessionID := strings.TrimSpace(req.Header.Get("Anthropic-Session-Id")); sessionID != "" {
		return "anthropic:" + sessionID
	}
	if sessionID := strings.TrimSpace(req.Header.Get("X-Session-Id")); sessionID != "" {
		return "session:" + sessionID
	}
	if threadID := extractJSONField(body, "thread_id"); threadID != "" {
		return "openai-thread:" + threadID
	}
	path := req.URL.Path
	if idx := strings.Index(path, "/threads/"); idx >= 0 {
		rest := strings.TrimPrefix(path[idx+len("/threads/"):], "/")
		if threadID, _, _ := strings.Cut(rest, "/"); threadID != "" {
			return "openai-thread:" + threadID
		}
	}
	host = strings.ToLower(stripPort(host))
	return "conn:" + host + ":" + req.RemoteAddr
}

func extractJSONField(body []byte, field string) string {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var payload map[string]any
	if err := dec.Decode(&payload); err != nil {
		return ""
	}
	value, ok := payload[field]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

// ReadRequestBody reads and restores an HTTP request body for capture and forwarding.
// It applies DefaultMaxBodyBytes. Prefer ReadRequestBodyLimited with the proxy
// configuration when a custom cap is in effect.
func ReadRequestBody(req *http.Request) ([]byte, error) {
	return ReadRequestBodyLimited(req, DefaultMaxBodyBytes)
}

// ReadRequestBodyLimited reads at most max bytes from the request body, restores
// the body for forwarding on success, and returns ErrBodyTooLarge when the body
// exceeds max. A non-positive max falls back to DefaultMaxBodyBytes.
func ReadRequestBodyLimited(req *http.Request, max int64) ([]byte, error) {
	if req == nil || req.Body == nil {
		return nil, nil
	}
	if max <= 0 {
		max = DefaultMaxBodyBytes
	}
	if req.ContentLength > max {
		_ = req.Body.Close()
		req.Body = http.NoBody
		return nil, &BodyTooLargeError{Limit: max, Observed: req.ContentLength}
	}
	body, err := readLimited(req.Body, max)
	if err != nil {
		_ = req.Body.Close()
		req.Body = http.NoBody
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

// ReadBodyLimited reads at most max bytes from r. It does not close r.
// A non-positive max falls back to DefaultMaxBodyBytes.
func ReadBodyLimited(r io.Reader, max int64) ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	if max <= 0 {
		max = DefaultMaxBodyBytes
	}
	return readLimited(r, max)
}

func readLimited(r io.Reader, max int64) ([]byte, error) {
	if max <= 0 || max > MaxBodyBytesLimit {
		return nil, fmt.Errorf("%w: invalid read limit %d", ErrInvalidConfig, max)
	}
	limited := io.LimitReader(r, max+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > max {
		return nil, &BodyTooLargeError{Limit: max, Observed: int64(len(body))}
	}
	return body, nil
}

// DrainAndClose drains a response body and closes the session on EOF when requested.
func DrainAndClose(ctx context.Context, body io.ReadCloser, onEOF func() error) error {
	if body == nil {
		return nil
	}
	defer body.Close()
	_, err := io.Copy(io.Discard, body)
	if err == io.EOF || err == nil {
		if onEOF != nil {
			return onEOF()
		}
	}
	return err
}

// ParseModelFromBody returns a model field when present in JSON request bodies.
func ParseModelFromBody(body []byte) string {
	return extractJSONField(body, "model")
}

// ParseUsageFromResponse extracts token usage from provider JSON responses.
func ParseUsageFromResponse(body []byte) (promptTokens, outputTokens, totalTokens int) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var payload map[string]any
	if err := dec.Decode(&payload); err != nil {
		return 0, 0, 0
	}
	usage, ok := payload["usage"].(map[string]any)
	if !ok {
		return 0, 0, 0
	}
	promptTokens = intFromAny(usage["prompt_tokens"], usage["input_tokens"])
	outputTokens = intFromAny(usage["completion_tokens"], usage["output_tokens"])
	totalTokens = intFromAny(usage["total_tokens"])
	if totalTokens == 0 {
		totalTokens = promptTokens + outputTokens
	}
	return promptTokens, outputTokens, totalTokens
}

// CountToolCallsFromResponse counts tool/function invocations in a provider
// response body. It recognises the Anthropic Messages format (top-level
// "content" blocks of type "tool_use"), the OpenAI Chat Completions format
// (choices[].message.tool_calls), and the OpenAI Responses format (top-level
// "output" items of type "function_call"/"tool_call"). An unparseable or
// unrecognised body yields 0, so the count is best-effort and never errors.
func CountToolCallsFromResponse(body []byte) int {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var payload map[string]any
	if err := dec.Decode(&payload); err != nil {
		return 0
	}

	count := 0
	// Anthropic Messages: typed content blocks.
	count += countTypedBlocks(payload["content"], "tool_use")
	// OpenAI Responses: typed output items.
	count += countTypedBlocks(payload["output"], "function_call", "tool_call")
	// OpenAI Chat Completions: tool_calls array per choice message.
	if choices, ok := payload["choices"].([]any); ok {
		for _, choice := range choices {
			choiceMap, ok := choice.(map[string]any)
			if !ok {
				continue
			}
			message, ok := choiceMap["message"].(map[string]any)
			if !ok {
				continue
			}
			if toolCalls, ok := message["tool_calls"].([]any); ok {
				count += len(toolCalls)
			}
		}
	}
	return count
}

// countTypedBlocks counts items in a JSON array whose "type" field matches any
// of the supplied type names.
func countTypedBlocks(value any, types ...string) int {
	blocks, ok := value.([]any)
	if !ok {
		return 0
	}
	count := 0
	for _, block := range blocks {
		blockMap, ok := block.(map[string]any)
		if !ok {
			continue
		}
		blockType, _ := blockMap["type"].(string)
		for _, want := range types {
			if blockType == want {
				count++
				break
			}
		}
	}
	return count
}

func intFromAny(values ...any) int {
	for _, value := range values {
		switch typed := value.(type) {
		case json.Number:
			i, err := typed.Int64()
			if err == nil {
				return int(i)
			}
		case float64:
			return int(typed)
		case int:
			return typed
		}
	}
	return 0
}

// recordedHeaders is a conservative evidence allowlist. Provider APIs invent
// new credential header names regularly, so a denylist can silently persist a
// secret into a tamper-evident bundle record.
var recordedHeaders = map[string]struct{}{
	"accept":               {},
	"accept-encoding":      {},
	"content-encoding":     {},
	"content-type":         {},
	"openai-request-id":    {},
	"anthropic-request-id": {},
	"request-id":           {},
	"retry-after":          {},
	"traceparent":          {},
	"user-agent":           {},
	"x-request-id":         {},
}

// recordedHeaderNames returns the sorted allowlist for the capture-scope
// attestation.
func recordedHeaderNames() []string {
	names := make([]string, 0, len(recordedHeaders))
	for name := range recordedHeaders {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func ScanHeaders(header http.Header) map[string]string {
	out := map[string]string{}
	for key, values := range header {
		if _, recorded := recordedHeaders[strings.ToLower(key)]; !recorded {
			continue
		}
		if len(values) > 0 {
			out[key] = values[0]
		}
	}
	return out
}

// ExtractAPIKey returns a bearer or provider API key from request headers.
func ExtractAPIKey(header http.Header) string {
	if header == nil {
		return ""
	}
	if key := strings.TrimSpace(header.Get("X-Api-Key")); key != "" {
		return key
	}
	auth := strings.TrimSpace(header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

// ReadHTTPRequest reads an HTTP request from a buffered connection.
func ReadHTTPRequest(reader *bufio.Reader) (*http.Request, error) {
	return http.ReadRequest(reader)
}

// IsTargetHost reports whether host matches configured capture targets.
func IsTargetHost(host string, targets []string) bool {
	host = strings.ToLower(stripPort(host))
	for _, target := range targets {
		target = strings.ToLower(strings.TrimSpace(target))
		if host == target || strings.HasSuffix(host, "."+target) {
			return true
		}
	}
	return false
}

// ProviderFromHost maps upstream hostnames to shorthand provider names.
func ProviderFromHost(host string) string {
	host = strings.ToLower(stripPort(host))
	switch {
	case strings.Contains(host, "openai"):
		return "openai"
	case strings.Contains(host, "anthropic"):
		return "anthropic"
	default:
		return host
	}
}

func newSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString(b[:8])
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmtUUID(b)
}

func fmtUUID(b [16]byte) string {
	hexBytes := hex.EncodeToString(b[:])
	return hexBytes[0:8] + "-" + hexBytes[8:12] + "-" + hexBytes[12:16] + "-" + hexBytes[16:20] + "-" + hexBytes[20:32]
}
