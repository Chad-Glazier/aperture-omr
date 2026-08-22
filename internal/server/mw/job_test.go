package mw

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

//
// Tests
//

func TestJobRegistrar(t *testing.T) {
	j := NewJobRegistrar(time.Hour)
	defer j.Close()

	t.Run("basic registration", func(t *testing.T) {
		const jobId = "do you believe in life after love?"

		req := httptest.NewRequest("POST", "/scan", nil)
		err := j.Register(jobId, req)
		assert.NilError(t, err)
		assert.Assert(t, j.IsRegistered(jobId))

		job, err := j.Get(jobId)
		assert.NilError(t, err)
		assert.Assert(t, job.Id == jobId)
		assert.Assert(t, job.Method == "POST")
		assert.Assert(t, job.Path == "/scan")
		assert.Assert(t, !job.Started.IsZero())
		assert.Assert(t, job.Progress == 0)
		assert.Assert(t, job.Finished.IsZero())
		assert.Assert(t, job.Success == false)
		assert.Assert(t, job.Notes == "")
	})

	t.Run("id conflict", func(t *testing.T) {
		const jobId = "I can feel something inside me sayin',"

		req := httptest.NewRequest("GET", "/", nil)
		err := j.Register(jobId, req)
		assert.NilError(t, err)

		err = j.Register(jobId, req)
		assert.Assert(t, err == ErrJobIdConflict)
	})

	t.Run("set progress", func(t *testing.T) {
		const jobId = "\"I really don't think you're strong enough, no\""

		req := httptest.NewRequest("GET", "/", nil)
		err := j.Register("job1", req)
		assert.NilError(t, err)

		err = j.SetProgress("job1", 0.35)
		assert.NilError(t, err)

		job, err := j.Get("job1")
		assert.NilError(t, err)
		assert.Assert(t, job.Progress == 0.35)
	})

	t.Run("progress missing job", func(t *testing.T) {
		_, err := j.Get("missing")
		assert.Assert(t, err == ErrJobNotFound)

		err = j.SetProgress("missing", 0.5)
		assert.Assert(t, err == ErrJobNotFound)
	})

	t.Run("write notes", func(t *testing.T) {
		const jobId = "Ser Geralt of Rivia"

		req := httptest.NewRequest("GET", "/", nil)
		err := j.Register(jobId, req)
		assert.NilError(t, err)

		err = j.WriteNotes(jobId, "fuck")
		assert.NilError(t, err)

		job, err := j.Get(jobId)
		assert.NilError(t, err)
		assert.Assert(t, job.Notes == "fuck")
	})

	t.Run("write notes on missing job", func(t *testing.T) {
		_, err := j.Get("missing")
		assert.Assert(t, err == ErrJobNotFound)

		err = j.WriteNotes("missing", "...")
		assert.Assert(t, err == ErrJobNotFound)
	})

	sampleResult := NewJobResult()
	sampleResultBody := []byte(`{ "mother": "yesterday?" }`)
	sampleResult.WriteHeader(http.StatusOK)
	sampleResult.Write(sampleResultBody)
	sampleResult.Headers.Add("Content-Type", "application/json")

	t.Run("finishing", func(t *testing.T) {
		const jobId = "Mersault"

		req := httptest.NewRequest("POST", "/scan", nil)
		err := j.Register(jobId, req)
		assert.NilError(t, err)

		err = j.SetProgress(jobId, 0.25)
		assert.NilError(t, err)

		err = j.SetFinished(jobId, true, sampleResult)
		assert.NilError(t, err)

		job, err := j.Get(jobId)
		assert.NilError(t, err)
		assert.Assert(t, job.Success)
		assert.Assert(t, job.Progress == 1.0)
		assert.Assert(t, !job.Finished.IsZero())

		result, err := j.GetResult(jobId)
		assert.NilError(t, err)
		assert.Assert(t, result.Status == sampleResult.Status)
		assert.Assert(t, result.Body.String() == string(sampleResultBody))
	})

	t.Run("finishing a missing job", func(t *testing.T) {
		err := j.SetFinished("missing", true, sampleResult)
		assert.Assert(t, err == ErrJobNotFound)
	})

	t.Run("progressing a finished job", func(t *testing.T) {
		const jobId = "the plague"

		req := httptest.NewRequest("GET", "/", nil)

		err := j.Register(jobId, req)
		assert.NilError(t, err)

		err = j.SetFinished(jobId, true, NewJobResult())
		assert.NilError(t, err)

		err = j.SetProgress(jobId, 0.5)
		assert.Assert(t, err == ErrJobFinished)

		err = j.SetFinished(jobId, false, NewJobResult())
		assert.Assert(t, err == ErrJobFinished)
	})

	t.Run("results of a missing job", func(t *testing.T) {
		_, err := j.GetResult("missing")
		assert.Assert(t, err == ErrJobNotFound)
	})
}

