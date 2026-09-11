// SPDX-License-Identifier: MIT
package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/canonicalize"
	"github.com/pcguest/atb/internal/event"
)

const (
	maxOperationPayloadBytes = 1 << 20 // 1 MiB before canonical hashing
	maxOperationTextBytes    = 4 << 10 // 4 KiB for retained or hashed metadata
)

// OperationInput contains observable metadata for one MCP request/response.
// Sensitive authorization, tracestate, and baggage values are accepted only
// so they can be committed by digest; they are never copied into event data.
type OperationInput struct {
	ProtocolVersion      string
	Method               string
	Name                 string
	Status               string
	Request              any
	Result               any
	OperationError       any
	ClientMetadata       any
	ClientCapabilities   any
	AuthorizationContext any
	Traceparent          string
	Tracestate           string
	Baggage              string
	TTLMS                *int
	CacheScope           string
	TaskHandle           string
	TaskState            string
	MRTRState            string
	Timestamp            time.Time
}

// BuildOperationEvent creates bounded MCP evidence without retaining request
// bodies, results, credentials, or private routing metadata.
func BuildOperationEvent(input OperationInput) (*event.Event, error) {
	if strings.TrimSpace(input.ProtocolVersion) == "" || strings.TrimSpace(input.Method) == "" {
		return nil, fmt.Errorf("mcp evidence: protocol version and method are required")
	}
	status := strings.TrimSpace(input.Status)
	switch status {
	case "success", "error", "cancelled", "pending":
	default:
		return nil, fmt.Errorf("mcp evidence: unsupported status %q", status)
	}
	if err := validateOperationText(input); err != nil {
		return nil, err
	}
	requestDigest, err := canonicalDigest(input.Request)
	if err != nil {
		return nil, fmt.Errorf("mcp evidence: request digest: %w", err)
	}
	data := map[string]any{
		"operation_id":     requestDigest,
		"protocol_version": strings.TrimSpace(input.ProtocolVersion),
		"method":           strings.TrimSpace(input.Method),
		"status":           status,
		"request_digest":   requestDigest,
	}
	copyText(data, "name", input.Name)
	if input.Result != nil {
		if data["result_digest"], err = canonicalDigest(input.Result); err != nil {
			return nil, fmt.Errorf("mcp evidence: result digest: %w", err)
		}
	}
	if input.OperationError != nil {
		if data["error_digest"], err = canonicalDigest(input.OperationError); err != nil {
			return nil, fmt.Errorf("mcp evidence: error digest: %w", err)
		}
	}
	for key, value := range map[string]any{
		"client_metadata_digest":       input.ClientMetadata,
		"capabilities_digest":          input.ClientCapabilities,
		"authorization_context_digest": input.AuthorizationContext,
	} {
		if value == nil {
			continue
		}
		if data[key], err = canonicalDigest(value); err != nil {
			return nil, fmt.Errorf("mcp evidence: %s: %w", key, err)
		}
	}
	if input.TTLMS != nil {
		if *input.TTLMS < 0 {
			return nil, fmt.Errorf("mcp evidence: ttl_ms must not be negative")
		}
		data["ttl_ms"] = *input.TTLMS
	}
	copyText(data, "cache_scope", input.CacheScope)
	copyText(data, "task_handle", input.TaskHandle)
	copyText(data, "task_state", input.TaskState)
	copyText(data, "mrtr_state", input.MRTRState)
	copyText(data, "traceparent", input.Traceparent)
	if input.Tracestate != "" {
		data["tracestate_digest"] = stringDigest(input.Tracestate)
	}
	if input.Baggage != "" {
		data["baggage_digest"] = stringDigest(input.Baggage)
	}

	timestamp := input.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	traceID, spanID := traceIDs(input.Traceparent)
	if strings.TrimSpace(input.Traceparent) != "" && traceID == "" {
		return nil, fmt.Errorf("mcp evidence: invalid traceparent")
	}
	return &event.Event{
		Type:      event.TypeMcpOperation,
		HashAlgo:  "sha256",
		Timestamp: timestamp.UTC().Format(time.RFC3339Nano),
		TraceID:   traceID,
		SpanID:    spanID,
		Data:      data,
	}, nil
}

func canonicalDigest(value any) (string, error) {
	canonical, err := canonicalize.Marshal(value)
	if err != nil {
		return "", err
	}
	if len(canonical) > maxOperationPayloadBytes {
		return "", fmt.Errorf("value exceeds %d-byte evidence limit", maxOperationPayloadBytes)
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func validateOperationText(input OperationInput) error {
	for name, value := range map[string]string{
		"protocol_version": input.ProtocolVersion,
		"method":           input.Method,
		"name":             input.Name,
		"traceparent":      input.Traceparent,
		"tracestate":       input.Tracestate,
		"baggage":          input.Baggage,
		"cache_scope":      input.CacheScope,
		"task_handle":      input.TaskHandle,
		"task_state":       input.TaskState,
		"mrtr_state":       input.MRTRState,
	} {
		if len(value) > maxOperationTextBytes {
			return fmt.Errorf("mcp evidence: %s exceeds %d-byte evidence limit", name, maxOperationTextBytes)
		}
	}
	return nil
}

func stringDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func copyText(data map[string]any, key, value string) {
	if value = strings.TrimSpace(value); value != "" {
		data[key] = value
	}
}

func traceIDs(traceparent string) (string, string) {
	parts := strings.Split(strings.TrimSpace(traceparent), "-")
	if len(parts) != 4 || len(parts[0]) != 2 || len(parts[1]) != 32 || len(parts[2]) != 16 || len(parts[3]) != 2 {
		return "", ""
	}
	if _, err := hex.DecodeString(parts[0] + parts[1] + parts[2] + parts[3]); err != nil || parts[0] == "ff" || allZeroHex(parts[1]) || allZeroHex(parts[2]) {
		return "", ""
	}
	return strings.ToLower(parts[1]), strings.ToLower(parts[2])
}

func allZeroHex(value string) bool {
	return strings.Trim(value, "0") == ""
}
