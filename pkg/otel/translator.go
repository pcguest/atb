// SPDX-License-Identifier: MIT
package otel

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/event"
)

// ErrNotImplemented indicates an optional transport or mapping path is absent.
var ErrNotImplemented = errors.New("otel: transport not implemented")

// ErrUnmappableSpan indicates a span is missing data required by the ATB AI
// trace envelope.
var ErrUnmappableSpan = errors.New("otel: span cannot be mapped")

// MappingError carries the field that made a span unmappable.
type MappingError struct {
	Field string
}

func (e *MappingError) Error() string {
	if e.Field == "" {
		return ErrUnmappableSpan.Error()
	}
	return fmt.Sprintf("otel: span cannot be mapped: missing %s", e.Field)
}

func (e *MappingError) Unwrap() error {
	return ErrUnmappableSpan
}

// Translator maps inbound OpenTelemetry spans into canonical ATB events.
type Translator interface {
	Translate(span OTelSpan) (*event.Event, error)
}

// DefaultTranslator maps recognised OpenTelemetry AI spans into the canonical
// ATB AI trace envelope.
type DefaultTranslator struct {
	// DefaultEventType is used when no span attribute selects an ATB event type.
	DefaultEventType string
}

var _ Translator = (*DefaultTranslator)(nil)

// NewDefaultTranslator returns the default translator implementation.
func NewDefaultTranslator() *DefaultTranslator {
	return &DefaultTranslator{}
}

// Translate maps span to an ATB event using DefaultTranslator.
func Translate(span OTelSpan) (*event.Event, error) {
	return DefaultTranslator{}.Translate(span)
}

// Translate implements Translator.
func (t DefaultTranslator) Translate(span OTelSpan) (*event.Event, error) {
	if span.TraceID == "" {
		return nil, &MappingError{Field: "trace_id"}
	}
	if span.SpanID == "" {
		return nil, &MappingError{Field: "span_id"}
	}
	if span.StartTime.IsZero() {
		return nil, &MappingError{Field: "start_time"}
	}

	eventType, err := t.eventType(span)
	if err != nil {
		return nil, err
	}
	phase := firstString(span.Attributes, "atb.phase", "ai.phase", "phase")
	if phase == "" {
		phase = inferPhase(span)
	}

	data := map[string]any{
		"trace_id":          span.TraceID,
		"span_id":           span.SpanID,
		"framework":         firstStringDefault("otel", span.Attributes, "atb.framework", "framework"),
		"framework_version": firstString(span.Attributes, "atb.framework_version", "framework_version", "otel.library.version"),
		"phase":             phase,
		"context":           contextForEvent(eventType, span),
		"privacy":           privacyForSpan(span),
		"timing":            timingForSpan(span),
		"status":            statusForSpan(span),
	}
	if span.ParentSpanID != "" {
		data["parent_span_id"] = span.ParentSpanID
	}
	if runID := firstString(span.Attributes, "atb.run_id", "run_id"); runID != "" {
		data["run_id"] = runID
	}

	return &event.Event{
		Type:         eventType,
		HashAlgo:     "sha256",
		Data:         data,
		Timestamp:    span.StartTime.UTC().Format(time.RFC3339),
		TraceID:      span.TraceID,
		SpanID:       span.SpanID,
		ParentSpanID: span.ParentSpanID,
	}, nil
}

func (t DefaultTranslator) eventType(span OTelSpan) (string, error) {
	if attr := firstString(span.Attributes, "atb.event_type", "atb.event.type", "ai.event_type", "ai.event.type"); attr != "" {
		return attr, nil
	}
	if t.DefaultEventType != "" {
		return t.DefaultEventType, nil
	}

	name := strings.ToLower(span.Name)
	switch {
	case strings.Contains(name, "llm"), strings.Contains(name, "chat"), strings.Contains(name, "completion"):
		return event.TypeAILLMCall, nil
	case strings.Contains(name, "tool"):
		return event.TypeAIToolExec, nil
	case strings.Contains(name, "chain"), strings.Contains(name, "step"):
		return event.TypeAIChainRun, nil
	default:
		return "", &MappingError{Field: "event_type"}
	}
}

