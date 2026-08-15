package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"
	"gotest.tools/v3/assert"
)

//
// Helper functions
//

func newTestResources(t *testing.T) ServerResources {
	t.Helper()

	s, err := NewLocalResources(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	return s
}

//
// Tests
//

func TestGetResourceUtilization(t *testing.T) {
	s := newTestResources(t)
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
	s := newTestResources(t)
	defer s.Close()

	var (
		r = httptest.NewRequest("GET", "/system/logs?limit=5", nil)
		w = httptest.NewRecorder()
	)

	GetLogs(s).ServeHTTP(w, r)
	assert.Assert(t, w.Result().StatusCode == http.StatusOK)
}

func TestGetCpuInfo(t *testing.T) {
	s := newTestResources(t)
	defer s.Close()

	var (
		r = httptest.NewRequest("GET", "/system/cpu?limit=5", nil)
		w = httptest.NewRecorder()
	)

	GetCpuInfo(s).ServeHTTP(w, r)
	assert.Assert(t, w.Result().StatusCode == http.StatusOK)
}

func TestGetMemoryInfo(t *testing.T) {
	s := newTestResources(t)
	defer s.Close()

	var (
		r = httptest.NewRequest("GET", "/system/memory?limit=5", nil)
		w = httptest.NewRecorder()
	)

	GetMemoryInfo(s).ServeHTTP(w, r)
	assert.Assert(t, w.Result().StatusCode == http.StatusOK)
}

func TestCheckAdminKey(t *testing.T) {
	s := newTestResources(t)
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
