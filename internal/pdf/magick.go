package pdf

// #cgo pkg-config: MagickWand
//
// #include <stdlib.h>
// #include <wand/MagickWand.h>
// #include "magick.h"
import "C"

import (
	"errors"
	"image"
	"unsafe"
)

func init() {
	C.MagickWandGenesis() // initialize ImageMagick
}

type Status int

const (
	OK Status = iota
	LoadingError
	StartupError
	PageNotFound
	RenderError
	OutOfMemory
	ExportError
	WrongPageCount
)

func (s Status) Error() string {
	switch s {
	case OK:
		return "ok"
	case LoadingError:
		return "loading error"
	case StartupError:
		return "startup error"
	case PageNotFound:
		return "page not found"
	case RenderError:
		return "render error"
	case OutOfMemory:
		return "out of memory"
	case ExportError:
		return "export error"
	case WrongPageCount:
		return "wrong page count"
	default:
		return "unknown error"
	}
}

func pageCount(filename string) (int, error) {
	cname := C.CString(filename)
	defer C.free(unsafe.Pointer(cname))

	n := C.pdf_file_page_count(cname)
	if n < 0 {
		return 0, errors.New("failed to load PDF")
	}

	return int(n), nil
}

func pageToGray(filename string, density int, pageIdx int) (*image.Gray, error) {
	cname := C.CString(filename)
	defer C.free(unsafe.Pointer(cname))

	var status C.PdfStatus

	img := C.pdf_file_page_to_gray(
		cname,
		C.size_t(density),
		C.size_t(pageIdx),
		&status,
	)

	if img == nil {
		return nil, Status(status)
	}

	defer C.gray_image_destroy(img)

	return copyToGoMemory(img), nil
}

func pdfBytesToGrays(pdf []byte, density int) ([]*image.Gray, error) {
	if len(pdf) == 0 {
		return nil, errors.New("empty PDF")
	}

	var status C.PdfStatus

	slice := C.pdf_bytes_to_gray_images(
		unsafe.Pointer(&pdf[0]),
		C.size_t(len(pdf)),
		C.int(density),
		&status,
	)

	if slice == nil {
		return nil, Status(status)
	}

	defer C.gray_image_slice_destroy(slice)

	length := int(slice.length)
	if length == 0 {
		return []*image.Gray{}, nil
	}

	// Turn the C array into a Go slice view.
	cImages := unsafe.Slice(
		(*C.GrayImage)(unsafe.Pointer(slice.items)),
		length,
	)

	images := make([]*image.Gray, length)

	for i, img := range cImages {
		width := int(img.width)
		height := int(img.height)

		// Since this is an 8-bit grayscale image:
		size := width * height

		images[i] = &image.Gray{
			Pix: C.GoBytes(
				img.pixels,
				C.int(size),
			),
			Stride: width,
			Rect:   image.Rect(0, 0, width, height),
		}
	}

	return images, nil
}

func copyToGoMemory(img *C.GrayImage) *image.Gray {
	width := int(img.width)
	height := int(img.height)

	size := width * height

	pixels := C.GoBytes(
		img.pixels,
		C.int(size),
	)

	return &image.Gray{
		Pix:    pixels,
		Stride: width,
		Rect:   image.Rect(0, 0, width, height),
	}
}
