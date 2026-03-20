package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromPath_Empty(t *testing.T) {
	cfg := loadConfigFromPath("/nonexistent/path.json")
	if len(cfg.Addresses) != 0 {
		t.Error("expected empty addresses for missing config")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-config.json")

	cfg := &Config{
		Addresses: []Address{
			{Chain: "evm", Address: "0xabc"},
			{Chain: "near", Address: "alice.near"},
		},
		NearIntentsAccount: "alice.near",
	}

	if err := saveConfigToPath(cfg, path); err != nil {
		t.Fatalf("save config: %v", err)
	}

	loaded := loadConfigFromPath(path)
	if len(loaded.Addresses) != 2 {
		t.Fatalf("expected 2 addresses, got %d", len(loaded.Addresses))
	}
	if loaded.Addresses[0].Chain != "evm" {
		t.Errorf("expected chain evm, got %s", loaded.Addresses[0].Chain)
	}
	if loaded.NearIntentsAccount != "alice.near" {
		t.Errorf("expected near_intents_account alice.near, got %s", loaded.NearIntentsAccount)
	}
}

func TestResolveAnkrAPIKey_EnvVar(t *testing.T) {
	os.Setenv("ANKR_API_KEY", "test-key-from-env")
	defer os.Unsetenv("ANKR_API_KEY")

	cfg := &Config{AnkrAPIKey: "config-key"}
	key := resolveAnkrAPIKey(cfg)
	if key != "test-key-from-env" {
		t.Errorf("expected env var to take precedence, got %s", key)
	}
}
