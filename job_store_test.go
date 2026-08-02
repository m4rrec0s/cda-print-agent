package main

import "testing"

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

	restarted, err := newPrintJobStore()
	if err != nil {
		t.Fatalf("restart store: %v", err)
	}
	jobs := restarted.resumableJobs()
	if len(jobs) != 1 || jobs[0].JobID != job.JobID {
		t.Fatalf("expected interrupted job to be resumable, got %#v", jobs)
	}
	if _, duplicate, err := restarted.receive(job); err != nil || duplicate {
		t.Fatalf("duplicate job must not be re-enqueued: duplicate=%t err=%v", duplicate, err)
	}
}
