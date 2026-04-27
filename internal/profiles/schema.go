// SPDX-License-Identifier: MIT
package profiles

import (
	"fmt"
	"math"
	"strings"
)

var expectedWeightKeys = map[string]struct{}{
	"EC": {},
	"FC": {},
	"RC": {},
	"TC": {},
	"SC": {},
	"XC": {},
	"AC": {},
	"GC": {},
}

type ProfileSchema struct {
	ID            string             `yaml:"id"`
	Version       int                `yaml:"version"`
	WorkflowClass string             `yaml:"workflow_class"`
	BlindSpots    []string           `yaml:"blind_spots"`
	Weights       map[string]float64 `yaml:"weights"`
	SupportsCAS   bool               `yaml:"supports_cas,omitempty"`
	SCMode        string             `yaml:"sc_mode,omitempty"`
	Required      []EventRule        `yaml:"required"`
	Optional      []EventRule        `yaml:"optional"`
	Relations     []RelationRule     `yaml:"relations"`
}

type RequiredWhenRule struct {
	WhenType  string `yaml:"when_type"`
	AtOrAfter bool   `yaml:"at_or_after,omitempty"`
	Message   string `yaml:"message,omitempty"`
}

type EventRule struct {
	Type         string             `yaml:"type"`
	Fields       []string           `yaml:"fields"`
	Message      string             `yaml:"message"`
	Severity     string             `yaml:"severity"`
	RequiredWhen []RequiredWhenRule `yaml:"required_when,omitempty"`
}

type RelationRule struct {
	Name    string `yaml:"name"`
	From    string `yaml:"from"`
	To      string `yaml:"to"`
	Field   string `yaml:"field"`
	Message string `yaml:"message"`
}

func ValidateSchema(s ProfileSchema) error {
	if strings.TrimSpace(s.ID) == "" {
		return fmt.Errorf("profile schema: id is required")
	}
	if s.Version < 1 {
		return fmt.Errorf("profile schema %q: version must be >= 1", s.ID)
	}
	if strings.TrimSpace(s.WorkflowClass) == "" {
		return fmt.Errorf("profile schema %q: workflow_class is required", s.ID)
	}
	if err := validateWeights(s.ID, s.Weights); err != nil {
		return err
	}
	if err := validateEventRules(s.ID, "required", s.Required); err != nil {
		return err
	}
	if err := validateEventRules(s.ID, "optional", s.Optional); err != nil {
		return err
	}
	if err := validateRequiredWhenReferences(s.ID, s.Required, s.Optional); err != nil {
		return err
	}
	return nil
}

func validateWeights(id string, weights map[string]float64) error {
	if len(weights) != len(expectedWeightKeys) {
		return fmt.Errorf("profile schema %q: weights keys must be exactly {EC,FC,RC,TC,SC,XC,AC,GC}", id)
	}

	sum := 0.0
	for key, value := range weights {
		if _, ok := expectedWeightKeys[key]; !ok {
			return fmt.Errorf("profile schema %q: unknown weight key %q", id, key)
		}
		sum += value
	}
	for key := range expectedWeightKeys {
		if _, ok := weights[key]; !ok {
			return fmt.Errorf("profile schema %q: missing weight key %q", id, key)
		}
	}
	if math.Abs(sum-1.0) > 1e-9 {
		return fmt.Errorf("profile schema %q: weights sum to %.12f, want 1.0", id, sum)
	}
	return nil
}

func validateEventRules(id string, section string, rules []EventRule) error {
	seen := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		ruleType := strings.TrimSpace(rule.Type)
		if ruleType == "" {
			return fmt.Errorf("profile schema %q: %s rule type is required", id, section)
		}
		if _, ok := seen[ruleType]; ok {
			return fmt.Errorf("profile schema %q: duplicate %s rule type %q", id, section, ruleType)
		}
		seen[ruleType] = struct{}{}

		switch strings.TrimSpace(rule.Severity) {
		case "critical", "warning":
		default:
			return fmt.Errorf("profile schema %q: %s rule %q has invalid severity %q", id, section, ruleType, rule.Severity)
		}

		if section == "required" && len(rule.RequiredWhen) > 0 {
			return fmt.Errorf("profile schema %q: required_when not supported on required rule %q", id, ruleType)
		}
		for _, r := range rule.RequiredWhen {
			when := strings.TrimSpace(r.WhenType)
			if when == "" {
				return fmt.Errorf("profile schema %q: %s rule %q has required_when with empty when_type", id, section, ruleType)
			}
			if when == ruleType {
				return fmt.Errorf("profile schema %q: %s rule %q has required_when that references itself", id, section, ruleType)
			}
		}
	}
	return nil
}

func validateRequiredWhenReferences(id string, required, optional []EventRule) error {
	eventTypes := make(map[string]struct{})

	for _, rule := range required {
		if t := strings.TrimSpace(rule.Type); t != "" {
			eventTypes[t] = struct{}{}
		}
	}
	for _, rule := range optional {
		if t := strings.TrimSpace(rule.Type); t != "" {
			eventTypes[t] = struct{}{}
		}
	}

	edges := make(map[string]map[string]struct{})
	addEdge := func(from, to string) {
		m, ok := edges[from]
		if !ok {
			m = make(map[string]struct{})
			edges[from] = m
		}
		m[to] = struct{}{}
	}

	process := func(section string, rules []EventRule) error {
		for _, rule := range rules {
			from := strings.TrimSpace(rule.Type)
			for _, rw := range rule.RequiredWhen {
				when := strings.TrimSpace(rw.WhenType)
				if when == "" {
					continue
				}
				if _, ok := eventTypes[when]; !ok {
					return fmt.Errorf("profile schema %q: %s rule %q has required_when referencing unknown event type %q", id, section, from, when)
				}
				addEdge(when, from)
			}
		}
		return nil
	}

	if err := process("required", required); err != nil {
		return err
	}
	if err := process("optional", optional); err != nil {
		return err
	}

	for from, tos := range edges {
		for to := range tos {
			if from == to {
				continue
			}
			if back, ok := edges[to]; ok {
				if _, has := back[from]; has {
					return fmt.Errorf("profile schema %q: required_when rules form a cycle between %q and %q", id, from, to)
				}
			}
		}
	}

	return nil
}
