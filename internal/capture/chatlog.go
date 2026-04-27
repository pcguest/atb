// SPDX-License-Identifier: MIT
package capture

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/canonicalize"
	"github.com/pcguest/atb/internal/event"
)

const (
	FormatGenericJSONL  = "generic-jsonl"
	FormatClaudeDesktop = "claude-desktop"
	FormatOpenAIJSONL   = "openai-jsonl"

	defaultPurposeTag   = "chatlog_import"
	defaultOutputFormat = "text/plain"
	suggestedProfileRAG = "atb.profile.rag_answer"
)

// ErrUnsupportedProvider indicates the requested chatlog format is not
// implemented or recognised. Treat as a user-facing input error.
var ErrUnsupportedProvider = errors.New("capture: unsupported provider")

// ErrMalformedChatlog indicates the chatlog content is structurally invalid
// (bad JSON, missing required fields, mapping prerequisites unmet). Treat as
// a user-facing input error.
var ErrMalformedChatlog = errors.New("capture: malformed chatlog")

// ErrUnknownTurn indicates a turn whose role cannot be mapped to any ATB
// event type. Callers may wrap with index context and decide whether to
// abort or skip; the importer skips these turns and logs to stderr.
var ErrUnknownTurn = errors.New("capture: unrecognised turn role")

// errUnsupportedChatlogFormat is the legacy alias retained for callers within
// the package that used to wrap with this sentinel; new code should match
// against ErrUnsupportedProvider.
//
//lint:ignore U1000 retained alias for backward compatibility with pre-Prompt-19 callers
var errUnsupportedChatlogFormat = ErrUnsupportedProvider

// ChatMessage is the normalised representation of one imported chatlog line.
type ChatMessage struct {
	Line                  int
	Role                  string
	Content               string
	Timestamp             string
	Model                 string
	ModelProvider         string
	ModelParametersDigest string
	PromptDigest          string
	OutputDigest          string
	OutputFormat          string
	ToolName              string
	ToolCallID            string
	ToolArgs              json.RawMessage
	ConversationID        string
	SessionID             string
	RequestID             string
	ActorIDHash           string
	PurposeTag            string
}

// EventSpec is one canonical ATB event ready to append into a bundle.
type EventSpec struct {
	Type      string
	Data      map[string]any
	Timestamp string
}

// MappingResult captures the mapped ATB events and import metadata.
type MappingResult struct {
	Events             []EventSpec
	SkippedRecords     int
	SuggestedProfileID string
}

// AppendSummary describes one append operation into a bundle.
type AppendSummary struct {
	BundlePath    string
	EventsWritten int
	LastRecord    bundle.Record
}

type rawChatMessage struct {
	Role                  string          `json:"role"`
	Content               string          `json:"content"`
	Timestamp             string          `json:"timestamp"`
	Model                 string          `json:"model,omitempty"`
	ModelProvider         string          `json:"model_provider,omitempty"`
	ModelParametersDigest string          `json:"model_parameters_digest,omitempty"`
	PromptDigest          string          `json:"prompt_digest,omitempty"`
	OutputDigest          string          `json:"output_digest,omitempty"`
	OutputFormat          string          `json:"output_format,omitempty"`
	ToolName              string          `json:"tool_name,omitempty"`
	ToolCallID            string          `json:"tool_call_id,omitempty"`
	ToolArgs              json.RawMessage `json:"tool_args,omitempty"`
	ConversationID        string          `json:"conversation_id,omitempty"`
	SessionID             string          `json:"session_id,omitempty"`
	RequestID             string          `json:"request_id,omitempty"`
	ActorIDHash           string          `json:"actor_id_hash,omitempty"`
	PurposeTag            string          `json:"purpose_tag,omitempty"`
}

// ParseChatlog dispatches to the requested chatlog parser. Format-level
// failures wrap ErrUnsupportedProvider; content-level failures wrap
// ErrMalformedChatlog.
func ParseChatlog(format string, r io.Reader) ([]ChatMessage, error) {
	switch strings.TrimSpace(strings.ToLower(format)) {
	case FormatGenericJSONL:
		messages, err := ParseGenericJSONL(r)
		if err != nil {
			if errors.Is(err, ErrMalformedChatlog) {
				return nil, err
			}
			return nil, fmt.Errorf("%w: %v", ErrMalformedChatlog, err)
		}
		return messages, nil
	case FormatClaudeDesktop, FormatOpenAIJSONL:
		return nil, fmt.Errorf("%w %q: Capture v1 only implements %s", ErrUnsupportedProvider, format, FormatGenericJSONL)
	default:
		return nil, fmt.Errorf("%w %q", ErrUnsupportedProvider, format)
	}
}

