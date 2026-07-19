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
	"sync/atomic"

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

	pdfs, err := splitEven(r, runtime.GOMAXPROCS(0))
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

// Renders a large PDF by dividing it into batches of a fixed size. Once a 
// given batch is processed, the callback will be invoked on the rendered 
// matrices. Notably, these matrices will be freed after the callback returns.
// 
// The parallelization argument determines how many batches will be processed 
// simultaneously. Setting it to zero will fall back to using GOMAXPROCS.
//
// Batches will preserve their original order. E.g., if the batch size is 3,
// pages 1-3 will be one batch, pages 4-6 will be another, and so on. If the
// original PDF has a number of pages that isn't a multiple of the batch size,
// an error will be returned. The pages of each batch are also passed in order,
// and the accompanying integer is the batch's index.
func RenderPageBatches(
	r io.ReadSeeker, 
	density int,
	batchSize int,
	parallelization int,
	callback func ([]*gocv.Mat, uint32),
) error {

	conf := pdfcpu.LoadConfiguration()
	pageCount, err := pdfcpu.PageCount(r, conf)
	if err != nil {
		return err
	}
	
	if pageCount%batchSize != 0	{
		return fmt.Errorf(
			"RenderPageBatches: PDF pages %d not divisible by batch size %d",
			pageCount, batchSize,
		)
	}

	spans, err := pdfcpu.SplitRaw(r, batchSize, conf)
	if err != nil {
		return err
	}
	
	nextSpanIdx := atomic.Uint32{}
	nextSpanIdx.Store(0)

	wg := errgroup.Group{}
	for range parallelization {
		wg.Go(func() (err error) {

			// We track the currently-allocated matrix slice so that, if the
			// callback panics, we can free it with the recovery call.
			var mats *MatSlice
			defer func() {
				if r := recover(); r != nil {
					if mats != nil {
						mats.Close()
					}
					err = fmt.Errorf("RenderPageBatches: panic recovered")
				}
			}()
			
			//
			// We keep each thread running until the pool of PDF batches is
			// complete.
			//

			for nextSpanIdx.Load() < uint32(len(spans)) {

				spanIdx := nextSpanIdx.Add(1)-1
				
				buf, err := io.ReadAll(spans[spanIdx].Reader)
				if err != nil {
					return err
				}
				spans[spanIdx] = nil // Help the GC?

				mats, err = pdfBytesToMats(buf, density)
				callback(mats.Mats, spanIdx)				
				mats.Close()
				buf = nil
			}

			return nil
		})
	}
	if err := wg.Wait(); err != nil {
		return err
	}

	return nil
}
