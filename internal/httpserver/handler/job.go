package handler

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

type JobDetails struct {
	Id       string
	Method   string
	Path     string
	Progress float64
	Started  time.Time
	Finished time.Time
	Success  bool
}

var (
	ErrJobNotFound   = errors.New("job not found")
	ErrJobIdConflict = errors.New("new job could not be created because the ID is already in use")
)

type JobRegistrar struct {
	jobs           map[string]*JobDetails
	mu             sync.RWMutex
	evictionTicker *time.Ticker
}

func (j *JobRegistrar) IsRegistered(id string) bool {
	j.mu.RLock()
	defer j.mu.RUnlock()

	_, ok := j.jobs[id]
	return ok
}

func (j *JobRegistrar) GetProgress(id string) (float64, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	job, ok := j.jobs[id]
	if !ok {
		return 0, ErrJobNotFound
	}

	return job.Progress, nil
}

func (j *JobRegistrar) SetProgress(id string, progress float64) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	job := j.jobs[id]
	if job == nil {
		return ErrJobNotFound
	}

	job.Progress = progress
	return nil
}

func (j *JobRegistrar) Register(
	id string,
	r *http.Request,
) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if _, ok := j.jobs[id]; ok {
		return ErrJobIdConflict
	}

	job := JobDetails{
		Id:      id,
		Method:  r.Method,
		Path:    r.URL.Path,
		Started: time.Now(),
	}
	j.jobs[id] = &job

	return nil
}

func (j *JobRegistrar) Finish(id string, success bool) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	job, ok := j.jobs[id]
	if !ok {
		return ErrJobNotFound
	}

	job.Finished = time.Now()
	job.Progress = 1.00
	job.Success = success
	return nil
}

func (j *JobRegistrar) evictAllOlderThan(age time.Duration) {
	j.mu.Lock()
	defer j.mu.Unlock()

	expired := make([]string, 0)
	now := time.Now()

	for key, job := range j.jobs {
		if now.Compare(job.Started.Add(age)) == 1 {
			expired = append(expired, key)
		}
	}

	for _, key := range expired {
		delete(j.jobs, key)
	}
}

func NewJobRegistrar(maxAge time.Duration) *JobRegistrar {
	j := &JobRegistrar{
		jobs: make(map[string]*JobDetails, 1<<8),
	}

	evictionTicker := time.NewTicker(maxAge)

	go func() {
		for range evictionTicker.C {
			j.evictAllOlderThan(maxAge)
		}
	}()

	j.evictionTicker = evictionTicker

	return j
}

func (j *JobRegistrar) Close() {
	j.evictionTicker.Stop()
}
