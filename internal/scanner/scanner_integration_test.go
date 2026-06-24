//go:build integration

package scanner

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestScan(t *testing.T) {
	if _, err := os.Stat("testdata/input.jpg"); err != nil {
		t.Skip("testdata not present, skipping integration test")
	}

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
	defer tmpl.Close()

	imgFile, err := os.Open("testdata/input.jpg")
	if err != nil {
		t.Fatalf("open image: %v", err)
	}
	defer imgFile.Close()

	data, err := Scan(imgFile, tmpl)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	defer data.Close()

	if data.Color.Cols() != tmpl.Width || data.Color.Rows() != tmpl.Height {
		t.Errorf("expected output %dx%d, got %dx%d",
			tmpl.Width, tmpl.Height, data.Color.Cols(), data.Color.Rows())
	}
}
