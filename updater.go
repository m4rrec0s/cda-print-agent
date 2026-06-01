package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/minio/selfupdate"
)

type VersionInfo struct {
	Version      string `json:"version"`
	DownloadURL  string `json:"downloadUrl"`
	ReleaseNotes string `json:"releaseNotes"`
}

func CheckUpdate(apiURL string, agentKey string, currentVersion string) (*VersionInfo, error) {
	if apiURL == "" {
		return nil, nil
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, apiURL+"/api/agent/version", nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao montar requisicao de versao: %w", err)
	}
	if agentKey != "" {
		req.Header.Set("X-API-Key", agentKey)
		req.Header.Set("X-Agent-Key", agentKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao verificar versao: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("backend retornou HTTP %d", resp.StatusCode)
	}

	var info VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	if info.Version == "" || info.DownloadURL == "" || info.Version == currentVersion {
		return nil, nil
	}

	return &info, nil
}

func ApplyUpdate(downloadURL string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download falhou: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download retornou HTTP %d", resp.StatusCode)
	}

	if err := selfupdate.Apply(resp.Body, selfupdate.Options{}); err != nil {
		return fmt.Errorf("falha ao aplicar update: %w", err)
	}

	log.Printf("event=update_applied download_url=%s", downloadURL)
	return nil
}

func RestartApp() error {
	cmd := newHiddenCommand(os.Args[0], os.Args[1:]...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("falha ao reiniciar: %w", err)
	}
	return nil
}
