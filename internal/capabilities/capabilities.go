// SPDX-License-Identifier: MIT

// Package capabilities derives versioned evidence capabilities from raw ATB
// events. A capability is an investigation aid, not a replacement record: the
// raw event remains the verifiable evidence and mappings never invent fields.
package capabilities

import (
	"sort"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

const MappingVersion = "atb.capability.v1"

// Evidence is a deterministic interpretation of one raw record.
type Evidence struct {
	Name           string         `json:"name"`
	MappingVersion string         `json:"mapping_version"`
	EventSequence  int            `json:"event_sequence"`
	RawEventType   string         `json:"raw_event_type"`
	Fields         map[string]any `json:"fields"`
}

// Derive returns canonical capabilities in event-sequence order.
func Derive(records []bundle.Record) []Evidence {
	out := []Evidence{}
	for _, record := range records {
		capability, ok := deriveRecord(record)
		if ok {
			out = append(out, capability)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].EventSequence < out[j].EventSequence
	})
	return out
}

func deriveRecord(record bundle.Record) (Evidence, bool) {
	if record.Event.Type != event.TypeAIRetrievalExecuted && record.Event.Type != event.TypeRAGRetrieval {
		return Evidence{}, false
	}

	data, ok := record.Event.Data.(map[string]any)
	if !ok {
		return Evidence{}, false
	}
	fields := map[string]any{}
	copyIfPresent(fields, data, "retrieval_id", "request_id", "query_digest", "index_id", "strategy",
		"result_set_digest", "selected_node_ids", "section_paths", "page_start", "page_end")
	return Evidence{
		Name:           "retrieval.performed",
		MappingVersion: MappingVersion,
		EventSequence:  record.Event.Sequence,
		RawEventType:   record.Event.Type,
		Fields:         fields,
	}, true
}

func copyIfPresent(destination, source map[string]any, keys ...string) {
	for _, key := range keys {
		value, ok := source[key]
		if !ok {
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			continue
		}
		destination[key] = value
	}
}
