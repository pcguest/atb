// SPDX-License-Identifier: MIT
package proxy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// ToolCall is a tool/function invocation the model requested in a response.
// Arguments holds the raw request payload; it is digested, never stored raw.
type ToolCall struct {
	Name      string
	Arguments []byte
}

// InputDigest returns the SHA-256 hex of the tool arguments, or "" when empty.
func (t ToolCall) InputDigest() string {
	if len(t.Arguments) == 0 {
		return ""
	}
	sum := sha256.Sum256(t.Arguments)
	return hex.EncodeToString(sum[:])
}

// ExtractToolCalls parses the tool/function calls a model requested in a
// response body. It recognises Anthropic Messages (tool_use content blocks),
// OpenAI Chat Completions (choices[].message.tool_calls), OpenAI Responses
// (output[] function_call items), and the streamed (SSE) forms of the Chat
// Completions and Anthropic Messages APIs. Unrecognised bodies yield no calls.
func ExtractToolCalls(body []byte) []ToolCall {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		// Not a single JSON object; it may be a Server-Sent Events stream.
		if bytes.Contains(body, []byte("data:")) {
			return extractStreamingToolCalls(body)
		}
		return nil
	}

	var calls []ToolCall

	// Anthropic Messages: top-level content blocks of type "tool_use".
	for _, block := range asSlice(payload["content"]) {
		m, ok := block.(map[string]any)
		if !ok || str(m["type"]) != "tool_use" {
			continue
		}
		calls = append(calls, ToolCall{
			Name:      str(m["name"]),
			Arguments: marshalArgs(m["input"]),
		})
	}

	// OpenAI Responses: output items of type "function_call".
	for _, item := range asSlice(payload["output"]) {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if t := str(m["type"]); t != "function_call" && t != "tool_call" {
			continue
		}
		calls = append(calls, ToolCall{
			Name:      str(m["name"]),
			Arguments: marshalArgs(m["arguments"]),
		})
	}

	// OpenAI Chat Completions: tool_calls per choice message.
	for _, choice := range asSlice(payload["choices"]) {
		cm, ok := choice.(map[string]any)
		if !ok {
			continue
		}
		message, ok := cm["message"].(map[string]any)
		if !ok {
			continue
		}
		for _, tc := range asSlice(message["tool_calls"]) {
			tcm, ok := tc.(map[string]any)
			if !ok {
				continue
			}
			fn, ok := tcm["function"].(map[string]any)
			if !ok {
				continue
			}
			calls = append(calls, ToolCall{
				Name:      str(fn["name"]),
				Arguments: marshalArgs(fn["arguments"]),
			})
		}
	}

	return calls
}

// extractStreamingToolCalls reassembles tool calls from a Server-Sent Events
// stream — OpenAI Chat Completions (choices[].delta.tool_calls deltas) and
// Anthropic Messages (content_block_start tool_use + input_json_delta). Tool
// names and accumulated argument fragments are recovered by index; partial
// streams still yield the tool name (the full body is digested separately).
func extractStreamingToolCalls(body []byte) []ToolCall {
	type acc struct {
		name string
		args []byte
	}
	byIndex := map[int]*acc{}
	order := []int{}
	at := func(i int) *acc {
		a, ok := byIndex[i]
		if !ok {
			a = &acc{}
			byIndex[i] = a
			order = append(order, i)
		}
		return a
	}

	for _, line := range bytes.Split(body, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		payload := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if len(payload) == 0 || string(payload) == "[DONE]" {
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal(payload, &ev); err != nil {
			continue
		}

		// Anthropic Messages streaming.
		switch str(ev["type"]) {
		case "content_block_start":
			if cb, ok := ev["content_block"].(map[string]any); ok && str(cb["type"]) == "tool_use" {
				at(intOf(ev["index"])).name = str(cb["name"])
			}
			continue
		case "content_block_delta":
			if d, ok := ev["delta"].(map[string]any); ok && str(d["type"]) == "input_json_delta" {
				a := at(intOf(ev["index"]))
				a.args = append(a.args, []byte(str(d["partial_json"]))...)
			}
			continue
		}

		// OpenAI Chat Completions streaming.
		for _, choice := range asSlice(ev["choices"]) {
			cm, ok := choice.(map[string]any)
			if !ok {
				continue
			}
			delta, ok := cm["delta"].(map[string]any)
			if !ok {
				continue
			}
			for _, tc := range asSlice(delta["tool_calls"]) {
				tcm, ok := tc.(map[string]any)
				if !ok {
					continue
				}
				a := at(intOf(tcm["index"]))
				if fn, ok := tcm["function"].(map[string]any); ok {
					if n := str(fn["name"]); n != "" {
						a.name = n
					}
					a.args = append(a.args, []byte(str(fn["arguments"]))...)
				}
			}
		}
	}

	var calls []ToolCall
	for _, i := range order {
		a := byIndex[i]
		if a.name == "" {
			continue
		}
		calls = append(calls, ToolCall{Name: a.name, Arguments: a.args})
	}
	return calls
}

func intOf(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i)
		}
	}
	return 0
}

// ToolResultError is a tool result the client reported as failed, carried in a
// subsequent request body. It maps to an ai.action.error accountability event.
type ToolResultError struct {
	ToolUseID string
	Detail    []byte
}

// DetailDigest returns the SHA-256 hex of the error detail, or "" when empty.
func (e ToolResultError) DetailDigest() string {
	if len(e.Detail) == 0 {
		return ""
	}
	sum := sha256.Sum256(e.Detail)
	return hex.EncodeToString(sum[:])
}

// ExtractToolResultErrors parses failed tool results from a request body.
// It currently recognises Anthropic tool_result content blocks with
// is_error=true; OpenAI tool messages carry no standard error flag, so they are
// not classified as failures here.
func ExtractToolResultErrors(body []byte) []ToolResultError {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}

	var errs []ToolResultError
	for _, msg := range asSlice(payload["messages"]) {
		mm, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		for _, block := range asSlice(mm["content"]) {
			bm, ok := block.(map[string]any)
			if !ok || str(bm["type"]) != "tool_result" {
				continue
			}
			if isErr, _ := bm["is_error"].(bool); !isErr {
				continue
			}
			id := strings.TrimSpace(str(bm["tool_use_id"]))
			if id == "" {
				continue // cannot satisfy ai.action.error required action_id
			}
			errs = append(errs, ToolResultError{
				ToolUseID: id,
				Detail:    marshalArgs(bm["content"]),
			})
		}
	}
	return errs
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func str(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

// marshalArgs renders tool arguments/content to bytes for digesting. String
// payloads (OpenAI arguments) are used as-is; structured payloads (Anthropic
// input) are JSON-encoded. Returns nil when empty.
func marshalArgs(v any) []byte {
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		if t == "" {
			return nil
		}
		return []byte(t)
	default:
		raw, err := json.Marshal(t)
		if err != nil || len(raw) == 0 || string(raw) == "null" {
			return nil
		}
		return raw
	}
}
