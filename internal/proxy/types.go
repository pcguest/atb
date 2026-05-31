// SPDX-License-Identifier: MIT
package proxy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/identity"
)

// recordBody writes privacy-safe body evidence into a captured event payload.
// By default only a SHA-256 digest and byte length are stored so an always-on
// recorder never persists raw prompts, completions, or PII. The raw body is
// included only when captureRaw is set (ProxyConfig.CaptureBodies).
func recordBody(data map[string]any, body []byte, captureRaw bool) {
	if len(body) == 0 {
		return
	}
	sum := sha256.Sum256(body)
	data["body_sha256"] = hex.EncodeToString(sum[:])
	data["body_bytes"] = len(body)
	if captureRaw {
		data["body"] = string(body)
	}
}

const (
	// TypeLLMRequest is the ATB event type for a captured upstream LLM request.
	// Aliased to the schema-generated constant so the proxy and the canonical
	// event registry (schemas/event.v1.json) cannot drift.
	TypeLLMRequest = event.TypeLLMRequest
	// TypeLLMResponse is the ATB event type for a captured upstream LLM response.
	TypeLLMResponse = event.TypeLLMResponse
)

// RequestRecord holds normalised metadata for a captured LLM API request.
type RequestRecord struct {
	SessionID   string
	Host        string
	Method      string
	Path        string
	Provider    string
	Model       string
	APIKey      string
	Headers     map[string]string
	Body        []byte
	RecordedAt  time.Time
	DisplayName string
	Email       string
	// CaptureBody retains the raw body in the recorded event; default false
	// records only a digest. Set from ProxyConfig.CaptureBodies.
	CaptureBody bool
}

// ResponseRecord holds normalised metadata for a captured LLM API response.
type ResponseRecord struct {
	SessionID    string
	Host         string
	Method       string
	Path         string
	Provider     string
	Model        string
	APIKey       string
	StatusCode   int
	Headers      map[string]string
	Body         []byte
	RecordedAt   time.Time
	DisplayName  string
	Email        string
	PromptTokens int
	OutputTokens int
	TotalTokens  int
	// CaptureBody retains the raw body in the recorded event; default false
	// records only a digest. Set from ProxyConfig.CaptureBodies.
	CaptureBody bool
}

// ToEvent maps the request record to a canonical ATB event envelope.
func (r RequestRecord) ToEvent() (*event.Event, error) {
	if strings.TrimSpace(r.SessionID) == "" {
		return nil, fmt.Errorf("proxy: request record missing session_id")
	}
	if strings.TrimSpace(r.Host) == "" {
		return nil, fmt.Errorf("proxy: request record missing host")
	}
	recordedAt := r.RecordedAt
	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
	}

	data := map[string]any{
		"session_id": r.SessionID,
		"host":       r.Host,
		"method":     r.Method,
		"path":       r.Path,
	}
	if r.Provider != "" {
		data["provider"] = r.Provider
	}
	if r.Model != "" {
		data["model"] = r.Model
	}
	recordBody(data, r.Body, r.CaptureBody)
	if len(r.Headers) > 0 {
		data["headers"] = r.Headers
	}
	identity.ApplyActor(data, identity.Identity{
		DisplayName: r.DisplayName,
		Email:       r.Email,
	}, r.APIKey)

	ev := &event.Event{
		Type:      TypeLLMRequest,
		Timestamp: recordedAt.UTC().Format(time.RFC3339Nano),
		Data:      data,
	}
	return ev, nil
}

// ToEvent maps the response record to a canonical ATB event envelope.
func (r ResponseRecord) ToEvent() (*event.Event, error) {
	if strings.TrimSpace(r.SessionID) == "" {
		return nil, fmt.Errorf("proxy: response record missing session_id")
	}
	if strings.TrimSpace(r.Host) == "" {
		return nil, fmt.Errorf("proxy: response record missing host")
	}
	recordedAt := r.RecordedAt
	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
	}

	data := map[string]any{
		"session_id":  r.SessionID,
		"host":        r.Host,
		"method":      r.Method,
		"path":        r.Path,
		"status_code": r.StatusCode,
	}
	if r.Provider != "" {
		data["provider"] = r.Provider
	}
	if r.Model != "" {
		data["model"] = r.Model
	}
	recordBody(data, r.Body, r.CaptureBody)
	if len(r.Headers) > 0 {
		data["headers"] = r.Headers
	}
	if r.PromptTokens > 0 || r.OutputTokens > 0 || r.TotalTokens > 0 {
		data["usage"] = map[string]int{
			"prompt_tokens":     r.PromptTokens,
			"completion_tokens": r.OutputTokens,
			"total_tokens":      r.TotalTokens,
		}
	}
	identity.ApplyActor(data, identity.Identity{
		DisplayName: r.DisplayName,
		Email:       r.Email,
	}, "")

	ev := &event.Event{
		Type:      TypeLLMResponse,
		Timestamp: recordedAt.UTC().Format(time.RFC3339Nano),
		Data:      data,
	}
	return ev, nil
}
