package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chad-Glazier/aperture-omr/internal/fstore"
	"github.com/Chad-Glazier/aperture-omr/internal/server/resources"
	"gotest.tools/v3/assert"
)

func TestGetImage(t *testing.T) {

	s := resources.NewTesting(t)
	defer s.Close()

	var (
		pTmplId = postCanonicalPreprocessingTemplate(s, t)
		scanId  = postCanonicalScan(s, t, pTmplId)
	)

	// Note: the canonical scan has 2 pages, so indices 0 and 1 should be
	// valid.

	tt := []struct {
		name              string
		scanId            string
		pageIdx           string
		expectStatus      int
		expectContentType string
	}{
		{
			name:              "canonical page 0",
			scanId:            scanId,
			pageIdx:           "0",
			expectStatus:      http.StatusOK,
			expectContentType: fstore.ImgContentType,
		},
		{
			name:              "canonical page 1",
			scanId:            scanId,
			pageIdx:           "1",
			expectStatus:      http.StatusOK,
			expectContentType: fstore.ImgContentType,
		},
		{
			name:         "incoherent page index",
			scanId:       scanId,
			pageIdx:      "-1",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "invalid page index",
			scanId:       scanId,
			pageIdx:      "3",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "unknown scan",
			scanId:       "hehehe",
			pageIdx:      "1",
			expectStatus: http.StatusNotFound,
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			var (
				r = httptest.NewRequest("GET", "/", nil)
				w = httptest.NewRecorder()
			)
			if test.scanId != "" {
				r.URL.RawQuery = "scan=" + test.scanId + "&"
			}
			if test.pageIdx != "" {
				r.URL.RawQuery += "page=" + test.pageIdx + "&"
			}

			GetImage(s).ServeHTTP(w, r)
			assert.Assert(t, w.Result().StatusCode == test.expectStatus)

			if w.Result().StatusCode >= 300 || w.Result().StatusCode < 200 {
				return
			}
			assert.Assert(t,
				test.expectContentType == w.Header().Get("Content-Type"),
			)
		})
	}
}

func TestGetSnippet(t *testing.T) {

	s := resources.NewTesting(t)
	defer s.Close()

	var (
		mTmplId = postCanonicalMarkingTemplate(s, t)
		pTmplId = postCanonicalPreprocessingTemplate(s, t)
		scanId  = postCanonicalScan(s, t, pTmplId)
	)

	tt := []struct {
		name              string
		scanId            string
		tmplId            string
		question          string
		expectStatus      int
		expectContentType string
	}{
		{
			name:              "canonical page 0 question",
			scanId:            scanId,
			tmplId:            mTmplId,
			question:          "Q1",
			expectStatus:      http.StatusOK,
			expectContentType: fstore.ImgContentType,
		},
		{
			name:              "canonical page 1 question",
			scanId:            scanId,
			tmplId:            mTmplId,
			question:          "I3",
			expectStatus:      http.StatusOK,
			expectContentType: fstore.ImgContentType,
		},
		{
			name:         "unknown question",
			scanId:       scanId,
			tmplId:       mTmplId,
			question:     "hehehe",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "unknown template",
			scanId:       scanId,
			tmplId:       "hehehe",
			question:     "Q1",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "unknown scan",
			scanId:       "hehehe",
			tmplId:       mTmplId,
			question:     "Q1",
			expectStatus: http.StatusNotFound,
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			var (
				r = httptest.NewRequest("GET", "/", nil)
				w = httptest.NewRecorder()
			)
			if test.scanId != "" {
				r.URL.RawQuery = "scan=" + test.scanId + "&"
			}
			if test.tmplId != "" {
				r.URL.RawQuery += "template=" + test.tmplId + "&"
			}
			if test.question != "" {
				r.URL.RawQuery += "question=" + test.question + "&"
			}

			GetSnippet(s).ServeHTTP(w, r)
			assert.Assert(t, w.Result().StatusCode == test.expectStatus)

			if w.Result().StatusCode >= 300 || w.Result().StatusCode < 200 {
				return
			}
			assert.Assert(t,
				test.expectContentType == w.Header().Get("Content-Type"),
			)
		})
	}
}
