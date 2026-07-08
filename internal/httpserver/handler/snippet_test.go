package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"ubco-team15/omr/internal/httpserver/dto"
)

//
// Tests.
//

func TestGetSnippet(t *testing.T) {

	s, err := NewDefaultResources(t.TempDir())
	if err != nil {
		t.Fatal("error initializing server resources: " + err.Error())
	}
	defer s.Close()

	//
	// We need some setup before we can get snippets. Namely,
	//
	//  1) Upload the preprocessing template.
	//  2) Upload the scan(s) of the exam.
	//  3) Upload the marking template.
	// 	4) Get snippets.
	//
	// It's not straightforward to test whether the question snippets are
	// correct without looking at them, so for this test we just confirm that
	// there is a snippet returned for each question.
	//

	//
	// 1) Upload the preprocessing template.
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
	preprocessingTemplateId, ok := v["templateId"]
	if !ok {
		t.Fatalf("templateId wasn't found in response body")
	}

	//
	// 2) Upload the scan of the exam.
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
	// 3) Upload the marking template.
	//

	req, err = makeJsonRequest(t, "testdata/marking_template.json")
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	PostMarkingTemplate(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body%s", rr.Code, rr.Body.String())
	}

	v = make(map[string]string)
	if err := json.Unmarshal(rr.Body.Bytes(), &v); err != nil {
		t.Fatalf("failed to parse JSON response: %s", err.Error())
	}
	markingTemplateId, ok := v["templateId"]
	if !ok {
		t.Fatalf("templateId wasn't found in response body")
	}

	//
	// 4) Get the snippets.
	//

	// Some queries we expect to return 400 because the request is missing
	// information.
	badQueries400 := []string{
		"",
		"question=1234",
		"question=1234&scan=1234",
		"question=1234&template=1234",
	}
	for _, query := range badQueries400 {
		req, err = http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		req.URL.RawQuery = query
		rr = httptest.NewRecorder()
		GetSnippet(s).ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf(
				"expected query %s to have 400 response, got %d",
				query,
				rr.Code,
			)
		}
	}

	// Some queries we expect to return 404 because the requested resources are
	// missing.
	badQueries404 := []string{
		"question=1234&scan=1234&template=1234",
		"question=1234&scan=" + scanId + "&template=1234",
		"question=1234&scan=1234&template=" + markingTemplateId,
	}
	for _, query := range badQueries404 {
		req, err = http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		req.URL.RawQuery = query
		rr = httptest.NewRecorder()
		GetSnippet(s).ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf(
				"expected query %s to have 404 response, got %d",
				query,
				rr.Code,
			)
		}
	}

	// Now, we test all requests we expect to be 200.

	buf, err := testData.ReadFile("testdata/marking_template.json")
	var tmpl dto.MarkingTemplate
	if err := json.Unmarshal(buf, &tmpl); err != nil {
		t.Fatal("error parsing marking template: " + err.Error())
	}

	for _, page := range tmpl.Pages {
		for _, question := range page.Questions {

			req, err = http.NewRequest(http.MethodGet, "/", nil)
			if err != nil {
				t.Fatal(err)
			}

			req.URL.RawQuery = fmt.Sprintf(
				"question=%s&scan=%s&template=%s",
				question.ID, scanId, markingTemplateId,
			)
			rr = httptest.NewRecorder()
			GetSnippet(s).ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf(
					"expected query %s to have 200 response, got %d",
					req.URL.RawQuery,
					rr.Code,
				)
			}

		}
	}

	//
	// Below, we test one more kind of error case: a snippet is requested for a
	// scan, but the template referenced has the wrong number of pages.
	//

	req, err = makeJsonRequest(t, "testdata/marking_template_single_page.json")
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	PostMarkingTemplate(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body%s", rr.Code, rr.Body.String())
	}

	v = make(map[string]string)
	if err := json.Unmarshal(rr.Body.Bytes(), &v); err != nil {
		t.Fatalf("failed to parse JSON response: %s", err.Error())
	}
	singlePageMarkingTemplate, ok := v["templateId"]
	if !ok {
		t.Fatalf("templateId wasn't found in response body")
	}

	req, err = http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.URL.RawQuery = fmt.Sprintf(
		"question=F1&scan=%s&template=%s",
		scanId, singlePageMarkingTemplate,
	)
	rr = httptest.NewRecorder()
	GetSnippet(s).ServeHTTP(rr, req)

	if rr.Code == http.StatusOK {
		t.Fatalf(
			"expected query %s to have bad response, got %d",
			req.URL.RawQuery,
			rr.Code,
		)
	}

}
