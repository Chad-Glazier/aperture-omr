package pdf

import (
	"testing"

	"gocv.io/x/gocv"
)

func TestPdfPageCount(t *testing.T) {
	count, err := pdfPageCount("testdata/sample.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Fatalf(
			"expected page count to be %d, got %d",
			5, count,
		)
	}
}

func TestPdfPageToGray(t *testing.T) {
	const samplePdfPath = "testdata/sample.pdf"

	page, err := pdfPageToGray(samplePdfPath, 0)
	if err != nil {
		t.Fatal(err)
	}

	mat, err := gocv.ImageGrayToMatGray(page)
	if err != nil {
		t.Fatal(err)
	}
	defer mat.Close()

	// This test cannot guarantee that a rendered image is "correct" without
	// manual inspection but using IMEncode at least guarantees that the
	// matrix is well-formed.
	buf, err := gocv.IMEncode(gocv.PNGFileExt, mat)
	if err != nil {
		t.Fatal(err)
	}
	defer buf.Close()

	//
	// Test that all valid indices work.
	//

	const pageCount = 5

	for i := range pageCount {
		if _, err := pdfPageToGray(samplePdfPath, i); err != nil {
			t.Fatal(err)
		}
	}

	//
	// Test that a bunch of invalid indices don't work.
	//

	for i := pageCount; i < pageCount+10; i++ {
		_, err := pdfPageToGray(samplePdfPath, i)
		if err != errIndexOutOfBounds {
			t.Fatal(err)
		}
	}
}

func TestPdfToGrayPages(t *testing.T) {

	const samplePdfPath = "testdata/sample.pdf"

	pages, err := pdfToGrayPages(samplePdfPath)
	if err != nil {
		t.Fatal(err)
	}

	for _, page := range pages {
		mat, err := gocv.ImageGrayToMatGray(page)
		if err != nil {
			t.Fatal(err)
		}
		defer mat.Close()

		if _, err := gocv.IMEncode(gocv.PNGFileExt, mat); err != nil {
			t.Fatal(err)
		}
	}
}
