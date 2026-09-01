package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"
	"github.com/Chad-Glazier/aperture-omr/internal/server/resources"
	"gotest.tools/v3/assert"
)

//
// Helper functions
//

func assertScanMarksAreEqual(t *testing.T, want, got dto.ScanMarks) {
	t.Helper()

	assert.Assert(t, want.ScanId == got.ScanId)
	assert.Assert(t, len(want.Questions) == len(got.Questions))
	for qId, wBubbles := range want.Questions {
		gBubbles, ok := got.Questions[qId]
		assert.Assert(t, ok)
		assert.Assert(t, len(gBubbles) == len(wBubbles))
		for bId := range wBubbles {
			_, ok := gBubbles[bId]
			assert.Assert(t, ok)
		}
	}
}

func assertIsCanonicalMarks(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	buf, err := testData.ReadFile("testdata/marks/canonical.json")
	assert.Assert(t, err == nil)

	var want dto.MarkResult
	err = json.Unmarshal(buf, &want)
	assert.Assert(t, err == nil)

	var got dto.MarkResult
	err = json.Unmarshal(w.Body.Bytes(), &got)
	assert.Assert(t, err == nil)

	assert.Assert(t, len(want.Results) == len(got.Results))

	for i := range want.Results {
		assertScanMarksAreEqual(t, want.Results[i], got.Results[i])
	}
}

//
// Tests
//

func TestRequestMarks(t *testing.T) {

	s := resources.NewTesting(t)
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

		f, err := os.Create("testdata/output/canonical_marks.json")
		assert.Assert(t, err == nil)
		defer f.Close()

		_, err = f.Write(w.Body.Bytes())
		assert.Assert(t, err == nil)

		assert.Assert(t, w.Result().StatusCode == http.StatusOK)

		var markResult dto.MarkResult
		err = json.Unmarshal(w.Body.Bytes(), &markResult)
		assert.Assert(t, err == nil)

		assertIsCanonicalMarks(t, w)
	})

	//
	// Next, we try a batch operation:
	//
	// 1) We upload the "5_funky_duplicate_exams.pdf" scan. This should yield 5
	//    scan IDs.
	//
	// 2) Next, we request marks for all 5 scans.
	//
	// 3) Finally, we check their marks. Since they are duplicates, their
	//    markings should all be equal to one another.
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

		var markResult dto.MarkResult
		err = json.Unmarshal(w.Body.Bytes(), &markResult)
		assert.Assert(t, err == nil)

		//
		// (3) Ensure their marks are all equivalent.
		//

		assert.Assert(t, len(markResult.Results) == 5)
		for i := 1; i < 5; i++ {
			assertScanMarksAreEqual(t,
				markResult.Results[i-1],
				markResult.Results[i],
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
