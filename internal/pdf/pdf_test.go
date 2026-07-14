package pdf

import (
	"embed"
	"testing"

	"gocv.io/x/gocv"
)

//go:embed testdata/*
var testData embed.FS

func TestRenderPageMats(t *testing.T) {

	r, err := testData.Open("testdata/sample.pdf")
	if err != nil {
		t.Fatal(err)
	}

	mats, err := RenderPageMats(r)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure that the matrices are well-formed images.
	for _, mat := range mats {
		buf, err := gocv.IMEncode(gocv.PNGFileExt, *mat)
		if err != nil {
			t.Fatal(err)
		}
		buf.Close()
	}

	// Ensure that there is one matrix per page.
	if len(mats) != 5 {
		t.Fatalf(
			"pdf had %d pages but only %d matrices were rendered",
			5, len(mats),
		)
	}
}

func TestRenderPageMatsWithMalformedData(t *testing.T) {

	//
	// This test checks how the renderer handles non-PDF data being passed to
	// it.
	//

	// Check a text file. This should not be parseable at all by the PDF
	// renderer.

	r, err := testData.Open("testdata/not_a_pdf.txt")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := RenderPageMats(r); err != ErrMalformedPdf {
		t.Fatal(err)
	}

	// Check an image file. The MagickWand library is able to handle all kinds
	// of images, so it's conceivable that would fail silently. We need to
	// ensure that it doesn't.

	r, err = testData.Open("testdata/not_a_pdf.jpg")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := RenderPageMats(r); err != ErrMalformedPdf {
		t.Fatal(err)
	}
}
