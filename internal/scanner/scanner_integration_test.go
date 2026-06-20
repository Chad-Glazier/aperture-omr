//go:build integration

package scanner

import (
	"encoding/json"
	"os"
	"testing"

	"gocv.io/x/gocv"
)

func TestScan(t *testing.T) {
	if _, err := os.Stat("testdata/input.jpg"); err != nil {
		t.Skip("testdata not present, skipping integration test")
	}

	tmplData, err := os.ReadFile("testdata/template.json")
	if err != nil {
		t.Fatalf("read template: %v", err)
	}

	var tmpl Template
	if err := json.Unmarshal(tmplData, &tmpl); err != nil {
		t.Fatalf("parse template: %v", err)
	}
	tmpl.Dir = "testdata"

	img := gocv.IMRead("testdata/input.jpg", gocv.IMReadColor)
	defer img.Close()
	if img.Empty() {
		t.Fatal("failed to read test input image")
	}

	data, err := Scan(&img, &tmpl)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	defer data.Close()

	if data.Color.Cols() != tmpl.Width || data.Color.Rows() != tmpl.Height {
		t.Errorf("expected output %dx%d, got %dx%d",
			tmpl.Width, tmpl.Height, data.Color.Cols(), data.Color.Rows())
	}
}
