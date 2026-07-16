package pdf

import (
	"testing"

	"gocv.io/x/gocv"
)

func TestPageCount(t *testing.T) {
	count, err := pageCount("testdata/sample.pdf")
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

	page, err := pageToGray(samplePdfPath, 74, 0)
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
		if _, err := pageToGray(samplePdfPath, 74, i); err != nil {
			t.Fatal(err)
		}
	}

	//
	// Test that a bunch of invalid indices don't work.
	//

	for i := pageCount; i < pageCount+10; i++ {
		_, err := pageToGray(samplePdfPath, 74, i)
		if err != PageNotFound {
			t.Fatal(err)
		}
	}
}

func TestPdfBytesToGrays(t *testing.T) {

	// The sample PDF has 5 pages.
	pdf, err := testData.ReadFile("testdata/sample.pdf")
	if err != nil {
		t.Fatal(err)
	}

	images, err := pdfBytesToGrays(pdf, 74)
	if err != nil {
		t.Fatalf("pdfBytesToGrays() returned error: %v", err)
	}

	if len(images) != 5 {
		t.Fatalf("expected %d images, got %d", 5, len(images))
	}

	for i, img := range images {
		if img == nil {
			t.Fatalf("page %d: image is nil", i+1)
		}

		b := img.Bounds()
		if b.Dx() == 0 || b.Dy() == 0 {
			t.Errorf("page %d: invalid image size %v", i+1, b)
		}
	}
}
