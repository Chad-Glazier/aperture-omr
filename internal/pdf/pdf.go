/*
This package wraps the ImageMagick library (which wraps GhostScript) to convert
PDFs into other formats.
*/
package pdf

import (
	"errors"
	"fmt"
	"image"
	"io"
	"runtime"
	"sync"
	"sync/atomic"

	pdfcpu "github.com/pdfcpu/pdfcpu/pkg/api"
	"gocv.io/x/gocv"
	"golang.org/x/sync/errgroup"
)

var (
	ErrMalformedPdf      = errors.New("the given file does not form a PDF")
	ErrPageCountMismatch = errors.New("the given PDF page count is not a multiple of the batch size")
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

var inUse sync.Mutex

// Represents a batch of pages in a PDF. If there was an error in the rendering
// of this batch it will be included in the Error field (and the Pages field
// will be nil).
//
// It's important that a batch be closed after the caller is done using it.
type Batch struct {
	mats  *PageMats
	Pages []*gocv.Mat
	Index uint32
	Error error
}

func (b *Batch) Close() {
	if b.mats != nil {
		b.mats.Close()
	}
	b.mats = nil
	b.Pages = nil
	b.Index = 0
	b.Error = nil
}

// Renders a large PDF by dividing it into batches of a fixed size. Once a
// given batch is processed and converted to matrices it will be passed
// through the returned channel (the total number of batches is given by the
// second return value). It's important that each batch is closed as soon as
// it's done being used.
//
// Batches will preserve their original order. E.g., if the batch size is 3,
// then the first batch (index 0) will include pages 1-3, the next batch will
// include pages 4-6, and so on. The pages within a batch also match their
// original order.
//
// The parallelization argument determines how many batches will be processed
// simultaneously. Setting it to zero will fall back to using GOMAXPROCS. It
// is worth noting that the amount of memory required by this procedure at any
// given time can be estimated by the following equation:
//
//     memory ∝ parallelization * (density * batchSize + overhead)
//
// If this function returns an error, it will either be because the reader does
// not describe a PDF (ErrMalformedPdf) or the page count of the PDF is not
// a multiple of the batch size (ErrPageCountMismatch). Other kinds of errors,
// like those that arise during the actual rendering of a PDF batch, are
// attached to the batch that concerns them. The occurrence of such an error
// will not halt this process.
func RenderPageBatches(
	r io.ReadSeeker,
	density int,
	batchSize int,
	parallelization int,
) (chan Batch, int, error) {

	// PDF rendering is hard, and it uses a lot of memory. We could, at some
	// point, implement a sophisticated scheduler to manage things and limit
	// the resources being used. In the meantime, we're just going to say "only
	// one process at a time".
	inUse.Lock()
	defer inUse.Unlock()

	if parallelization <= 0 {
		parallelization = runtime.GOMAXPROCS(0)
	}

	conf := pdfcpu.LoadConfiguration()
	pageCount, err := pdfcpu.PageCount(r, conf)
	if err != nil {
		return nil, 0, ErrMalformedPdf
	}

	if pageCount%batchSize != 0 {
		return nil, 0, ErrPageCountMismatch
	}

	spans, err := pdfcpu.SplitRaw(r, batchSize, conf)
	if err != nil {
		return nil, 0, ErrMalformedPdf
	}

	nextSpanIdx := atomic.Uint32{}
	threadCount := atomic.Int32{}

	ch := make(chan Batch, parallelization)

	for range parallelization {
		go func() {

			//
			// We keep each thread running until the pool of PDF batches is
			// complete.
			//

			threadCount.Add(1)

			for nextSpanIdx.Load() < uint32(len(spans)) {
				spanIdx := nextSpanIdx.Add(1) - 1

				buf, err := io.ReadAll(spans[spanIdx].Reader)
				if err != nil {
					ch <- Batch{Index: spanIdx, Error: err}
					continue
				}

				mats, err := pdfBytesToMats(buf, density)
				if err != nil {
					ch <- Batch{Index: spanIdx, Error: err}
					continue
				}

				ch <- Batch{
					mats:  mats,
					Pages: mats.Pages,
					Index: spanIdx,
					Error: nil,
				}
			}

			if threadCount.Add(-1) == 0 {
				close(ch)
			}

		}()
	}

	return ch, len(spans), nil
}
