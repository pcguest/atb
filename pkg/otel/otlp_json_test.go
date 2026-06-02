// SPDX-License-Identifier: MIT
package otel_test

import (
	"errors"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/pkg/otel"
)

func TestDecodeTraceJSON_mapsLLMSpan(t *testing.T) {
	t.Parallel()

	// A representative OTLP/JSON ExportTraceServiceRequest: hex ids, unix-nano
	// timestamps as strings, int64 attribute as a string, kind/status as int
	// enums, and gen_ai.system carried at resource level.
	payload := []byte(`{
      "resourceSpans": [{
        "resource": {
          "attributes": [
            {"key": "gen_ai.system", "value": {"stringValue": "openai"}}
          ]
        },
        "scopeSpans": [{
          "scope": {"name": "my-instrumentation", "version": "1.2.3"},
          "spans": [{
            "traceId": "0102030405060708090a0b0c0d0e0f10",
            "spanId": "0102030405060708",
            "name": "gen_ai.chat",
            "kind": 3,
            "startTimeUnixNano": "1772622902000000000",
            "endTimeUnixNano": "1772622903200000000",
            "status": {"code": 1},
            "attributes": [
              {"key": "gen_ai.request.model", "value": {"stringValue": "gpt-4.1-mini"}},
              {"key": "gen_ai.usage.input_tokens", "value": {"intValue": "12"}},
              {"key": "gen_ai.usage.output_tokens", "value": {"intValue": "28"}},
              {"key": "gen_ai.usage.total_tokens", "value": {"intValue": "40"}},
              {"key": "gen_ai.request.temperature", "value": {"doubleValue": 0.7}}
            ]
          }]
        }]
      }]
    }`)

	traces, err := otel.DecodeTraceJSON(payload)
	if err != nil {
		t.Fatalf("DecodeTraceJSON: %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("traces = %d, want 1", len(traces))
	}
	if traces[0].TraceID != "0102030405060708090a0b0c0d0e0f10" {
		t.Fatalf("TraceID = %q", traces[0].TraceID)
	}
	if len(traces[0].Spans) != 1 {
		t.Fatalf("spans = %d, want 1", len(traces[0].Spans))
	}
	span := traces[0].Spans[0]
	if span.Kind != "client" {
		t.Fatalf("Kind = %q, want client", span.Kind)
	}
	if span.StatusCode != "ok" {
		t.Fatalf("StatusCode = %q, want ok", span.StatusCode)
	}
	if span.EndTime.Sub(span.StartTime) != 1200*time.Millisecond {
		t.Fatalf("duration = %v, want 1.2s", span.EndTime.Sub(span.StartTime))
	}
	// Resource-level attribute is visible to the span.
	if span.Attributes["gen_ai.system"] != "openai" {
		t.Fatalf("gen_ai.system = %v, want openai", span.Attributes["gen_ai.system"])
	}
	// int64 attribute decoded from its JSON-string encoding.
	if span.Attributes["gen_ai.usage.total_tokens"] != int64(40) {
		t.Fatalf("total_tokens = %#v, want int64(40)", span.Attributes["gen_ai.usage.total_tokens"])
	}

	// The decoded span feeds the existing translator end-to-end.
	ev, err := otel.Translate(span)
	if err != nil {
		t.Fatalf("Translate decoded span: %v", err)
	}
	if ev.Type != event.TypeAILLMCall {
		t.Fatalf("event type = %q, want %q", ev.Type, event.TypeAILLMCall)
	}
	data := ev.Data.(map[string]any)
	ctx := data["context"].(map[string]any)
	if ctx["provider"] != "openai" || ctx["model"] != "gpt-4.1-mini" {
		t.Fatalf("context = %#v", ctx)
	}
	usage := ctx["token_usage"].(map[string]any)
	if usage["total_tokens"] != int64(40) {
		t.Fatalf("total_tokens = %#v, want int64(40)", usage["total_tokens"])
	}
}

func TestDecodeTraceJSON_stringEnumsAndErrorStatus(t *testing.T) {
	t.Parallel()

	// proto3 JSON renders enums by their string name; status is an error.
	payload := []byte(`{
      "resourceSpans": [{
        "scopeSpans": [{
          "spans": [{
            "traceId": "aabbccddeeff00112233445566778899",
            "spanId": "1122334455667788",
            "name": "tool.run",
            "kind": "SPAN_KIND_INTERNAL",
            "startTimeUnixNano": "1772622902000000000",
            "status": {"code": "STATUS_CODE_ERROR", "message": "boom"},
            "attributes": [
              {"key": "tool.name", "value": {"stringValue": "shell"}}
            ]
          }]
        }]
      }]
    }`)

	traces, err := otel.DecodeTraceJSON(payload)
	if err != nil {
		t.Fatalf("DecodeTraceJSON: %v", err)
	}
	span := traces[0].Spans[0]
	if span.Kind != "internal" {
		t.Fatalf("Kind = %q, want internal", span.Kind)
	}
	if span.StatusCode != "error" {
		t.Fatalf("StatusCode = %q, want error", span.StatusCode)
	}
	if span.StatusMessage != "boom" {
		t.Fatalf("StatusMessage = %q, want boom", span.StatusMessage)
	}

	ev, err := otel.Translate(span)
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	data := ev.Data.(map[string]any)
	status := data["status"].(map[string]any)
	if status["ok"] != false {
		t.Fatalf("status.ok = %v, want false", status["ok"])
	}
	if data["phase"] != "error" {
		t.Fatalf("phase = %v, want error", data["phase"])
	}
}

func TestDecodeTraceJSON_groupsSpansByTrace(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
      "resourceSpans": [{
        "scopeSpans": [{
          "spans": [
            {"traceId": "aaaa", "spanId": "01", "name": "x", "startTimeUnixNano": "1"},
            {"traceId": "bbbb", "spanId": "02", "name": "y", "startTimeUnixNano": "2"},
            {"traceId": "aaaa", "spanId": "03", "name": "z", "startTimeUnixNano": "3"}
          ]
        }]
      }]
    }`)

	traces, err := otel.DecodeTraceJSON(payload)
	if err != nil {
		t.Fatalf("DecodeTraceJSON: %v", err)
	}
	if len(traces) != 2 {
		t.Fatalf("traces = %d, want 2", len(traces))
	}
	// First-seen order: aaaa before bbbb.
	if traces[0].TraceID != "aaaa" || traces[1].TraceID != "bbbb" {
		t.Fatalf("order = [%q, %q], want [aaaa, bbbb]", traces[0].TraceID, traces[1].TraceID)
	}
	if len(traces[0].Spans) != 2 {
		t.Fatalf("trace aaaa spans = %d, want 2", len(traces[0].Spans))
	}
	// Span order preserved within the trace.
	if traces[0].Spans[0].SpanID != "01" || traces[0].Spans[1].SpanID != "03" {
		t.Fatalf("span order = [%q, %q], want [01, 03]", traces[0].Spans[0].SpanID, traces[0].Spans[1].SpanID)
	}
}

func TestDecodeTraceJSON_instrumentationLibraryFallback(t *testing.T) {
	t.Parallel()

	// Pre-1.0 OTLP field names: instrumentationLibrarySpans / instrumentationLibrary.
	payload := []byte(`{
      "resourceSpans": [{
        "instrumentationLibrarySpans": [{
          "instrumentationLibrary": {"name": "legacy-lib", "version": "0.9"},
          "spans": [
            {"traceId": "cccc", "spanId": "0a", "name": "llm.completion", "startTimeUnixNano": "5"}
          ]
        }]
      }]
    }`)

	traces, err := otel.DecodeTraceJSON(payload)
	if err != nil {
		t.Fatalf("DecodeTraceJSON: %v", err)
	}
	if len(traces) != 1 || len(traces[0].Spans) != 1 {
		t.Fatalf("traces/spans = %d/%v", len(traces), traces)
	}
	if got := traces[0].Spans[0].Attributes["otel.library.version"]; got != "0.9" {
		t.Fatalf("otel.library.version = %v, want 0.9", got)
	}
}

func TestDecodeTraceJSON_emptyAndInvalid(t *testing.T) {
	t.Parallel()

	traces, err := otel.DecodeTraceJSON([]byte(`{"resourceSpans": []}`))
	if err != nil {
		t.Fatalf("empty payload error: %v", err)
	}
	if traces != nil {
		t.Fatalf("empty payload traces = %v, want nil", traces)
	}

	if _, err := otel.DecodeTraceJSON([]byte(`{not json`)); err == nil {
		t.Fatal("invalid JSON = nil error, want error")
	}
}

func TestDecodeTraceJSON_throughReceiver(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
      "resourceSpans": [{
        "scopeSpans": [{
          "spans": [
            {"traceId": "dddd", "spanId": "10", "name": "gen_ai.chat", "startTimeUnixNano": "1",
             "attributes": [{"key": "gen_ai.system", "value": {"stringValue": "anthropic"}}]},
            {"traceId": "dddd", "spanId": "11", "name": "unmappable", "startTimeUnixNano": "2"}
          ]
        }]
      }]
    }`)

	traces, err := otel.DecodeTraceJSON(payload)
	if err != nil {
		t.Fatalf("DecodeTraceJSON: %v", err)
	}
	r := &otel.Receiver{Translator: otel.DefaultTranslator{}}
	// The unmappable span (no event type derivable) aborts with a typed error,
	// matching Receiver's existing contract.
	if _, err := r.Receive(t.Context(), traces[0]); !errors.Is(err, otel.ErrUnmappableSpan) {
		t.Fatalf("Receive err = %v, want ErrUnmappableSpan", err)
	}
}
