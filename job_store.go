package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	jobStatusReceived         = "RECEIVED"
	jobStatusPrinting         = "PRINTING"
	jobStatusPrinted          = "PRINTED"
	jobStatusFailed           = "FAILED"
	localTerminalJobRetention = 90 * 24 * time.Hour
)

type persistedPrintJob struct {
	Job            PrintJob  `json:"job"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	LastError      string    `json:"lastError,omitempty"`
	CompletedFiles []int     `json:"completedFiles,omitempty"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type printJobStore struct {
	mu   sync.Mutex
	path string
	jobs map[string]persistedPrintJob
}

func newPrintJobStore() (*printJobStore, error) {
	store := &printJobStore{
		path: filepath.Join(getConfigDir(), "print-jobs.json"),
		jobs: make(map[string]persistedPrintJob),
	}
	data, err := os.ReadFile(store.path)
	if os.IsNotExist(err) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read local print queue: %w", err)
	}
	if err := json.Unmarshal(data, &store.jobs); err != nil {
		return nil, fmt.Errorf("decode local print queue: %w", err)
	}

	retentionCutoff := time.Now().Add(-localTerminalJobRetention)
	for id, entry := range store.jobs {
		if (entry.Status == jobStatusPrinted || entry.Status == jobStatusFailed) && !entry.UpdatedAt.IsZero() && entry.UpdatedAt.Before(retentionCutoff) {
			delete(store.jobs, id)
			continue
		}
		if entry.Status == jobStatusPrinting {
			entry.Status = jobStatusReceived
			entry.LastError = "Agente reiniciado durante impressão; job será retomado"
			entry.UpdatedAt = time.Now()
			store.jobs[id] = entry
		}
	}
	if err := store.saveLocked(); err != nil {
		return nil, err
	}
	return store, nil
}

func (store *printJobStore) receive(job PrintJob) (persistedPrintJob, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if existing, ok := store.jobs[job.JobID]; ok {
		if existing.Status != jobStatusFailed {
			return existing, false, nil
		}
		updated := existing
		updated.Status = jobStatusReceived
		updated.LastError = ""
		updated.UpdatedAt = time.Now()
		updated.Job = job
		store.jobs[job.JobID] = updated
		if err := store.saveLocked(); err != nil {
			store.jobs[job.JobID] = existing
			return existing, false, err
		}
		return updated, true, nil
	}
	now := time.Now()
	entry := persistedPrintJob{Job: job, Status: jobStatusReceived, CreatedAt: now, UpdatedAt: now}
	store.jobs[job.JobID] = entry
	if err := store.saveLocked(); err != nil {
		delete(store.jobs, job.JobID)
		return persistedPrintJob{}, false, err
	}
	return entry, true, nil
}

func (store *printJobStore) dashboardJobs() []persistedPrintJob {
	store.mu.Lock()
	defer store.mu.Unlock()
	jobs := make([]persistedPrintJob, 0, len(store.jobs))
	for _, entry := range store.jobs {
		if entry.CreatedAt.IsZero() {
			entry.CreatedAt = entry.UpdatedAt
		}
		jobs = append(jobs, entry)
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].UpdatedAt.After(jobs[j].UpdatedAt) })
	return jobs
}

func (store *printJobStore) start(jobID string) (PrintJob, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	entry, ok := store.jobs[jobID]
	if !ok || entry.Status != jobStatusReceived {
		return PrintJob{}, false, nil
	}
	previous := entry
	entry.Status = jobStatusPrinting
	entry.LastError = ""
	entry.UpdatedAt = time.Now()
	store.jobs[jobID] = entry
	if err := store.saveLocked(); err != nil {
		store.jobs[jobID] = previous
		return PrintJob{}, false, err
	}
	entry.Job.CompletedFiles = append([]int(nil), entry.CompletedFiles...)
	return entry.Job, true, nil
}

func (store *printJobStore) complete(jobID string, err error) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	entry, ok := store.jobs[jobID]
	if !ok {
		return fmt.Errorf("local print job %s not found", jobID)
	}
	previous := entry
	entry.UpdatedAt = time.Now()
	if err != nil {
		entry.Status = jobStatusFailed
		entry.LastError = err.Error()
	} else {
		entry.Status = jobStatusPrinted
		entry.LastError = ""
	}
	store.jobs[jobID] = entry
	if saveErr := store.saveLocked(); saveErr != nil {
		store.jobs[jobID] = previous
		return saveErr
	}
	return nil
}

func (store *printJobStore) markFileComplete(jobID string, fileIndex int) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	entry, ok := store.jobs[jobID]
	if !ok {
		return fmt.Errorf("local print job %s not found", jobID)
	}
	if fileIndex < 0 || fileIndex >= len(entry.Job.Files) {
		return fmt.Errorf("file index %d outside local print job %s", fileIndex, jobID)
	}
	for _, completed := range entry.CompletedFiles {
		if completed == fileIndex {
			return nil
		}
	}
	previous := entry
	entry.CompletedFiles = append(entry.CompletedFiles, fileIndex)
	entry.UpdatedAt = time.Now()
	store.jobs[jobID] = entry
	if err := store.saveLocked(); err != nil {
		store.jobs[jobID] = previous
		return err
	}
	return nil
}

func (store *printJobStore) resumableJobs() []PrintJob {
	store.mu.Lock()
	defer store.mu.Unlock()
	jobs := make([]persistedPrintJob, 0)
	for _, entry := range store.jobs {
		if entry.Status == jobStatusReceived {
			jobs = append(jobs, entry)
		}
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].UpdatedAt.Before(jobs[j].UpdatedAt) })
	result := make([]PrintJob, 0, len(jobs))
	for _, entry := range jobs {
		result = append(result, entry.Job)
	}
	return result
}

func (store *printJobStore) terminalJobs() []persistedPrintJob {
	store.mu.Lock()
	defer store.mu.Unlock()
	jobs := make([]persistedPrintJob, 0)
	for _, entry := range store.jobs {
		if entry.Status == jobStatusPrinted || entry.Status == jobStatusFailed {
			jobs = append(jobs, entry)
		}
	}
	return jobs
}

func (store *printJobStore) saveLocked() error {
	data, err := json.Marshal(store.jobs)
	if err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(store.path), ".print-jobs-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary print queue: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if err := tempFile.Chmod(0600); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("secure temporary print queue: %w", err)
	}
	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("write temporary print queue: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("sync temporary print queue: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temporary print queue: %w", err)
	}
	if err := os.Rename(tempPath, store.path); err != nil {
		return fmt.Errorf("replace print queue: %w", err)
	}
	return nil
}
