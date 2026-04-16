// Package profiles — DSL v1 profile loader.
//
// DSL v1 scope (what a user-defined profile file can express):
//
//   - id:              required; must not collide with a built-in profile ID.
//   - version:         optional integer >= 1; defaults to 1.
//   - description:     used as workflow_class in reports; defaults to the last
//     dot-separated segment of the id.
//   - required_events: list of event type strings; each absent event type is a
//     critical failure.
//   - warning_events:  list of event type strings; each absent event type is a
//     warning, not a critical failure.
//   - cas_weights:     optional map from CAS sub-score key to weight. Allowed keys
//     are {EC, FC, RC, TC, SC, XC, AC, GC}. Unspecified keys default
//     to 0.0. If present, the specified values must sum to 1.0 (±1e-9).
//     If omitted, the profile does not produce a CAS score.
//
// Explicitly out of scope for v1:
//   - Per-event required-field checks (no "fields" list on event rules).
//   - required_when temporal conditions.
//   - Relation rules (cross-event ID binding checks).
//   - sc_mode or blind_spot declarations.
//   - Arbitrary predicates or cross-bundle correlation.
//   - Advanced temporal logic beyond simple event presence.
//
// A DSL profile compiles to the same ProfileSchema used by built-in profiles
// and goes through the same verification and CAS machinery. There is no
// separate evaluation code path for DSL profiles.
package profiles

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DSLProfile is the YAML structure for a user-defined obligation profile.
// It is the authoritative input type for DSL v1 profile files.
type DSLProfile struct {
	ID             string             `yaml:"id"`
	Version        int                `yaml:"version"`
	Description    string             `yaml:"description"`
	RequiredEvents []string           `yaml:"required_events"`
	WarningEvents  []string           `yaml:"warning_events"`
	CASWeights     map[string]float64 `yaml:"cas_weights"`
}

// LoadDSLProfile reads a DSL v1 profile file from path and returns the
// compiled ProfileSchema. Returns a precise, user-readable error when the
// file is missing, cannot be parsed, or contains invalid values.
func LoadDSLProfile(path string) (ProfileSchema, error) {
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return ProfileSchema{}, fmt.Errorf("load DSL profile %q: %w", path, err)
	}
	schema, err := ParseDSLProfile(content)
	if err != nil {
		return ProfileSchema{}, fmt.Errorf("DSL profile %q: %w", path, err)
	}
	return schema, nil
}

// ParseDSLProfile parses and validates DSL v1 profile YAML and returns the
// compiled ProfileSchema. It enforces the DSL v1 constraints; it does NOT
// call ValidateSchema on the result (DSL compilation guarantees validity).
func ParseDSLProfile(data []byte) (ProfileSchema, error) {
	var dsl DSLProfile
	if err := yaml.Unmarshal(data, &dsl); err != nil {
		return ProfileSchema{}, fmt.Errorf("parse: %w", err)
	}

	dsl.ID = strings.TrimSpace(dsl.ID)
	dsl.Description = strings.TrimSpace(dsl.Description)

	if dsl.ID == "" {
		return ProfileSchema{}, fmt.Errorf("id is required")
	}
	if dsl.Version < 1 {
		dsl.Version = 1
	}

	// Collision check: DSL profiles must not shadow built-in profiles.
	if HasSchema(dsl.ID) {
		return ProfileSchema{}, fmt.Errorf("profile id %q collides with a built-in profile; DSL profiles must use a distinct id", dsl.ID)
	}

	// Validate CAS weights when present.
	supportsCAS := len(dsl.CASWeights) > 0
	weights, err := compileDSLWeights(dsl.ID, dsl.CASWeights, supportsCAS)
	if err != nil {
		return ProfileSchema{}, err
	}

	// Compile required_events → critical EventRules.
	required, err := compileDSLEvents(dsl.ID, dsl.RequiredEvents, "required_events", "critical")
	if err != nil {
		return ProfileSchema{}, err
	}

	// Compile warning_events → warning EventRules.
	optional, err := compileDSLEvents(dsl.ID, dsl.WarningEvents, "warning_events", "warning")
	if err != nil {
		return ProfileSchema{}, err
	}

	workflowClass := dsl.Description
	if workflowClass == "" {
		workflowClass = workflowClassFromID(dsl.ID)
	}

	return ProfileSchema{
		ID:            dsl.ID,
		Version:       dsl.Version,
		WorkflowClass: workflowClass,
		SupportsCAS:   supportsCAS,
		Weights:       weights,
		Required:      required,
		Optional:      optional,
	}, nil
}

// compileDSLWeights validates cas_weights and returns the full 8-key weight map
// with unspecified keys set to 0.0.
func compileDSLWeights(id string, raw map[string]float64, supportsCAS bool) (map[string]float64, error) {
	// Always produce a full 8-key map so downstream callers get a consistent shape.
	full := make(map[string]float64, len(expectedWeightKeys))
	for k := range expectedWeightKeys {
		full[k] = 0.0
	}
	if !supportsCAS {
		return full, nil
	}

	var sum float64
	for key, value := range raw {
		k := strings.TrimSpace(key)
		if _, ok := expectedWeightKeys[k]; !ok {
			return nil, fmt.Errorf("profile %q: unknown CAS weight key %q (allowed: EC FC RC TC SC XC AC GC)", id, key)
		}
		full[k] = value
		sum += value
	}
	if math.Abs(sum-1.0) > 1e-9 {
		return nil, fmt.Errorf("profile %q: cas_weights must sum to 1.0 (got %.12f)", id, sum)
	}
	return full, nil
}

// compileDSLEvents converts a flat list of event type strings into EventRules.
func compileDSLEvents(id string, types []string, field string, severity string) ([]EventRule, error) {
	seen := make(map[string]struct{}, len(types))
	rules := make([]EventRule, 0, len(types))
	for _, et := range types {
		et = strings.TrimSpace(et)
		if et == "" {
			return nil, fmt.Errorf("profile %q: %s contains an empty event type", id, field)
		}
		if _, dup := seen[et]; dup {
			return nil, fmt.Errorf("profile %q: %s contains duplicate event type %q", id, field, et)
		}
		seen[et] = struct{}{}
		rules = append(rules, EventRule{
			Type:     et,
			Severity: severity,
			Message:  et + " missing",
		})
	}
	return rules, nil
}

// workflowClassFromID derives a human-readable workflow class from a
// dot-namespaced profile id. Returns the last dot-separated segment.
// Example: "atb.profile.custom.demo" → "demo".
func workflowClassFromID(id string) string {
	parts := strings.Split(id, ".")
	return parts[len(parts)-1]
}
