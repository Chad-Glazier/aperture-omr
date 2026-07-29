package pdf

import (
	"bytes"
	"embed"
	"testing"

	"gocv.io/x/gocv"
)

//go:embed testdata/*
var testData embed.FS

func TestRenderPageBatches_OK(t *testing.T) {

	// The large sample PDF has 88 pages.
	buf, err := testData.ReadFile("testdata/sample_large.pdf")
	if err != nil {
		t.Fatal(err)
	}

	batches, nBatches, err := RenderPageBatches(
		bytes.NewReader(buf),
		74,
		2,
		0,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}

	if nBatches != 88/2 {
		t.Fatalf(
			"pdf had %d pages but only %d %d-page batches were prepared",
			88, nBatches, 2,
		)
	}

	nBatchesRendered := 0
	for batch := range batches {
		nBatchesRendered++

		if batch.Error != nil {
			t.Fatal(batch.Error)
		}

		// Ensure that each page's matrix is a well-formed image.
		for _, page := range batch.Pages {
			_, err := gocv.IMEncode(gocv.PNGFileExt, *page)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	// Ensure that each page was rendered.
	if nBatchesRendered != 88/2 {
		t.Fatalf(
			"pdf had %d pages but only %d %d-page batches were rendered",
			88, nBatchesRendered, 2,
		)
	}
}

func TestRenderPageBatches_MalformedData(t *testing.T) {

	//
	// This test checks how the renderer handles non-PDF data being passed to
	// it.
	//

	// Check a text file. This should not be parseable at all by the PDF
	// renderer.
	buf, err := testData.ReadFile("testdata/not_a_pdf.txt")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = RenderPageBatches(
		bytes.NewReader(buf),
		74,
		2,
		0,
		0,
	)
	if err != ErrMalformedPdf {
		t.Fatal("expected ErrMalformedPdf error")
	}

	// Check an image file. The MagickWand library is able to handle all kinds
	// of images, so it's conceivable that it would fail silently. We need to
	// ensure that it doesn't.
	buf, err = testData.ReadFile("testdata/not_a_pdf.jpg")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = RenderPageBatches(
		bytes.NewReader(buf),
		74,
		2,
		0,
		0,
	)
	if err != ErrMalformedPdf {
		t.Fatal("expected ErrMalformedPdf error")
	}
}

func TestRenderPageBatches_PageMismatch(t *testing.T) {
	// The large sample PDF has 88 pages.
	buf, err := testData.ReadFile("testdata/sample_large.pdf")
	if err != nil {
		t.Fatal(err)
	}

	// 3 does not divide 88, so we expect this to err.
	_, _, err = RenderPageBatches(
		bytes.NewReader(buf),
		74,
		3,
		0,
		0,
	)
	if err != ErrPageCountMismatch {
		t.Fatalf("expected ErrPageCountMismatch, got %s", err.Error())
	}
}
