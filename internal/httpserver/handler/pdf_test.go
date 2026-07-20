package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"ubco-team15/omr/internal/httpserver/dto"
)

//
// Helper functions
//

func newScanPdfRequest(
	t *testing.T,
	pTmplId string,
	pdfPath string,
) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if pTmplId != "" {
		err := writer.WriteField("preprocessingTemplate", pTmplId)
		if err != nil {
			t.Fatal(err)
		}
	}

	if pdfPath != "" {
		part, err := writer.CreateFormFile("pdf", "exams.pdf")
		if err != nil {
			t.Fatal(err)
		}

		buf, err := testData.ReadFile(pdfPath)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := part.Write(buf); err != nil {
			t.Fatal(err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		&body,
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req
}

// Get the marks for scans and return the results.
func getMarkResults(
	t *testing.T,
	s ServerResources,
	scanIds []string,
	mTmplId string,
) *dto.MarkingResult {

	scanList := strings.Builder{}
	scanList.WriteString("[")
	for i, scanId := range scanIds {
		fmt.Fprintf(&scanList, "\"%s\"", scanId)
		if i != len(scanIds)-1 {
			scanList.WriteString(",")
		}
	}
	scanList.WriteString("]")

	body := fmt.Appendf(nil,
		`{
			"template": "%s",
			"scans": %s
		}`,
		mTmplId,
		scanList.String(),
	)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	PostMarkingJob(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	marks := &dto.MarkingResult{}
	if err := json.Unmarshal(rr.Body.Bytes(), marks); err != nil {
		t.Fatal("failed to unmarshal response body")
	}

	return marks
}

//
// Tests
//

func TestPostScanPdf_Normal(t *testing.T) {

	s, err := NewLocalResources(t.TempDir())
	if err != nil {
		t.Fatal("error initializing server resources: " + err.Error())
	}
	defer s.Close()

	pTmplId := postNewPreprocessingTemplate(t, s)

	//
	// Normal path
	//

	req := newScanPdfRequest(t, pTmplId, "testdata/batches/3_normal_exams.pdf")
	rr := httptest.NewRecorder()
	PostScanPdf(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

}

func TestPostScanPdf_Funky(t *testing.T) {

	s, err := NewLocalResources(t.TempDir())
	if err != nil {
		t.Fatal("error initializing server resources: " + err.Error())
	}
	defer s.Close()

	pTmplId := postNewPreprocessingTemplate(t, s)

	//
	// Test an exam that has some pages in the wrong order and upside-down.
	//

	req := newScanPdfRequest(t, pTmplId, "testdata/batches/1_funky_exam.pdf")
	rr := httptest.NewRecorder()
	PostScanPdf(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

}

func TestPostScanPdf_FunkyBatch(t *testing.T) {

	s, err := NewLocalResources(t.TempDir())
	if err != nil {
		t.Fatal("error initializing server resources: " + err.Error())
	}
	defer s.Close()

	pTmplId := postNewPreprocessingTemplate(t, s)
	mTmplId := postNewMarkingTemplate(t, s)

	//
	// This test uploads some PDF scans that have some funkiness (some pages
	// are upside-down, some are in the wrong order), but they're all the same
	// exam. This means that we should be able to verify that the exams were
	// parsed correctly by marking them after their upload and then checking
	// whether the markings are equal.
	//

	req := newScanPdfRequest(
		t,
		pTmplId,
		"testdata/batches/5_funky_duplicate_exams.pdf",
	)
	rr := httptest.NewRecorder()
	PostScanPdf(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var scanIds []string
	json.Unmarshal(rr.Body.Bytes(), &scanIds)

	marks := getMarkResults(t, s, scanIds, mTmplId)
	for i := 1; i < len(marks.Scans); i++ {
		if !reflect.DeepEqual(marks.Scans[i].Marks, marks.Scans[i-1].Marks) {
			t.Fatalf(
				"marks for scan %d and %d aren't equal",
				i, i-1,
			)
		}
	}
}

func TestPostScanPdf_BadInputs(t *testing.T) {

	s, err := NewLocalResources(t.TempDir())
	if err != nil {
		t.Fatal("error initializing server resources: " + err.Error())
	}
	defer s.Close()

	pTmplId := postNewPreprocessingTemplate(t, s)

	//
	// This test just runs through a set of bad inputs to the endpoint.
	//

	// 400: No template ID.
	req := newScanPdfRequest(t, "", "testdata/batches/1_funky_exam.pdf")

	rr := httptest.NewRecorder()
	PostScanPdf(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	// 400: No PDF.
	req = newScanPdfRequest(t, pTmplId, "")

	rr = httptest.NewRecorder()
	PostScanPdf(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	// 400: Malformed PDF.
	req = newScanPdfRequest(t, pTmplId, "testdata/pages/exam0page0.jpeg")

	rr = httptest.NewRecorder()
	PostScanPdf(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	// 400: Page count mismatch between scan and template.
	req = newScanPdfRequest(
		t, pTmplId, "testdata/batches/1_half_of_an_exam.pdf",
	)

	rr = httptest.NewRecorder()
	PostScanPdf(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	// 404: Unrecognized template ID.
	req = newScanPdfRequest(
		t, "big chungus", "testdata/batches/1_funky_exam.pdf",
	)

	rr = httptest.NewRecorder()
	PostScanPdf(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
