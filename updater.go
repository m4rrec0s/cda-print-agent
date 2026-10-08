package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	parsedURL, err := url.Parse(downloadURL)
	if err != nil || parsedURL.Host == "" || !isSecureUpdateURL(parsedURL) {
		return fmt.Errorf("URL de atualização inválida ou insegura")
	}

	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("montar download de atualização: %w", err)
	}
	client := &http.Client{
		Timeout: 5 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !isSecureUpdateURL(req.URL) {
				return fmt.Errorf("redirecionamento de atualização para URL insegura")
			}
			if len(via) >= 10 {
				return fmt.Errorf("muitos redirecionamentos de atualização")
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download falhou: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download retornou HTTP %d", resp.StatusCode)
	}
	if !isSecureUpdateURL(resp.Request.URL) {
		return fmt.Errorf("download de atualização não usou conexão segura")
	}

	const maxUpdateSize = 200 << 20
	binary, err := io.ReadAll(io.LimitReader(resp.Body, maxUpdateSize+1))
	if err != nil {
		return fmt.Errorf("ler binário da atualização: %w", err)
	}
	if len(binary) > maxUpdateSize {
		return fmt.Errorf("binário de atualização excede limite de %d MiB", maxUpdateSize>>20)
	}
	if len(binary) == 0 {
		return fmt.Errorf("binário de atualização vazio")
	}

	if err := selfupdate.Apply(bytes.NewReader(binary), selfupdate.Options{}); err != nil {
		return fmt.Errorf("falha ao aplicar update: %w", err)
	}

	log.Printf("event=update_applied")
	return nil
}

func isSecureUpdateURL(candidate *url.URL) bool {
	if candidate == nil {
		return false
	}
	if strings.EqualFold(candidate.Scheme, "https") {
		return true
	}
	if !strings.EqualFold(candidate.Scheme, "http") {
		return false
	}
	host := strings.Trim(candidate.Hostname(), "[]")
	return strings.EqualFold(host, "localhost") || host == "127.0.0.1" || host == "::1"
}

func RestartApp() error {
	cmd := newHiddenCommand(os.Args[0], os.Args[1:]...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("falha ao reiniciar: %w", err)
	}
	return nil
}
