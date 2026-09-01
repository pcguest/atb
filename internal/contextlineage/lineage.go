// SPDX-License-Identifier: MIT

// Package contextlineage builds a bounded investigation view over observable
// context evidence. It does not execute retrieval, caching, compaction, or
// assembly and deliberately has no chain-of-thought representation.
package contextlineage

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

// MetadataProvenance states how a metadata value entered the evidence record.
type MetadataProvenance string

const (
	ProvenanceAsserted MetadataProvenance = "asserted"
	ProvenanceDerived  MetadataProvenance = "derived"
	ProvenanceAttested MetadataProvenance = "attested"
)

// Unit is a content unit bound by digest.
type Unit struct {
	ID                 string                        `json:"id"`
	Kind               string                        `json:"kind"`
	Digest             string                        `json:"digest"`
	Source             string                        `json:"source,omitempty"`
	ParentIDs          []string                      `json:"parent_ids"`
	Scope              string                        `json:"scope,omitempty"`
	Sensitivity        string                        `json:"sensitivity,omitempty"`
	CreatedAt          string                        `json:"created_at,omitempty"`
	MetadataProvenance map[string]MetadataProvenance `json:"metadata_provenance,omitempty"`
	EventSequence      int                           `json:"event_sequence"`
}

// Operation is an observed change or use of context units.
type Operation struct {
	ID                  string   `json:"id"`
	Type                string   `json:"type"`
	InputUnitIDs        []string `json:"input_unit_ids"`
	OutputUnitIDs       []string `json:"output_unit_ids"`
	Method              string   `json:"method,omitempty"`
	ProcessorID         string   `json:"processor_id,omitempty"`
	ProcessorVersion    string   `json:"processor_version,omitempty"`
	InputTokenCount     *int     `json:"input_token_count,omitempty"`
	OutputTokenCount    *int     `json:"output_token_count,omitempty"`
	PreservedReferences []string `json:"preserved_references,omitempty"`
	DroppedReferences   []string `json:"dropped_references,omitempty"`
	InvocationID        string   `json:"invocation_id,omitempty"`
	EventSequence       int      `json:"event_sequence"`
}

// Lineage is the deterministic context investigation model for a bundle.
type Lineage struct {
	Units      []Unit      `json:"units"`
	Operations []Operation `json:"operations"`
	Warnings   []string    `json:"warnings"`
}

// Build derives context lineage from raw records without changing them.
func Build(records []bundle.Record) Lineage {
	lineage := Lineage{Units: []Unit{}, Operations: []Operation{}, Warnings: []string{}}
	unitIDs := map[string]int{}
	operationIDs := map[string]int{}

	for _, record := range records {
		data, ok := record.Event.Data.(map[string]any)
		if !ok {
			continue
		}
		switch record.Event.Type {
		case event.TypeAIContextUnit:
			unit, valid := unitFromData(record.Event.Sequence, data)
			if !valid {
				lineage.Warnings = append(lineage.Warnings, fmt.Sprintf("seq %d: malformed context unit", record.Event.Sequence))
				continue
			}
			if previous, exists := unitIDs[unit.ID]; exists {
				lineage.Warnings = append(lineage.Warnings, fmt.Sprintf("seq %d: duplicate context unit %q first seen at seq %d", record.Event.Sequence, unit.ID, previous))
				continue
			}
			unitIDs[unit.ID] = record.Event.Sequence
			lineage.Units = append(lineage.Units, unit)
		case event.TypeAIContextOperation:
			operation, valid := operationFromData(record.Event.Sequence, data)
			if !valid {
				lineage.Warnings = append(lineage.Warnings, fmt.Sprintf("seq %d: malformed context operation", record.Event.Sequence))
				continue
			}
			if previous, exists := operationIDs[operation.ID]; exists {
				lineage.Warnings = append(lineage.Warnings, fmt.Sprintf("seq %d: duplicate context operation %q first seen at seq %d", record.Event.Sequence, operation.ID, previous))
				continue
			}
			operationIDs[operation.ID] = record.Event.Sequence
			lineage.Operations = append(lineage.Operations, operation)
		}
	}

	for _, operation := range lineage.Operations {
		for _, unitID := range append(append([]string{}, operation.InputUnitIDs...), operation.OutputUnitIDs...) {
			if _, exists := unitIDs[unitID]; !exists {
				lineage.Warnings = append(lineage.Warnings, fmt.Sprintf("operation %q references unavailable context unit %q", operation.ID, unitID))
			}
		}
	}
	sort.Strings(lineage.Warnings)
	return lineage
}

func unitFromData(sequence int, data map[string]any) (Unit, bool) {
	unit := Unit{
		ID:                 text(data, "unit_id"),
		Kind:               text(data, "kind"),
		Digest:             text(data, "digest"),
		Source:             text(data, "source"),
		ParentIDs:          texts(data, "parent_unit_ids"),
		Scope:              text(data, "scope"),
		Sensitivity:        text(data, "sensitivity"),
		CreatedAt:          text(data, "created_at"),
		MetadataProvenance: provenance(data["metadata_provenance"]),
		EventSequence:      sequence,
	}
	return unit, unit.ID != "" && unit.Kind != "" && unit.Digest != ""
}

func operationFromData(sequence int, data map[string]any) (Operation, bool) {
	operation := Operation{
		ID:                  text(data, "operation_id"),
		Type:                text(data, "operation"),
		InputUnitIDs:        texts(data, "input_unit_ids"),
		OutputUnitIDs:       texts(data, "output_unit_ids"),
		Method:              text(data, "method"),
		ProcessorID:         text(data, "processor_id"),
		ProcessorVersion:    text(data, "processor_version"),
		InputTokenCount:     integer(data, "input_token_count"),
		OutputTokenCount:    integer(data, "output_token_count"),
		PreservedReferences: texts(data, "preserved_references"),
		DroppedReferences:   texts(data, "dropped_references"),
		InvocationID:        text(data, "invocation_id"),
		EventSequence:       sequence,
	}
	validType := operation.Type == "select" || operation.Type == "transform" || operation.Type == "compact" ||
		operation.Type == "cache_reuse" || operation.Type == "assemble"
	return operation, operation.ID != "" && validType
}

func text(data map[string]any, key string) string {
	value, _ := data[key].(string)
	return strings.TrimSpace(value)
}

func texts(data map[string]any, key string) []string {
	result := []string{}
	switch values := data[key].(type) {
	case []string:
		for _, value := range values {
			if value = strings.TrimSpace(value); value != "" {
				result = append(result, value)
			}
		}
	case []any:
		for _, raw := range values {
			if value, ok := raw.(string); ok {
				if value = strings.TrimSpace(value); value != "" {
					result = append(result, value)
				}
			}
		}
	}
	return result
}

func integer(data map[string]any, key string) *int {
	switch value := data[key].(type) {
	case int:
		return &value
	case float64:
		converted := int(value)
		if value == float64(converted) && converted >= 0 {
			return &converted
		}
	}
	return nil
}

func provenance(raw any) map[string]MetadataProvenance {
	values, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	result := map[string]MetadataProvenance{}
	for key, rawValue := range values {
		value, ok := rawValue.(string)
		if !ok {
			continue
		}
		switch MetadataProvenance(value) {
		case ProvenanceAsserted, ProvenanceDerived, ProvenanceAttested:
			result[key] = MetadataProvenance(value)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
