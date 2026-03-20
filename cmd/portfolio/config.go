package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const ConfigFileName = ".portfolio.json"

type Address struct {
	Chain   string `json:"chain"`
	Address string `json:"address"`
}

type Config struct {
	Addresses          []Address `json:"addresses"`
	AnkrAPIKey         string    `json:"ankr_api_key,omitempty"`
	HeliusAPIKey       string    `json:"helius_api_key,omitempty"`
	NearIntentsAccount string    `json:"near_intents_account,omitempty"`
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ConfigFileName
	}
	return filepath.Join(home, ConfigFileName)
}

func loadConfig() *Config {
	return loadConfigFromPath(configPath())
}

func loadConfigFromPath(path string) *Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return &Config{}
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not parse %s: %v\n", path, err)
		return &Config{}
	}
	return &cfg
}

func saveConfig(cfg *Config) error {
	return saveConfigToPath(cfg, configPath())
}

func saveConfigToPath(cfg *Config, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func resolveAnkrAPIKey(cfg *Config) string {
	if env := os.Getenv("ANKR_API_KEY"); env != "" {
		return env
	}
	return cfg.AnkrAPIKey
}

func resolveHeliusAPIKey(cfg *Config) string {
	if env := os.Getenv("HELIUS_API_KEY"); env != "" {
		return env
	}
	return cfg.HeliusAPIKey
}
