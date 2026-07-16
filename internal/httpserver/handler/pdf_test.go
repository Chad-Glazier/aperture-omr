package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
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

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/scan/pdf",
		&body,
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req
}

//
// Tests
//

func TestPostScanPdf_OK(t *testing.T) {

	s, err := NewLocalResources(t.TempDir())
	if err != nil {
		t.Fatal("error initializing server resources: " + err.Error())
	}
	defer s.Close()

	pTmplId := postNewPreprocessingTemplate(t, s)

	//
	// Normal path
	//

	req := newScanPdfRequest(t, pTmplId, "testdata/batches/exams.pdf")
	rr := httptest.NewRecorder()
	PostMarkingJob(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	
}
