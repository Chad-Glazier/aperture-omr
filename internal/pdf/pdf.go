package pdf

import (
	"errors"
	"io"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"

	"github.com/Chad-Glazier/aperture-omr/internal/omr"
	"github.com/gen2brain/go-fitz"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const (
	MaxDpi              uint = 300
	MaxConcurrentBlocks uint = 6
)

var (
	ErrBadInput          = errors.New("pdf: nonsensical input. Likely a programmer error")
	ErrPageOutOfBounds   = errors.New("pdf: page index is out of bounds for the document")
	ErrMalformedPdf      = errors.New("pdf: the given file does not form a PDF")
	ErrPageCountMismatch = errors.New("pdf: the given PDF page count is not a multiple of the block size")
	ErrInvalidBlockSize  = errors.New("pdf: the given block size does not divide the given n")
)

type pageBlock struct {
	pages []omr.Mat
	err   error
	from  uint
	thru  uint
}

var _ omr.PageSet = (*pageBlock)(nil)

func (p pageBlock) Pages() []omr.Mat {
	return p.pages
}

func (p pageBlock) Metadata() map[string]string {
	return map[string]string{
		"from": strconv.FormatUint(uint64(p.from), 10),
		"thru": strconv.FormatUint(uint64(p.thru), 10),
	}
}

func (p pageBlock) Error() error {
	return p.err
}

func RenderPageBlocks(
	r io.ReadSeeker,
	dpi uint,
	blockSize uint,
	parallelism uint,
) (<-chan omr.PageSet, uint, error) {

	dpi = min(MaxDpi, dpi)

	if parallelism == 0 {
		parallelism = min(MaxConcurrentBlocks, uint(runtime.GOMAXPROCS(0))/2)
	}

	parallelism = min(MaxConcurrentBlocks, parallelism)

	// The number of allotted threads should be scaled down when the block size
	// is above 2. Most exams won't be more than two pages, but the possibility
	// still needs to be addressed because it can spike the memory if we hold
	// a bunch of large blocks simultaneously. (It's also worth mentioning that
	// this equation can return a value greater than MaxConcurrentBlocks when
	// blocks are 1-page, but that's fine.)
	parallelism = uint(max(1, int(parallelism)+2-int(blockSize)))

	docs, err := blockPartitionPdf(r, blockSize, parallelism)
	if err != nil {
		return nil, 0, err
	}

	numBlocks := uint(0)
	for _, doc := range docs {
		numBlocks += uint(len(doc.subset.blocks))
	}

	threadCount := atomic.Int32{}
	out := make(chan omr.PageSet)

	for i := range docs {
		go func() {
			threadCount.Add(1)
			defer func() {
				if threadCount.Add(-1) == 0 {
					close(out)
				}
			}()

			doc := docs[i]
			defer doc.close()

			for i, block := range doc.subset.blocks {
				startIdx := uint(i) * blockSize
				mats, err := renderPages(doc.doc, dpi, startIdx, blockSize)
				if err != nil {
					out <- pageBlock{
						from: block.from,
						thru: block.thru,
						err:  err,
					}
					continue
				}

				out <- pageBlock{
					pages: mats,
					from:  block.from,
					thru:  block.thru,
				}
			}
		}()
	}

	return out, numBlocks, nil
}

type blockedSubdoc struct {
	subset blockedSubset
	doc    *fitz.Document
	file   *os.File
}

func (b *blockedSubdoc) close() {
	b.doc.Close()
	b.file.Close()
	os.Remove(b.file.Name())
}

func blockPartitionPdf(
	pdf io.ReadSeeker,
	blockSize,
	numSubsets uint,
) ([]blockedSubdoc, error) {
	ctx, err := splittingContext(pdf)
	if err != nil {
		return nil, ErrMalformedPdf
	}

	subsets, err := blockPartition(uint(ctx.PageCount), blockSize, numSubsets)
	if err != nil {
		return nil, ErrPageCountMismatch
	}

	subDocs := make([]blockedSubdoc, 0, len(subsets))
	for _, subset := range subsets {

		f, err := os.CreateTemp("", "tmp_pdf_subset_*.pdf")
		if err != nil {
			for j := range subDocs {
				subDocs[j].close()
			}
			panic(err)
		}

		err = pageSpan(f, ctx, subset.from, subset.thru)
		if err != nil {
			f.Close()
			os.Remove(f.Name())
			for j := range subDocs {
				subDocs[j].close()
			}
			return nil, err
		}

		doc, err := fitz.New(f.Name())
		if err != nil {
			f.Close()
			os.Remove(f.Name())
			for j := range subDocs {
				subDocs[j].close()
			}
			return nil, ErrMalformedPdf
		}

		subDocs = append(subDocs, blockedSubdoc{
			subset: subset,
			doc:    doc,
			file:   f,
		})
	}

	return subDocs, nil
}

// Represents a discrete piece of a region. The boundaries are 1-based and
// inclusive.
type block struct {
	from uint
	thru uint
}

type blockedSubset struct {
	from   uint
	thru   uint
	blocks []block
}

// Returns a set of boundaries which form a roughly-even partition of the
// numbers 1 through n, divided into blocks. All blocks will be whole. I.e., if
// the block size is 4, then all subsets will have a total size that is a
// multiple of 4.
//
// The number of subsets is an upper bound. Empty subsets will be omitted from
// the returned slice.
//
// If the block size does not divide the given n, then a partitioning is
// impossible and [ErrInvalidBlockSize] is returned.
func blockPartition(n, blockSize, numSubsets uint) ([]blockedSubset, error) {
	if n%blockSize != 0 {
		return nil, ErrInvalidBlockSize
	}

	numBlocks := n / blockSize
	numSubsets = min(numBlocks, numSubsets)
	blocksPerRegion := make([]uint, numSubsets)
	idx := uint(0)
	for numBlocks > 0 {
		numBlocks--
		blocksPerRegion[idx]++

		idx += 1
		idx %= numSubsets
	}

	regions := make([]blockedSubset, numSubsets)
	from := uint(1)
	for i, blocks := range blocksPerRegion {
		regions[i] = blockedSubset{
			blocks: make([]block, blocks),
		}
		regions[i].from = from
		for j := range blocks {
			thru := from + blockSize - 1
			regions[i].blocks[j].from = from
			regions[i].blocks[j].thru = thru
			from = thru + 1
		}
		regions[i].thru = from - 1
	}

	return regions, nil
}

// Renders a specific number of pages from the given document beginning from
// the start index.
//
// If an error is returned, it will [ErrPageOutOfBounds] or an unnamed error
// from GoCV or Fitz.
func renderPages(
	doc *fitz.Document,
	dpi,
	startIdx,
	count uint,
) ([]omr.Mat, error) {

	if startIdx+count > uint(doc.NumPage()) || startIdx == uint(doc.NumPage()) {
		return nil, ErrPageOutOfBounds
	}

	mats := make([]omr.Mat, 0, count)

	for i := startIdx; i < startIdx+count; i++ {
		img, err := doc.ImageDPI(int(i), float64(dpi))
		if err != nil {
			omr.CloseAll(mats)
			return nil, err
		}

		mat, err := omr.RgbaToMat(img)
		if err != nil {
			omr.CloseAll(mats)
			return nil, err
		}

		mats = append(mats, mat)
	}

	return mats, nil
}

// Prepares a pdfcpu context for PDF splitting.
//
// If an error is returned, it will be [ErrMalformedPdf].
func splittingContext(r io.ReadSeeker) (*model.Context, error) {

	conf := model.NewDefaultConfiguration()
	conf.Cmd = model.SPLIT

	ctx, err := api.ReadValidateAndOptimize(r, conf)
	if err != nil {
		return nil, ErrMalformedPdf
	}

	return ctx, nil
}

// A reimplementation of pdfcpu's unexported "pageSpan" function. The "from"
// and "thru" values are 1-based page numbers, and the bounds are inclusive.
//
// If an error is returned, it will be [ErrPageOutOfBounds], [ErrBadInput]
// (when from > thru), or an unnamed error from pdfcpu.
func pageSpan(
	dst io.Writer,
	ctx *model.Context,
	from, thru uint,
) error {
	if from < 1 || thru > uint(ctx.PageCount) {
		return ErrPageOutOfBounds
	}

	if from > thru {
		return ErrBadInput
	}

	ctxNew, err := pdfcpu.ExtractPages(
		ctx,
		api.PagesForPageRange(int(from), int(thru)),
		false,
	)
	if err != nil {
		return nil
	}

	if err := api.WriteContext(ctxNew, dst); err != nil {
		return err
	}

	ctxNew.Write.Fp.Close()
	return nil
}
