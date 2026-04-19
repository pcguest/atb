package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	profiledsl "github.com/pcguest/atb/internal/profiles"
	verifypkg "github.com/pcguest/atb/internal/verify"
	"gopkg.in/yaml.v3"
)

var errProfilesHelp = errors.New("profiles help requested")

type profilesValidateConfig struct {
	Files  []string
	Dirs   []string
	Format string
}

// profileValidationReport is the stable JSON schema for `atb profiles validate`.
// It is a top-level object keyed by profile ID. When a file cannot provide an
// ID, the cleaned file path is used as the key. Each value contains `id`,
// `valid`, and `errors`.
type profileValidationReport map[string]profileValidationEntry

type profileValidationEntry struct {
	ID     string   `json:"id"`
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
}

type profileValidationKind int

const (
	profileValidationKindUnknown profileValidationKind = iota
	profileValidationKindSchema
	profileValidationKindLegacy
	profileValidationKindDSL
)

func cmdProfiles() {
	os.Exit(runProfiles(os.Args[2:], os.Stdout, os.Stderr))
}

func runProfiles(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printProfilesUsage(stderr)
		return exitUserError
	}

	switch args[0] {
	case "-h", "--help":
		printProfilesUsage(stdout)
		return exitSuccess
	case "validate":
		return runProfilesValidate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "atb profiles: unknown subcommand %q\n", args[0])
		printProfilesUsage(stderr)
		return exitUserError
	}
}

func runProfilesValidate(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseProfilesValidateArgs(args)
	if err != nil {
		if errors.Is(err, errProfilesHelp) {
			printProfilesValidateUsage(stdout)
			return exitSuccess
		}
		fmt.Fprintf(stderr, "atb profiles validate: %v\n", err)
		printProfilesValidateUsage(stderr)
		return exitUserError
	}

	report, err := buildProfileValidationReport(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "atb profiles validate: %v\n", err)
		return exitUserError
	}

	if cfg.Format == verifyFormatJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(stderr, "atb profiles validate: encode json output: %v\n", err)
			return exitSystemError
		}
	} else {
		writeProfileValidationText(stdout, report)
	}

	if profileValidationSucceeded(report) {
		return exitSuccess
	}
	return exitUserError
}

func parseProfilesValidateArgs(args []string) (profilesValidateConfig, error) {
	cfg := profilesValidateConfig{Format: verifyFormatText}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errProfilesHelp
		case arg == "--file":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --file")
			}
			i++
			cfg.Files = append(cfg.Files, filepath.Clean(args[i]))
		case strings.HasPrefix(arg, "--file="):
			cfg.Files = append(cfg.Files, filepath.Clean(strings.TrimPrefix(arg, "--file=")))
		case arg == "--dir":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --dir")
			}
			i++
			cfg.Dirs = append(cfg.Dirs, filepath.Clean(args[i]))
		case strings.HasPrefix(arg, "--dir="):
			cfg.Dirs = append(cfg.Dirs, filepath.Clean(strings.TrimPrefix(arg, "--dir=")))
		case arg == "--format" || arg == "-f":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for %s", arg)
			}
			i++
			cfg.Format = strings.ToLower(strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "--format="):
			cfg.Format = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--format=")))
		case strings.HasPrefix(arg, "-"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			return cfg, fmt.Errorf("unexpected argument %q", arg)
		}
	}

	if cfg.Format != verifyFormatText && cfg.Format != verifyFormatJSON {
		return cfg, fmt.Errorf("invalid format %q (expected text|json)", cfg.Format)
	}
	return cfg, nil
}

func printProfilesUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb profiles <subcommand>")
	fmt.Fprintln(w, "Subcommands:")
	fmt.Fprintln(w, "  validate  Validate built-in and supplied profile definitions")
}

func printProfilesValidateUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb profiles validate [--file <path>] [--dir <path>] [--format text|json]")
	fmt.Fprintln(w, "  --file <path>  validate one additional profile definition")
	fmt.Fprintln(w, "  --dir <path>   validate all *.yaml and *.yml files in a directory")
}