// ParseGenericJSONL parses the Capture v1 generic JSONL schema.
func ParseGenericJSONL(r io.Reader) ([]ChatMessage, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), bundle.MaxLineSizeBytes)

	messages := make([]ChatMessage, 0)
	line := 0
	for scanner.Scan() {
		line++
		rawLine := strings.TrimSpace(scanner.Text())
		if rawLine == "" {
			continue
		}

		rawBytes := []byte(rawLine)
		if bytes.TrimSpace(rawBytes)[0] == '[' {
			var entries []json.RawMessage
			if err := json.Unmarshal(rawBytes, &entries); err != nil {
				return nil, fmt.Errorf("parse generic JSONL line %d: %w", line, err)
			}
			for _, entry := range entries {
				message, err := parseRawChatMessage(line, entry)
				if err != nil {
					return nil, err
				}
				messages = append(messages, message)
			}
			continue
		}

		message, err := parseRawChatMessage(line, rawBytes)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse generic JSONL: %w", err)
	}
	return messages, nil
}

func parseRawChatMessage(line int, raw []byte) (ChatMessage, error) {
	var input rawChatMessage
	if err := json.Unmarshal(raw, &input); err != nil {
		return ChatMessage{}, fmt.Errorf("parse generic JSONL line %d: %w", line, err)
	}

	return normaliseRawChatMessage(line, input)
}

func normaliseRawChatMessage(line int, input rawChatMessage) (ChatMessage, error) {
	message := ChatMessage{
		Line:                  line,
		Role:                  strings.TrimSpace(strings.ToLower(input.Role)),
		Content:               strings.TrimSpace(input.Content),
		Timestamp:             strings.TrimSpace(input.Timestamp),
		Model:                 strings.TrimSpace(input.Model),
		ModelProvider:         strings.TrimSpace(input.ModelProvider),
		ModelParametersDigest: strings.TrimSpace(input.ModelParametersDigest),
		PromptDigest:          strings.TrimSpace(input.PromptDigest),
		OutputDigest:          strings.TrimSpace(input.OutputDigest),
		OutputFormat:          strings.TrimSpace(input.OutputFormat),
		ToolName:              strings.TrimSpace(input.ToolName),
		ToolCallID:            strings.TrimSpace(input.ToolCallID),
		ToolArgs:              input.ToolArgs,
		ConversationID:        strings.TrimSpace(input.ConversationID),
		SessionID:             strings.TrimSpace(input.SessionID),
		RequestID:             strings.TrimSpace(input.RequestID),
		ActorIDHash:           strings.TrimSpace(input.ActorIDHash),
		PurposeTag:            strings.TrimSpace(input.PurposeTag),
	}

	switch message.Role {
	case "user", "assistant", "system":
	case "tool":
	default:
		// Unknown or absent roles are not rejected at parse time; the mapper
		// surfaces them via ErrUnknownTurn so import policy can skip them.
		return message, nil
	}

	if message.Content == "" {
		return ChatMessage{}, fmt.Errorf("parse generic JSONL line %d: missing content", line)
	}
	if message.Timestamp == "" {
		return ChatMessage{}, fmt.Errorf("parse generic JSONL line %d: missing timestamp", line)
	}
	if _, err := time.Parse(time.RFC3339, message.Timestamp); err != nil {
		return ChatMessage{}, fmt.Errorf("parse generic JSONL line %d: invalid timestamp %q: %w", line, message.Timestamp, err)
	}

	if message.Role == "tool" && message.ToolName == "" {
		return ChatMessage{}, fmt.Errorf("parse generic JSONL line %d: tool records require tool_name", line)
	}

	return message, nil
}

// MapMessagesToEvents converts parsed chat messages into canonical ATB events.
// Mapping failures wrap ErrMalformedChatlog so CLI callers can classify them
// as user-facing input errors.
func MapMessagesToEvents(messages []ChatMessage) (MappingResult, error) {
	result, err := mapMessagesToEventsInner(messages)
	if err != nil {
		return MappingResult{}, fmt.Errorf("%w: %w", ErrMalformedChatlog, err)
	}
	return result, nil
}

// FilterKnownTurns returns messages with recognised roles and the zero-based
// turn indices that were excluded. It lets importers choose skip policy while
// keeping MapMessagesToEvents strict for unmappable turns.
func FilterKnownTurns(messages []ChatMessage) ([]ChatMessage, []int) {
	filtered := make([]ChatMessage, 0, len(messages))
	unknown := make([]int, 0)
	for index, message := range messages {
		if isKnownTurnRole(message.Role) {
			filtered = append(filtered, message)
			continue
		}
		unknown = append(unknown, index)
	}
	return filtered, unknown
}

