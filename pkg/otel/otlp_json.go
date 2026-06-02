// SPDX-License-Identifier: MIT
package otel

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DecodeTraceJSON decodes an OTLP/JSON `ExportTraceServiceRequest` payload into
// OTelTrace batches grouped by trace id, ready for Receiver.Receive.
//
// It implements the OTLP/JSON wire encoding directly — hex-encoded trace and
// span ids, unix-nano timestamps carried as JSON strings, the AnyValue union,
// and span kind / status code expressed as either their integer enum or their
// proto string name — without importing the OpenTelemetry SDK, so pkg/otel
// stays dependency-free (see types.go).
//
// Resource-level and scope-level attributes are merged into each span's
// attribute map, with span attributes taking precedence, so context recorded
// once per resource (for example gen_ai.system) is visible to the translator.
// Traces are returned in first-seen order; spans keep their order within a
// trace. A payload with no spans yields a nil slice and no error.
func DecodeTraceJSON(data []byte) ([]OTelTrace, error) {
	var req otlpExportTraceJSON
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("otel: decode OTLP/JSON: %w", err)
	}

	var order []string
	byTrace := map[string][]OTelSpan{}

	for _, rs := range req.ResourceSpans {
		resourceAttrs := keyValuesToMap(rs.Resource.Attributes)
		scopeSpans := rs.ScopeSpans
		if len(scopeSpans) == 0 {
			// Pre-1.0 OTLP named this field instrumentationLibrarySpans.
			scopeSpans = rs.InstrumentationLibrarySpans
		}
		for _, ss := range scopeSpans {
			scope := ss.Scope
			if scope.Name == "" && scope.Version == "" {
				scope = ss.InstrumentationLibrary
			}
			for _, raw := range ss.Spans {
				span, err := raw.toOTelSpan(resourceAttrs, scope)
				if err != nil {
					return nil, err
				}
				if _, seen := byTrace[span.TraceID]; !seen {
					order = append(order, span.TraceID)
				}
				byTrace[span.TraceID] = append(byTrace[span.TraceID], span)
			}
		}
	}

	if len(order) == 0 {
		return nil, nil
	}
	traces := make([]OTelTrace, 0, len(order))
	for _, traceID := range order {
		traces = append(traces, OTelTrace{TraceID: traceID, Spans: byTrace[traceID]})
	}
	return traces, nil
}

// --- OTLP/JSON wire structs (only the fields ATB consumes) ---

type otlpExportTraceJSON struct {
	ResourceSpans []otlpResourceSpans `json:"resourceSpans"`
}

type otlpResourceSpans struct {
	Resource                    otlpResource     `json:"resource"`
	ScopeSpans                  []otlpScopeSpans `json:"scopeSpans"`
	InstrumentationLibrarySpans []otlpScopeSpans `json:"instrumentationLibrarySpans"`
}

type otlpResource struct {
	Attributes []otlpKeyValue `json:"attributes"`
}

type otlpScopeSpans struct {
	Scope                  otlpScope  `json:"scope"`
	InstrumentationLibrary otlpScope  `json:"instrumentationLibrary"`
	Spans                  []otlpSpan `json:"spans"`
}

type otlpScope struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type otlpSpan struct {
	TraceID           string          `json:"traceId"`
	SpanID            string          `json:"spanId"`
	ParentSpanID      string          `json:"parentSpanId"`
	Name              string          `json:"name"`
	Kind              json.RawMessage `json:"kind"`
	StartTimeUnixNano string          `json:"startTimeUnixNano"`
	EndTimeUnixNano   string          `json:"endTimeUnixNano"`
	Attributes        []otlpKeyValue  `json:"attributes"`
	Status            otlpStatus      `json:"status"`
	Events            []otlpSpanEvent `json:"events"`
}

type otlpStatus struct {
	Code    json.RawMessage `json:"code"`
	Message string          `json:"message"`
}

type otlpSpanEvent struct {
	TimeUnixNano string         `json:"timeUnixNano"`
	Name         string         `json:"name"`
	Attributes   []otlpKeyValue `json:"attributes"`
}

type otlpKeyValue struct {
	Key   string       `json:"key"`
	Value otlpAnyValue `json:"value"`
}

type otlpAnyValue struct {
	StringValue *string          `json:"stringValue"`
	BoolValue   *bool            `json:"boolValue"`
	IntValue    json.RawMessage  `json:"intValue"`
	DoubleValue *float64         `json:"doubleValue"`
	ArrayValue  *otlpArrayValue  `json:"arrayValue"`
	KvlistValue *otlpKvlistValue `json:"kvlistValue"`
	BytesValue  *string          `json:"bytesValue"`
}

