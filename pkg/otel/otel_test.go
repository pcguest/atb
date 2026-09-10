// SPDX-License-Identifier: MIT
package otel_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/pkg/otel"
)

func TestNewDefaultTranslator(t *testing.T) {
	t.Parallel()
	if got := otel.NewDefaultTranslator(); got == nil {
		t.Fatal("NewDefaultTranslator() returned nil")
	}
}

func TestTranslate_mapsLLMSpan(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 9, 9, 15, 2, 0, time.UTC)
	end := start.Add(1200 * time.Millisecond)

	got, err := otel.Translate(otel.OTelSpan{
		TraceID:   "0102030405060708090a0b0c0d0e0f10",
		SpanID:    "0102030405060708",
		Name:      "gen_ai.chat",
		StartTime: start,
		EndTime:   end,
		Attributes: map[string]any{
			"gen_ai.system":              "openai",
			"gen_ai.request.model":       "gpt-4.1-mini",
			"gen_ai.usage.input_tokens":  12,
			"gen_ai.usage.output_tokens": 28,
			"gen_ai.usage.total_tokens":  40,
			"privacy.mode":               "hash",
			"deployment.region":          "australia-southeast1",
			"tool.arguments":             `{"account":"customer-17"}`,
		},
	})
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}
	if got.Type != event.TypeAILLMCall {
		t.Fatalf("Type = %q, want %q", got.Type, event.TypeAILLMCall)
	}
	if got.TraceID != "0102030405060708090a0b0c0d0e0f10" {
		t.Fatalf("TraceID = %q", got.TraceID)
	}
	data, ok := got.Data.(map[string]any)
	if !ok {
		t.Fatalf("Data type = %T, want map[string]any", got.Data)
	}
	if data["phase"] != "end" {
		t.Fatalf("phase = %v, want end", data["phase"])
	}
	context, ok := data["context"].(map[string]any)
	if !ok {
		t.Fatalf("context type = %T, want map[string]any", data["context"])
	}
	if context["provider"] != "openai" {
		t.Fatalf("provider = %v, want openai", context["provider"])
	}
	tokenUsage := context["token_usage"].(map[string]any)
	if tokenUsage["total_tokens"] != int64(40) {
		t.Fatalf("total_tokens = %v, want 40", tokenUsage["total_tokens"])
	}
	attributes := data["otel_attributes"].(map[string]any)
	if attributes["deployment.region"] != "australia-southeast1" {
		t.Fatalf("deployment.region = %v", attributes["deployment.region"])
	}
	arguments := attributes["tool.arguments"].(map[string]any)
	if arguments["redacted"] != true || arguments["sha256"] == "" {
		t.Fatalf("tool.arguments = %#v, want digest-only value", arguments)
	}
	if strings.Contains(fmt.Sprint(arguments), "customer-17") {
		t.Fatalf("tool.arguments leaked raw content: %#v", arguments)
	}
}

func TestTranslateMapsCurrentGenAISemanticsWithoutRawContent(t *testing.T) {
	t.Parallel()
	got, err := otel.Translate(otel.OTelSpan{
		TraceID:   "0102030405060708090a0b0c0d0e0f10",
		SpanID:    "0102030405060708",
		Name:      "retrieval policy-handbook",
		StartTime: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		Attributes: map[string]any{
			"gen_ai.operation.name":       "retrieval",
			"gen_ai.provider.name":        "example",
			"gen_ai.data_source.id":       "policy-handbook",
			"gen_ai.retrieval.query.text": "secret approval question",
			"gen_ai.retrieval.documents":  []any{map[string]any{"id": "doc-1", "score": 0.9}},
			"gen_ai.input.messages":       "private input sentinel",
			"gen_ai.output.messages":      "private output sentinel",
		},
	})
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}
	if got.Type != event.TypeAIRetrievalExecuted {
		t.Fatalf("Type = %q, want %q", got.Type, event.TypeAIRetrievalExecuted)
	}
	data := got.Data.(map[string]any)
	context := data["context"].(map[string]any)
	query := context["query"].(map[string]any)
	if query["sha256"] == "" || strings.Contains(fmt.Sprint(context), "secret approval question") {
		t.Fatalf("retrieval context must be digest-only: %#v", context)
	}
	for _, secret := range []string{"secret approval question", "doc-1", "private input sentinel", "private output sentinel"} {
		if strings.Contains(fmt.Sprint(got), secret) {
			t.Fatalf("translated event retained sensitive content %q", secret)
		}
	}
}

func TestTranslate_ignoresUnknownEventTypeHint(t *testing.T) {
	t.Parallel()
	got, err := otel.Translate(otel.OTelSpan{
		TraceID:   "0102030405060708090a0b0c0d0e0f10",
		SpanID:    "0102030405060708",
		Name:      "gen_ai.chat",
		StartTime: time.Date(2026, 3, 9, 9, 15, 2, 0, time.UTC),
		Attributes: map[string]any{
			"atb.event_type": "hostile.invented.type",
			"gen_ai.system":  "openai",
		},
	})
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}
	if got.Type != event.TypeAILLMCall {
		t.Fatalf("Type = %q, want mapped %q not the unallowlisted hint", got.Type, event.TypeAILLMCall)
	}
}

func TestTranslate_returnsTypedErrorForUnmappableSpan(t *testing.T) {
	t.Parallel()
	_, err := otel.Translate(otel.OTelSpan{
		SpanID: "0102030405060708",
		Name:   "test.span",
	})
	if !errors.Is(err, otel.ErrUnmappableSpan) {
		t.Fatalf("Translate() error = %v, want %v", err, otel.ErrUnmappableSpan)
	}
}

func TestLegacyReceiverSurfaceRemainsCompatible(t *testing.T) {
	t.Parallel()

	var transport otel.InboundTransport = otel.StubTransport{}
	if err := transport.Receive(t.Context(), otel.OTelTrace{}); !errors.Is(err, otel.ErrNotImplemented) {
		t.Fatalf("StubTransport.Receive() error = %v, want %v", err, otel.ErrNotImplemented)
	}

	result := otel.ReceiverResult{Errors: []error{otel.ErrNotImplemented}}
	if len(result.Errors) != 1 {
		t.Fatalf("ReceiverResult.Errors length = %d, want 1", len(result.Errors))
	}
}

func TestReceiver_returnsUnmappableSpanError(t *testing.T) {
	t.Parallel()
	r := &otel.Receiver{
		Translator: otel.DefaultTranslator{},
	}
	trace := otel.OTelTrace{
		TraceID: "0102030405060708090a0b0c0d0e0f10",
		Spans: []otel.OTelSpan{
			{TraceID: "0102030405060708090a0b0c0d0e0f10", SpanID: "0102030405060708", Name: "a"},
			{TraceID: "0102030405060708090a0b0c0d0e0f10", SpanID: "0908070605040302", Name: "b"},
		},
	}
	got, err := r.Receive(t.Context(), trace)
	if !errors.Is(err, otel.ErrUnmappableSpan) {
		t.Fatalf("Receive() error = %v, want %v", err, otel.ErrUnmappableSpan)
	}
	if got.SkippedCount != 0 {
		t.Fatalf("SkippedCount = %d, want 0", got.SkippedCount)
	}
}
