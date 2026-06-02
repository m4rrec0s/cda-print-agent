package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type AgentConfig struct {
	WSURL         string `json:"wsUrl"`
	APIURL        string `json:"apiUrl"`
	AgentKey      string `json:"agentKey"`
	HotFolderPath string `json:"hotFolderPath"`
}

func getConfigDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData, _ = os.UserConfigDir()
	}
	dir := filepath.Join(appData, "CestoDAmore")
	os.MkdirAll(dir, 0700)
	return dir
}

func getConfigPath() string {
	return filepath.Join(getConfigDir(), "config.json")
}

func LoadConfigFromFile() (*AgentConfig, error) {
	data, err := os.ReadFile(getConfigPath())
	if err != nil {
		return nil, err
	}
	var cfg AgentConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveConfigFile(cfg AgentConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getConfigPath(), data, 0600)
}

func ConfigExists() bool {
	_, err := os.Stat(getConfigPath())
	return err == nil
}

func ValidateConfig(cfg *AgentConfig) error {
	if cfg.WSURL != "" {
		if !strings.HasPrefix(cfg.WSURL, "ws://") && !strings.HasPrefix(cfg.WSURL, "wss://") {
			return fmt.Errorf("wsUrl invalido: deve comecar com ws:// ou wss:// (valor atual: %q)", cfg.WSURL)
		}
		if _, err := url.Parse(cfg.WSURL); err != nil {
			return fmt.Errorf("wsUrl invalido: %w", err)
		}
	}

	if cfg.APIURL != "" {
		if _, err := url.Parse(cfg.APIURL); err != nil {
			return fmt.Errorf("apiUrl invalido: %w", err)
		}
		parsed, _ := url.Parse(cfg.APIURL)
		if parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("apiUrl invalido: deve ser uma URL completa (ex: https://api.cestodamore.com.br), valor atual: %q", cfg.APIURL)
		}
	}

	return nil
}
