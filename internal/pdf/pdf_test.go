package pdf

import (
	"bytes"
	"embed"
	"fmt"
	"reflect"
	"testing"

	"github.com/Chad-Glazier/aperture-omr/internal/omr"
	"github.com/gen2brain/go-fitz"
	"gotest.tools/v3/assert"
)

//go:embed testdata/*
var testData embed.FS

func TestRenderPageBlocks_OK(t *testing.T) {

	// Note that the large sample PDF has 88 pages.
	buf, err := testData.ReadFile("testdata/sample_large.pdf")
	assert.Assert(t, err == nil)

	batches, nBatches, err := RenderPageBlocks(
		bytes.NewReader(buf),
		74,
		2,
		0,
	)
	assert.Assert(t, err == nil)
	assert.Assert(t, nBatches == 88/2)

	nBatchesRendered := 0
	for batch := range batches {
		nBatchesRendered++

		assert.Assert(t, batch.Error() == nil)

		// Ensure that each page's matrix is a well-formed image.
		for _, page := range batch.Pages() {
			w := bytes.Buffer{}
			_, err := omr.EncodeMatToImage(&w, omr.ImageEncodingJpeg, page)
			assert.Assert(t, err == nil)
		}
	}

	// Ensure that each page was rendered.
	assert.Assert(t, nBatchesRendered == 88/2)
}

func TestRenderPageBlocks_MalformedData(t *testing.T) {

	//
	// This test checks how the renderer handles non-PDF data being passed to
	// it.
	//

	// Check a text file. This should not be parseable at all by the PDF
	// renderer.
	buf, err := testData.ReadFile("testdata/not_a_pdf.txt")
	assert.Assert(t, err == nil)

	_, _, err = RenderPageBlocks(
		bytes.NewReader(buf),
		74,
		2,
		0,
	)
	assert.Assert(t, err == ErrMalformedPdf)

	// Check an image file. This should also not be parseable.
	buf, err = testData.ReadFile("testdata/not_a_pdf.jpg")
	assert.Assert(t, err == nil)

	_, _, err = RenderPageBlocks(
		bytes.NewReader(buf),
		74,
		2,
		0,
	)
	assert.Assert(t, err == ErrMalformedPdf)
}

func TestRenderPageBlocks_PageMismatch(t *testing.T) {
	// The large sample PDF has 88 pages.
	buf, err := testData.ReadFile("testdata/sample_large.pdf")
	assert.Assert(t, err == nil)

	// 3 does not divide 88, so we expect this to err.
	_, _, err = RenderPageBlocks(
		bytes.NewReader(buf),
		74,
		3,
		0,
	)
	assert.Assert(t, err == ErrPageCountMismatch)
}

//
// Test the helper functions.
//

