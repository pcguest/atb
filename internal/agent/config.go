// SPDX-License-Identifier: MIT
package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultListenAddr = "127.0.0.1:6180"
	legacyDataDir     = "./data/agent"
	configFileName    = "config.json"
	configDirName     = ".atb"
)

// Config holds runtime settings for the local ATB Agent process.
type Config struct {
	ListenAddr string
	DataDir    string
	Version    string
}

type sharedConfigFile struct {
	Agent *agentSettingsFile `json:"agent,omitempty"`
}

type agentSettingsFile struct {
	ListenAddr string `json:"listen_addr,omitempty"`
	DataDir    string `json:"data_dir,omitempty"`
}

// DefaultConfigPath returns the preferred user-level ATB config file path.
func DefaultConfigPath(homeDir string) string {
	if home := strings.TrimSpace(homeDir); home != "" {
		return filepath.Join(home, configDirName, configFileName)
	}
	return filepath.Join(configDirName, configFileName)
}

// DefaultDataDir returns the default agent workspace root when no override is set.
func DefaultDataDir(homeDir string) string {
	if home := strings.TrimSpace(homeDir); home != "" {
		return filepath.Join(home, configDirName, "agent")
	}
	return legacyDataDir
}

// LoadConfig reads agent settings from the process environment and optional config files.
func LoadConfig(version string) (Config, error) {
	return LoadConfigFromEnv(version, os.Getenv)
}

// LoadConfigFromEnv parses agent settings using the supplied environment lookup.
// Priority: environment variables, then config file (user home, then cwd), then defaults.
func LoadConfigFromEnv(version string, getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	if version == "" {
		return Config{}, fmt.Errorf("version is required")
	}

	homeDir := strings.TrimSpace(getenv("HOME"))
	fileCfg := loadAgentSettingsFromFiles(homeDir, getenv)

	listenAddr := defaultListenAddr
	if fileCfg != nil && strings.TrimSpace(fileCfg.ListenAddr) != "" {
		listenAddr = strings.TrimSpace(fileCfg.ListenAddr)
	}
	if v := strings.TrimSpace(getenv("ATB_AGENT_LISTEN_ADDR")); v != "" {
		listenAddr = v
	}

	dataDir := DefaultDataDir(homeDir)
	if fileCfg != nil && strings.TrimSpace(fileCfg.DataDir) != "" {
		dataDir = strings.TrimSpace(fileCfg.DataDir)
	}
	if v := strings.TrimSpace(getenv("ATB_AGENT_DATA_DIR")); v != "" {
		dataDir = v
	}

	return Config{
		ListenAddr: listenAddr,
		DataDir:    dataDir,
		Version:    version,
	}, nil
}

func loadAgentSettingsFromFiles(homeDir string, getenv func(string) string) *agentSettingsFile {
	paths := agentConfigFilePaths(homeDir)
	for _, path := range paths {
		cfg, err := readAgentSettingsFile(path)
		if err == nil && cfg != nil {
			return cfg
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			continue
		}
	}
	_ = getenv
	return nil
}

func agentConfigFilePaths(homeDir string) []string {
	paths := make([]string, 0, 2)
	if home := strings.TrimSpace(homeDir); home != "" {
		paths = append(paths, filepath.Join(home, configDirName, configFileName))
	}
	paths = append(paths, filepath.Join(configDirName, configFileName))
	return paths
}

func readAgentSettingsFile(path string) (*agentSettingsFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg sharedConfigFile
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if cfg.Agent == nil {
		return nil, nil
	}
	return cfg.Agent, nil
}
