// SPDX-License-Identifier: MIT
package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromEnvDefaults(t *testing.T) {
	cfg, err := LoadConfigFromEnv("1.11.0", func(string) string { return "" })
	if err != nil {
		t.Fatalf("LoadConfigFromEnv: %v", err)
	}
	if cfg.ListenAddr != defaultListenAddr {
		t.Fatalf("ListenAddr = %q, want %q", cfg.ListenAddr, defaultListenAddr)
	}
	if cfg.DataDir != legacyDataDir {
		t.Fatalf("DataDir = %q, want %q", cfg.DataDir, legacyDataDir)
	}
	if cfg.Version != "1.11.0" {
		t.Fatalf("Version = %q, want %q", cfg.Version, "1.11.0")
	}
}

func TestLoadConfigFromEnvHomeDefaultDataDir(t *testing.T) {
	getenv := func(key string) string {
		if key == "HOME" {
			return "/home/tester"
		}
		return ""
	}
	cfg, err := LoadConfigFromEnv("1.11.0", getenv)
	if err != nil {
		t.Fatalf("LoadConfigFromEnv: %v", err)
	}
	want := filepath.Join("/home/tester", ".atb", "agent")
	if cfg.DataDir != want {
		t.Fatalf("DataDir = %q, want %q", cfg.DataDir, want)
	}
}

func TestLoadConfigFromEnvOverrides(t *testing.T) {
	env := map[string]string{
		"ATB_AGENT_LISTEN_ADDR": "127.0.0.1:7001",
		"ATB_AGENT_DATA_DIR":    "/tmp/atb-agent",
	}
	getenv := func(key string) string {
		return env[key]
	}

	cfg, err := LoadConfigFromEnv("1.11.0", getenv)
	if err != nil {
		t.Fatalf("LoadConfigFromEnv: %v", err)
	}
	if cfg.ListenAddr != "127.0.0.1:7001" {
		t.Fatalf("ListenAddr = %q, want %q", cfg.ListenAddr, "127.0.0.1:7001")
	}
	if cfg.DataDir != "/tmp/atb-agent" {
		t.Fatalf("DataDir = %q, want %q", cfg.DataDir, "/tmp/atb-agent")
	}
}

func TestLoadConfigFromEnvFileOverridesDefaults(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, ".atb", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0750); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	payload := []byte(`{
  "version": 1,
  "agent": {
    "listen_addr": "127.0.0.1:7002",
    "data_dir": "/tmp/from-config"
  }
}`)
	if err := os.WriteFile(configPath, payload, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cfg, err := LoadConfigFromEnv("1.11.0", func(string) string { return "" })
	if err != nil {
		t.Fatalf("LoadConfigFromEnv: %v", err)
	}
	if cfg.ListenAddr != "127.0.0.1:7002" {
		t.Fatalf("ListenAddr = %q, want file override", cfg.ListenAddr)
	}
	if cfg.DataDir != "/tmp/from-config" {
		t.Fatalf("DataDir = %q, want file override", cfg.DataDir)
	}
}

func TestLoadConfigFromEnvEnvBeatsFile(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, ".atb", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0750); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	payload := []byte(`{"agent":{"listen_addr":"127.0.0.1:7002","data_dir":"/tmp/from-config"}}`)
	if err := os.WriteFile(configPath, payload, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	env := map[string]string{
		"ATB_AGENT_LISTEN_ADDR": "127.0.0.1:7003",
		"ATB_AGENT_DATA_DIR":    "/tmp/from-env",
	}
	getenv := func(key string) string {
		return env[key]
	}

	cfg, err := LoadConfigFromEnv("1.11.0", getenv)
	if err != nil {
		t.Fatalf("LoadConfigFromEnv: %v", err)
	}
	if cfg.ListenAddr != "127.0.0.1:7003" {
		t.Fatalf("ListenAddr = %q, want env override", cfg.ListenAddr)
	}
	if cfg.DataDir != "/tmp/from-env" {
		t.Fatalf("DataDir = %q, want env override", cfg.DataDir)
	}
}

func TestLoadConfigFromEnvRequiresVersion(t *testing.T) {
	_, err := LoadConfigFromEnv("", func(string) string { return "" })
	if err == nil {
		t.Fatal("expected error for empty version")
	}
}

func TestDefaultConfigPath(t *testing.T) {
	want := filepath.Join("/home/tester", ".atb", "config.json")
	if got := DefaultConfigPath("/home/tester"); got != want {
		t.Fatalf("DefaultConfigPath = %q, want %q", got, want)
	}
}
