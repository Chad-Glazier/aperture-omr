package pdf

import (
	"bytes"
	"embed"
	"sync/atomic"
	"testing"

	pdfcpu "github.com/pdfcpu/pdfcpu/pkg/api"
	"gocv.io/x/gocv"
)

//go:embed testdata/*
var testData embed.FS

func TestRenderPageMats(t *testing.T) {

	// The large sample PDF has 88 pages.
	buf, err := testData.ReadFile("testdata/sample_large.pdf")
	if err != nil {
		t.Fatal(err)
	}

	mats, err := RenderPageMats(bytes.NewReader(buf), 74)
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
	if len(mats) != 88 {
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
	buf, err := testData.ReadFile("testdata/not_a_pdf.txt")
	if err != nil {
		t.Fatal(err)
	}

	_, err = RenderPageMats(bytes.NewReader(buf), 74)
	if err != ErrMalformedPdf {
		t.Fatal(err)
	}

	// Check an image file. The MagickWand library is able to handle all kinds
	// of images, so it's conceivable that would fail silently. We need to
	// ensure that it doesn't.
	buf, err = testData.ReadFile("testdata/not_a_pdf.jpg")
	if err != nil {
		t.Fatal(err)
	}

	_, err = RenderPageMats(bytes.NewReader(buf), 74)
	if err != ErrMalformedPdf {
		t.Fatal(err)
	}
}

func TestSplitEven(t *testing.T) {

	// The sample PDF has 5 pages.
	buf, err := testData.ReadFile("testdata/sample.pdf")
	if err != nil {
		t.Fatal(err)
	}

	//
	// Try dividing 5 pages among 3 buckets.
	//

	pdfs, err := splitEven(bytes.NewReader(buf), 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(pdfs) != 3 {
		t.Fatalf("expected %d PDF files, got %d", 3, len(pdfs))
	}

	conf := pdfcpu.LoadConfiguration()
	for _, pdf := range pdfs {
		if err := pdfcpu.Validate(bytes.NewReader(pdf), conf); err != nil {
			t.Fatal(err)
		}
	}

	//
	// Try dividing 5 pages among 1 bucket.
	//

	pdfs, err = splitEven(bytes.NewReader(buf), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(pdfs) != 1 {
		t.Fatalf("expected %d PDF files, got %d", 1, len(pdfs))
	}

	for _, pdf := range pdfs {
		if err := pdfcpu.Validate(bytes.NewReader(pdf), conf); err != nil {
			t.Fatal(err)
		}
	}

	//
	// Try dividing 5 pages among 6 buckets.
	//

	pdfs, err = splitEven(bytes.NewReader(buf), 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(pdfs) != 5 {
		t.Fatalf("expected %d PDF files, got %d", 5, len(pdfs))
	}

	for _, pdf := range pdfs {
		if err := pdfcpu.Validate(bytes.NewReader(pdf), conf); err != nil {
			t.Fatal(err)
		}
	}
}


func TestRenderPageBatches(t *testing.T) {

	// The large sample PDF has 88 pages.
	buf, err := testData.ReadFile("testdata/sample_large.pdf")
	if err != nil {
		t.Fatal(err)
	}

	matsProcessed := atomic.Int32{}
	err = RenderPageBatches(
		bytes.NewReader(buf), 
		300,
		2,
		8,
		func(mats []*gocv.Mat, batchIdx uint32) {
			// Ensure that the matrices are well-formed images.
			for _, mat := range mats {
				buf, err := gocv.IMEncode(gocv.PNGFileExt, *mat)
				matsProcessed.Add(1)
				if err != nil {
					t.Fatal(err)
				}
				buf.Close()
			}
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure that there is one matrix per page.
	if matsProcessed.Load() != 88 {
		t.Fatalf(
			"pdf had %d pages but only %d matrices were rendered",
			5, matsProcessed.Load(),
		)
	}
}
