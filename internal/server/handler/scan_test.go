package handler

import (
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Chad-Glazier/aperture-omr/internal/omr"
	"github.com/Chad-Glazier/aperture-omr/internal/pdf"
	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"
	"github.com/Chad-Glazier/aperture-omr/internal/server/resources"
	"gotest.tools/v3/assert"
)

//
// Helper functions
//

// Posts a scan to the server. This should always succeed and return the ID of
// the new scan. On an unexpected failure, the test will be terminated.
//
// To get the correct preprocessing template ID for this function, see
// [postCanonicalPreprocessingTemplate].
func postCanonicalScan(
	s resources.ServerResources,
	t *testing.T,
	preprocessingTemplateId string,
) string {
	t.Helper()

	body, err := testData.Open("testdata/scans/canonical.pdf")
	assert.Assert(t, err == nil)

	var (
		r = httptest.NewRequest("POST", "/", body)
		w = httptest.NewRecorder()
	)
	r.URL.RawQuery = "preprocessTemplate=" + preprocessingTemplateId
	r.URL.RawQuery += "&dpi=250"
	r.Header.Set("Content-Type", "application/pdf")

	PostScanPdf(s).ServeHTTP(w, r)
	assert.Assert(t, w.Result().StatusCode == http.StatusOK)

	var result dto.ScanResult
	err = json.Unmarshal(w.Body.Bytes(), &result)
	assert.Assert(t, err == nil)
	assert.Assert(t, len(result.ScanIds) == 1)
	assert.Assert(t, len(result.Errors) == 0)
	assert.Assert(t, result.ScanIds[0] != "")

	return result.ScanIds[0]
}

//
// Tests
//

func TestVisualizeScanPdf(t *testing.T) {
	s := resources.NewTesting(t)
	defer s.Close()

	pTmplId := postCanonicalPreprocessingTemplate(s, t)

	pTmplOut0, err := os.Create("testdata/output/visualizedPreprocessingTemplate_page0.png")
	assert.Assert(t, err == nil)
	defer pTmplOut0.Close()
	pTmplOut1, err := os.Create("testdata/output/visualizedPreprocessingTemplate_page1.png")
	assert.Assert(t, err == nil)
	defer pTmplOut1.Close()

	tmpl, err := s.LoadPreprocessingTemplate(pTmplId)
	assert.Assert(t, err == nil)

	tmpl, err = omr.ScalePreprocessingTemplate(omr.FitMethodContain, tmpl, 1890, 2677)
	assert.Assert(t, err == nil)

	img0, err := tmpl.Image(0)
	assert.Assert(t, err == nil)
	img1, err := tmpl.Image(1)
	assert.Assert(t, err == nil)

	err = png.Encode(pTmplOut0, img0)
	assert.Assert(t, err == nil)
	err = png.Encode(pTmplOut1, img1)
	assert.Assert(t, err == nil)

	r, err := os.Open("testdata/scans/1_funky_exam.pdf")
	assert.Assert(t, err == nil)
	defer r.Close()

	examCh, nExams, err := pdf.RenderPageBlocks(r, 250, 2, 1)
	assert.Assert(t, err == nil)
	assert.Assert(t, nExams == 1)
	pageSet := <-examCh
	assert.Assert(t, pageSet.Error() == nil)
	omr.CloseAllInStream(examCh)
	
	pages := pageSet.Pages()
	assert.Assert(t, len(pages) == 2)

	renderedPagesOut, err := os.Create("testdata/output/renderedPages.png")
	assert.Assert(t, err == nil)
	omr.VisualizeSideBySide(renderedPagesOut, pages[0], pages[1])

	out0, err := os.Create("testdata/output/visualizedPreprocessing_page0.png")
	err = omr.VisualizePreprocess(out0, tmpl, pages, 0)
	assert.Assert(t, err == nil)
	out1, err := os.Create("testdata/output/visualizedPreprocessing_page1.png")
	err = omr.VisualizePreprocess(out1, tmpl, pages, 1)
	assert.Assert(t, err == nil)

	preprocessed, err := omr.Preprocess(tmpl, pages)
	assert.Assert(t, err == nil)
	defer omr.CloseAll(preprocessed)

	mTmplId := postCanonicalMarkingTemplate(s, t)
	mTmpl, err := s.LoadMarkingTemplate(mTmplId)
	assert.Assert(t, err == nil)

	markOut0, err := os.Create("testdata/output/markedPages0.png")
	assert.Assert(t, err == nil)
	err = omr.VisualizeMark(markOut0, mTmpl, preprocessed, 0)
	assert.Assert(t, err == nil)

	markOut1, err := os.Create("testdata/output/markedPages1.png")
	assert.Assert(t, err == nil)
	err = omr.VisualizeMark(markOut1, mTmpl, preprocessed, 1)
	assert.Assert(t, err == nil)
}

