package profiles

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
)

type CriticalFailure struct {
	Kind   string
	Detail string
}

type EvaluationResult struct {
	Pass               bool
	CriticalFailures   []CriticalFailure
	RequiredWarnings   []string
	InformationalNotes []string
}

func Evaluate(schema ProfileSchema, records []bundle.Record) EvaluationResult {
	result := EvaluationResult{
		CriticalFailures:   []CriticalFailure{},
		RequiredWarnings:   []string{},
		InformationalNotes: []string{},
	}

	recordsByType := indexRecordsByType(records)

	for _, rule := range schema.Required {
		evaluateRequiredRule(&result, recordsByType[rule.Type], rule)
		appendTemporalConditionNotes(&result, rule)
	}
	for _, rule := range schema.Optional {
		evaluateOptionalRule(&result, recordsByType[rule.Type], rule)
		appendTemporalConditionNotes(&result, rule)
	}
	for _, rule := range schema.Relations {
		evaluateRelationRule(&result, recordsByType, rule)
	}

	result.InformationalNotes = append(result.InformationalNotes, schema.BlindSpots...)
	finaliseEvaluationResult(&result)
	return result
}

func evaluateRequiredRule(result *EvaluationResult, records []bundle.Record, rule EventRule) {
	if len(records) == 0 {
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind:   "missing_event",
			Detail: messageOrDefault(rule.Message, fmt.Sprintf("%s missing", rule.Type)),
		})
		return
	}
	if len(rule.Fields) == 0 {
		return
	}
	if len(missingFields(dataMap(records[0].Event.Data), rule.Fields)) > 0 {
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind:   "missing_field",
			Detail: messageOrDefault(rule.Message, fmt.Sprintf("%s missing required fields", rule.Type)),
		})
	}
}

func evaluateOptionalRule(result *EvaluationResult, records []bundle.Record, rule EventRule) {
	if len(records) == 0 {
		result.RequiredWarnings = append(result.RequiredWarnings, messageOrDefault(rule.Message, fmt.Sprintf("%s missing", rule.Type)))
		return
	}
	if len(rule.Fields) == 0 {
		return
	}
	if len(missingFields(dataMap(records[0].Event.Data), rule.Fields)) > 0 {
		result.RequiredWarnings = append(result.RequiredWarnings, messageOrDefault(rule.Message, fmt.Sprintf("%s missing required fields", rule.Type)))
	}
}

func evaluateRelationRule(result *EvaluationResult, recordsByType map[string][]bundle.Record, rule RelationRule) {
	fromRecords := recordsByType[rule.From]
	toRecords := recordsByType[rule.To]
	if len(fromRecords) == 0 {
		return
	}

	toByField := indexByField(toRecords, rule.Field)
	for _, fromRecord := range fromRecords {
		fieldValue := fieldString(fromRecord, rule.Field)
		if fieldValue != "" && len(toByField[fieldValue]) > 0 {
			continue
		}
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind:   "relation_violation",
			Detail: messageOrDefault(rule.Message, relationDefaultMessage(rule)),
		})
	}
}

func appendTemporalConditionNotes(result *EvaluationResult, rule EventRule) {
	if len(rule.TemporalConditions) == 0 {
		return
	}
	result.InformationalNotes = append(result.InformationalNotes,
		fmt.Sprintf("temporal_conditions: %s ignored in v1", rule.Type))
}

func messageOrDefault(message string, fallback string) string {
	if strings.TrimSpace(message) != "" {
		return message
	}
	return fallback
}

func relationDefaultMessage(rule RelationRule) string {
	if strings.TrimSpace(rule.Name) != "" {
		return rule.Name
	}
	return fmt.Sprintf("%s %s must match on %s", rule.From, rule.To, rule.Field)
}

func dataMap(data any) map[string]any {
	if data == nil {
		return nil
	}
	m, ok := data.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

func missingFields(data map[string]any, fields []string) []string {
	missing := make([]string, 0, len(fields))
	for _, field := range fields {
		if !hasField(data, field) {
			missing = append(missing, field)
		}
	}
	return missing
}

func hasField(data map[string]any, key string) bool {
	if data == nil {
		return false
	}
	value, ok := data[key]
	if !ok {
		return false
	}
	if stringValue, ok := value.(string); ok {
		return stringValue != ""
	}
	return value != nil
}

func indexRecordsByType(records []bundle.Record) map[string][]bundle.Record {
	index := make(map[string][]bundle.Record, len(records))
	for _, record := range records {
		index[record.Event.Type] = append(index[record.Event.Type], record)
	}
	return index
}

func indexByField(records []bundle.Record, field string) map[string][]bundle.Record {
	index := make(map[string][]bundle.Record, len(records))
	for _, record := range records {
		value := fieldString(record, field)
		if value == "" {
			continue
		}
		index[value] = append(index[value], record)
	}
	return index
}

func fieldString(record bundle.Record, field string) string {
	data := dataMap(record.Event.Data)
	if data == nil {
		return ""
	}
	value, ok := data[field].(string)
	if !ok {
		return ""
	}
	return value
}

func finaliseEvaluationResult(result *EvaluationResult) {
	result.RequiredWarnings = appendUniqueStrings(nil, result.RequiredWarnings...)
	result.InformationalNotes = appendUniqueStrings(nil, result.InformationalNotes...)
	sort.Strings(result.RequiredWarnings)
	sort.Strings(result.InformationalNotes)
	sort.Slice(result.CriticalFailures, func(i, j int) bool {
		if result.CriticalFailures[i].Kind == result.CriticalFailures[j].Kind {
			return result.CriticalFailures[i].Detail < result.CriticalFailures[j].Detail
		}
		return result.CriticalFailures[i].Kind < result.CriticalFailures[j].Kind
	})
	result.Pass = len(result.CriticalFailures) == 0
}

func appendUniqueStrings(dst []string, values ...string) []string {
	seen := make(map[string]struct{}, len(dst)+len(values))
	for _, value := range dst {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		dst = append(dst, value)
	}
	return dst
}
