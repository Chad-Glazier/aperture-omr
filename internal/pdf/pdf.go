/*
This package wraps the ImageMagick library to convert PDFs into other formats.
*/
package pdf

import (
	"errors"
	"fmt"
	"image"
	"io"
	"runtime"

	pdfcpu "github.com/pdfcpu/pdfcpu/pkg/api"
	"gocv.io/x/gocv"
	"golang.org/x/sync/errgroup"
)

var (
	ErrMalformedPdf = errors.New("the given file does not form a PDF")
)

// Interprets the reader as a stream of bytes representing a PDF file. The
// output is a slice of grayscale OpenCV matrices representing the PDF's pages
// in their original order.
//
// The density parameter is the DPI used when rasterizing the image. Higher DPI
// are much slower to compute but provide higher quality images. As a rule of
// thumb, you should keep it between 300 (high resolution) and 74 (low
// resolution).
//
// Returns ErrMalformedPdf if the given file was not a proper PDF.
func RenderPageMats(r io.ReadSeeker, density int) ([]*gocv.Mat, error) {

	pdfs, err := splitEven(r, runtime.NumCPU())
	if err != nil {
		return nil, ErrMalformedPdf
	}

	results := make([][]*image.Gray, len(pdfs))

	wg := errgroup.Group{}
	for i, pdf := range pdfs {
		wg.Go(func() error {

			images, err := pdfBytesToGrays(pdf, density)
			if err != nil {
				return err
			}

			results[i] = images
			return nil

		})
	}
	if err := wg.Wait(); err != nil {
		return nil, err
	}

	mats := make([]*gocv.Mat, 0)
	for i := range results {
		for j := range results[i] {

			mat, err := gocv.ImageGrayToMatGray(results[i][j])
			if err != nil {
				for _, m := range mats {
					m.Close()
				}
				return nil, err
			}

			mats = append(mats, &mat)

		}
	}

	return mats, nil

}

// Divides a single PDF buffer into multiple valid PDF files. Pages from the
// original are evenly distributed among the returned sub-PDFs.
//
// In the case that the original PDF has fewer pages than the number of buckets
// specified, the number of sub-PDFs made will be equal to the number of pages.
func splitEven(pdfData io.ReadSeeker, buckets int) ([][]byte, error) {

	if buckets <= 0 {
		return nil, fmt.Errorf("splitEven: number of buckets must be positive")
	}

	conf := pdfcpu.LoadConfiguration()
	pageCount, err := pdfcpu.PageCount(pdfData, conf)
	if err != nil {
		return nil, err
	}

	span := pageCount / buckets
	if pageCount%buckets != 0 {
		span++
	}

	results, err := pdfcpu.SplitRaw(pdfData, span, conf)
	if err != nil {
		return nil, err
	}

	bufs := make([][]byte, len(results))
	for i := range bufs {
		buf, err := io.ReadAll(results[i].Reader)
		if err != nil {
			return nil, err
		}
		bufs[i] = buf
	}

	return bufs, nil
}
