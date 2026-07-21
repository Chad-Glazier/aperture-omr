package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetImage(t *testing.T) {

	s, err := NewLocalResources(t.TempDir())
	if err != nil {
		t.Fatal("error initializing server resources: " + err.Error())
	}
	defer s.Close()

	//
	// 1) Upload the preprocessing template.
	//

	req, err := makeMultipartRequest(
		t,
		"testdata/preprocessing_template.json",
		multipartImage{name: "page0anchor0", filename: "testdata/anchors/anchor.jpeg"},
		multipartImage{name: "page0anchor1", filename: "testdata/anchors/anchor.jpeg"},
		multipartImage{name: "page0anchor2", filename: "testdata/anchors/anchor.jpeg"},
		multipartImage{name: "page1anchor0", filename: "testdata/anchors/anchor.jpeg"},
		multipartImage{name: "page1anchor1", filename: "testdata/anchors/anchor.jpeg"},
		multipartImage{name: "page1anchor2", filename: "testdata/anchors/anchor.jpeg"},
	)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	PostPreprocessingTemplate(s).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("failed to upload preprocessing template")
	}

	v := make(map[string]string)
	if err := json.Unmarshal(rr.Body.Bytes(), &v); err != nil {
		t.Fatalf("failed to parse JSON response: %s", err.Error())
	}
	preprocessingTemplateId, ok := v["templateId"]
	if !ok {
		t.Fatalf("templateId wasn't found in response body")
	}

	//
	// 2) Upload the scan of the exam (two pages).
	//

	req, err = makeMultipartRequest(
		t,
		"",
		multipartImage{name: "page0", filename: "testdata/pages/exam0page0.jpeg"},
		multipartImage{name: "page1", filename: "testdata/pages/exam0page1.jpeg"},
	)
	if err != nil {
		t.Fatal(err)
	}
	req.URL.RawQuery += "template=" + preprocessingTemplateId

	rr = httptest.NewRecorder()
	PostScan(s).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	v = make(map[string]string)
	if err := json.Unmarshal(rr.Body.Bytes(), &v); err != nil {
		t.Fatalf("failed to parse JSON response: %s", err.Error())
	}
	scanId, ok := v["scanId"]
	if !ok {
		t.Fatalf("scanId wasn't found in response body")
	}

	//
	// 3) Get the page images.
	//

	badQueries400 := []string{
		"",
		"scan=" + scanId,
		"page=0",
		"scan=" + scanId + "&page=notanumber",
	}
	for _, query := range badQueries400 {
		req, err = http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.URL.RawQuery = query
		rr = httptest.NewRecorder()
		GetImage(s).ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf(
				"expected query %s to have 400 response, got %d",
				query, rr.Code,
			)
		}
	}

	badQueries404 := []string{
		"scan=1234&page=0",
		"scan=" + scanId + "&page=99",
	}
	for _, query := range badQueries404 {
		req, err = http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.URL.RawQuery = query
		rr = httptest.NewRecorder()
		GetImage(s).ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf(
				"expected query %s to have 404 response, got %d",
				query, rr.Code,
			)
		}
	}

	for pageIdx := range 2 {
		req, err = http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.URL.RawQuery = fmt.Sprintf("scan=%s&page=%d", scanId, pageIdx)
		rr = httptest.NewRecorder()
		GetImage(s).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf(
				"expected query %s to have 200 response, got %d",
				req.URL.RawQuery, rr.Code,
			)
		}
		if ct := rr.Header().Get("Content-Type"); ct != imgType {
			t.Fatalf("expected Content-Type %s, got %s", imgType, ct)
		}
		if rr.Body.Len() == 0 {
			t.Fatalf("expected a non-empty image body for page %d", pageIdx)
		}
	}
}
