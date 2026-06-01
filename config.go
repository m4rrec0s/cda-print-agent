package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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