func buildProfileValidationReport(cfg profilesValidateConfig) (profileValidationReport, error) {
	report := make(profileValidationReport)
	sources := make(map[string][]string)

	for _, profile := range verifypkg.AllProfiles() {
		schema, err := profiledsl.LoadSchema(profile.ID())
		entry := profileValidationEntry{ID: profile.ID(), Errors: []string{}}
		if err != nil {
			entry.Errors = append(entry.Errors, err.Error())
		} else {
			entry.Errors = append(entry.Errors, validateSchemaDefinition(schema, true)...)
		}
		report[profile.ID()] = finaliseProfileValidationEntry(report[profile.ID()], entry)
		sources[profile.ID()] = append(sources[profile.ID()], "built-in")
	}

	paths, err := collectProfileValidationPaths(cfg)
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		key, entry, err := validateProfilePath(path)
		if err != nil {
			return nil, err
		}
		report[key] = finaliseProfileValidationEntry(report[key], entry)
		sources[key] = append(sources[key], path)
	}

	for key, entry := range report {
		if len(sources[key]) > 1 {
			entry.Errors = append(entry.Errors, fmt.Sprintf("duplicate profile ID %q", key))
		}
		sort.Strings(entry.Errors)
		entry.Valid = len(entry.Errors) == 0
		report[key] = entry
	}

	return report, nil
}

func collectProfileValidationPaths(cfg profilesValidateConfig) ([]string, error) {
	paths := make([]string, 0, len(cfg.Files))
	paths = append(paths, cfg.Files...)

	for _, dir := range cfg.Dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext != ".yaml" && ext != ".yml" {
				continue
			}
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}

	sort.Strings(paths)
	return paths, nil
}

func validateProfilePath(path string) (string, profileValidationEntry, error) {
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", profileValidationEntry{}, err
	}

	key := extractProfileValidationID(content)
	if key == "" {
		key = filepath.Clean(path)
	}
	entry := profileValidationEntry{ID: extractProfileValidationID(content), Errors: []string{}}

	switch detectProfileValidationKind(content) {
	case profileValidationKindDSL:
		schema, err := profiledsl.ParseDSLProfile(content)
		if err != nil {
			entry.Errors = append(entry.Errors, err.Error())
			return key, entry, nil
		}
		entry.ID = schema.ID
		return schema.ID, profileValidationEntry{
			ID:     schema.ID,
			Errors: validateSchemaDefinition(schema, false),
		}, nil
	case profileValidationKindSchema:
		schema, err := profiledsl.ParseSchema(content)
		if err != nil {
			entry.Errors = append(entry.Errors, err.Error())
			return key, entry, nil
		}
		entry.ID = schema.ID
		return schema.ID, profileValidationEntry{
			ID:     schema.ID,
			Errors: validateSchemaDefinition(schema, true),
		}, nil
	default:
		profile, err := verifypkg.ResolveProfile(path)
		if err != nil {
			entry.Errors = append(entry.Errors, err.Error())
			return key, entry, nil
		}
		entry.ID = profile.ID()
		return profile.ID(), profileValidationEntry{
			ID:     profile.ID(),
			Errors: validateLoadedProfile(profile),
		}, nil
	}
}

func finaliseProfileValidationEntry(dst, src profileValidationEntry) profileValidationEntry {
	if dst.ID == "" {
		dst.ID = src.ID
	}
	dst.Errors = append(dst.Errors, src.Errors...)
	return dst
}

func detectProfileValidationKind(content []byte) profileValidationKind {
	var raw map[string]any
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return profileValidationKindUnknown
	}

	switch {
	case hasValidationKeys(raw, "required_events", "warning_events", "cas_weights"):
		return profileValidationKindDSL
	case hasValidationKeys(raw, "workflow_class", "required", "optional", "relations", "blind_spots", "weights", "supports_cas", "sc_mode"):
		return profileValidationKindSchema
	case hasValidationKeys(raw, "display_name", "detect", "obligations"):
		return profileValidationKindLegacy
	default:
		return profileValidationKindUnknown
	}
}

