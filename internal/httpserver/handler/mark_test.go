package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"ubco-team15/omr/internal/httpserver/dto"
)

func TestPostMark(t *testing.T) {

	s, err := NewDefaultResources(t.TempDir())
	if err != nil {
		t.Fatal("error initializing server resources: " + err.Error())
	}
	defer s.Close()

	//
	// The setup for the marking endpoint requires using a couple other
	// endpoints. The steps are:
	//
	//  1) Upload the preprocessing template.
	//  2) Upload the scan(s) of the exam.
	//  3) Upload the marking template.
	//  4) Post the marking job.
	//
	// After that, we can check that the marks are what we expect.
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
	// 4) Post the marking job.
	//

	body := fmt.Appendf(nil,
		`{
			"template": "%s",
			"scans": [
				"%s"
			]
		}`,
		markingTemplateId,
		scanId,
	)
	req = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr = httptest.NewRecorder()
	PostMarkingJob(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	//
	// Check the marks.
	//

	actual := &dto.MarkingResult{}
	if err := json.Unmarshal(rr.Body.Bytes(), actual); err != nil {
		t.Fatal("failed to unmarshal response body")
	}

	expected := &dto.MarkingResult{}
	buf, err := testData.ReadFile("testdata/answers/exam0.json")
	if err != nil {
		t.Fatal("failed to read answers file")
	}
	if err := json.Unmarshal(buf, expected); err != nil {
		t.Fatalf("failed to unmarshal answers file")
	}

	if len(actual.Scans) == 0 {
		t.Fatal("response body had no scans")
	}
	if len(expected.Scans) != len(actual.Scans) {
		t.Fatalf(
			"expected %d scans but got %d",
			len(expected.Scans), len(actual.Scans),
		)
	}

	for i := range actual.Scans {
		if actual.Scans[i].ScanId != scanId {
			t.Fatalf(
				"expected result scanId to be %s, got %s",
				scanId, actual.Scans[i].ScanId,
			)
		}
		if len(expected.Scans[i].Marks) != len(actual.Scans[i].Marks) {
			t.Fatalf(
				"expected %d scans but got %d",
				len(expected.Scans), len(actual.Scans),
			)
		}

		// Note: this current test expects answers to be in a specific order,
		// but that's not actually important. We can remove that constraint
		// later if it becomes relevant.
		for j := range len(actual.Scans[i].Marks) {
			actualMark := actual.Scans[i].Marks[j]
			expectedMark := expected.Scans[i].Marks[j]
			if actualMark.QuestionId != expectedMark.QuestionId {
				t.Fatal("question ID mismatch")
			}
			if actualMark.Flagged != expectedMark.Flagged {
				t.Fatalf(
					"question %s: expected flagged=%t, got flagged=%t",
					actualMark.QuestionId,
					expectedMark.Flagged,
					actualMark.Flagged,
				)
			}
			if len(actualMark.Selected) != len(expectedMark.Selected) {
				t.Fatalf(
					"question %s: expected %d selections, got %d",
					actualMark.QuestionId,
					len(expectedMark.Selected),
					len(actualMark.Selected),
				)
			}

			for k := range len(actualMark.Selected) {
				if actualMark.Selected[k] != expectedMark.Selected[k] {
					t.Fatalf(
						"question %s: expected selection %s, got %s",
						actualMark.QuestionId,
						expectedMark.Selected[k],
						actualMark.Selected[k],
					)
				}
			}
		}
	}
}
