package handler

import (
	"bytes"
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

func assertScanMarksAreEqual(t *testing.T, want, got dto.Scan) {
	t.Helper()

	// Note: this implementation assumes that two equivalent markings will have
	// equivalent ordering for their questions and selections. This isn't
	// necessary to satisfy the API, so if the implementation changes (e.g., to
	// mark questions in a random order, or in parallel), this comparison will
	// need to be changed.

	assert.Assert(t, len(want.Marks) == len(got.Marks))
	for i := range want.Marks {
		var (
			w = want.Marks[i]
			g = got.Marks[i]
		)
		assert.Assert(t, w.QuestionId == g.QuestionId)
		assert.Assert(t, w.Flagged == g.Flagged)
		assert.Assert(t, len(w.Selected) == len(g.Selected))
		for j := range w.Selected {
			assert.Assert(t, w.Selected[j] == g.Selected[j])
		}
	}
}

func assertIsCanonicalMarks(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	buf, err := testData.ReadFile("testdata/marks/canonical.json")
	assert.Assert(t, err == nil)

	var want dto.MarkingResult
	err = json.Unmarshal(buf, &want)
	assert.Assert(t, err == nil)

	var got dto.MarkingResult
	err = json.Unmarshal(w.Body.Bytes(), &got)
	assert.Assert(t, err == nil)

	assert.Assert(t, want.PagesMarked == got.PagesMarked)
	assert.Assert(t, len(want.Scans) == len(got.Scans))

	for i := range want.Scans {
		assertScanMarksAreEqual(t, want.Scans[i], got.Scans[i])
	}
}

//
// Tests
//

func TestRequestMarks(t *testing.T) {

	s := newTestResources(t)
	defer s.Close()

	var (
		pTmplId = postCanonicalPreprocessingTemplate(s, t)
		mTmplId = postCanonicalMarkingTemplate(s, t)
		scanId  = postCanonicalScan(s, t, pTmplId)
	)

	//
	// First, we check the canonical case.
	//

	t.Run("canonical", func(t *testing.T) {

		buf, err := json.Marshal(dto.MarkingJobRequest{
			TemplateId: mTmplId,
			ScanIds:    []string{scanId},
		})
		assert.Assert(t, err == nil)

		var (
			r = httptest.NewRequest("POST", "/", bytes.NewReader(buf))
			w = httptest.NewRecorder()
		)
		r.Header.Set("Content-Type", "application/json")

		RequestMarks(s).ServeHTTP(w, r)
		assert.Assert(t, w.Result().StatusCode == http.StatusOK)

		var markResult dto.MarkingResult
		err = json.Unmarshal(w.Body.Bytes(), &markResult)
		assert.Assert(t, err == nil)

		assertIsCanonicalMarks(t, w)
	})

	//
	// Next, we try a batch operation:
	// 1) first, we upload the "5_funky_duplicate_exams.pdf" scan. This should
	//    yield 5 scan IDs.
	// 2) next, we request marks for all 5 at once.
	// 3) finally, we check their marks. Since they are duplicates, their
	//    markings should all be equivalent.
	//

	t.Run("funky bunch", func(t *testing.T) {

		//
		// (1) Post the funky batch of duplicate exams.
		//

		body, err := testData.Open(
			"testdata/scans/5_funky_duplicate_exams.pdf",
		)
		assert.Assert(t, err == nil)

		var (
			r = httptest.NewRequest("POST", "/", body)
			w = httptest.NewRecorder()
		)
		r.URL.RawQuery = "preprocessingTemplate=" + pTmplId + "&"
		r.URL.RawQuery += "dpi=250"
		r.Header.Set("Content-Type", "application/pdf")

		PostScanPdf(s).ServeHTTP(w, r)
		assert.Assert(t, w.Result().StatusCode == http.StatusOK)

		var result dto.ScanResult
		err = json.Unmarshal(w.Body.Bytes(), &result)
		assert.Assert(t, err == nil)
		assert.Assert(t, len(result.ScanIds) == 5)

		//
		// (2) Request their marks.
		//

		buf, err := json.Marshal(dto.MarkingJobRequest{
			TemplateId: mTmplId,
			ScanIds:    result.ScanIds,
		})
		assert.Assert(t, err == nil)

		r = httptest.NewRequest("POST", "/", bytes.NewReader(buf))
		w = httptest.NewRecorder()
		r.Header.Set("Content-Type", "application/json")

		RequestMarks(s).ServeHTTP(w, r)
		assert.Assert(t, w.Result().StatusCode == http.StatusOK)

		var markResult dto.MarkingResult
		err = json.Unmarshal(w.Body.Bytes(), &markResult)
		assert.Assert(t, err == nil)

		//
		// (3) Ensure their marks are all equivalent.
		//

		assert.Assert(t, len(markResult.Scans) == 5)
		for i := 1; i < 5; i++ {
			assertScanMarksAreEqual(t,
				markResult.Scans[i-1],
				markResult.Scans[i],
			)
		}
	})

	//
	// Finally, we test out a few bad inputs.
	//

	tt := []struct {
		name         string
		tmplId       string
		scanId       string
		expectStatus int
	}{
		{
			name:         "empty template ID",
			tmplId:       "",
			scanId:       scanId,
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "empty scan ID",
			tmplId:       mTmplId,
			scanId:       "",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "unrecognized template ID",
			tmplId:       "hehehe",
			scanId:       scanId,
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "unrecognized scan ID",
			tmplId:       mTmplId,
			scanId:       "hehehe",
			expectStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, test := range tt {
		buf, err := json.Marshal(dto.MarkingJobRequest{
			TemplateId: test.tmplId,
			ScanIds:    []string{test.scanId},
		})
		assert.Assert(t, err == nil)

		var (
			r = httptest.NewRequest("POST", "/", bytes.NewReader(buf))
			w = httptest.NewRecorder()
		)
		r.Header.Set("Content-Type", "application/json")

		RequestMarks(s).ServeHTTP(w, r)
		assert.Assert(t, w.Result().StatusCode == test.expectStatus)
	}
}