func hasValidationKeys(values map[string]any, keys ...string) bool {
	for _, key := range keys {
		if _, ok := values[key]; ok {
			return true
		}
	}
	return false
}

func extractProfileValidationID(content []byte) string {
	var raw struct {
		ID string `yaml:"id"`
	}
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return ""
	}
	return strings.TrimSpace(raw.ID)
}

func validateSchemaDefinition(schema profiledsl.ProfileSchema, requireBlindSpots bool) []string {
	errors := make([]string, 0)
	if strings.TrimSpace(schema.ID) == "" {
		errors = append(errors, "id is required")
	}
	if schema.Version < 1 {
		errors = append(errors, "version must be >= 1")
	}
	if strings.TrimSpace(schema.WorkflowClass) == "" {
		errors = append(errors, "workflow_class is required")
	}
	if len(schema.Required)+len(schema.Optional)+len(schema.Relations) == 0 {
		errors = append(errors, "at least one obligation is required")
	}
	if requireBlindSpots && len(schema.BlindSpots) == 0 {
		errors = append(errors, "at least one exclusion is required")
	}
	if schema.SupportsCAS {
		errors = append(errors, validateWeightVector(schema.ID, schema.Weights)...)
	}
	return errors
}

func validateLoadedProfile(profile verifypkg.Profile) []string {
	if profile == nil {
		return []string{"profile failed to load"}
	}

	errors := make([]string, 0)
	if strings.TrimSpace(profile.ID()) == "" {
		errors = append(errors, "id is required")
	}
	if profile.Version() < 1 {
		errors = append(errors, "version must be >= 1")
	}
	if strings.TrimSpace(profile.WorkflowClass()) == "" {
		errors = append(errors, "workflow_class is required")
	}

	result := profile.Evaluate(nil)
	if len(result.CriticalFailures)+len(result.RequiredWarnings) == 0 {
		errors = append(errors, "at least one obligation is required")
	}

	if weightSum(profile.DefaultWeights()) > 0 {
		errors = append(errors, validateWeightVector(profile.ID(), profile.DefaultWeights())...)
	}
	return errors
}

func validateWeightVector(id string, weights map[string]float64) []string {
	const tolerance = 1e-9

	expected := []string{"AC", "EC", "FC", "GC", "RC", "SC", "TC", "XC"}
	errors := make([]string, 0)
	sum := 0.0

	for _, key := range expected {
		value, ok := weights[key]
		if !ok {
			errors = append(errors, fmt.Sprintf("profile %q: missing CAS weight key %q", id, key))
			continue
		}
		sum += value
	}
	for key := range weights {
		if !containsValidationKey(expected, key) {
			errors = append(errors, fmt.Sprintf("profile %q: unknown CAS weight key %q", id, key))
		}
	}
	if len(errors) > 0 {
		return errors
	}
	if math.Abs(sum-1.0) > tolerance {
		errors = append(errors, fmt.Sprintf("profile %q: CAS weights sum to %.12f, want 1.0", id, sum))
	}
	return errors
}

func weightSum(weights map[string]float64) float64 {
	total := 0.0
	for _, value := range weights {
		total += value
	}
	return total
}

func containsValidationKey(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func writeProfileValidationText(w io.Writer, report profileValidationReport) {
	keys := make([]string, 0, len(report))
	for key := range report {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		entry := report[key]
		if entry.Valid {
			fmt.Fprintf(w, "OK %s\n", key)
			continue
		}
		fmt.Fprintf(w, "FAILED %s: %s\n", key, strings.Join(entry.Errors, "; "))
	}
}

func profileValidationSucceeded(report profileValidationReport) bool {
	for _, entry := range report {
		if !entry.Valid {
			return false
		}
	}
	return true
}