func TestPostScanPdf(t *testing.T) {

	s := resources.NewTesting(t)
	defer s.Close()

	pTmplId := postCanonicalPreprocessingTemplate(s, t)

	tt := []struct {
		name             string
		fileName         string
		dpi              string
		expectStatus     int
		expectNumScanIds int
		templateId       string
	}{
		{
			name:             "canonical",
			fileName:         "canonical.pdf",
			dpi:              "250",
			expectStatus:     http.StatusOK,
			expectNumScanIds: 1,
			templateId:       pTmplId,
		},
		{
			name:         "fake pdf",
			fileName:     "not_a_real_pdf.pdf",
			dpi:          "",
			expectStatus: http.StatusUnsupportedMediaType,
			templateId:   pTmplId,
		},
		{
			name:         "dpi too low",
			fileName:     "canonical.pdf",
			dpi:          "50",
			expectStatus: http.StatusUnprocessableEntity,
			templateId:   pTmplId,
		},
		{
			name:         "unrecognized template",
			fileName:     "canonical.pdf",
			dpi:          "250",
			expectStatus: http.StatusNotFound,
			templateId:   "hehehe",
		},
		{
			name:         "no template id",
			fileName:     "canonical.pdf",
			dpi:          "250",
			expectStatus: http.StatusBadRequest,
			templateId:   "",
		},
		{
			name:             "default dpi",
			fileName:         "canonical.pdf",
			dpi:              "",
			expectStatus:     http.StatusOK,
			templateId:       pTmplId,
			expectNumScanIds: 1,
		},
		{
			name:             "1 funky exam",
			fileName:         "1_funky_exam.pdf",
			dpi:              "250",
			expectStatus:     http.StatusOK,
			expectNumScanIds: 1,
			templateId:       pTmplId,
		},
		{
			name:             "5 funky exams",
			fileName:         "5_funky_duplicate_exams.pdf",
			dpi:              "250",
			expectStatus:     http.StatusOK,
			expectNumScanIds: 5,
			templateId:       pTmplId,
		},
		{
			name:             "3 normal exams",
			fileName:         "3_normal_exams.pdf",
			dpi:              "250",
			expectStatus:     http.StatusOK,
			expectNumScanIds: 3,
			templateId:       pTmplId,
		},
		{
			name:             "1 funky exam",
			fileName:         "1_funky_exam.pdf",
			dpi:              "250",
			expectStatus:     http.StatusOK,
			expectNumScanIds: 1,
			templateId:       pTmplId,
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {

			body, err := testData.Open("testdata/scans/" + test.fileName)
			assert.Assert(t, err == nil)

			var (
				r = httptest.NewRequest("POST", "/", body)
				w = httptest.NewRecorder()
			)
			r.URL.RawQuery = "preprocessTemplate=" + test.templateId
			if test.dpi != "" {
				r.URL.RawQuery += "&dpi=" + test.dpi
			}
			r.Header.Set("Content-Type", "application/pdf")

			PostScanPdf(s).ServeHTTP(w, r)

			assert.Assert(t, w.Result().StatusCode == test.expectStatus)
			if test.expectStatus >= 300 || test.expectStatus < 200 {
				return
			}

			var result dto.ScanResult
			err = json.Unmarshal(w.Body.Bytes(), &result)
			assert.Assert(t, err == nil)
			assert.Assert(t, len(result.ScanIds) == test.expectNumScanIds)
		})
	}
}

func TestDeleteScans(t *testing.T) {

	s := resources.NewTesting(t)
	defer s.Close()

	pTemplId := postCanonicalPreprocessingTemplate(s, t)
	scanId := postCanonicalScan(s, t, pTemplId)

	tt := []struct {
		name         string
		scanId       string
		expectStatus int
	}{
		{
			name:         "canonical scan",
			scanId:       scanId,
			expectStatus: http.StatusOK,
		},
		{
			name:         "empty query",
			scanId:       "",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "nonexistent template",
			scanId:       "hehehe",
			expectStatus: http.StatusOK,
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			var (
				w = httptest.NewRecorder()
				r = httptest.NewRequest("DELETE", "/", nil)
			)
			r.URL.RawQuery = "id=" + test.scanId

			DeleteScan(s).ServeHTTP(w, r)
			assert.Assert(t, w.Result().StatusCode == test.expectStatus)

			_, err := s.LoadScan(test.scanId)
			assert.Assert(t, err != nil)
		})
	}
}