func TestBlockPartition(t *testing.T) {
	tt := []struct {
		name            string
		n               uint
		blockSize       uint
		numberOfSubsets uint
		want            []blockedSubset
		err             error
	}{
		{
			name:            "single subset",
			n:               12,
			blockSize:       4,
			numberOfSubsets: 1,
			want: []blockedSubset{
				{
					from: 1,
					thru: 12,
					blocks: []block{
						{from: 1, thru: 4},
						{from: 5, thru: 8},
						{from: 9, thru: 12},
					},
				},
			},
		},
		{
			name:            "two equal subsets",
			n:               12,
			blockSize:       2,
			numberOfSubsets: 2,
			want: []blockedSubset{
				{
					from: 1,
					thru: 6,
					blocks: []block{
						{from: 1, thru: 2},
						{from: 3, thru: 4},
						{from: 5, thru: 6},
					},
				},
				{
					from: 7,
					thru: 12,
					blocks: []block{
						{from: 7, thru: 8},
						{from: 9, thru: 10},
						{from: 11, thru: 12},
					},
				},
			},
		},
		{
			name:            "three equal subsets",
			n:               12,
			blockSize:       2,
			numberOfSubsets: 3,
			want: []blockedSubset{
				{
					from: 1,
					thru: 4,
					blocks: []block{
						{from: 1, thru: 2},
						{from: 3, thru: 4},
					},
				},
				{
					from: 5,
					thru: 8,
					blocks: []block{
						{from: 5, thru: 6},
						{from: 7, thru: 8},
					},
				},
				{
					from: 9,
					thru: 12,
					blocks: []block{
						{from: 9, thru: 10},
						{from: 11, thru: 12},
					},
				},
			},
		},
		{
			name:            "uneven subsets",
			n:               20,
			blockSize:       2,
			numberOfSubsets: 3,
			want: []blockedSubset{
				{
					from: 1,
					thru: 8,
					blocks: []block{
						{from: 1, thru: 2},
						{from: 3, thru: 4},
						{from: 5, thru: 6},
						{from: 7, thru: 8},
					},
				},
				{
					from: 9,
					thru: 14,
					blocks: []block{
						{from: 9, thru: 10},
						{from: 11, thru: 12},
						{from: 13, thru: 14},
					},
				},
				{
					from: 15,
					thru: 20,
					blocks: []block{
						{from: 15, thru: 16},
						{from: 17, thru: 18},
						{from: 19, thru: 20},
					},
				},
			},
		},
		{
			name:            "one block per subset",
			n:               12,
			blockSize:       4,
			numberOfSubsets: 3,
			want: []blockedSubset{
				{
					from:   1,
					thru:   4,
					blocks: []block{{from: 1, thru: 4}},
				},
				{
					from:   5,
					thru:   8,
					blocks: []block{{from: 5, thru: 8}},
				},
				{
					from:   9,
					thru:   12,
					blocks: []block{{from: 9, thru: 12}},
				},
			},
		},
		{
			name:            "invalid block size",
			n:               10,
			blockSize:       4,
			numberOfSubsets: 2,
			err:             ErrInvalidBlockSize,
		},
		{
			name:            "more subsets than blocks",
			n:               8,
			blockSize:       4,
			numberOfSubsets: 3,
			want: []blockedSubset{
				{
					from:   1,
					thru:   4,
					blocks: []block{{from: 1, thru: 4}},
				},
				{
					from:   5,
					thru:   8,
					blocks: []block{{from: 5, thru: 8}},
				},
			},
		},
		{
			name:            "zero pages",
			n:               0,
			blockSize:       1,
			numberOfSubsets: 3,
			want:            []blockedSubset{},
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			got, err := blockPartition(
				test.n,
				test.blockSize,
				test.numberOfSubsets,
			)

			assert.Assert(t, err == test.err)

			if err != nil {
				return
			}
			assert.Assert(t, reflect.DeepEqual(got, test.want))
		})
	}
}

func TestRenderPages(t *testing.T) {

	f, err := testData.Open("testdata/sample_large.pdf")
	assert.Assert(t, err == nil)

	doc, err := fitz.NewFromReader(f)
	assert.Assert(t, err == nil)

	tt := []struct {
		startIdx uint
		count    int
		err      error
	}{
		{1, 2, nil},
		{6, 88, ErrPageOutOfBounds},
		{0, 2, nil},
		{4, 2, nil},
		{88, 0, ErrPageOutOfBounds},
	}

	for _, test := range tt {
		name := fmt.Sprintf(
			"startIdx=%d count=%d (88 pages)",
			test.startIdx, test.count,
		)
		if test.err != nil {
			name += fmt.Sprintf("; error=%s", test.err.Error())
		}
		t.Run(name, func(t *testing.T) {
			mats, err := renderPages(doc, 74, test.startIdx, uint(test.count))
			defer omr.CloseAll(mats)
			
			assert.Assert(t, err == test.err)
			if err != nil {
				return
			}
			assert.Assert(t, len(mats) == test.count)
		})
	}
}

func TestPageSpan(t *testing.T) {
	buf, err := testData.ReadFile("testdata/sample_large.pdf")
	if err != nil {
		t.Fatal(err)
	}
	r := bytes.NewReader(buf)

	ctx, err := splittingContext(r)
	if err != nil {
		t.Fatal(err)
	}

	tt := []struct {
		from uint
		thru uint
		err  error
	}{
		{1, 2, nil},
		{6, 88, nil},
		{0, 2, ErrPageOutOfBounds},
		{4, 2, ErrBadInput},
		{56, 102, ErrPageOutOfBounds},
	}

	for _, test := range tt {
		name := fmt.Sprintf(
			"from=%d thru=%d (88 pages)",
			test.from, test.thru,
		)
		if test.err != nil {
			name += fmt.Sprintf("; error=%s", test.err.Error())
		}
		t.Run(name, func(t *testing.T) {
			buf := bytes.Buffer{}
			err := pageSpan(&buf, ctx, test.from, test.thru)
			if err != test.err {
				t.Errorf("err: got %v, want %v", err, test.err)
			}

			if test.err != nil {
				return
			}
		})
	}
}
