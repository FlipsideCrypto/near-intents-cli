package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".near-intents.json")
	os.WriteFile(path, []byte(`{"token":"file-token","api_endpoint":"https://custom.api"}`), 0644)

	cfg := loadConfigFromPath(path)
	if cfg.Token != "file-token" {
		t.Errorf("expected file-token, got %s", cfg.Token)
	}
	if cfg.APIEndpoint != "https://custom.api" {
		t.Errorf("expected custom endpoint, got %s", cfg.APIEndpoint)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	cfg := loadConfigFromPath("/nonexistent/path.json")
	if cfg.Token != "" {
		t.Error("expected empty token for missing file")
	}
	if cfg.APIEndpoint != DefaultAPIEndpoint {
		t.Errorf("expected default endpoint, got %s", cfg.APIEndpoint)
	}
}

func TestResolveTokenPrecedence(t *testing.T) {
	cfg := &Config{Token: "file-token"}

	// Env var overrides file
	os.Setenv("NEAR_INTENTS_JWT_TOKEN", "env-token")
	defer os.Unsetenv("NEAR_INTENTS_JWT_TOKEN")

	token := resolveToken("", cfg)
	if token != "env-token" {
		t.Errorf("expected env-token, got %s", token)
	}

	// Flag overrides env
	token = resolveToken("flag-token", cfg)
	if token != "flag-token" {
		t.Errorf("expected flag-token, got %s", token)
	}
}
