/*
This package wraps the ImageMagick library (which wraps GhostScript) to convert
PDFs into other formats.
*/
package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"gocv.io/x/gocv"
)

var (
	ErrMalformedPdf      = errors.New("the given file does not form a PDF")
	ErrPageCountMismatch = errors.New("the given PDF page count is not a multiple of the batch size")
)

// Represents a batch of pages in a PDF. If there was an error in the rendering
// of this batch it will be included in the Error field (and the Pages field
// will be nil).
//
// It's important that a batch be closed after the caller is done using it.
type Batch struct {
	mats  *PageMats
	Pages []*gocv.Mat
	Index uint32
	// The page number (1-based) of the original PDF where this batch starts
	// (inclusive).
	From uint32
	// The page number (1-based) of the original PDF where this batch ends
	// (inclusive).
	Thru  uint32
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

var inUse sync.Mutex

// Renders a large PDF by dividing it into batches of a fixed size. Once a
// given batch is processed and converted to matrices it will be passed
// through the returned channel (the total number of batches is given by the
// second return value). It's important that each batch is closed as soon as
// it's done being used. Notably, the reader given to this function may also be
// closed when it returns.
//
// Batches will preserve their original order. E.g., if the batch size is 3,
// then the first batch (index 0) will include pages 1-3, the next batch will
// include pages 4-6, and so on. The pages within a batch also match their
// original order.
//
// The parallelization argument determines how many batches will be processed
// simultaneously. Setting it to zero will fall back to using GOMAXPROCS.
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

	//
	// PDF rendering is hard, and it uses a lot of memory. We could, at some
	// point, implement a sophisticated scheduler to manage things and limit
	// the resources being used. In the meantime, we're just going to say "only
	// one render at a time".
	//

	inUse.Lock()
	defer inUse.Unlock()

	if parallelization <= 0 {
		parallelization = runtime.GOMAXPROCS(0)
	}
	//
	// We split the PDF below and then run the threads.
	//

	conf := api.LoadConfiguration()

	spans, err := Split(r, batchSize, conf)
	if err != nil {
		return nil, 0, err
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

			for {
				spanIdx := nextSpanIdx.Add(1) - 1
				if spanIdx >= uint32(len(spans)) {
					break
				}

				pdf := spans[spanIdx]

				mats, err := pdfBytesToMats(
					pdf.Buf.Bytes(),
					density,
				)
				pdf.Buf.Reset()
				if err != nil {
					ch <- Batch{
						Index: spanIdx,
						Error: err,
						From:  pdf.From,
						Thru:  pdf.Thru,
					}
					continue
				}

				ch <- Batch{
					mats:  mats,
					Pages: mats.Pages,
					Index: spanIdx,
					From:  pdf.From,
					Thru:  pdf.Thru,
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

// Represents a complete PDF file that was formed from a larger one.
type SubPdf struct {
	Buf bytes.Buffer
	// The page number (1-based) of the original PDF where this sub-PDF starts.
	From uint32
	// The page number (1-based) of the original PDF where this sub-PDF ends
	// (inclusive).
	Thru uint32
}

// This function splits a PDF into a number of smaller PDFs, each with a number
// of pages equal to the given "span." I.e., if the span is 1, then the
// resulting PDFs each have one page.
//
// If the given reader doesn't form a proper PDF, then ErrMalformedPdf will be
// returned. If the given span does not divide the total pages in the PDF, then
// ErrPageCountMismatch will be returned.
//
// Note that the package we're using (pdfcpu) has a version of this function
// already, "SplitRaw." This function internally builds a buffer but it only
// exposes a reader. Since we need a buffer, using that function would require
// us to first copy everything from that reader (BAD!).
func Split(
	r io.ReadSeeker,
	span int,
	conf *model.Configuration,
) (pdfs []SubPdf, err error) {
	if span <= 0 {
		return nil, fmt.Errorf("Split: page span must be positive")
	}
	if r == nil {
		return nil, fmt.Errorf("Split: received nil reader")
	}

	defer fault.Catch(&err)

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.SPLIT

	ctx, err := api.ReadValidateAndOptimize(r, conf)
	if err != nil {
		return nil, ErrMalformedPdf
	}

	if ctx.PageCount%span != 0 {
		return nil, ErrPageCountMismatch
	}

	pdfs = make([]SubPdf, ctx.PageCount/span)
	for i := range pdfs {
		start := i * span
		from := start + 1
		thru := start + span
		pdf, err := pageSpan(ctx, from, thru)
		if err != nil {
			return nil, err
		}
		pdfs[i] = pdf
	}

	return pdfs, nil
}

// A reimplementation of pdfcpu's unexported "pageSpan" function.
func pageSpan(
	ctx *model.Context,
	from, thru int,
) (SubPdf, error) {
	ctxNew, err := pdfcpu.ExtractPages(
		ctx,
		api.PagesForPageRange(from, thru),
		false,
	)
	if err != nil {
		return SubPdf{}, err
	}

	var b bytes.Buffer
	if err := api.WriteContext(ctxNew, &b); err != nil {
		return SubPdf{}, err
	}

	ctxNew.Write.Fp.Close()

	return SubPdf{
		From: uint32(from),
		Thru: uint32(thru),
		Buf:  b,
	}, nil
}
