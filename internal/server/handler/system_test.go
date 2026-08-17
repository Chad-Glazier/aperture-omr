package handler

import (
	"strconv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"
	"github.com/Chad-Glazier/aperture-omr/internal/server/resources"
	"gotest.tools/v3/assert"
)

//
// Tests
//

func TestGetResourceUtilization(t *testing.T) {
	s := resources.NewTesting(t)
	defer s.Close()

	var (
		r = httptest.NewRequest("GET", "/system/utilization?limit=5", nil)
		w = httptest.NewRecorder()
	)

	GetResourceUtilization(s).ServeHTTP(w, r)
	assert.Assert(t, w.Result().StatusCode == http.StatusOK)

	var result dto.ResourceUtilization
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.Assert(t, err == nil)

	assert.Assert(t, len(result.CpuHistory) <= 5)
	assert.Assert(t, len(result.MemoryHistory) <= 5)
}

func TestGetLogs(t *testing.T) {
	s := resources.NewTesting(t)
	defer s.Close()

	var (
		r = httptest.NewRequest("GET", "/system/logs?limit=5", nil)
		w = httptest.NewRecorder()
	)

	GetLogs(s).ServeHTTP(w, r)
	assert.Assert(t, w.Result().StatusCode == http.StatusOK)
}

func TestGetCpuInfo(t *testing.T) {
	s := resources.NewTesting(t)
	defer s.Close()

	var (
		r = httptest.NewRequest("GET", "/system/cpu?limit=5", nil)
		w = httptest.NewRecorder()
	)

	GetCpuInfo(s).ServeHTTP(w, r)
	assert.Assert(t, w.Result().StatusCode == http.StatusOK)
}

func TestGetMemoryInfo(t *testing.T) {
	s := resources.NewTesting(t)
	defer s.Close()

	var (
		r = httptest.NewRequest("GET", "/system/memory?limit=5", nil)
		w = httptest.NewRecorder()
	)

	GetMemoryInfo(s).ServeHTTP(w, r)
	assert.Assert(t, w.Result().StatusCode == http.StatusOK)
}

func TestCheckAdminKey(t *testing.T) {
	s := resources.NewTesting(t)
	defer s.Close()

	s.SetAdminKey("secret")

	t.Run("valid", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("OMR-Admin-Key", "secret")

		w := httptest.NewRecorder()

		CheckAdminKey(s).ServeHTTP(w, r)
		assert.Assert(t, w.Result().StatusCode == http.StatusOK)
	})

	t.Run("invalid", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("OMR-Admin-Key", "hehehe")

		w := httptest.NewRecorder()

		CheckAdminKey(s).ServeHTTP(w, r)
		assert.Assert(t, w.Result().StatusCode == http.StatusUnauthorized)
	})
}

func TestDeleteOldScans(t *testing.T) {
	s := resources.NewTesting(t)
	defer s.Close()

	s.SetAdminKey("secret")

	pTmplId := postCanonicalPreprocessingTemplate(s, t)

	//
	// We create 6 exams and record their creation dates, spacing them out by
	// 100ms each.
	//

	scanIds := make([]string, 6)
	creationTimes := make([]int64, len(scanIds))
	for i := range scanIds {
		scanIds[i] = postCanonicalScan(s, t, pTmplId)
		creationTimes[i] = time.Now().UnixMilli()
		time.Sleep(100*time.Millisecond)
	}

	//
	// Now, we attempt to delete the scans, two at a time.
	//

	for i := 0; i < len(scanIds); i += 2 {
		var (
			r = httptest.NewRequest("DELETE", "/", nil)
			w = httptest.NewRecorder()
		)
		r.Header.Set("OMR-Admin-Key", "secret")
		r.URL.RawQuery = "unixMilli="+strconv.FormatInt(creationTimes[i], 10) 

		DeleteScansOlderThan(s).ServeHTTP(w, r)
		assert.Assert(t, w.Result().StatusCode == http.StatusOK)

		for j := range i+1 {
			f, err := s.OpenScanPicture(scanIds[j], 0)
			if err == nil {
				f.Close()
			}
			assert.Assert(t, err == resources.ErrNotFound)
		}
		for j := i+1; j < len(scanIds); j++ {
			f, err := s.OpenScanPicture(scanIds[j], 0)
			if err == nil {
				f.Close()
			}
			assert.Assert(t, err == nil)
		}
	}
}
