package handler

import (
	"bytes"
	"embed"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

//go:embed testdata/*
var testData embed.FS

//
// Helper functions.
//

type multipartImage struct {
	name     string
	filename string
}

func makeMultipartRequest(
	t *testing.T,
	templateFilename string,
	images ...multipartImage,
) (*http.Request, error) {
	t.Helper()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	template, err := testData.ReadFile(templateFilename)
	if err != nil {
		return nil, err
	}

	if err := w.WriteField("template", string(template)); err != nil {
		return nil, err
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
	s, err := NewServerResources()
	if err != nil {
		t.Fatal("error initializing server resources: " + err.Error())
	}
	defer s.Cleanup()

	req, err := makeMultipartRequest(
		t,
		"testdata/preprocessing_template.json",
		multipartImage{
			name: "page0anchor0", 
			filename: "testdata/anchors/page0anchor0.jpg",
		},
		multipartImage{
			name: "page0anchor1", 
			filename: "testdata/anchors/page0anchor1.jpg",
		},		
		multipartImage{
			name: "page0anchor2", 
			filename: "testdata/anchors/page0anchor2.jpg",
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

	// TODO: validate that the data was stored.
}
