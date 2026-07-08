// SPDX-License-Identifier: MIT
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/retentionaudit"
)

const (
	configVersion           = 1
	configDirName           = ".atb"
	configFileName          = "config.json"
	retentionDefaultArchive = "archive.atb"
	retentionDefaultCutoff  = "file_mtime"

	// EUAIActRetentionMinDays is 6 months, conservative (non-leap), for EU AI Act Article 19 retention checks.
	EUAIActRetentionMinDays = 183
)

type atbConfig struct {
	Version   int              `json:"version"`
	Retention *retentionPolicy `json:"retention,omitempty"`
	Push      *pushSettings    `json:"push,omitempty"`
}

type retentionPolicy struct {
	Days        int      `json:"days"`
	ArchiveDir  string   `json:"archive_dir,omitempty"`
	Scope       []string `json:"scope,omitempty"`
	CutoffBasis string   `json:"cutoff_basis,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
}

type pushSettings struct {
	Target            string `json:"target,omitempty"`
	EndpointURL       string `json:"endpoint_url,omitempty"`
	Region            string `json:"region,omitempty"`
	LockMode          string `json:"lock_mode,omitempty"`
	LockUntil         string `json:"lock_until,omitempty"`
	CredentialsSource string `json:"credentials_source,omitempty"`
}

type configCommandArgs struct {
	Subcommand                 string
	Days                       int
	AllowBelowStatutoryMinimum bool
}

func cmdConfig() {
	os.Exit(runConfig(os.Args[2:], os.Stdout, os.Stderr))
}

func runConfig(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseConfigArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "atb config: %v\n", err)
		printConfigUsage(stdout)
		return exitUserError
	}

	if cfg.Subcommand != "retention" {
		fmt.Fprintf(stderr, "atb config: unsupported subcommand %q\n", cfg.Subcommand)
		printConfigUsage(stdout)
		return exitUserError
	}

	if cfg.Days < EUAIActRetentionMinDays {
		if !cfg.AllowBelowStatutoryMinimum {
			fmt.Fprintf(stderr, "atb config: retention period %d days is below the EU AI Act Article 19 minimum of %d days (approximately 6 months). To suppress this error and accept compliance responsibility, pass --allow-below-eu-minimum.\n", cfg.Days, EUAIActRetentionMinDays)
			return exitUserError
		}
		fmt.Fprintf(stderr, "WARNING: retention period %d days is below the EU AI Act Article 19 minimum of %d days. Operator has accepted compliance responsibility.\n", cfg.Days, EUAIActRetentionMinDays)
	}

	configPath := defaultConfigPath()
	existing, err := loadATBConfig(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(stderr, "atb config: load config: %v\n", err)
		return exitSystemError
	}
	if errors.Is(err, os.ErrNotExist) {
		existing = atbConfig{Version: configVersion}
	}
	if existing.Version == 0 {
		existing.Version = configVersion
	}
	now := time.Now().UTC()
	previousPolicy := existing.Retention
	nextPolicy := defaultRetentionPolicy(cfg.Days, now)
	nextDigest, err := retentionaudit.Digest(nextPolicy)
	if err != nil {
		fmt.Fprintf(stderr, "atb config: prepare retention audit: %v\n", err)
		return exitSystemError
	}
	eventType := event.TypeDataRetentionPolicySet
	auditData := map[string]any{
		"policy_id":                   "local.default",
		"days":                        nextPolicy.Days,
		"archive_dir":                 nextPolicy.ArchiveDir,
		"scope":                       nextPolicy.Scope,
		"cutoff_basis":                nextPolicy.CutoffBasis,
		"config_digest":               nextDigest,
		"accepted_below_eu_minimum":   cfg.Days < EUAIActRetentionMinDays,
		"atb_enforces_storage_policy": false,
	}
	if previousPolicy != nil {
		previousDigest, digestErr := retentionaudit.Digest(previousPolicy)
		if digestErr != nil {
			fmt.Fprintf(stderr, "atb config: prepare previous retention audit: %v\n", digestErr)
			return exitSystemError
		}
		eventType = event.TypeDataRetentionPolicyChanged
		auditData["previous_config_digest"] = previousDigest
	}
	existing.Retention = nextPolicy

	previousRaw, readErr := os.ReadFile(configPath) // #nosec G304 -- config path is project-local and controlled by the CLI
	configExisted := readErr == nil

	if err := saveATBConfig(configPath, existing); err != nil {
		fmt.Fprintf(stderr, "atb config: save config: %v\n", err)
		return exitSystemError
	}
	if err := retentionaudit.Append(retentionaudit.DefaultPath(), eventType, auditData, now); err != nil {
		fmt.Fprintf(stderr, "atb config: retention audit logging failed: %v\n", err)
		if rollbackErr := rollbackConfig(configPath, previousRaw, configExisted); rollbackErr != nil {
			fmt.Fprintf(stderr, "atb config: rollback failed: %v; retention policy in %s is saved without an audit event\n", rollbackErr, configPath)
		} else {
			fmt.Fprintf(stderr, "atb config: previous config restored; retention policy was not applied\n")
		}
		return exitSystemError
	}

	fmt.Fprintf(stdout, "✓ Retention set: %d day(s) in %s\n", cfg.Days, configPath)
	fmt.Fprintf(stdout, "✓ Retention audit: %s\n", retentionaudit.DefaultPath())
	return exitSuccess
}

func parseConfigArgs(args []string) (configCommandArgs, error) {
	cfg := configCommandArgs{}
	if len(args) == 0 {
		return cfg, fmt.Errorf("usage: atb config retention --days <n>")
	}
	cfg.Subcommand = strings.ToLower(strings.TrimSpace(args[0]))
	if cfg.Subcommand != "retention" {
		return cfg, fmt.Errorf("unknown config subcommand %q", args[0])
	}

	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--days":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --days")
			}
			days, err := parsePositiveDays(args[i+1])
			if err != nil {
				return cfg, err
			}
			cfg.Days = days
			i++
		case strings.HasPrefix(arg, "--days="):
			days, err := parsePositiveDays(strings.TrimPrefix(arg, "--days="))
			if err != nil {
				return cfg, err
			}
			cfg.Days = days
		case arg == "--allow-below-eu-minimum":
			cfg.AllowBelowStatutoryMinimum = true
		case strings.HasPrefix(arg, "--"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			return cfg, fmt.Errorf("unexpected argument %q", arg)
		}
	}

	if cfg.Days <= 0 {
		return cfg, fmt.Errorf("--days must be > 0")
	}

	return cfg, nil
}

func parsePositiveDays(raw string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid --days value %q", raw)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("--days must be > 0")
	}
	return parsed, nil
}

func defaultConfigPath() string {
	return filepath.Join(configDirName, configFileName)
}

func defaultRetentionPolicy(days int, now time.Time) *retentionPolicy {
	return &retentionPolicy{
		Days:        days,
		ArchiveDir:  retentionDefaultArchive,
		Scope:       []string{"run.atb/*.atb"},
		CutoffBasis: retentionDefaultCutoff,
		UpdatedAt:   now.UTC().Format(time.RFC3339),
	}
}

func loadRetentionPolicy(configPath string) (*retentionPolicy, error) {
	cfg, err := loadATBConfig(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if cfg.Retention == nil {
		return nil, nil
	}
	return cfg.Retention, nil
}

func loadPushSettings(configPath string) (*pushSettings, error) {
	cfg, err := loadATBConfig(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if cfg.Push == nil {
		return nil, nil
	}
	return cfg.Push, nil
}

func loadATBConfig(path string) (atbConfig, error) {
	var cfg atbConfig
	raw, err := os.ReadFile(path) // #nosec G304 -- config path is project-local and controlled by the CLI
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Version == 0 {
		cfg.Version = configVersion
	}
	return cfg, nil
}

func saveATBConfig(path string, cfg atbConfig) error {
	if cfg.Version == 0 {
		cfg.Version = configVersion
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil { // #nosec G301 -- project-local config dir
		return fmt.Errorf("mkdir config dir: %w", err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// rollbackConfig undoes a config save whose paired audit append failed, so a
// retention policy is never committed without its audit event.
func rollbackConfig(path string, previousRaw []byte, existed bool) error {
	if !existed {
		return os.Remove(path)
	}
	return os.WriteFile(path, previousRaw, 0600) // #nosec G306 G703 -- restores the project-local config file this command just overwrote
}

func printConfigUsage(stdout io.Writer) {
	fmt.Fprintln(stdout, "Usage: atb config retention --days <n> [--allow-below-eu-minimum]")
}
