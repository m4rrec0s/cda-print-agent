package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/phpdave11/gofpdf"
)

type PrinterInfo struct {
	Name          string `json:"Name"`
	PrinterStatus int    `json:"PrinterStatus"`
}

type SizeConfig struct {
	WidthMm  int    `json:"widthMm"`
	HeightMm int    `json:"heightMm"`
	Label    string `json:"label"`
}

type PrintJobFile struct {
	Name          string     `json:"name"`
	DriveFileID   string     `json:"driveFileId"`
	SubfolderName string     `json:"subfolderName"`
	Type          string     `json:"type"`
	PrinterRole   string     `json:"printerRole"`
	SizeConfig    SizeConfig `json:"sizeConfig"`
}

type PrintJob struct {
	JobID         string         `json:"jobId,omitempty"`
	OrderID       string         `json:"orderId"`
	CustomerName  string         `json:"customerName"`
	DriveFolderID string         `json:"driveFolderId"`
	Files         []PrintJobFile `json:"files"`
}

type JobFileStatus struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type JobUIEvent struct {
	Kind         string          `json:"kind"`
	JobID        string          `json:"jobId"`
	CustomerName string          `json:"customerName"`
	Status       string          `json:"status"`
	Message      string          `json:"message"`
	Files        []JobFileStatus `json:"files"`
	Timestamp    string          `json:"timestamp"`
}

type downloadURLResponse struct {
	DownloadURL string `json:"downloadUrl"`
}

type StepCallback func(stepType string, fileIndex int, errMsg string)

func GetPrinters() ([]string, error) {
	if runtime.GOOS != "windows" {
		return []string{}, nil
	}

	cmd := newHiddenCommand("powershell", "-NoProfile", "-Command",
		"Get-Printer | Select-Object Name, PrinterStatus | ConvertTo-Json")
	output, err := cmd.Output()
	if err != nil {
		return []string{}, err
	}

	var printers []PrinterInfo
	if err := json.Unmarshal(output, &printers); err != nil {
		var single PrinterInfo
		if singleErr := json.Unmarshal(output, &single); singleErr != nil {
			return []string{}, err
		}
		printers = []PrinterInfo{single}
	}

	names := make([]string, 0, len(printers))
	for _, p := range printers {
		// PrinterStatus: 0=Idle, 1=Paused, 2=Error, 3=PendingDeletion, 8=PowerSave
		// Incluir todas para exibição, logando o status
		log.Printf("event=printer_found name=%q status=%d", p.Name, p.PrinterStatus)
		names = append(names, p.Name)
	}

	return names, nil
}

func GetPrintersWithStatus() ([]PrinterInfo, error) {
	if runtime.GOOS != "windows" {
		return []PrinterInfo{}, nil
	}

	cmd := newHiddenCommand("powershell", "-NoProfile", "-Command",
		"Get-Printer | Select-Object Name, PrinterStatus | ConvertTo-Json")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var printers []PrinterInfo
	if err := json.Unmarshal(output, &printers); err != nil {
		var single PrinterInfo
		if singleErr := json.Unmarshal(output, &single); singleErr != nil {
			return nil, err
		}
		printers = []PrinterInfo{single}
	}

	for _, p := range printers {
		log.Printf("event=printer_status name=%q status=%d", p.Name, p.PrinterStatus)
	}

	return printers, nil
}

type PrinterResolver func(role string) string

