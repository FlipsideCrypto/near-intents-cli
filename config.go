package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	DefaultAPIEndpoint = "https://1click.chaindefuser.com"
	DefaultSigningURL  = "https://swap.flipsidecrypto.xyz"
	ConfigFileName     = ".near-intents.json"
)

type Config struct {
	Token          string `json:"token,omitempty"`
	APIEndpoint    string `json:"api_endpoint,omitempty"`
	SigningBaseURL string `json:"signing_base_url,omitempty"`
}

func loadConfig() *Config {
	home, err := os.UserHomeDir()
	if err != nil {
		return defaultConfig()
	}
	return loadConfigFromPath(filepath.Join(home, ConfigFileName))
}

func loadConfigFromPath(path string) *Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultConfig()
	}
	cfg := defaultConfig()
	json.Unmarshal(data, cfg)
	if cfg.APIEndpoint == "" {
		cfg.APIEndpoint = DefaultAPIEndpoint
	}
	if cfg.SigningBaseURL == "" {
		cfg.SigningBaseURL = DefaultSigningURL
	}
	return cfg
}

func defaultConfig() *Config {
	return &Config{
		APIEndpoint:    DefaultAPIEndpoint,
		SigningBaseURL: DefaultSigningURL,
	}
}

func resolveToken(flagValue string, cfg *Config) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv("NEAR_INTENTS_JWT_TOKEN"); env != "" {
		return env
	}
	return cfg.Token
}