type otlpArrayValue struct {
	Values []otlpAnyValue `json:"values"`
}

type otlpKvlistValue struct {
	Values []otlpKeyValue `json:"values"`
}

// --- conversion ---

func (s otlpSpan) toOTelSpan(resourceAttrs map[string]any, scope otlpScope) (OTelSpan, error) {
	start, err := unixNanoToTime(s.StartTimeUnixNano)
	if err != nil {
		return OTelSpan{}, fmt.Errorf("otel: span %s start time: %w", s.SpanID, err)
	}
	end, err := unixNanoToTime(s.EndTimeUnixNano)
	if err != nil {
		return OTelSpan{}, fmt.Errorf("otel: span %s end time: %w", s.SpanID, err)
	}

	// Merge resource → scope → span attributes; span attributes win.
	attrs := map[string]any{}
	for k, v := range resourceAttrs {
		attrs[k] = v
	}
	if scope.Name != "" {
		attrs["otel.scope.name"] = scope.Name
		attrs["otel.library.name"] = scope.Name
	}
	if scope.Version != "" {
		attrs["otel.scope.version"] = scope.Version
		attrs["otel.library.version"] = scope.Version
	}
	for k, v := range keyValuesToMap(s.Attributes) {
		attrs[k] = v
	}

	out := OTelSpan{
		TraceID:       s.TraceID,
		SpanID:        s.SpanID,
		ParentSpanID:  s.ParentSpanID,
		Name:          s.Name,
		Kind:          spanKindToString(s.Kind),
		StartTime:     start,
		EndTime:       end,
		StatusCode:    statusCodeToString(s.Status.Code),
		StatusMessage: s.Status.Message,
		Attributes:    attrs,
	}
	for _, e := range s.Events {
		ts, err := unixNanoToTime(e.TimeUnixNano)
		if err != nil {
			return OTelSpan{}, fmt.Errorf("otel: span %s event %q time: %w", s.SpanID, e.Name, err)
		}
		out.Events = append(out.Events, OTelSpanEvent{
			Name:       e.Name,
			Timestamp:  ts,
			Attributes: keyValuesToMap(e.Attributes),
		})
	}
	return out, nil
}

func keyValuesToMap(kvs []otlpKeyValue) map[string]any {
	if len(kvs) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(kvs))
	for _, kv := range kvs {
		if kv.Key == "" {
			continue
		}
		out[kv.Key] = kv.Value.toAny()
	}
	return out
}

func (v otlpAnyValue) toAny() any {
	switch {
	case v.StringValue != nil:
		return *v.StringValue
	case v.BoolValue != nil:
		return *v.BoolValue
	case len(v.IntValue) > 0:
		// OTLP/JSON encodes int64 as a string; tolerate a bare JSON number too.
		s := strings.Trim(string(v.IntValue), `"`)
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
		return s
	case v.DoubleValue != nil:
		return *v.DoubleValue
	case v.ArrayValue != nil:
		out := make([]any, 0, len(v.ArrayValue.Values))
		for _, item := range v.ArrayValue.Values {
			out = append(out, item.toAny())
		}
		return out
	case v.KvlistValue != nil:
		return keyValuesToMap(v.KvlistValue.Values)
	case v.BytesValue != nil:
		return *v.BytesValue
	default:
		return nil
	}
}

func unixNanoToTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" {
		return time.Time{}, nil
	}
	nanos, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid unix-nano %q: %w", raw, err)
	}
	return time.Unix(0, nanos).UTC(), nil
}

// spanKindToString maps an OTLP span kind (integer enum or proto string name)
// to the short lower-case form the translator expects.
func spanKindToString(raw json.RawMessage) string {
	s := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	if s == "" {
		return ""
	}
	switch strings.ToUpper(s) {
	case "1", "SPAN_KIND_INTERNAL":
		return "internal"
	case "2", "SPAN_KIND_SERVER":
		return "server"
	case "3", "SPAN_KIND_CLIENT":
		return "client"
	case "4", "SPAN_KIND_PRODUCER":
		return "producer"
	case "5", "SPAN_KIND_CONSUMER":
		return "consumer"
	case "0", "SPAN_KIND_UNSPECIFIED":
		return "unspecified"
	default:
		return strings.ToLower(s)
	}
}

// statusCodeToString maps an OTLP status code (integer enum or proto string
// name) to "ok" / "error" / "unset"; the translator treats "error" as a failure.
func statusCodeToString(raw json.RawMessage) string {
	s := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	switch strings.ToUpper(s) {
	case "1", "STATUS_CODE_OK":
		return "ok"
	case "2", "STATUS_CODE_ERROR":
		return "error"
	default:
		return "unset"
	}
}