func ProcessPrintJob(
	ctx context.Context,
	apiURL string,
	agentKey string,
	hotFolderPath string,
	resolvePrinter PrinterResolver,
	job PrintJob,
	emit func(JobUIEvent),
	onStep StepCallback,
) error {
	if hotFolderPath == "" {
		return fmt.Errorf("HOT_FOLDER_PATH nao configurado")
	}

	if apiURL == "" {
		return fmt.Errorf("API_URL nao configurado - verifique as configuracoes do agente")
	}

	if _, err := url.Parse(apiURL); err != nil {
		return fmt.Errorf("API_URL invalido (%q) - verifique as configuracoes do agente", apiURL)
	}

	if err := os.MkdirAll(hotFolderPath, 0755); err != nil {
		return fmt.Errorf("criar hot folder: %w", err)
	}

	jobDir := filepath.Join(os.TempDir(), "cda-print-agent", safeFileName(job.JobID))
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		return fmt.Errorf("criar pasta temporaria: %w", err)
	}
	defer os.RemoveAll(jobDir)

	statuses := make([]JobFileStatus, len(job.Files))
	for index, file := range job.Files {
		statuses[index] = JobFileStatus{
			Name:   file.Name,
			Type:   file.Type,
			Status: "pending",
		}
	}

	emitJobEvent(emit, job, "started", "received", "Job recebido pelo agente", statuses)
	downloadedPaths := make([]string, len(job.Files))

	for index, file := range job.Files {
		statuses[index].Status = "downloading"
		emitJobEvent(emit, job, "file", "downloading", fmt.Sprintf("Baixando %s", file.Name), statuses)
		onStep("DOWNLOADING", index, "")

		tempPath, err := downloadDriveFile(ctx, apiURL, agentKey, job.JobID, file)
		if err != nil {
			statuses[index].Status = "failed"
			statuses[index].Error = err.Error()
			emitJobEvent(emit, job, "failed", "failed", err.Error(), statuses)
			onStep("FAILED", index, err.Error())
			return err
		}

		statuses[index].Status = "downloaded"
		emitJobEvent(emit, job, "file", "downloaded", fmt.Sprintf("Download concluido: %s", file.Name), statuses)
		onStep("DOWNLOADED", index, "")
		downloadedPaths[index] = tempPath
	}

	// Check if any file needs PDF fallback
	needsPDFFallback := false
	for _, file := range job.Files {
		printerName := resolvePrinter(file.PrinterRole)
		if printerName == "pdf_fallback" {
			needsPDFFallback = true
			break
		}
	}

	if needsPDFFallback {
		return handlePDFFallback(ctx, job, hotFolderPath, jobDir, downloadedPaths, statuses, emit, onStep)
	}

	return handlePrintToFolder(ctx, job, hotFolderPath, downloadedPaths, statuses, emit, onStep, resolvePrinter)
}

func handlePDFFallback(
	ctx context.Context,
	job PrintJob,
	hotFolderPath string,
	jobDir string,
	downloadedPaths []string,
	statuses []JobFileStatus,
	emit func(JobUIEvent),
	onStep StepCallback,
) error {
	for index, file := range job.Files {
		statuses[index].Status = "generating_pdf"
		emitJobEvent(emit, job, "file", "generating_pdf", fmt.Sprintf("Gerando PDF: %s", file.Name), statuses)
		onStep("GENERATING_PDF", index, "")

		outputPath := filepath.Join(hotFolderPath, safeFileName(fmt.Sprintf("%s_%s_%s.pdf", job.JobID, file.Type, strings.TrimSuffix(file.Name, filepath.Ext(file.Name)))))
		if err := generatePDF(ctx, downloadedPaths[index], outputPath); err != nil {
			statuses[index].Status = "failed"
			statuses[index].Error = err.Error()
			emitJobEvent(emit, job, "failed", "failed", fmt.Sprintf("Falha ao gerar PDF: %s", err.Error()), statuses)
			onStep("FAILED", index, err.Error())
			return err
		}

		statuses[index].Status = "pdf_generated"
		emitJobEvent(emit, job, "file", "pdf_generated", fmt.Sprintf("PDF gerado: %s", file.Name), statuses)
		onStep("PDF_GENERATED", index, "")
	}

	emitJobEvent(emit, job, "completed", "printed", "PDFs gerados com sucesso (fallback)", statuses)
	return nil
}

