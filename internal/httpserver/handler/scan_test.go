package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostScan(t *testing.T) {

	s, err := NewDefaultResources(t.TempDir())
	if err != nil {
		t.Fatal("error initializing server resources: " + err.Error())
	}
	defer s.Close()

	//
	// Uploading scans requires that we first have a preprocessing template.
	//

	req, err := makeMultipartRequest(
		t,
		"testdata/preprocessing_template.json",
		multipartImage{
			name:     "page0anchor0",
			filename: "testdata/anchors/anchor.jpeg",
		},
		multipartImage{
			name:     "page0anchor1",
			filename: "testdata/anchors/anchor.jpeg",
		},
		multipartImage{
			name:     "page0anchor2",
			filename: "testdata/anchors/anchor.jpeg",
		},
		multipartImage{
			name:     "page1anchor0",
			filename: "testdata/anchors/anchor.jpeg",
		},
		multipartImage{
			name:     "page1anchor1",
			filename: "testdata/anchors/anchor.jpeg",
		},
		multipartImage{
			name:     "page1anchor2",
			filename: "testdata/anchors/anchor.jpeg",
		},
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
	templateId, ok := v["templateId"]
	if !ok {
		t.Fatalf("templateId wasn't found in response body")
	}

	//
	// Now we can try to post the scan. We'll test out a couple of error cases
	// and then the normal path.
	//

	//
	// 400: No pages.
	//

	req, err = makeMultipartRequest(
		t,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	req.URL.RawQuery += "template=" + templateId

	rr = httptest.NewRecorder()
	PostScan(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	//
	// 400: Wrong number of pages.
	//

	req, err = makeMultipartRequest(
		t,
		"",
		multipartImage{
			name:     "page0",
			filename: "testdata/pages/exam0page0.jpeg",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	req.URL.RawQuery += "template=" + templateId

	rr = httptest.NewRecorder()
	PostScan(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	//
	// 404: Unrecognized template.
	//

	req, err = makeMultipartRequest(
		t,
		"",
		multipartImage{
			name:     "page0",
			filename: "testdata/pages/exam0page0.jpeg",
		},
		multipartImage{
			name:     "page1",
			filename: "testdata/pages/exam0page1.jpeg",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	req.URL.RawQuery += "template=" + "chickenturtleduck"

	rr = httptest.NewRecorder()
	PostScan(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	//
	// 200: Normal path.
	//

	req, err = makeMultipartRequest(
		t,
		"",
		multipartImage{
			name:     "page0",
			filename: "testdata/pages/exam0page0.jpeg",
		},
		multipartImage{
			name:     "page1",
			filename: "testdata/pages/exam0page1.jpeg",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	req.URL.RawQuery += "template=" + templateId

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

	pages, err := s.LoadScan(scanId)
	if err != nil {
		t.Fatalf("scan wasn't saved: %s", err.Error())
	}
	if len(pages) != 2 {
		t.Fatal("scan pages failed to load")
	}
}
