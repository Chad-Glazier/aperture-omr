package handler

import (
	"embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"
	"github.com/Chad-Glazier/aperture-omr/internal/server/resources"
	"gotest.tools/v3/assert"
)

//go:embed testdata/*
var testData embed.FS

//
// Helper functions
//

// Posts a marking template to the server. This should always succeed and
// return the ID of the new template. On an unexpected failure, the test will
// be terminated.
func postCanonicalMarkingTemplate(s resources.ServerResources, t *testing.T) string {
	t.Helper()

	body, err := testData.Open("testdata/marking_templates/canonical.json")
	assert.Assert(t, err == nil)

	var (
		r = httptest.NewRequest("POST", "/", body)
		w = httptest.NewRecorder()
	)
	r.Header.Set("Content-Type", "application/json")

	PostMarkingTemplate(s).ServeHTTP(w, r)

	assert.Assert(t, w.Result().StatusCode == http.StatusOK)

	var respBody dto.IdResponse
	err = json.Unmarshal(w.Body.Bytes(), &respBody)
	assert.Assert(t, err == nil)
	assert.Assert(t, respBody.Id != "")

	return respBody.Id
}

// Posts a preprocessing template to the server. This should always succeed and
// return the ID of the new template. On an unexpected failure, the test will
// be terminated.
func postCanonicalPreprocessingTemplate(
	s resources.ServerResources, t *testing.T,
) string {
	t.Helper()

	body, err := testData.Open(
		"testdata/preprocessing_templates/canonical.json",
	)
	assert.Assert(t, err == nil)

	var (
		r = httptest.NewRequest("POST", "/", body)
		w = httptest.NewRecorder()
	)
	r.Header.Set("Content-Type", "application/json")

	PostPreprocessingTemplate(s).ServeHTTP(w, r)
	assert.Assert(t, w.Result().StatusCode == http.StatusOK)

	var respBody dto.IdResponse
	err = json.Unmarshal(w.Body.Bytes(), &respBody)
	assert.Assert(t, err == nil)
	assert.Assert(t, respBody.Id != "")

	return respBody.Id
}

//
// Tests
//

func TestPostMarkingTemplate(t *testing.T) {

	s := resources.NewTesting(t)
	defer s.Close()

	tt := []struct {
		name         string
		templateName string
		expectStatus int
	}{
		{
			name:         "canonical template",
			templateName: "canonical.json",
			expectStatus: http.StatusOK,
		},
		{
			name:         "malformed template",
			templateName: "bad.json",
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			body, err := testData.Open(
				"testdata/marking_templates/" + test.templateName,
			)
			assert.Assert(t, err == nil)

			var (
				w = httptest.NewRecorder()
				r = httptest.NewRequest("POST", "/", body)
			)
			r.Header.Set("Content-Type", "application/json")

			PostMarkingTemplate(s).ServeHTTP(w, r)
			assert.Assert(t, w.Result().StatusCode == test.expectStatus)
		})
	}
}

func TestDeleteMarkingTemplate(t *testing.T) {

	s := resources.NewTesting(t)
	defer s.Close()

	id := postCanonicalMarkingTemplate(s, t)

	tt := []struct {
		name         string
		templateId   string
		expectStatus int
	}{
		{
			name:         "canonical template",
			templateId:   id,
			expectStatus: http.StatusOK,
		},
		{
			name:         "empty query",
			templateId:   "",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "nonexistent template",
			templateId:   "hehehe",
			expectStatus: http.StatusOK,
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			var (
				w = httptest.NewRecorder()
				r = httptest.NewRequest("DELETE", "/", nil)
			)
			r.URL.RawQuery = "id=" + test.templateId
			r.Header.Set("Content-Type", "application/json")

			DeleteMarkingTemplate(s).ServeHTTP(w, r)
			assert.Assert(t, w.Result().StatusCode == test.expectStatus)

			_, err := s.LoadMarkingTemplate(test.templateId)
			assert.Assert(t, err != nil)
		})
	}
}

func TestPostPreprocessingTemplate(t *testing.T) {

	//
	// TODO: Implement more robust tests. Right now, we just check that the
	// canonical preprocessing template works.
	//

	s := resources.NewTesting(t)
	defer s.Close()

	t.Run("canonical", func(t *testing.T) {
		postCanonicalPreprocessingTemplate(s, t)
	})
}

func TestDeletePreprocessingTemplate(t *testing.T) {

	s := resources.NewTesting(t)
	defer s.Close()

	id := postCanonicalPreprocessingTemplate(s, t)

	tt := []struct {
		name         string
		templateId   string
		expectStatus int
	}{
		{
			name:         "canonical template",
			templateId:   id,
			expectStatus: http.StatusOK,
		},
		{
			name:         "empty query",
			templateId:   "",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "nonexistent template",
			templateId:   "hehehe",
			expectStatus: http.StatusOK,
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			var (
				w = httptest.NewRecorder()
				r = httptest.NewRequest("DELETE", "/", nil)
			)
			r.URL.RawQuery = "id=" + test.templateId
			r.Header.Set("Content-Type", "application/json")

			DeletePreprocessingTemplate(s).ServeHTTP(w, r)
			assert.Assert(t, w.Result().StatusCode == test.expectStatus)

			_, err := s.LoadMarkingTemplate(test.templateId)
			assert.Assert(t, err != nil)
		})
	}
}