func printFileToPrinterWindows(filePath string, printerName string) error {
	if printerName == "" {
		return fmt.Errorf("nenhuma impressora selecionada")
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	log.Printf(
		"event=print_file file=%s ext=%s printer=%s",
		filePath,
		ext,
		printerName,
	)

	if ext == ".pdf" {
		return printPDFViaSumatra(filePath, printerName)
	}

	switch ext {
	case ".png", ".jpg", ".jpeg", ".bmp":
		return printImage(filePath, printerName)
	case ".doc", ".docx":
		return printWord(filePath, printerName)
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}
}

func printPDFViaSumatra(filePath string, printerName string) error {
	sumatraPath, err := getSumatraPath()
	if err != nil {
		log.Printf("event=sumatra_extract_failed error=%q — fallback para PrintTo", err)
		return printViaStartProcess(filePath, printerName)
	}

	log.Printf("event=pdf_print_sumatra file=%q printer=%q sumatra=%q", filePath, printerName, sumatraPath)

	cmd := newHiddenCommand(sumatraPath,
		"-print-to", printerName,
		"-silent",
		filePath,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Printf("event=sumatra_print_failed error=%q stderr=%q", err.Error(), stderr.String())
		return printViaStartProcess(filePath, printerName)
	}

	log.Printf("event=pdf_print_success_sumatra file=%q printer=%q", filePath, printerName)
	return nil
}

func printImage(filePath string, printerName string) error {
	cmd := newHiddenCommand("mspaint",
		"/pt",
		filePath,
		printerName,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("event=mspaint_failed error=%q fallback=sumatra", err.Error())
		if sumatraErr := printPDFViaSumatra(filePath, printerName); sumatraErr != nil {
			log.Printf("event=sumatra_image_failed error=%q fallback=PrintTo", sumatraErr.Error())
			return printViaStartProcess(filePath, printerName)
		}
		return nil
	}

	log.Printf("event=print_sent_via_mspaint file=%q printer=%q", filePath, printerName)
	return nil
}

func printWord(filePath string, printerName string) error {
	log.Printf("event=word_print_started file=%s", filePath)

	script := `
$ErrorActionPreference = "Stop"
$filePath = $env:CDA_PRINT_FILE
$printerName = $env:CDA_PRINT_PRINTER
$word = $null
$doc = $null
$originalPrinter = $null
try {
  $word = New-Object -ComObject Word.Application
  $word.Visible = $false
  $word.DisplayAlerts = 0
  if ($printerName) {
    $originalPrinter = $word.ActivePrinter
    try {
      $word.ActivePrinter = $printerName
    }
    catch {
      $printer = Get-CimInstance Win32_Printer | Where-Object { $_.Name -eq $printerName } | Select-Object -First 1
      if ($printer -eq $null) { throw }
      $port = $printer.PortName
      if ($port -notmatch ':$') { $port = $port + ':' }
      $word.ActivePrinter = "$($printer.Name) on $port"
    }
  }
  $doc = $word.Documents.Open($filePath, $false, $true)
  $background = $false
  $doc.PrintOut([ref]$background)
  Start-Sleep -Seconds 3
}
finally {
  if ($doc -ne $null) {
    $doc.Close($false)
    [System.Runtime.InteropServices.Marshal]::ReleaseComObject($doc) | Out-Null
  }
  if ($word -ne $null) {
    if ($originalPrinter) { $word.ActivePrinter = $originalPrinter }
    $word.Quit()
    [System.Runtime.InteropServices.Marshal]::ReleaseComObject($word) | Out-Null
  }
  [GC]::Collect()
  [GC]::WaitForPendingFinalizers()
}
`
	cmd := newHiddenCommand("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(os.Environ(),
		"CDA_PRINT_FILE="+filePath,
		"CDA_PRINT_PRINTER="+printerName,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Word PrintOut falhou: %w | stderr: %s", err, stderr.String())
	}

	log.Printf("event=word_print_success file=%s", filePath)
	return nil
}

func printPDF(filePath string, printerName string) error {
	log.Printf("event=pdf_print_started file=%s", filePath)

	script := `
$ErrorActionPreference = "Stop"
$filePath = $env:CDA_PRINT_FILE
$process = Start-Process -FilePath $filePath -Verb Print -WindowStyle Hidden -PassThru
if ($process -ne $null) {
  $process.WaitForExit(30000) | Out-Null
}
Start-Sleep -Seconds 3
`
	cmd := newHiddenCommand("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(os.Environ(), "CDA_PRINT_FILE="+filePath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ShellExecute print falhou: %w | stderr: %s", err, stderr.String())
	}

	log.Printf("event=pdf_print_success file=%s", filePath)
	return nil
}

func printViaStartProcess(filePath string, printerName string) error {
	script := fmt.Sprintf(
		`Start-Process -FilePath "%s" -Verb PrintTo -ArgumentList '"%s"' -WindowStyle Hidden -Wait`,
		filePath, printerName,
	)
	cmd := newHiddenCommand("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("PrintTo falhou: %w | stderr: %s", err, stderr.String())
	}
	return nil
}

