package pdf

// #cgo pkg-config: MagickWand
//
// #include <stdlib.h>
// #include "magick.h"
import "C"

import (
	"context"
	"errors"
	"image"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sync/errgroup"
)

var (
	errIndexOutOfBounds = errors.New("page index out of bounds")
	errUnkown          = errors.New("unknown error rendering pdf")
)

// Determines the dots per inch (DPI) when rendering the PDF. Lower values are
// much faster to compute but lead to poorer quality images. 300 DPI would be
// very high resolution while 74 DPI would be very low resolution.
const density = 200

// Counts the number of pages in a PDF without rendering it. This function is
// still noticeably slow; don't use it unless necessary.
func pdfPageCount(path string) (int, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	
	pageCount := int(C.pdf_file_page_count(cPath))

	if pageCount == -1 {
		return 0, errors.New("error loading file")
	}

	return pageCount, nil
}

// Renders a single page from a PDF file to a grayscale OpenCV matrix.
func pdfPageToGray(path string, page int) (*image.Gray, error) {

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var img C.GrayImage
	status := C.pdf_file_page_to_gray(
		cPath,
		C.int(density),
		C.int(page),
		&img,
	)
	if status != C.PDF_OK {
		switch status {
		case C.PDF_PAGE_NOT_FOUND:
			return nil, errIndexOutOfBounds
		case C.PDF_LOADING_ERROR:
			return nil, ErrMalformedPdf
		default:
			return nil, errUnkown
		}
	}

	defer C.free_gray_image(&img)

	width := int(img.width)
	height := int(img.height)

	pix := C.GoBytes(
		unsafe.Pointer(img.pixels),
		C.int(width*height),
	)

	return &image.Gray{
		Pix:    pix,
		Stride: width,
		Rect:   image.Rect(0, 0, width, height),
	}, nil
}

// Renders all pages from a PDF file into a slice of grayscale images, each of
// which represents a page. The order of pages in the slice matches the order
// that they appear in the PDF.
func pdfToGrayPages(path string) ([]*image.Gray, error) {
	workers := runtime.NumCPU()

	var nextPage atomic.Int64
	var finished atomic.Bool

	results := make(map[int]*image.Gray)
	var mu sync.Mutex

	wg, _ := errgroup.WithContext(context.Background())

	for range workers {
		wg.Go(func() error {
			for {
				if finished.Load() {
					return nil
				}

				page := int(nextPage.Add(1) - 1)

				img, err := pdfPageToGray(path, page)
				if err == errIndexOutOfBounds {
					finished.Store(true)
					return nil
				}

				if err != nil {
					return err
				}

				mu.Lock()
				results[page] = img
				mu.Unlock()
			}
		})
	}

	if err := wg.Wait(); err != nil {
		return nil, err
	}

	pages := make([]*image.Gray, len(results))
	for idx, img := range results {
		pages[idx] = img
	}

	return pages, nil
}
