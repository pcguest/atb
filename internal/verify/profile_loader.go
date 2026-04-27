// SPDX-License-Identifier: MIT
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	profiledsl "github.com/pcguest/atb/internal/profiles"
	"gopkg.in/yaml.v3"
)

type profileFileConfig struct {
	ID          string                   `yaml:"id"`
	DisplayName string                   `yaml:"display_name"`
	Description string                   `yaml:"description"`
	Detect      profileDetectConfig      `yaml:"detect"`
	Obligations profileObligationsConfig `yaml:"obligations"`
	Relations   []profileRelationConfig  `yaml:"relations"`
}

type profileDetectConfig struct {
	EventTypes []string `yaml:"event_types"`
}

type profileObligationsConfig struct {
	Critical []profileRequirementConfig `yaml:"critical"`
	Required []profileRequirementConfig `yaml:"required"`
}

type profileRequirementConfig struct {
	EventType string `yaml:"event_type"`
	Message   string `yaml:"message"`
}

type profileRelationConfig struct {
	From    string `yaml:"from"`
	To      string `yaml:"to"`
	Message string `yaml:"message"`
}

type externalProfile struct {
	config profileFileConfig
}

type externalSchemaProfile struct {
	schema profiledsl.ProfileSchema
}

type profileFileFormat int

const (
	profileFileFormatUnknown profileFileFormat = iota
	profileFileFormatSchema
	profileFileFormatLegacy
	profileFileFormatDSL
)

// ResolveProfile resolves either a built-in profile ID or a profile file path.
func ResolveProfile(spec string) (Profile, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}
	if isProfilePath(spec) {
		return loadProfileFromFile(spec)
	}

	profile := ProfileByID(spec)
	if profile == nil {
		return nil, fmt.Errorf("unrecognised profile %q", spec)
	}
	return profile, nil
}