func handlePrintToFolder(
	ctx context.Context,
	job PrintJob,
	hotFolderPath string,
	downloadedPaths []string,
	statuses []JobFileStatus,
	emit func(JobUIEvent),
	onStep StepCallback,
	resolvePrinter PrinterResolver,
) error {
	for index, file := range job.Files {
		printerName := resolvePrinter(file.PrinterRole)
		log.Printf("event=resolve_printer role=%s printer=%q file=%q", file.PrinterRole, printerName, file.Name)

		finalPath := downloadedPaths[index]

		if file.PrinterRole == "photo" && !strings.HasSuffix(strings.ToLower(finalPath), ".pdf") {
			pdfPath := strings.TrimSuffix(finalPath, filepath.Ext(finalPath)) + ".pdf"
			if err := convertToPDFForDNP(finalPath, pdfPath); err != nil {
				log.Printf("event=pdf_convert_gofpdf_failed file=%q error=%q — tentando ImageMagick", file.Name, err)
				if err2 := tryImageMagick(ctx, finalPath, pdfPath); err2 != nil {
					statuses[index].Status = "failed"
					statuses[index].Error = fmt.Sprintf("falha ao converter para PDF: %s", err.Error())
					emitJobEvent(emit, job, "failed", "failed", fmt.Sprintf("Falha ao converter %s para PDF", file.Name), statuses)
					onStep("FAILED", index, err.Error())
					return fmt.Errorf("conversao PDF obrigatoria falhou para %s: %w", file.Name, err)
				}
			}
			os.Remove(finalPath)
			finalPath = pdfPath
			file.Name = strings.TrimSuffix(file.Name, filepath.Ext(file.Name)) + ".pdf"
			log.Printf("event=pdf_converted file=%q output=%q", file.Name, pdfPath)
		}

		statuses[index].Status = "moving"
		emitJobEvent(emit, job, "file", "moving", fmt.Sprintf("Movendo %s para pasta de impressao", file.Name), statuses)
		onStep("MOVING", index, "")

		destPath, err := moveToHotFolder(finalPath, hotFolderPath, job.JobID, file)
		if err != nil {
			statuses[index].Status = "failed"
			statuses[index].Error = err.Error()
			emitJobEvent(emit, job, "failed", "failed", err.Error(), statuses)
			onStep("FAILED", index, err.Error())
			return err
		}

		if runtime.GOOS == "windows" && printerName != "" {
			onStep("SENDING_TO_PRINTER", index, "")
			log.Printf("event=printing_file file=%q printer=%q", destPath, printerName)
			if err := printFileToPrinterWindows(destPath, printerName); err != nil {
				log.Printf("event=print_failed file=%q printer=%q error=%q", destPath, printerName, err.Error())
				statuses[index].Status = "failed"
				statuses[index].Error = err.Error()
				emitJobEvent(emit, job, "file", "failed", fmt.Sprintf("Falha ao imprimir %s: %s", file.Name, err.Error()), statuses)
				onStep("FAILED", index, err.Error())
				return fmt.Errorf("falha ao imprimir %s: %w", file.Name, err)
			}
			log.Printf("event=print_success file=%q printer=%q", destPath, printerName)
			statuses[index].Status = "printed"
			emitJobEvent(emit, job, "file", "printed", fmt.Sprintf("Arquivo enviado para impressora: %s", file.Name), statuses)
			onStep("FILE_PRINTED", index, "")
		} else {
			log.Printf("event=hot_folder_only role=%s file=%q printer_name=%q", file.PrinterRole, file.Name, printerName)
			statuses[index].Status = "printed"
			emitJobEvent(emit, job, "file", "printing", fmt.Sprintf("Arquivo entregue ao hot folder: %s", file.Name), statuses)
			onStep("FILE_PRINTED", index, "")
		}
	}

	emitJobEvent(emit, job, "completed", "printed", "Todos os arquivos foram processados", statuses)
	return nil
}

func generatePDF(ctx context.Context, imagePath string, outputPath string) error {
	if err := tryImageMagick(ctx, imagePath, outputPath); err == nil {
		return nil
	}

	log.Printf("event=pdf_fallback_copy image_path=%q output_path=%q", imagePath, outputPath)
	if err := copyFile(imagePath, outputPath); err != nil {
		return fmt.Errorf("copiar imagem como fallback: %w", err)
	}
	return nil
}

func convertToPDFForDNP(imagePath string, outputPath string) error {
	const w, h = 100.0, 150.0

	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr: "mm",
		Size:    gofpdf.SizeType{Wd: w, Ht: h},
	})
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	ext := strings.ToLower(filepath.Ext(imagePath))
	imageType := "PNG"
	switch ext {
	case ".jpg", ".jpeg":
		imageType = "JPEG"
	case ".png":
		imageType = "PNG"
	case ".gif":
		imageType = "GIF"
	}

	pdf.ImageOptions(imagePath, 0, 0, w, h, false,
		gofpdf.ImageOptions{ImageType: imageType}, 0, "")

	return pdf.OutputFileAndClose(outputPath)
}

