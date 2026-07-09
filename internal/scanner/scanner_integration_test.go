
package scanner

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"gocv.io/x/gocv"
)

func loadTestTemplate(t *testing.T) *Template {
	t.Helper()

	f, err := os.Open("testdata/template.json")
	if err != nil {
		t.Fatalf("open template: %v", err)
	}
	defer f.Close()

	tmpl, err := LoadTemplate(f, func(path string) (io.ReadCloser, error) {
		return os.Open(filepath.Join("testdata", path))
	})
	if err != nil {
		t.Fatalf("load template: %v", err)
	}
	t.Cleanup(tmpl.Close)

	return tmpl
}

func TestScan(t *testing.T) {
	if _, err := os.Stat("testdata/input.jpg"); err != nil {
		t.Skip("testdata not present, skipping integration test")
	}

	tmpl := loadTestTemplate(t)

	imgFile, err := os.Open("testdata/input.jpg")
	if err != nil {
		t.Fatalf("open image: %v", err)
	}
	defer imgFile.Close()

	results, err := Scan([]io.Reader{imgFile}, tmpl)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	defer results[0].Close()

	if results[0].Picture.Cols() != tmpl.Width || results[0].Picture.Rows() != tmpl.Height {
		t.Errorf("expected output %dx%d, got %dx%d",
			tmpl.Width, tmpl.Height, results[0].Picture.Cols(), results[0].Picture.Rows())
	}
}

// TestScanUpsideDown feeds a page rotated 180° through Scan and expects it
// to succeed anyway: recoverUpsideDown should rotate the frame back and
// re-match before giving up.
func TestScanUpsideDown(t *testing.T) {
	if _, err := os.Stat("testdata/input.jpg"); err != nil {
		t.Skip("testdata not present, skipping integration test")
	}

	tmpl := loadTestTemplate(t)

	img := gocv.IMRead("testdata/input.jpg", gocv.IMReadGrayScale)
	if img.Empty() {
		t.Fatalf("read testdata/input.jpg")
	}
	defer img.Close()

	flipped := gocv.NewMat()
	defer flipped.Close()

	if err := gocv.Rotate(img, &flipped, gocv.Rotate180Clockwise); err != nil {
		t.Fatalf("rotate: %v", err)
	}

	buf, err := gocv.IMEncode(gocv.JPEGFileExt, flipped)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	defer buf.Close()

	results, err := Scan([]io.Reader{bytes.NewReader(buf.GetBytes())}, tmpl)
	if err != nil {
		t.Fatalf("Scan failed on upside-down page: %v", err)
	}
	defer results[0].Close()

	if results[0].Picture.Cols() != tmpl.Width || results[0].Picture.Rows() != tmpl.Height {
		t.Errorf("expected output %dx%d, got %dx%d",
			tmpl.Width, tmpl.Height, results[0].Picture.Cols(), results[0].Picture.Rows())
	}
}