func contextForEvent(eventType string, span OTelSpan) map[string]any {
	switch eventType {
	case event.TypeAILLMCall:
		return llmContext(span)
	case event.TypeAIToolExec:
		return toolContext(span)
	case event.TypeAIChainRun:
		return chainContext(span)
	default:
		return map[string]any{
			"span_name": span.Name,
			"span_kind": span.Kind,
		}
	}
}

func llmContext(span OTelSpan) map[string]any {
	ctx := map[string]any{}
	if provider := firstString(span.Attributes, "gen_ai.system", "llm.provider", "ai.provider"); provider != "" {
		ctx["provider"] = provider
	}
	if model := firstString(span.Attributes, "gen_ai.request.model", "gen_ai.response.model", "llm.model", "ai.model"); model != "" {
		ctx["model"] = model
	}
	addTextDigest(ctx, "prompt", span.Attributes, "gen_ai.prompt", "prompt.text", "gen_ai.prompt.sha256", "prompt.sha256")
	addTextDigest(ctx, "completion", span.Attributes, "gen_ai.completion", "completion.text", "gen_ai.completion.sha256", "completion.sha256")
	addTokenUsage(ctx, span.Attributes)
	if temperature, ok := firstFloat(span.Attributes, "gen_ai.request.temperature", "temperature"); ok {
		ctx["temperature"] = temperature
	}
	if maxTokens, ok := firstInt(span.Attributes, "gen_ai.request.max_tokens", "max_tokens"); ok {
		ctx["max_tokens"] = maxTokens
	}
	if finishReason := firstString(span.Attributes, "gen_ai.response.finish_reason", "finish_reason"); finishReason != "" {
		ctx["finish_reason"] = finishReason
	}
	return ctx
}

func toolContext(span OTelSpan) map[string]any {
	ctx := map[string]any{}
	if name := firstString(span.Attributes, "tool.name", "ai.tool.name", "tool_name"); name != "" {
		ctx["tool_name"] = name
	}
	if version := firstString(span.Attributes, "tool.version", "tool_version"); version != "" {
		ctx["tool_version"] = version
	}
	addTextDigest(ctx, "input", span.Attributes, "tool.input", "input.text", "tool.input.sha256", "input.sha256")
	addTextDigest(ctx, "output", span.Attributes, "tool.output", "output.text", "tool.output.sha256", "output.sha256")
	return ctx
}

func chainContext(span OTelSpan) map[string]any {
	ctx := map[string]any{}
	if name := firstString(span.Attributes, "chain.name", "ai.chain.name", "chain_name"); name != "" {
		ctx["chain_name"] = name
	}
	if inputKeys := firstStringSlice(span.Attributes, "chain.input_keys", "input_keys"); len(inputKeys) > 0 {
		ctx["input_keys"] = inputKeys
	}
	if outputKeys := firstStringSlice(span.Attributes, "chain.output_keys", "output_keys"); len(outputKeys) > 0 {
		ctx["output_keys"] = outputKeys
	}
	if stepCount, ok := firstInt(span.Attributes, "chain.step_count", "step_count"); ok {
		ctx["step_count"] = stepCount
	}
	return ctx
}

func privacyForSpan(span OTelSpan) map[string]any {
	mode := firstStringDefault("off", span.Attributes, "privacy.mode", "atb.privacy.mode")
	redactionEnabled, ok := firstBool(span.Attributes, "privacy.redaction_enabled", "atb.privacy.redaction_enabled")
	if !ok {
		redactionEnabled = mode != "off"
	}
	return map[string]any{
		"mode":              mode,
		"redaction_enabled": redactionEnabled,
		"pii_ruleset":       firstStringDefault("phase4-gdpr-v1", span.Attributes, "privacy.pii_ruleset", "atb.privacy.pii_ruleset"),
	}
}

