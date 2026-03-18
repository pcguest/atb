package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	configVersion           = 1
	configDirName           = ".atb"
	configFileName          = "config.json"
	retentionDefaultArchive = "archive.atb"
	retentionDefaultCutoff  = "file_mtime"
)

type atbConfig struct {
	Version   int              `json:"version"`
	Retention *retentionPolicy `json:"retention,omitempty"`
}

type retentionPolicy struct {
	Days        int      `json:"days"`
	ArchiveDir  string   `json:"archive_dir,omitempty"`
	Scope       []string `json:"scope,omitempty"`
	CutoffBasis string   `json:"cutoff_basis,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
}

type configCommandArgs struct {
	Subcommand string
	Days       int
}

func cmdConfig() {
	cfg, err := parseConfigArgs(os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb config: %v\n", err)
		printConfigUsage()
		os.Exit(exitUserError)
	}

	if cfg.Subcommand != "retention" {
		fmt.Fprintf(os.Stderr, "atb config: unsupported subcommand %q\n", cfg.Subcommand)
		printConfigUsage()
		os.Exit(exitUserError)
	}

	configPath := defaultConfigPath()
	existing, err := loadATBConfig(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "atb config: load config: %v\n", err)
		os.Exit(exitSystemError)
	}
	if errors.Is(err, os.ErrNotExist) {
		existing = atbConfig{Version: configVersion}
	}
	if existing.Version == 0 {
		existing.Version = configVersion
	}
	existing.Retention = defaultRetentionPolicy(cfg.Days, time.Now().UTC())

	if err := saveATBConfig(configPath, existing); err != nil {
		fmt.Fprintf(os.Stderr, "atb config: save config: %v\n", err)
		os.Exit(exitSystemError)
	}

	fmt.Printf("✓ Retention set: %d day(s) in %s\n", cfg.Days, configPath)
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

func printConfigUsage() {
	fmt.Println("Usage: atb config retention --days <n>")
}