func isProfilePath(spec string) bool {
	if strings.ContainsAny(spec, `/\`) {
		return true
	}

	switch strings.ToLower(filepath.Ext(spec)) {
	case ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func loadProfileFromFile(path string) (Profile, error) {
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("load profile %q: %w", path, err)
	}

	format, err := detectProfileFileFormat(content)
	if err != nil {
		return nil, fmt.Errorf("parse profile %q: %w", path, err)
	}

	if format == profileFileFormatDSL {
		return loadDSLProfileFromFile(path, content)
	}

	if format == profileFileFormatSchema {
		return loadSchemaProfileFromFile(path, content)
	}

	var cfg profileFileConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		if format == profileFileFormatLegacy {
			return nil, fmt.Errorf("parse profile %q: %w", path, err)
		}
		return loadSchemaProfileFromFile(path, content)
	}
	if err := cfg.validate(path); err != nil {
		if format != profileFileFormatLegacy {
			return loadSchemaProfileFromFile(path, content)
		}
		return nil, err
	}

	cfg.normalise()
	return &externalProfile{config: cfg}, nil
}

func loadDSLProfileFromFile(path string, content []byte) (Profile, error) {
	schema, err := profiledsl.ParseDSLProfile(content)
	if err != nil {
		return nil, fmt.Errorf("parse profile %q: %w", path, err)
	}
	return &externalSchemaProfile{schema: schema}, nil
}

func detectProfileFileFormat(content []byte) (profileFileFormat, error) {
	var raw map[string]any
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return profileFileFormatUnknown, err
	}

	switch {
	case hasAnyKeys(raw, "required_events", "warning_events", "cas_weights"):
		return profileFileFormatDSL, nil
	case hasAnyKeys(raw, "version", "workflow_class", "required", "optional"):
		return profileFileFormatSchema, nil
	case hasAnyKeys(raw, "display_name", "detect", "obligations"):
		return profileFileFormatLegacy, nil
	default:
		return profileFileFormatUnknown, nil
	}
}

func hasAnyKeys(values map[string]any, keys ...string) bool {
	for _, key := range keys {
		if _, ok := values[key]; ok {
			return true
		}
	}
	return false
}

func loadSchemaProfileFromFile(path string, content []byte) (Profile, error) {
	schema, err := profiledsl.ParseSchema(content)
	if err != nil {
		return nil, fmt.Errorf("parse profile %q: %w", path, err)
	}
	return &externalSchemaProfile{schema: schema}, nil
}

func (c *profileFileConfig) validate(path string) error {
	switch {
	case strings.TrimSpace(c.ID) == "":
		return fmt.Errorf("profile %q: missing required field %q", path, "id")
	case strings.TrimSpace(c.DisplayName) == "":
		return fmt.Errorf("profile %q: missing required field %q", path, "display_name")
	case len(c.Obligations.Critical) == 0:
		return fmt.Errorf("profile %q: at least one critical obligation is required", path)
	}

	for i, obligation := range c.Obligations.Critical {
		if strings.TrimSpace(obligation.EventType) == "" {
			return fmt.Errorf("profile %q: critical obligation %d missing required field %q", path, i, "event_type")
		}
	}
	for i, obligation := range c.Obligations.Required {
		if strings.TrimSpace(obligation.EventType) == "" {
			return fmt.Errorf("profile %q: required obligation %d missing required field %q", path, i, "event_type")
		}
	}
	for i, relation := range c.Relations {
		switch {
		case strings.TrimSpace(relation.From) == "":
			return fmt.Errorf("profile %q: relation %d missing required field %q", path, i, "from")
		case strings.TrimSpace(relation.To) == "":
			return fmt.Errorf("profile %q: relation %d missing required field %q", path, i, "to")
		}
	}

	return nil
}

func (c *profileFileConfig) normalise() {
	c.ID = strings.TrimSpace(c.ID)
	c.DisplayName = strings.TrimSpace(c.DisplayName)
	c.Description = strings.TrimSpace(c.Description)

	for i := range c.Detect.EventTypes {
		c.Detect.EventTypes[i] = strings.TrimSpace(c.Detect.EventTypes[i])
	}
	for i := range c.Obligations.Critical {
		c.Obligations.Critical[i].EventType = strings.TrimSpace(c.Obligations.Critical[i].EventType)
		c.Obligations.Critical[i].Message = strings.TrimSpace(c.Obligations.Critical[i].Message)
	}
	for i := range c.Obligations.Required {
		c.Obligations.Required[i].EventType = strings.TrimSpace(c.Obligations.Required[i].EventType)
		c.Obligations.Required[i].Message = strings.TrimSpace(c.Obligations.Required[i].Message)
	}
	for i := range c.Relations {
		c.Relations[i].From = strings.TrimSpace(c.Relations[i].From)
		c.Relations[i].To = strings.TrimSpace(c.Relations[i].To)
		c.Relations[i].Message = strings.TrimSpace(c.Relations[i].Message)
	}
}

func (p *externalProfile) ID() string { return p.config.ID }

func (p *externalProfile) Version() int { return 1 }

func (p *externalProfile) WorkflowClass() string { return p.config.DisplayName }

func (p *externalProfile) Evaluate(records []bundle.Record) ProfileResult {
	result := ProfileResult{
		ProfileID:          p.ID(),
		Version:            p.Version(),
		WorkflowClass:      p.WorkflowClass(),
		CriticalFailures:   []CriticalFailure{},
		RequiredWarnings:   []string{},
		InformationalNotes: []string{},
	}

	recordsByType := indexRecordsByType(records)
	for _, obligation := range p.config.Obligations.Critical {
		if len(recordsByType[obligation.EventType]) > 0 {
			continue
		}
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind:   "missing_event",
			Detail: obligationMessage(obligation.EventType, obligation.Message),
		})
	}
	for _, obligation := range p.config.Obligations.Required {
		if len(recordsByType[obligation.EventType]) > 0 {
			continue
		}
		result.RequiredWarnings = append(result.RequiredWarnings, obligationMessage(obligation.EventType, obligation.Message))
	}
	for _, relation := range p.config.Relations {
		if relationSatisfied(recordsByType[relation.From], recordsByType[relation.To]) {
			continue
		}
		result.CriticalFailures = append(result.CriticalFailures, CriticalFailure{
			Kind:   "relation_violation",
			Detail: relationMessage(relation),
		})
	}
	if p.config.Description != "" {
		result.InformationalNotes = append(result.InformationalNotes, p.config.Description)
	}

	finaliseProfileResult(&result)
	return result
}

func (p *externalProfile) BlindSpots() []string {
	return nil
}

func (p *externalProfile) DefaultWeights() map[string]float64 {
	return map[string]float64{
		"EC": 0,
		"FC": 0,
		"RC": 0,
		"TC": 0,
		"SC": 0,
		"XC": 0,
		"AC": 0,
		"GC": 0,
	}
}

func (p *externalSchemaProfile) ID() string { return p.schema.ID }

func (p *externalSchemaProfile) Version() int { return p.schema.Version }

func (p *externalSchemaProfile) WorkflowClass() string { return p.schema.WorkflowClass }

func (p *externalSchemaProfile) Evaluate(records []bundle.Record) ProfileResult {
	return evaluateSchemaProfile(p.schema, records)
}

func (p *externalSchemaProfile) BlindSpots() []string {
	return copyStringSlice(p.schema.BlindSpots)
}

func (p *externalSchemaProfile) DefaultWeights() map[string]float64 {
	return copyFloatMap(p.schema.Weights)
}

// profileSchema exposes the underlying ProfileSchema for CAS-aware callers.
// It satisfies the profileWithSchema interface defined in the verify package.
func (p *externalSchemaProfile) profileSchema() profiledsl.ProfileSchema {
	return p.schema
}

func obligationMessage(eventType, message string) string {
	if message != "" {
		return message
	}
	return fmt.Sprintf("%s missing", eventType)
}

func relationMessage(relation profileRelationConfig) string {
	if relation.Message != "" {
		return relation.Message
	}
	return fmt.Sprintf("%s must precede %s", relation.From, relation.To)
}

func relationSatisfied(fromRecords, toRecords []bundle.Record) bool {
	if len(toRecords) == 0 {
		return true
	}
	if len(fromRecords) == 0 {
		return false
	}

	fromSequences := make([]int, 0, len(fromRecords))
	for _, record := range fromRecords {
		fromSequences = append(fromSequences, record.Event.Sequence)
	}
	sort.Ints(fromSequences)

	for _, toRecord := range toRecords {
		matched := false
		for _, fromSequence := range fromSequences {
			if fromSequence < toRecord.Event.Sequence {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}