func TestJobRegistrarEviction(t *testing.T) {
	j := NewJobRegistrar(time.Hour)
	defer j.Close()

	j.jobs["old"] = &JobDetails{
		Id:      "old",
		Started: time.Now().Add(-24 * time.Hour),
	}
	j.jobs["new"] = &JobDetails{
		Id:      "new",
		Started: time.Now(),
	}

	j.evictAllOlderThan(time.Hour)
	assert.Assert(t, !j.IsRegistered("old"))
	assert.Assert(t, j.IsRegistered("new"))
}

func TestJobRegistrarListHandler(t *testing.T) {
	j := NewJobRegistrar(time.Hour)
	defer j.Close()

	var (
		r = httptest.NewRequest("GET", "/", nil)
		w = httptest.NewRecorder()
	)

	for _, jobId := range []string{"a", "b", "c"} {
		err := j.Register(jobId, r)
		assert.NilError(t, err)
	}

	j.jobs["a"].Started = time.Now().Add(-2 * time.Hour)
	j.jobs["b"].Started = time.Now()
	j.jobs["c"].Started = time.Now().Add(-1 * time.Hour)

	j.ListHandler().ServeHTTP(w, r)
	assert.Assert(t, w.Result().StatusCode == http.StatusOK)

	var jobs []*JobDetails
	err := json.Unmarshal(w.Body.Bytes(), &jobs)
	assert.NilError(t, err)
	assert.Assert(t, len(jobs) == 3)
	assert.Assert(t, jobs[0].Id == "a") // They should be sorted from oldest to
	assert.Assert(t, jobs[1].Id == "c") // newest.
	assert.Assert(t, jobs[2].Id == "b")
}

func TestJobRegistrarResultHandler(t *testing.T) {
	j := NewJobRegistrar(time.Hour)
	defer j.Close()

	const jobId = "snake-eater"

	err := j.Register(jobId, httptest.NewRequest("GET", "/", nil))
	assert.NilError(t, err)

	sampleResult := NewJobResult()
	sampleResultBody := []byte(`{ "still": "in a dream" }`)
	sampleResult.WriteHeader(http.StatusCreated)
	sampleResult.Write(sampleResultBody)
	sampleResult.Headers.Add("Content-Type", "application/json")

	err = j.SetFinished(jobId, true, sampleResult)
	assert.NilError(t, err)

	var (
		r = httptest.NewRequest("GET", "/result?id="+jobId, nil)
		w = httptest.NewRecorder()
	)
	j.ResultHandler().ServeHTTP(w, r)

	assert.Assert(t, w.Result().StatusCode == http.StatusCreated)
	assert.Assert(t, w.Header().Get("Content-Type") == "application/json")
	assert.Assert(t, w.Body.String() == string(sampleResultBody))
}

func TestJobRegistrarResultHandlerIncomplete(t *testing.T) {
	j := NewJobRegistrar(time.Hour)
	defer j.Close()

	const jobId = "lazy"

	err := j.Register(jobId, httptest.NewRequest("GET", "/", nil))
	assert.NilError(t, err)

	var (
		r = httptest.NewRequest("GET", "/result?id="+jobId, nil)
		w = httptest.NewRecorder()
	)
	j.ResultHandler().ServeHTTP(w, r)

	assert.Assert(t, w.Result().StatusCode == http.StatusNotFound)
}

func TestJobResultResponseWriter(t *testing.T) {
	r := NewJobResult()

	r.Headers.Set("Among", "Us")
	assert.Assert(t, r.Headers.Get("Among") == "Us")

	_, err := r.Write([]byte("body"))
	assert.NilError(t, err)
	assert.Assert(t, r.Body.String() == "body")

	r.WriteHeader(http.StatusCreated)
	assert.Assert(t, r.Status == http.StatusCreated)
}

func TestJobRegistrarJob(t *testing.T) {
	j := NewJobRegistrar(time.Hour)
	defer j.Close()

	handler := j.AsyncJob(func(
		w http.ResponseWriter,
		r *http.Request,
		res JobResources,
	) {
		res.SetProgress(0.5)
		res.SetNotes("processing")

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("done"))
	})

	var (
		r = httptest.NewRequest("POST", "/", nil)
		w = httptest.NewRecorder()
	)
	handler.ServeHTTP(w, r)
	assert.Assert(t, w.Result().StatusCode == http.StatusAccepted)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NilError(t, err)

	id := response["id"]
	assert.Assert(t, id != "")

	time.Sleep(100 * time.Millisecond) // wait for the job to finish
	job, err := j.Get(id)
	assert.NilError(t, err)
	assert.Assert(t, job.Success)
	assert.Assert(t, job.Progress == 1.0)
	assert.Assert(t, !job.Finished.IsZero())

	result, err := j.GetResult(id)
	assert.NilError(t, err)
	assert.Assert(t, result.Body.String() == "done")
	assert.Assert(t, result.Status == http.StatusCreated)
}
