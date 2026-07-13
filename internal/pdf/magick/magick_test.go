package magick

import (
	"testing"

	"gocv.io/x/gocv"
)

func TestPdfPageToGray(t *testing.T) {
	const sampleImagePath = "testdata/sample.pdf"

	page, err := PdfPageToGray(sampleImagePath, 0)
	if err != nil {
		t.Fatal(err)
	}

	mat, err := gocv.ImageGrayToMatGray(page)
	if err != nil {
		t.Fatal(err)
	}

	// This test cannot guarantee that a rendered image is "correct", but using
	// IMEncode at least guarantees that the matrix is well-formed.
	if _, err := gocv.IMEncode(gocv.PNGFileExt, mat); err != nil {
		t.Fatal(err)
	}

	// 
	// Test that all valid indices work.
	//

	const pageCount = 5

	for i := range pageCount {
		if _, err := PdfPageToGray(sampleImagePath, i); err != nil {
			t.Fatal(err)
		}
	}

	for i := pageCount; i < pageCount+100; i++ {
		_, err := PdfPageToGray(sampleImagePath, i)
		if err != ErrIndexOutOfBounds {
			t.Fatal(err)
		}
	}
}


func TestPdfToGrayPages(t *testing.T) {

	const sampleImagePath = "testdata/sample_large.pdf"

	pages, err := PdfToGrayPages(sampleImagePath)
	if err != nil {
		t.Fatal(err)
	}

	for _, page := range pages {
		mat, err := gocv.ImageGrayToMatGray(page)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := gocv.IMEncode(gocv.PNGFileExt, mat); err != nil {
			t.Fatal(err)
		}
	}

}
