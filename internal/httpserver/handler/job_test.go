package handler

import (
	"errors"
	"net/http/httptest"
	"testing"
	"time"
)

//
// Helpers
//

func newRegistrar(t *testing.T) *JobRegistrar {
	t.Helper()

	r := NewJobRegistrar(time.Hour)
	t.Cleanup(r.Close)
	return r
}

//
// Tests
//

func TestJobRegistrarRegister(t *testing.T) {
	j := newRegistrar(t)

	req := httptest.NewRequest("POST", "/scan", nil)

	if err := j.Register("job1", req); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if !j.IsRegistered("job1") {
		t.Fatal("job should be registered")
	}

	job := j.jobs["job1"]

	if job.Id != "job1" {
		t.Errorf("Id = %q, want %q", job.Id, "job1")
	}

	if job.Method != "POST" {
		t.Errorf("Method = %q, want POST", job.Method)
	}

	if job.Path != "/scan" {
		t.Errorf("Path = %q, want /scan", job.Path)
	}

	if job.Started.IsZero() {
		t.Error("Started was not initialized")
	}

	if job.Progress != 0 {
		t.Errorf("Progress = %v, want 0", job.Progress)
	}
}

func TestJobRegistrarRegisterConflict(t *testing.T) {
	j := newRegistrar(t)

	req := httptest.NewRequest("GET", "/", nil)

	if err := j.Register("job1", req); err != nil {
		t.Fatal(err)
	}

	err := j.Register("job1", req)
	if !errors.Is(err, ErrJobIdConflict) {
		t.Fatalf("expected ErrJobIdConflict, got %v", err)
	}
}

func TestJobRegistrarGetSetProgress(t *testing.T) {
	j := newRegistrar(t)

	req := httptest.NewRequest("GET", "/", nil)

	if err := j.Register("job1", req); err != nil {
		t.Fatal(err)
	}

	if err := j.SetProgress("job1", 0.35); err != nil {
		t.Fatal(err)
	}

	progress, err := j.GetProgress("job1")
	if err != nil {
		t.Fatal(err)
	}

	if progress != 0.35 {
		t.Errorf("progress = %v, want 0.35", progress)
	}
}

func TestJobRegistrarProgressMissingJob(t *testing.T) {
	j := newRegistrar(t)

	if _, err := j.GetProgress("missing"); !errors.Is(err, ErrJobNotFound) {
		t.Fatalf("expected ErrJobNotFound, got %v", err)
	}

	if err := j.SetProgress("missing", 0.5); !errors.Is(err, ErrJobNotFound) {
		t.Fatalf("expected ErrJobNotFound, got %v", err)
	}
}

func TestJobRegistrarFinish(t *testing.T) {
	j := newRegistrar(t)

	req := httptest.NewRequest("POST", "/scan", nil)

	if err := j.Register("job1", req); err != nil {
		t.Fatal(err)
	}

	if err := j.SetProgress("job1", 0.25); err != nil {
		t.Fatal(err)
	}

	if err := j.Finish("job1", true); err != nil {
		t.Fatal(err)
	}

	job := j.jobs["job1"]

	if !job.Success {
		t.Error("Success = false, want true")
	}

	if job.Progress != 1.0 {
		t.Errorf("Progress = %v, want 1.0", job.Progress)
	}

	if job.Finished.IsZero() {
		t.Error("Finished was not set")
	}
}

func TestJobRegistrarFinishMissingJob(t *testing.T) {
	j := newRegistrar(t)

	err := j.Finish("missing", true)
	if !errors.Is(err, ErrJobNotFound) {
		t.Fatalf("expected ErrJobNotFound, got %v", err)
	}
}

func TestJobRegistrarEvictAllOlderThan(t *testing.T) {
	j := newRegistrar(t)

	now := time.Now()

	j.jobs["old"] = &JobDetails{
		Id:      "old",
		Started: now.Add(-2 * time.Hour),
	}

	j.jobs["new"] = &JobDetails{
		Id:      "new",
		Started: now,
	}

	j.evictAllOlderThan(time.Hour)

	if j.IsRegistered("old") {
		t.Error("old job should have been evicted")
	}

	if !j.IsRegistered("new") {
		t.Error("new job should not have been evicted")
	}
}

func TestJobRegistrarEvictEmpty(t *testing.T) {
	j := newRegistrar(t)

	j.evictAllOlderThan(time.Hour)

	if len(j.jobs) != 0 {
		t.Errorf("len(j.jobs) = %d, want 0", len(j.jobs))
	}
}
