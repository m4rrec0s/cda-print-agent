package main

import (
	"os"
	"testing"
	"time"
)

func TestPrintJobStorePersistsAndRecoversInterruptedJob(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	store, err := newPrintJobStore()
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	job := PrintJob{JobID: "job-1", OrderID: "order-1", CustomerName: "Cliente", DriveFolderID: "folder-1", Files: []PrintJobFile{{Name: "art.jpg", DriveFileID: "file-1", Type: "foto", PrinterRole: "photo"}}}
	if _, _, err := store.receive(job); err != nil {
		t.Fatalf("persist received job: %v", err)
	}
	if _, started, err := store.start(job.JobID); err != nil || !started {
		t.Fatalf("start persisted job: started=%t err=%v", started, err)
	}
	if err := store.markFileComplete(job.JobID, 0); err != nil {
		t.Fatalf("persist file checkpoint: %v", err)
	}

	restarted, err := newPrintJobStore()
	if err != nil {
		t.Fatalf("restart store: %v", err)
	}
	jobs := restarted.resumableJobs()
	if len(jobs) != 1 || jobs[0].JobID != job.JobID {
		t.Fatalf("expected interrupted job to be resumable, got %#v", jobs)
	}
	resumed, started, err := restarted.start(job.JobID)
	if err != nil || !started {
		t.Fatalf("start recovered job: started=%t err=%v", started, err)
	}
	if len(resumed.CompletedFiles) != 1 || resumed.CompletedFiles[0] != 0 {
		t.Fatalf("expected file checkpoint to survive restart, got %#v", resumed.CompletedFiles)
	}
	if _, duplicate, err := restarted.receive(job); err != nil || duplicate {
		t.Fatalf("duplicate job must not be re-enqueued: duplicate=%t err=%v", duplicate, err)
	}
}

func TestPrintJobStoreReceiveRollsBackMemoryWhenPersistenceFails(t *testing.T) {
	store := &printJobStore{
		path: t.TempDir() + "/missing/print-jobs.json",
		jobs: make(map[string]persistedPrintJob),
	}
	job := PrintJob{JobID: "job-failed-write"}
	if _, _, err := store.receive(job); err == nil {
		t.Fatal("expected persistence error")
	}
	if _, exists := store.jobs[job.JobID]; exists {
		t.Fatal("job remained in memory after persistence failure")
	}
}

func TestPrintJobStorePrunesOldTerminalJobs(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	store, err := newPrintJobStore()
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store.jobs["old"] = persistedPrintJob{
		Job:       PrintJob{JobID: "old"},
		Status:    jobStatusPrinted,
		UpdatedAt: time.Now().Add(-localTerminalJobRetention - time.Hour),
	}
	if err := store.saveLocked(); err != nil {
		t.Fatalf("persist old terminal job: %v", err)
	}
	reloaded, err := newPrintJobStore()
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	if _, exists := reloaded.jobs["old"]; exists {
		t.Fatal("old terminal job was not pruned")
	}
}

func TestMoveToHotFolderPublishesFileAndRejectsDuplicate(t *testing.T) {
	dir := t.TempDir()
	source := dir + "/source.pdf"
	if err := os.WriteFile(source, []byte("complete document"), 0600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	job := PrintJobFile{Name: "ticket.pdf", Type: "ticket"}
	path, err := moveToHotFolder(source, dir, "job-1", job)
	if err != nil {
		t.Fatalf("move to hot folder: %v", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read published file: %v", err)
	}
	if string(contents) != "complete document" {
		t.Fatalf("unexpected published content: %q", contents)
	}
	if _, err := moveToHotFolder(path, dir, "job-1", job); err == nil {
		t.Fatal("expected duplicate destination to be rejected")
	}
}
