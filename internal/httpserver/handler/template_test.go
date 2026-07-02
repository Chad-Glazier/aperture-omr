package handler

import (
	"bytes"
	"embed"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

//
// Helper functions.
//

//go:embed testdata/*
var testData embed.FS

type multipartImage struct {
	name     string
	filename string
}

func makeJsonRequest(
	t *testing.T,
	templateFilename string,
) (*http.Request, error) {
	t.Helper()

	template, err := testData.Open(templateFilename)
	if err != nil {
		return nil, err
	}
	defer template.Close()

	req := httptest.NewRequest(http.MethodPost, "/", template)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func makeMultipartRequest(
	t *testing.T,
	templateFilename string,
	images ...multipartImage,
) (*http.Request, error) {
	t.Helper()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	if templateFilename != "" {
		template, err := testData.ReadFile(templateFilename)
		if err != nil {
			return nil, err
		}

		if err := w.WriteField("template", string(template)); err != nil {
			return nil, err
		}
	}

	for _, image := range images {
		imgWriter, err := w.CreateFormFile(image.name, image.name)
		if err != nil {
			return nil, err
		}
		f, err := testData.Open(image.filename)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		if _, err := io.Copy(imgWriter, f); err != nil {
			return nil, err
		}
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())

	return req, nil
}

//
// Tests.
//

func TestPostPreprocessingTemplate_OK(t *testing.T) {
	s, err := NewServerResources(t)
	if err != nil {
		t.Fatal("error initializing server resources: " + err.Error())
	}
	defer s.Cleanup()

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
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	v := make(map[string]string)
	if err := json.Unmarshal(rr.Body.Bytes(), &v); err != nil {
		t.Fatalf("failed to parse JSON response: %s", err.Error())
	}
	templateId, ok := v["templateId"]
	if !ok {
		t.Fatalf("templateId wasn't found in response body")
	}

	if _, err := s.LoadPreprocessingTemplate(templateId); err != nil {
		t.Fatal("Preprocessing template not found in database")
	}

	const pageCount = 2
	const anchorsPerPage = 3

	for i := range pageCount {
		for j := range anchorsPerPage {
			if _, err := s.LoadAnchor(templateId, i, j); err != nil {
				t.Fatalf("expected page%danchor%d was not found", i, j)
			}
		}
	}
}

func TestPostPreprocessingTemplate_BAD1(t *testing.T) {

	//
	// This request should be bad because the template specifies two pages,
	// but the request only has enough anchors for one.
	//

	s, err := NewServerResources(t)
	if err != nil {
		t.Fatal("error initializing server resources: " + err.Error())
	}
	defer s.Cleanup()

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
	)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	PostPreprocessingTemplate(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPostPreprocessingTemplate_BAD2(t *testing.T) {

	//
	// This request should be bad because there's only 2 anchors for one of
	// the pages.
	//

	s, err := NewServerResources(t)
	if err != nil {
		t.Fatal("error initializing server resources: " + err.Error())
	}
	defer s.Cleanup()

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
	)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	PostPreprocessingTemplate(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPostMarkingTemplate_OK(t *testing.T) {

	s, err := NewServerResources(t)
	if err != nil {
		t.Fatal("error initializing server resources: " + err.Error())
	}
	defer s.Cleanup()

	req, err := makeJsonRequest(t, "testdata/marking_template.json")
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	PostMarkingTemplate(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body%s", rr.Code, rr.Body.String())
	}

	v := make(map[string]string)
	if err := json.Unmarshal(rr.Body.Bytes(), &v); err != nil {
		t.Fatalf("failed to parse JSON response: %s", err.Error())
	}
	templateId, ok := v["templateId"]
	if !ok {
		t.Fatalf("templateId wasn't found in response body")
	}

	if _, err := s.LoadMarkingTemplate(templateId); err != nil {
		t.Fatal("Preprocessing template not found in database")
	}
}

//
// There are a lot more error cases that could be tested, but for now I'm just
// focusing on the normal path.
//
