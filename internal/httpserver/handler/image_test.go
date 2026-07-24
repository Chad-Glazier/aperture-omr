package handler

import (
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
	// Setup
	//

	pTmplId := postNewPreprocessingTemplate(t, s)
	scanId := postNewScan(t, s, pTmplId)

	//
	// Get the page images.
	//

	badQueries400 := []string{
		"",
		"scan=" + scanId,
		"page=0",
		"scan=" + scanId + "&page=notanumber",
	}
	for _, query := range badQueries400 {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.URL.RawQuery = query
		rr := httptest.NewRecorder()
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
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.URL.RawQuery = query
		rr := httptest.NewRecorder()
		GetImage(s).ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf(
				"expected query %s to have 404 response, got %d",
				query, rr.Code,
			)
		}
	}

	for pageIdx := range 2 {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.URL.RawQuery = fmt.Sprintf("scan=%s&page=%d", scanId, pageIdx)
		rr := httptest.NewRecorder()
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
