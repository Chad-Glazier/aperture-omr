package pdf

import (
	"testing"

	"gocv.io/x/gocv"
)

func TestPdfBytesToMats_OK(t *testing.T) {

	// The sample PDF has 5 pages.
	pdf, err := testData.ReadFile("testdata/sample.pdf")
	if err != nil {
		t.Fatal(err)
	}

	mats, err := pdfBytesToMats(pdf, 74)
	if err != nil {
		t.Fatalf("pdfBytesToMats() returned error: %v", err)
	}
	defer mats.Close()

	if len(mats.Pages) != 5 {
		t.Fatalf("expected %d mats, got %d", 5, len(mats.Pages))
	}

	for i := range mats.Pages {
		_, err := gocv.IMEncode(gocv.PNGFileExt, *mats.Pages[i])
		if err != nil {
			t.Fatal(err)
		}
	}
}
