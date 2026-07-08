package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"ubco-team15/omr/internal/httpserver/resources"

	"gocv.io/x/gocv"
)

// marker returns JPEG bytes for a small template: a white square inset by
// 5px on a black background, giving TmCcoeffNormed a non-zero standard
// deviation to divide by.
func markerImage(size int) []byte {
	m := gocv.NewMatWithSizeFromScalar(gocv.NewScalar(0, 0, 0, 0), size, size, gocv.MatTypeCV8UC1)
	defer m.Close()
	gocv.Rectangle(&m,
		image.Rect(5, 5, size-5, size-5),
		color.RGBA{R: 255, G: 255, B: 255, A: 255}, -1)

	buf, err := gocv.IMEncode(gocv.JPEGFileExt, m)
	if err != nil {
		panic(err)
	}
	defer buf.Close()
	return bytes.Clone(buf.GetBytes())
}

// page renders a canvas with a copy of the marker stamped at each center,
// encoded as JPEG.
func page(canvasSize, markSize int, centers []image.Point) []byte {
	canvas := gocv.NewMatWithSizeFromScalar(gocv.NewScalar(0, 0, 0, 0), canvasSize, canvasSize, gocv.MatTypeCV8UC1)
	defer canvas.Close()

	mark := gocv.NewMatWithSizeFromScalar(gocv.NewScalar(0, 0, 0, 0), markSize, markSize, gocv.MatTypeCV8UC1)
	defer mark.Close()
	gocv.Rectangle(&mark,
		image.Rect(5, 5, markSize-5, markSize-5),
		color.RGBA{R: 255, G: 255, B: 255, A: 255}, -1)

	for _, c := range centers {
		roi := canvas.Region(image.Rect(
			c.X-markSize/2, c.Y-markSize/2, c.X-markSize/2+markSize, c.Y-markSize/2+markSize,
		))
		mark.CopyTo(&roi)
		roi.Close()
	}

	buf, err := gocv.IMEncode(gocv.JPEGFileExt, canvas)
	if err != nil {
		panic(err)
	}
	defer buf.Close()
	return bytes.Clone(buf.GetBytes())
}

// rawMultipartRequest builds a multipart POST request from in-memory bytes,
// unlike makeMultipartRequest (template_test.go) which reads fixed files out
// of the embedded testdata FS — this test needs images generated at runtime.
func rawMultipartRequest(t *testing.T, templateJSON string, images map[string][]byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	if templateJSON != "" {
		if err := w.WriteField("template", templateJSON); err != nil {
			t.Fatal(err)
		}
	}
	for name, data := range images {
		fw, err := w.CreateFormFile(name, name+".jpg")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

// TestPostScanOutOfOrder feeds a two-page scan through PostScan with the
// pages swapped (page0 holds what belongs in page1's slot, and vice versa)
// and checks the response carries the dedicated out-of-order error code
// rather than a generic low-quality one.
func TestPostScanOutOfOrder(t *testing.T) {
	s, err := resources.NewTestingResources(t)
	if err != nil {
		t.Fatal("error initializing server resources: " + err.Error())
	}
	defer s.Cleanup()

	const canvasSize = 300
	const markSize = 40

	// Page A's layout: top-left, top-right, bottom-left.
	// Page B's layout: top-left, top-right, bottom-right — shares two of
	// page A's three anchors, so only the third tells them apart.
	pageACenters := []image.Point{{X: 60, Y: 60}, {X: 240, Y: 60}, {X: 60, Y: 240}}
	pageBCenters := []image.Point{{X: 60, Y: 60}, {X: 240, Y: 60}, {X: 240, Y: 240}}

	anchorJSON := func(c image.Point) string {
		return fmt.Sprintf(
			`{"center":{"x":%d,"y":%d},"roi":{"min":{"x":%d,"y":%d},"max":{"x":%d,"y":%d}}}`,
			c.X, c.Y, c.X-50, c.Y-50, c.X+50, c.Y+50,
		)
	}
	pageJSON := func(centers []image.Point) string {
		anchors := make([]string, len(centers))
		for i, c := range centers {
			anchors[i] = anchorJSON(c)
		}
		return `{"anchors":[` + strings.Join(anchors, ",") + `]}`
	}

	templateJSON := fmt.Sprintf(
		`{"width":%d,"height":%d,"config":{"blurSize":5,"morphCloseSize":3,`+
			`"minAnchorConfidence":0.6,"adaptiveBlockSize":31,"adaptiveC":-5},`+
			`"pages":[%s,%s]}`,
		canvasSize, canvasSize, pageJSON(pageACenters), pageJSON(pageBCenters),
	)

	markerJPEG := markerImage(markSize)
	req := rawMultipartRequest(t, templateJSON, map[string][]byte{
		"page0anchor0": markerJPEG,
		"page0anchor1": markerJPEG,
		"page0anchor2": markerJPEG,
		"page1anchor0": markerJPEG,
		"page1anchor1": markerJPEG,
		"page1anchor2": markerJPEG,
	})

	rr := httptest.NewRecorder()
	PostPreprocessingTemplate(s).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("failed to upload preprocessing template: status=%d body=%s", rr.Code, rr.Body.String())
	}

	v := make(map[string]string)
	if err := json.Unmarshal(rr.Body.Bytes(), &v); err != nil {
		t.Fatalf("failed to parse JSON response: %s", err.Error())
	}
	templateId, ok := v["templateId"]
	if !ok {
		t.Fatalf("templateId wasn't found in response body")
	}

	// Swap the pages: page0's slot (page A's layout) gets a page-B scan, and
	// vice versa — e.g. the back page fed in before the front page.
	req = rawMultipartRequest(t, "", map[string][]byte{
		"page0": page(canvasSize, markSize, pageBCenters),
		"page1": page(canvasSize, markSize, pageACenters),
	})
	req.URL.RawQuery = "template=" + templateId

	rr = httptest.NewRecorder()
	PostScan(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	assertErrorCode(t, rr, ErrCodePageOutOfOrder)
}
