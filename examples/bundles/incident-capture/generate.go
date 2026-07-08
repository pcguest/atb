// Command generate_incident_capture writes incident-capture.atb in this
// directory: a deterministic, capture-shaped ATB bundle modelling an agent
// incident as `atb intercept` would record it — an LLM exchange in which the
// model invoked a privileged tool with no human approval and the tool failed.
//
// It exercises the capture/accountability event types end to end:
// atb.llm.request, atb.llm.response, atb.tool.call, ai.action.error,
// atb.exchange.complete, atb.session.close.
//
// Usage: go run ./examples/bundles/incident-capture/
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

const (
	sessionID = "sess-incident-7731"
	host      = "api.anthropic.com"
	model     = "claude-3-5-sonnet"
	actorID   = "agent-support-bot"
	toolUseID = "toolu_delete_1"
)

func digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	outDir := filepath.Join(root, "examples", "bundles", "incident-capture")
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		fatal(err)
	}

	b, err := buildIncidentBundle()
	if err != nil {
		fatal(err)
	}

	bundlePath := filepath.Join(outDir, "incident-capture.atb")
	if err := b.Save(bundlePath); err != nil {
		fatal(err)
	}
	fmt.Printf("✓ wrote %s (%d records)\n", bundlePath, len(b.Records))
}

// buildIncidentBundle constructs the deterministic incident bundle. It is
// shared by main and by generate_test.go so the forensic behaviour (integrity
// plus the tool_without_approval oversight signal) is covered by `go test`
// without depending on the gitignored .atb artefact.
func buildIncidentBundle() (*bundle.Bundle, error) {
	b, err := bundle.New()
	if err != nil {
		return nil, err
	}

	base := time.Date(2026, 5, 28, 9, 0, 0, 0, time.UTC)
	step := func(i int) string { return base.Add(time.Duration(i) * time.Second).Format(time.RFC3339) }
	add := func(eventType string, data map[string]any, ts string) {
		if err != nil {
			return
		}
		err = b.AppendWithOptions(eventType, data, &bundle.AppendOptions{Timestamp: ts})
	}

	reqBody := `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"purge the test accounts"}]}`
	respBody := `{"content":[{"type":"tool_use","id":"toolu_delete_1","name":"delete_user_records","input":{"scope":"all"}}]}`
	toolErrBody := `{"messages":[{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_delete_1","is_error":true,"content":"refused: production guard"}]}]}`

	// Turn 1: model asks for a privileged tool, no approval recorded.
	add(event.TypeLLMRequest, map[string]any{
		"session_id": sessionID, "host": host, "method": "POST", "path": "/v1/messages",
		"provider": "anthropic", "model": model,
		"body_sha256": digest(reqBody), "body_bytes": len(reqBody),
		"actor": map[string]any{"display_name": actorID},
	}, step(0))
	add(event.TypeLLMResponse, map[string]any{
		"session_id": sessionID, "host": host, "method": "POST", "path": "/v1/messages",
		"status_code": 200, "provider": "anthropic", "model": model,
		"body_sha256": digest(respBody), "body_bytes": len(respBody),
		"usage": map[string]int{"prompt_tokens": 24, "completion_tokens": 18, "total_tokens": 42},
	}, step(1))
	add(event.TypeToolCall, map[string]any{
		"session_id": sessionID, "tool_name": "delete_user_records",
		"tool_input_digest": digest(`{"scope":"all"}`), "actor_id": actorID,
	}, step(2))

	// Turn 2: the client reports the tool failed.
	add(event.TypeLLMRequest, map[string]any{
		"session_id": sessionID, "host": host, "method": "POST", "path": "/v1/messages",
		"provider": "anthropic", "model": model,
		"body_sha256": digest(toolErrBody), "body_bytes": len(toolErrBody),
		"actor": map[string]any{"display_name": actorID},
	}, step(3))
	add(event.TypeAIActionError, map[string]any{
		"session_id": sessionID, "action_id": toolUseID, "error_class": "failed",
		"error_detail_digest": digest("refused: production guard"),
	}, step(4))
	add(event.TypeHumanOverride, map[string]any{
		"session_id": sessionID, "override_reason": "operator confirmed production guard should remain active",
		"actor_id": "reviewer-42", "overridden_action_id": toolUseID,
		"identity_evidence": map[string]any{
			"identity_provider": "https://login.example.test",
			"subject":           "reviewer-42",
			"auth_context":      "mfa",
			"assertion_type":    "jwt",
			"assertion_digest":  "sha256:" + digest("fixture-reviewer-assertion"),
		},
	}, step(5))

	add(event.TypeExchangeComplete, map[string]any{
		"session_id": sessionID, "exchange_id": "1", "request_event_id": digest("req-1"),
		"actor_id": actorID, "completed_at": step(6), "tool_calls_count": 1,
		"model": model, "input_tokens": 24, "output_tokens": 18,
	}, step(6))
	add(event.TypeSessionClose, map[string]any{
		"session_id": sessionID, "actor_id": actorID, "model": model,
		"exchange_count": 1, "total_tokens": 42, "closed_at": step(7),
	}, step(7))

	if err != nil {
		return nil, err
	}
	return b, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