func isKnownTurnRole(role string) bool {
	switch role {
	case "system", "user", "tool", "assistant":
		return true
	default:
		return false
	}
}

func mapMessagesToEventsInner(messages []ChatMessage) (MappingResult, error) {
	result := MappingResult{
		Events: make([]EventSpec, 0, len(messages)*2),
	}
	if len(messages) == 0 {
		return result, nil
	}

	promptWindow := make([]string, 0, len(messages))
	requestCounter := 0
	currentRequestID := ""
	globalNamespace := chatNamespace(messages)
	modelTurnSeen := false

	for index, message := range messages {
		switch message.Role {
		case "system":
			// system turns inform context only; not recorded as events.
			promptWindow = append(promptWindow, promptWindowEntry(message))
			result.SkippedRecords++
		case "user":
			requestCounter++
			requestID := message.RequestID
			if requestID == "" {
				requestID = generatedRequestID(globalNamespace, requestCounter)
			}
			currentRequestID = requestID

			data := map[string]any{
				"request_id":      requestID,
				"actor_id_hash":   fallbackActorIDHash(message, globalNamespace, requestCounter),
				"purpose_tag":     firstNonEmpty(message.PurposeTag, defaultPurposeTag),
				"input_digest":    digestText(message.Content),
				"input_format":    defaultOutputFormat,
				"source_role":     message.Role,
				"source_line_num": message.Line,
			}
			addChatContext(data, message)
			result.Events = append(result.Events, EventSpec{Type: event.TypeAIRequestReceived, Data: data, Timestamp: message.Timestamp})
			promptWindow = append(promptWindow, promptWindowEntry(message))
		case "tool":
			if currentRequestID == "" {
				return MappingResult{}, fmt.Errorf("map chatlog line %d: tool record arrived before a user turn", message.Line)
			}
			toolArgsValue, toolArgsText, err := decodeToolArgs(message.ToolArgs)
			if err != nil {
				return MappingResult{}, fmt.Errorf("map chatlog line %d: decode tool_args: %w", message.Line, err)
			}
			data := map[string]any{
				"request_id":         currentRequestID,
				"tool_name":          message.ToolName,
				"tool_output":        message.Content,
				"tool_output_digest": digestText(message.Content),
				"source_role":        message.Role,
				"source_line_num":    message.Line,
			}
			if message.ToolCallID != "" {
				data["tool_call_id"] = message.ToolCallID
			}
			if toolArgsText != "" {
				data["tool_args_digest"] = digestText(toolArgsText)
			}
			if toolArgsValue != nil {
				data["tool_args"] = toolArgsValue
			}
			addChatContext(data, message)
			result.Events = append(result.Events, EventSpec{Type: event.TypeAIToolExec, Data: data, Timestamp: message.Timestamp})
			promptWindow = append(promptWindow, promptWindowEntry(message))
		case "assistant":
			if currentRequestID == "" {
				return MappingResult{}, fmt.Errorf("map chatlog line %d: assistant record arrived before a user turn", message.Line)
			}

			outputDigest := firstNonEmpty(message.OutputDigest, digestText(message.Content))
			outputFormat := firstNonEmpty(message.OutputFormat, defaultOutputFormat)

			if message.Model != "" {
				data := map[string]any{
					"request_id":              currentRequestID,
					"model_provider":          firstNonEmpty(message.ModelProvider, guessModelProvider(message.Model)),
					"model_id":                message.Model,
					"model_parameters_digest": firstNonEmpty(message.ModelParametersDigest, digestCanonicalValue(map[string]any{})),
					"prompt_digest":           firstNonEmpty(message.PromptDigest, digestText(strings.Join(promptWindow, "\n"))),
					"source_role":             message.Role,
					"source_line_num":         message.Line,
				}
				addChatContext(data, message)
				result.Events = append(result.Events, EventSpec{Type: event.TypeAIModelInvoked, Data: data, Timestamp: message.Timestamp})

				outputData := map[string]any{
					"request_id":      currentRequestID,
					"output_digest":   outputDigest,
					"output_format":   outputFormat,
					"source_role":     message.Role,
					"source_line_num": message.Line,
				}
				addChatContext(outputData, message)
				result.Events = append(result.Events, EventSpec{Type: event.TypeAIModelOutput, Data: outputData, Timestamp: message.Timestamp})
				modelTurnSeen = true
			}

			responseData := map[string]any{
				"request_id":      currentRequestID,
				"output_digest":   outputDigest,
				"output_format":   outputFormat,
				"source_role":     message.Role,
				"source_line_num": message.Line,
			}
			addChatContext(responseData, message)
			result.Events = append(result.Events, EventSpec{Type: event.TypeAIResponseSent, Data: responseData, Timestamp: message.Timestamp})
			promptWindow = append(promptWindow, promptWindowEntry(message))
		default:
			return MappingResult{}, fmt.Errorf("capture: turn %d: %w", index, ErrUnknownTurn)
		}
	}

	if modelTurnSeen {
		result.SuggestedProfileID = suggestedProfileRAG
	}
	return result, nil
}

