package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configDir = ".config/bookleaf"
const configFile = "config.json"
const defaultApiUrl = "https://bookleaf-assignment-atharva.vercel.app"

type Auth struct {
	AccessToken string `json:"accessToken"`
	UserID      string `json:"userId"`
	Role        string `json:"role"`
	Email       string `json:"email,omitempty"`
}

type Config struct {
	Auth    *Auth  `json:"auth"`
	APIURL  string `json:"apiUrl"`
	UseJSON bool   `json:"-"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	dir := filepath.Join(home, configDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("mkdir config: %w", err)
	}
	return filepath.Join(dir, configFile), nil
}

func Load() (*Config, error) {
	cfg := &Config{
		APIURL: defaultApiUrl,
	}

	path, err := configPath()
	if err != nil {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	if envURL := os.Getenv("BOOKLEAF_API_URL"); envURL != "" {
		cfg.APIURL = envURL
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func ClearAuth() error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	cfg.Auth = nil
	return Save(cfg)
}

func Path() (string, error) {
	return configPath()
}