func timingForSpan(span OTelSpan) map[string]any {
	timing := map[string]any{
		"started_at": span.StartTime.UTC().Format(time.RFC3339),
		"ended_at":   nil,
		"latency_ms": nil,
	}
	if !span.EndTime.IsZero() {
		timing["ended_at"] = span.EndTime.UTC().Format(time.RFC3339)
		timing["latency_ms"] = int64(span.EndTime.Sub(span.StartTime) / time.Millisecond)
	}
	return timing
}

func statusForSpan(span OTelSpan) map[string]any {
	isError := strings.EqualFold(span.StatusCode, "error") || firstString(span.Attributes, "error", "exception.message") != ""
	var errText any
	if isError {
		errText = firstStringDefault(span.StatusMessage, span.Attributes, "error", "exception.message")
	}
	return map[string]any{
		"ok":    !isError,
		"error": errText,
	}
}

func inferPhase(span OTelSpan) string {
	if strings.EqualFold(span.StatusCode, "error") {
		return "error"
	}
	if !span.EndTime.IsZero() {
		return "end"
	}
	return "start"
}

func addTextDigest(ctx map[string]any, key string, attrs map[string]any, textKey string, altTextKey string, digestKey string, altDigestKey string) {
	text := firstString(attrs, textKey, altTextKey)
	digest := firstString(attrs, digestKey, altDigestKey)
	if text == "" && digest == "" {
		return
	}
	payload := map[string]any{}
	if text != "" {
		payload["text"] = text
	}
	if digest != "" {
		payload["sha256"] = digest
	}
	ctx[key] = payload
}

func addTokenUsage(ctx map[string]any, attrs map[string]any) {
	promptTokens, hasPrompt := firstInt(attrs, "gen_ai.usage.input_tokens", "llm.usage.prompt_tokens", "prompt_tokens")
	completionTokens, hasCompletion := firstInt(attrs, "gen_ai.usage.output_tokens", "llm.usage.completion_tokens", "completion_tokens")
	totalTokens, hasTotal := firstInt(attrs, "gen_ai.usage.total_tokens", "llm.usage.total_tokens", "total_tokens")
	if !hasPrompt && !hasCompletion && !hasTotal {
		return
	}
	usage := map[string]any{}
	if hasPrompt {
		usage["prompt_tokens"] = promptTokens
	}
	if hasCompletion {
		usage["completion_tokens"] = completionTokens
	}
	if hasTotal {
		usage["total_tokens"] = totalTokens
	}
	ctx["token_usage"] = usage
}

func firstStringDefault(defaultValue string, attrs map[string]any, keys ...string) string {
	if value := firstString(attrs, keys...); value != "" {
		return value
	}
	return defaultValue
}

func firstString(attrs map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := attrs[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case string:
			return value
		case fmt.Stringer:
			return value.String()
		default:
			return fmt.Sprint(value)
		}
	}
	return ""
}

func firstBool(attrs map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		raw, ok := attrs[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case bool:
			return value, true
		case string:
			parsed, err := strconv.ParseBool(value)
			if err == nil {
				return parsed, true
			}
		}
	}
	return false, false
}

func firstInt(attrs map[string]any, keys ...string) (int64, bool) {
	for _, key := range keys {
		raw, ok := attrs[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case int:
			return int64(value), true
		case int64:
			return value, true
		case float64:
			return int64(value), true
		case string:
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func firstFloat(attrs map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		raw, ok := attrs[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case float64:
			return value, true
		case float32:
			return float64(value), true
		case int:
			return float64(value), true
		case string:
			parsed, err := strconv.ParseFloat(value, 64)
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func firstStringSlice(attrs map[string]any, keys ...string) []string {
	for _, key := range keys {
		raw, ok := attrs[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case []string:
			return append([]string{}, value...)
		case []any:
			out := make([]string, 0, len(value))
			for _, item := range value {
				out = append(out, fmt.Sprint(item))
			}
			return out
		case string:
			parts := strings.Split(value, ",")
			out := make([]string, 0, len(parts))
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					out = append(out, trimmed)
				}
			}
			return out
		}
	}
	return []string{}
}