// AppendEventsToBundleInMemory appends events to an already-loaded *bundle.Bundle
// without loading or saving. It returns the number of events appended and the
// first error encountered. On error, no events are guaranteed to have been
// appended (it stops at the first failure).
func AppendEventsToBundleInMemory(b *bundle.Bundle, events []EventSpec) (int, error) {
	for i, spec := range events {
		if err := b.AppendWithOptions(spec.Type, spec.Data, &bundle.AppendOptions{Timestamp: spec.Timestamp}); err != nil {
			return i, err
		}
	}
	return len(events), nil
}

// AppendEventsToBundle appends the mapped events into the selected bundle path.
func AppendEventsToBundle(path string, events []EventSpec) (AppendSummary, error) {
	if len(events) == 0 {
		return AppendSummary{}, fmt.Errorf("append events: no events supplied")
	}

	b, err := loadOrCreateBundle(path)
	if err != nil {
		return AppendSummary{}, err
	}
	for _, spec := range events {
		if err := b.AppendWithOptions(spec.Type, spec.Data, &bundle.AppendOptions{Timestamp: spec.Timestamp}); err != nil {
			return AppendSummary{}, fmt.Errorf("append events: %w", err)
		}
	}
	if err := b.Save(path); err != nil {
		return AppendSummary{}, fmt.Errorf("append events: save: %w", err)
	}
	return AppendSummary{
		BundlePath:    path,
		EventsWritten: len(events),
		LastRecord:    b.Records[len(b.Records)-1],
	}, nil
}

func loadOrCreateBundle(path string) (*bundle.Bundle, error) {
	b, err := bundle.Load(path)
	if err == nil {
		return b, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return bundle.New()
}

func fallbackActorIDHash(message ChatMessage, namespace string, requestCounter int) string {
	if message.ActorIDHash != "" {
		return message.ActorIDHash
	}
	seed := fmt.Sprintf("%s:%d:%s", namespace, requestCounter, message.Role)
	return digestText(seed)
}

func addChatContext(data map[string]any, message ChatMessage) {
	if message.ConversationID != "" {
		data["conversation_id"] = message.ConversationID
	}
	if message.SessionID != "" {
		data["session_id"] = message.SessionID
	}
}

func chatNamespace(messages []ChatMessage) string {
	for _, message := range messages {
		if message.ConversationID != "" {
			return message.ConversationID
		}
		if message.SessionID != "" {
			return message.SessionID
		}
	}
	return "chatlog"
}

func promptWindowEntry(message ChatMessage) string {
	if message.Role == "tool" {
		return fmt.Sprintf("tool[%s]: %s", message.ToolName, message.Content)
	}
	return fmt.Sprintf("%s: %s", message.Role, message.Content)
}

func generatedRequestID(namespace string, requestCounter int) string {
	return fmt.Sprintf("req-%s-%03d", shortDigest(namespace), requestCounter)
}

func shortDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:4])
}

func digestText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func digestCanonicalValue(value any) string {
	canonical, err := canonicalize.Marshal(value)
	if err != nil {
		return digestText(fmt.Sprintf("%v", value))
	}
	return digestText(string(canonical))
}

func decodeToolArgs(raw json.RawMessage) (any, string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, "", nil
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, "", err
	}
	if value == nil {
		return nil, "", nil
	}
	if text, ok := value.(string); ok {
		return text, text, nil
	}

	canonical, err := canonicalize.Marshal(value)
	if err != nil {
		return nil, "", err
	}
	return value, string(canonical), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func guessModelProvider(model string) string {
	trimmed := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasPrefix(trimmed, "gpt-"), strings.HasPrefix(trimmed, "o1"), strings.HasPrefix(trimmed, "o3"), strings.HasPrefix(trimmed, "o4"):
		return "openai"
	case strings.HasPrefix(trimmed, "claude-"):
		return "anthropic"
	case strings.HasPrefix(trimmed, "gemini"), strings.HasPrefix(trimmed, "models/gemini"):
		return "google"
	default:
		return "unknown"
	}
}
