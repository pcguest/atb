// SPDX-License-Identifier: MIT
package otel

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

type testStringer string

func (s testStringer) String() string { return string(s) }

func TestAnyValueAndSpanEnumConversions(t *testing.T) {
	text := "text"
	boolean := true
	double := 1.5
	bytesValue := "AQI="
	tests := []struct {
		name string
		in   otlpAnyValue
		want any
	}{
		{name: "string", in: otlpAnyValue{StringValue: &text}, want: "text"},
		{name: "bool", in: otlpAnyValue{BoolValue: &boolean}, want: true},
		{name: "integer", in: otlpAnyValue{IntValue: json.RawMessage(`"42"`)}, want: int64(42)},
		{name: "invalid integer", in: otlpAnyValue{IntValue: json.RawMessage(`"not-int"`)}, want: "not-int"},
		{name: "double", in: otlpAnyValue{DoubleValue: &double}, want: 1.5},
		{name: "array", in: otlpAnyValue{ArrayValue: &otlpArrayValue{Values: []otlpAnyValue{{StringValue: &text}}}}, want: []any{"text"}},
		{name: "kvlist", in: otlpAnyValue{KvlistValue: &otlpKvlistValue{Values: []otlpKeyValue{{Key: "k", Value: otlpAnyValue{BoolValue: &boolean}}}}}, want: map[string]any{"k": true}},
		{name: "bytes", in: otlpAnyValue{BytesValue: &bytesValue}, want: "AQI="},
		{name: "empty", in: otlpAnyValue{}, want: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.in.toAny(); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got=%#v want=%#v", got, tc.want)
			}
		})
	}

	kinds := map[string]string{
		"":                        "",
		"0":                       "unspecified",
		"SPAN_KIND_INTERNAL":      "internal",
		"2":                       "server",
		"SPAN_KIND_CLIENT":        "client",
		"4":                       "producer",
		"SPAN_KIND_CONSUMER":      "consumer",
		`"SPAN_KIND_UNSPECIFIED"`: "unspecified",
		"CUSTOM":                  "custom",
	}
	for input, want := range kinds {
		if got := spanKindToString(json.RawMessage(input)); got != want {
			t.Fatalf("spanKindToString(%q)=%q want=%q", input, got, want)
		}
	}
}

func TestTranslatorAttributeCoercionsAndChainContext(t *testing.T) {
	attrs := map[string]any{
		"string":        "value",
		"stringer":      testStringer("stringed"),
		"number":        7,
		"bool":          true,
		"bool-string":   "false",
		"bad-bool":      "maybe",
		"int":           1,
		"int64":         int64(2),
		"float-int":     3.9,
		"string-int":    "4",
		"bad-int":       "four",
		"float":         1.25,
		"float32":       float32(2.5),
		"float-fromint": 3,
		"string-float":  "4.75",
		"bad-float":     "four",
		"strings":       []string{"a", "b"},
		"anys":          []any{"a", 2},
		"csv":           "a, b, ,c",
	}
	if firstString(attrs, "missing", "string") != "value" ||
		firstString(attrs, "stringer") != "stringed" ||
		firstString(attrs, "number") != "7" {
		t.Fatal("string coercion failed")
	}
	if got, ok := firstBool(attrs, "bool"); !ok || !got {
		t.Fatalf("bool=%v,%v", got, ok)
	}
	if got, ok := firstBool(attrs, "bool-string"); !ok || got {
		t.Fatalf("bool string=%v,%v", got, ok)
	}
	if _, ok := firstBool(attrs, "bad-bool"); ok {
		t.Fatal("invalid bool parsed")
	}
	for key, want := range map[string]int64{"int": 1, "int64": 2, "float-int": 3, "string-int": 4} {
		if got, ok := firstInt(attrs, key); !ok || got != want {
			t.Fatalf("firstInt(%s)=%d,%v", key, got, ok)
		}
	}
	if _, ok := firstInt(attrs, "bad-int"); ok {
		t.Fatal("invalid int parsed")
	}
	for key, want := range map[string]float64{"float": 1.25, "float32": 2.5, "float-fromint": 3, "string-float": 4.75} {
		if got, ok := firstFloat(attrs, key); !ok || got != want {
			t.Fatalf("firstFloat(%s)=%v,%v", key, got, ok)
		}
	}
	if _, ok := firstFloat(attrs, "bad-float"); ok {
		t.Fatal("invalid float parsed")
	}
	if got := firstStringSlice(attrs, "strings"); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("strings=%v", got)
	}
	if got := firstStringSlice(attrs, "anys"); !reflect.DeepEqual(got, []string{"a", "2"}) {
		t.Fatalf("anys=%v", got)
	}
	if got := firstStringSlice(attrs, "csv"); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("csv=%v", got)
	}

	ctx := chainContext(OTelSpan{Attributes: map[string]any{
		"chain.name":        "review",
		"chain.input_keys":  []any{"prompt", "policy"},
		"chain.output_keys": "decision, reason",
		"chain.step_count":  "3",
	}})
	if ctx["chain_name"] != "review" || ctx["step_count"] != int64(3) {
		t.Fatalf("chain context=%v", ctx)
	}
	if got := fmt.Sprint(ctx["output_keys"]); got != "[decision reason]" {
		t.Fatalf("output keys=%s", got)
	}
	if got := fmt.Sprint(ctx["input_keys"]); got != "[prompt policy]" {
		t.Fatalf("input keys=%s", got)
	}
}

func TestTextDigestBranches(t *testing.T) {
	ctx := map[string]any{}
	addTextDigest(ctx, "none", map[string]any{}, "text", "alt", "digest", "alt-digest")
	if len(ctx) != 0 {
		t.Fatalf("empty digest context=%v", ctx)
	}
	addTextDigest(ctx, "text", map[string]any{"text": "prompt"}, "text", "alt", "digest", "alt-digest")
	addTextDigest(ctx, "digest", map[string]any{"digest": "sha256:value"}, "text", "alt", "digest", "alt-digest")
	addTextDigest(ctx, "both", map[string]any{"alt": "completion", "alt-digest": "sha256:both"}, "text", "alt", "digest", "alt-digest")
	if len(ctx) != 3 {
		t.Fatalf("digest context=%v", ctx)
	}
}
