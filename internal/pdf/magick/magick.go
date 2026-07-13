/*
This package wraps the ImageMagick library to perform PDF to image conversion.
*/
package magick

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
	ErrIndexOutOfBounds = errors.New("page index out of bounds")
	ErrUnknown          = errors.New("unknown error rendering pdf")
)

// Determines the dots per inch (DPI) when rendering the PDF. Lower values are
// much faster to compute but lead to poorer quality images. 300 DPI would be
// very high resolution while 74 DPI would be low resolution.
const density = 200

// Renders a single page from a PDF file to a grayscale image.
func PdfPageToGray(path string, page int) (*image.Gray, error) {

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
			return nil, ErrIndexOutOfBounds
		default:
			return nil, ErrUnknown
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

func PdfToGrayPages(path string) (map[int]*image.Gray, error) {
	workers := runtime.NumCPU()

	var nextPage atomic.Int64
	var finished atomic.Bool

	results := make(map[int]*image.Gray)
	var mu sync.Mutex

	g, _ := errgroup.WithContext(context.Background())

	for i := 0; i < workers; i++ {
		g.Go(func() error {
			for {
				if finished.Load() {
					return nil
				}

				page := int(nextPage.Add(1) - 1)

				img, err := PdfPageToGray(path, page)

				if errors.Is(err, ErrIndexOutOfBounds) {
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

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}