func tryImageMagick(ctx context.Context, imagePath string, outputPath string) error {
	cmd := exec.CommandContext(ctx, "convert", imagePath, outputPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("convert falhou: %w | stderr: %s", err, stderr.String())
	}
	if _, err := os.Stat(outputPath); err != nil {
		return fmt.Errorf("PDF nao foi criado pelo convert")
	}
	log.Printf("event=pdf_generated_via_imagemagick path=%q", outputPath)
	return nil
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}

func downloadDriveFile(
	ctx context.Context,
	apiURL string,
	agentKey string,
	jobID string,
	file PrintJobFile,
) (string, error) {
	downloadURL, err := requestDownloadURL(ctx, apiURL, agentKey, file.DriveFileID)
	if err != nil {
		return "", err
	}

	jobDir := filepath.Join(os.TempDir(), "cda-print-agent", safeFileName(jobID))
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		return "", fmt.Errorf("criar pasta temporaria: %w", err)
	}

	tempPath := filepath.Join(jobDir, safeFileName(file.Name))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("criar request de download: %w", err)
	}

	client := http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("baixar arquivo %s: %w", file.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download de %s retornou HTTP %d", file.Name, resp.StatusCode)
	}

	out, err := os.Create(tempPath)
	if err != nil {
		return "", fmt.Errorf("criar arquivo temporario: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("salvar download temporario: %w", err)
	}

	log.Printf("event=file_downloaded job_id=%s file=%q path=%q", jobID, file.Name, tempPath)
	return tempPath, nil
}

func requestDownloadURL(ctx context.Context, apiURL string, agentKey string, driveFileID string) (string, error) {
	if apiURL == "" {
		return "", fmt.Errorf("apiUrl nao configurado - verifique as configuracoes do agente")
	}

	parsedBase, err := url.Parse(apiURL)
	if err != nil || parsedBase.Scheme == "" || parsedBase.Host == "" {
		return "", fmt.Errorf("apiUrl invalido: %q - verifique as configuracoes do agente", apiURL)
	}

	endpoint := strings.TrimRight(apiURL, "/") + "/api/print/files/" + url.PathEscape(driveFileID) + "/download-url"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("criar request da URL de download: %w", err)
	}
	if agentKey != "" {
		req.Header.Set("X-Agent-Key", agentKey)
		req.Header.Set("X-API-Key", agentKey)
	}

	client := http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("obter URL de download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("API retornou HTTP %d ao obter URL de download", resp.StatusCode)
	}

	var payload downloadURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("ler resposta da URL de download: %w", err)
	}
	if payload.DownloadURL == "" {
		return "", fmt.Errorf("API nao retornou downloadUrl")
	}

	return payload.DownloadURL, nil
}

func moveToHotFolder(tempPath string, hotFolderPath string, jobID string, file PrintJobFile) (string, error) {
	destinationName := safeFileName(fmt.Sprintf("%s_%s_%s", jobID, file.Type, file.Name))
	destinationPath := filepath.Join(hotFolderPath, destinationName)

	if err := os.Rename(tempPath, destinationPath); err == nil {
		log.Printf("event=file_moved_to_hot_folder job_id=%s file=%q destination=%q", jobID, file.Name, destinationPath)
		return destinationPath, nil
	}

	source, err := os.Open(tempPath)
	if err != nil {
		return "", fmt.Errorf("abrir arquivo temporario: %w", err)
	}
	defer source.Close()

	destination, err := os.Create(destinationPath)
	if err != nil {
		return "", fmt.Errorf("criar arquivo no hot folder: %w", err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return "", fmt.Errorf("copiar arquivo para hot folder: %w", err)
	}

	if err := os.Remove(tempPath); err != nil {
		log.Printf("event=temp_file_remove_failed path=%q error=%q", tempPath, err.Error())
	}

	log.Printf("event=file_copied_to_hot_folder job_id=%s file=%q destination=%q", jobID, file.Name, destinationPath)
	return destinationPath, nil
}

func emitJobEvent(
	emit func(JobUIEvent),
	job PrintJob,
	kind string,
	status string,
	message string,
	files []JobFileStatus,
) {
	snapshot := make([]JobFileStatus, len(files))
	copy(snapshot, files)
	emit(JobUIEvent{
		Kind:         kind,
		JobID:        job.JobID,
		CustomerName: job.CustomerName,
		Status:       status,
		Message:      message,
		Files:        snapshot,
		Timestamp:    time.Now().Format("15:04:05"),
	})
}

func safeFileName(value string) string {
	replacer := strings.NewReplacer("\\", "-", "/", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")
	clean := strings.TrimSpace(replacer.Replace(value))
	if clean == "" {
		return "arquivo"
	}
	return clean
}
