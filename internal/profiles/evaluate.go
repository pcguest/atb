// SPDX-License-Identifier: MIT
package profiles

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

type CriticalFailure struct {
	ID     string
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
	}
	for _, rule := range schema.Optional {
		evaluateOptionalRule(&result, recordsByType[rule.Type], rule)
	}
	for _, rule := range schema.Relations {
		evaluateRelationRule(&result, recordsByType, rule)
	}
	evaluateRequiredWhenRules(&result, schema, recordsByType)

	result.InformationalNotes = append(result.InformationalNotes, schema.BlindSpots...)
	finaliseEvaluationResult(&result)
	return result
}

func evaluateRequiredRule(result *EvaluationResult, records []bundle.Record, rule EventRule) {
	if len(records) == 0 {
		appendRuleFinding(
			result,
			rule,
			requiredEventID(rule.Type),
			"missing_event",
			messageOrDefault(rule.Message, fmt.Sprintf("%s missing", rule.Type)),
		)
		return
	}
	if len(rule.Fields) == 0 {
		return
	}
	missing := missingFields(dataMap(records[0].Event.Data), rule.Fields)
	if len(missing) > 0 {
		appendRuleFinding(
			result,
			rule,
			requiredFieldID(rule.Type, missing[0]),
			"missing_field",
			messageOrDefault(rule.Message, fmt.Sprintf("%s missing required fields", rule.Type)),
		)
	}
}

func evaluateOptionalRule(result *EvaluationResult, records []bundle.Record, rule EventRule) {
	if len(records) == 0 {
		appendRuleFinding(
			result,
			rule,
			optionalEventID(rule.Type),
			"missing_event",
			messageOrDefault(rule.Message, fmt.Sprintf("%s missing", rule.Type)),
		)
		return
	}
	if len(rule.Fields) == 0 {
		return
	}
	missing := missingFields(dataMap(records[0].Event.Data), rule.Fields)
	if len(missing) > 0 {
		appendRuleFinding(
			result,
			rule,
			requiredFieldID(rule.Type, missing[0]),
			"missing_field",
			messageOrDefault(rule.Message, fmt.Sprintf("%s missing required fields", rule.Type)),
		)
	}
}

func appendRuleFinding(result *EvaluationResult, rule EventRule, id string, kind string, detail string) {
	if strings.EqualFold(rule.Severity, "warning") {
		result.RequiredWarnings = append(result.RequiredWarnings, detail)
		return
	}

	result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
		ID:     id,
		Kind:   kind,
		Detail: detail,
	})
}

func evaluateRelationRule(result *EvaluationResult, recordsByType map[string][]bundle.Record, rule RelationRule) {
	fromRecords := recordsByType[rule.From]
	toRecords := recordsByType[rule.To]
	if len(fromRecords) == 0 {
		return
	}
	if len(toRecords) == 0 {
		return
	}

	toByField := indexByField(toRecords, rule.Field)
	for _, fromRecord := range fromRecords {
		fieldValue := fieldString(fromRecord, rule.Field)
		linkedRecords := toByField[fieldValue]
		if fieldValue != "" && len(linkedRecords) > 0 && len(rule.Predicates) == 0 {
			continue
		}
		if fieldValue != "" && len(linkedRecords) > 0 {
			if failure, ok := relationPredicateFailure(rule, fromRecord, linkedRecords); ok {
				result.CriticalFailures = append(result.CriticalFailures, failure)
			}
			continue
		}
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			ID:     relationID(rule),
			Kind:   "relation_violation",
			Detail: messageOrDefault(rule.Message, relationDefaultMessage(rule)),
		})
	}
}

func relationPredicateFailure(
	rule RelationRule,
	fromRecord bundle.Record,
	linkedRecords []bundle.Record,
) (CriticalFailure, bool) {
	var firstFailure CriticalFailure
	for _, toRecord := range linkedRecords {
		failure, ok := firstRelationPredicateFailure(rule, fromRecord, toRecord)
		if !ok {
			return CriticalFailure{}, false
		}
		if firstFailure == (CriticalFailure{}) {
			firstFailure = failure
		}
	}
	return firstFailure, firstFailure != (CriticalFailure{})
}

func firstRelationPredicateFailure(
	rule RelationRule,
	fromRecord bundle.Record,
	toRecord bundle.Record,
) (CriticalFailure, bool) {
	for _, predicate := range rule.Predicates {
		target := toRecord
		if predicate.Side == "from" {
			target = fromRecord
		}

		actual := fieldString(target, predicate.Field)
		expected := predicate.Equals
		if strings.EqualFold(actual, expected) {
			continue
		}

		return CriticalFailure{
			ID:     relationPredicateID(rule, predicate),
			Kind:   "relation_violation",
			Detail: relationPredicateMessage(rule, predicate, actual),
		}, true
	}
	return CriticalFailure{}, false
}

