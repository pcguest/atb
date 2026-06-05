// SPDX-License-Identifier: MIT
package otel_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/pkg/otel"
)

// twoSpanTracePayload is an OTLP/JSON ExportTraceServiceRequest with two spans
// in one trace that the DefaultTranslator can both map: an LLM chat span and a
// tool span.
const twoSpanTracePayload = `{
  "resourceSpans": [{
    "resource": {"attributes": [{"key": "gen_ai.system", "value": {"stringValue": "openai"}}]},
    "scopeSpans": [{
      "scope": {"name": "instr", "version": "1.0.0"},
      "spans": [
        {
          "traceId": "0102030405060708090a0b0c0d0e0f10",
          "spanId": "0102030405060708",
          "name": "gen_ai.chat",
          "kind": 3,
          "startTimeUnixNano": "1772622902000000000",
          "endTimeUnixNano": "1772622903200000000",
          "status": {"code": 1},
          "attributes": [{"key": "gen_ai.request.model", "value": {"stringValue": "gpt-4.1-mini"}}]
        },
        {
          "traceId": "0102030405060708090a0b0c0d0e0f10",
          "spanId": "1112131415161718",
          "parentSpanId": "0102030405060708",
          "name": "tool.run",
          "kind": 3,
          "startTimeUnixNano": "1772622903300000000",
          "endTimeUnixNano": "1772622903400000000",
          "status": {"code": 1}
        }
      ]
    }]
  }]
}`

func TestReceiveJSON_translatesEverySpan(t *testing.T) {
	t.Parallel()

	r := &otel.Receiver{Translator: otel.DefaultTranslator{}}
	result, err := r.ReceiveJSON(context.Background(), []byte(twoSpanTracePayload))
	if err != nil {
		t.Fatalf("ReceiveJSON: %v", err)
	}
	if len(result.Events) != 2 {
		t.Fatalf("events = %d, want 2", len(result.Events))
	}
	if got := result.Events[0].Type; got != event.TypeAILLMCall {
		t.Fatalf("event[0].Type = %q, want %q", got, event.TypeAILLMCall)
	}
	if got := result.Events[1].Type; got != event.TypeAIToolExec {
		t.Fatalf("event[1].Type = %q, want %q", got, event.TypeAIToolExec)
	}
	// The W3C trace linkage from the OTLP spans is carried onto the events so a
	// bundle append can preserve it.
	if result.Events[0].TraceID != "0102030405060708090a0b0c0d0e0f10" {
		t.Fatalf("event[0].TraceID = %q", result.Events[0].TraceID)
	}
	if result.Events[1].ParentSpanID != "0102030405060708" {
		t.Fatalf("event[1].ParentSpanID = %q, want the chat span id", result.Events[1].ParentSpanID)
	}
}

func TestReceiveJSON_emptyPayloadYieldsNoEvents(t *testing.T) {
	t.Parallel()

	r := &otel.Receiver{Translator: otel.DefaultTranslator{}}
	result, err := r.ReceiveJSON(context.Background(), []byte(`{"resourceSpans": []}`))
	if err != nil {
		t.Fatalf("ReceiveJSON: %v", err)
	}
	if len(result.Events) != 0 {
		t.Fatalf("events = %d, want 0", len(result.Events))
	}
}

func TestReceiveJSON_malformedJSONReturnsDecodeError(t *testing.T) {
	t.Parallel()

	r := &otel.Receiver{Translator: otel.DefaultTranslator{}}
	if _, err := r.ReceiveJSON(context.Background(), []byte(`{not json`)); err == nil {
		t.Fatal("ReceiveJSON: expected an error for malformed JSON, got nil")
	}
}

func TestReceiveJSON_unmappableSpanAbortsWithDefaultTranslator(t *testing.T) {
	t.Parallel()

	// A span the DefaultTranslator cannot type (no AI-shaped name, no
	// atb.event_type attribute) surfaces ErrUnmappableSpan rather than being
	// silently dropped — the ingest is strict about what it records.
	payload := `{"resourceSpans":[{"scopeSpans":[{"spans":[{
      "traceId":"0102030405060708090a0b0c0d0e0f10","spanId":"0102030405060708",
      "name":"database.query","startTimeUnixNano":"1772622902000000000"}]}]}]}`
	r := &otel.Receiver{Translator: otel.DefaultTranslator{}}
	if _, err := r.ReceiveJSON(context.Background(), []byte(payload)); !errors.Is(err, otel.ErrUnmappableSpan) {
		t.Fatalf("ReceiveJSON error = %v, want ErrUnmappableSpan", err)
	}
}

// skipTranslator is a tolerant Translator that asks the receiver to skip every
// span by returning ErrNotImplemented, exercising ReceiveJSON's SkippedCount
// aggregation across traces.
type skipTranslator struct{}

func (skipTranslator) Translate(span otel.OTelSpan) (*event.Event, error) {
	_ = span
	return nil, otel.ErrNotImplemented
}

func TestReceiveJSON_skipCountAggregatesAcrossSpans(t *testing.T) {
	t.Parallel()

	r := &otel.Receiver{Translator: skipTranslator{}}
	result, err := r.ReceiveJSON(context.Background(), []byte(twoSpanTracePayload))
	if err != nil {
		t.Fatalf("ReceiveJSON: %v", err)
	}
	if len(result.Events) != 0 {
		t.Fatalf("events = %d, want 0", len(result.Events))
	}
	if result.SkippedCount != 2 {
		t.Fatalf("SkippedCount = %d, want 2", result.SkippedCount)
	}
}
