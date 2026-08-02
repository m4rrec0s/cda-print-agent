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
	jobStatusReceived = "RECEIVED"
	jobStatusPrinting = "PRINTING"
	jobStatusPrinted  = "PRINTED"
	jobStatusFailed   = "FAILED"
)

type persistedPrintJob struct {
	Job       PrintJob  `json:"job"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	LastError string    `json:"lastError,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
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
	for id, entry := range store.jobs {
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
		if existing.Status == jobStatusFailed {
			existing.Status = jobStatusReceived
			existing.LastError = ""
			existing.UpdatedAt = time.Now()
			existing.Job = job
			store.jobs[job.JobID] = existing
			return existing, true, store.saveLocked()
		}
		return existing, false, nil
	}
	now := time.Now()
	entry := persistedPrintJob{Job: job, Status: jobStatusReceived, CreatedAt: now, UpdatedAt: now}
	store.jobs[job.JobID] = entry
	return entry, true, store.saveLocked()
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
	entry.Status = jobStatusPrinting
	entry.LastError = ""
	entry.UpdatedAt = time.Now()
	store.jobs[jobID] = entry
	return entry.Job, true, store.saveLocked()
}

func (store *printJobStore) complete(jobID string, err error) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	entry, ok := store.jobs[jobID]
	if !ok {
		return fmt.Errorf("local print job %s not found", jobID)
	}
	entry.UpdatedAt = time.Now()
	if err != nil {
		entry.Status = jobStatusFailed
		entry.LastError = err.Error()
	} else {
		entry.Status = jobStatusPrinted
		entry.LastError = ""
	}
	store.jobs[jobID] = entry
	return store.saveLocked()
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
	temp := store.path + ".tmp"
	if err := os.WriteFile(temp, data, 0600); err != nil {
		return err
	}
	return os.Rename(temp, store.path)
}