func evaluateRequiredWhenRules(
	result *EvaluationResult,
	schema ProfileSchema,
	recordsByType map[string][]bundle.Record,
) {
	allRules := make([]EventRule, 0, len(schema.Required)+len(schema.Optional))
	allRules = append(allRules, schema.Required...)
	allRules = append(allRules, schema.Optional...)

	for _, rule := range allRules {
		if len(rule.RequiredWhen) == 0 {
			continue
		}

		targetType := rule.Type
		targetRecords := recordsByType[targetType]

		for _, cond := range rule.RequiredWhen {
			conditionType := cond.WhenType
			conditionRecords := recordsByType[conditionType]
			if len(conditionRecords) == 0 {
				continue
			}

			if len(targetRecords) == 0 {
				detail := cond.Message
				if strings.TrimSpace(detail) == "" {
					detail = fmt.Sprintf("required_when: %s required when %s present", targetType, conditionType)
				}
				result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
					ID:     requiredWhenID(targetType, conditionType),
					Kind:   "missing_event",
					Detail: detail,
				})
				continue
			}

			if !cond.AtOrAfter {
				continue
			}

			condTime, condPresent, condErr := parseEventTimestamp(conditionRecords[0])
			if !condPresent || condErr != nil {
				result.RequiredWarnings = append(result.RequiredWarnings, requiredWhenConditionTimestampWarning(targetType, conditionType, condPresent, condErr))
				continue
			}

			anyValidTargetTS := false
			anyAtOrAfter := false
			for _, rec := range targetRecords {
				ts, present, err := parseEventTimestamp(rec)
				if !present || err != nil {
					continue
				}
				anyValidTargetTS = true
				if !ts.Before(condTime) {
					anyAtOrAfter = true
					break
				}
			}

			if !anyValidTargetTS {
				result.RequiredWarnings = append(result.RequiredWarnings, requiredWhenTargetTimestampWarning(targetType, conditionType))
				continue
			}

			if !anyAtOrAfter {
				result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
					ID:     requiredWhenID(targetType, conditionType),
					Kind:   "temporal_violation",
					Detail: fmt.Sprintf("required_when: %s must occur at or after %s", targetType, conditionType),
				})
			}
		}
	}
}

func requiredWhenConditionTimestampWarning(targetType string, conditionType string, present bool, err error) string {
	switch {
	case err != nil:
		return fmt.Sprintf("required_when: unable to enforce ordering between %s and %s because %s timestamp is invalid", targetType, conditionType, conditionType)
	case !present:
		return fmt.Sprintf("required_when: unable to enforce ordering between %s and %s because %s timestamp is missing", targetType, conditionType, conditionType)
	default:
		return fmt.Sprintf("required_when: unable to enforce ordering between %s and %s because %s timestamp is unavailable", targetType, conditionType, conditionType)
	}
}

func requiredWhenTargetTimestampWarning(targetType string, conditionType string) string {
	return fmt.Sprintf("required_when: unable to enforce ordering between %s and %s because %s timestamps are missing or invalid", targetType, conditionType, targetType)
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

func requiredEventID(eventType string) string {
	return "required:" + eventType
}

func optionalEventID(eventType string) string {
	return "optional:" + eventType
}

func requiredFieldID(eventType string, field string) string {
	return "required:" + eventType + ":field:" + field
}

func relationID(rule RelationRule) string {
	if name := strings.TrimSpace(rule.Name); name != "" {
		return "relation:" + name
	}
	return "relation:" + rule.From + ":" + rule.To + ":" + rule.Field
}

func relationPredicateID(rule RelationRule, predicate RelationPredicate) string {
	return relationID(rule) + ":predicate:" + predicate.Field + ":" + predicate.Equals
}

func relationPredicateMessage(rule RelationRule, predicate RelationPredicate, actual string) string {
	return fmt.Sprintf(
		"%s: predicate %s.%s got %q, want %q",
		relationDefaultMessage(rule),
		predicate.Side,
		predicate.Field,
		actual,
		predicate.Equals,
	)
}

func requiredWhenID(targetType string, conditionType string) string {
	return "required_when:" + targetType + ":when:" + conditionType
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

func parseEventTimestamp(record bundle.Record) (time.Time, bool, error) {
	raw := strings.TrimSpace(record.Event.Timestamp)
	if raw == "" {
		return time.Time{}, false, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, true, err
	}
	return parsed, true, nil
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
