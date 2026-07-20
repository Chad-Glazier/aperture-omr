package pdf

// #cgo pkg-config: Magick++
//
// #include <stdlib.h>
// #include <wand/MagickWand.h>
// #include "magick.h"
//
// #include <inttypes.h>
// static void PrintResourceLimits(void) {
//     printf("Memory: %" PRIu64 "\n", (uint64_t)GetMagickResourceLimit(MemoryResource));
//     printf("Map:    %" PRIu64 "\n", (uint64_t)GetMagickResourceLimit(MapResource));
//     printf("Disk:   %" PRIu64 "\n", (uint64_t)GetMagickResourceLimit(DiskResource));
//     printf("Area:   %" PRIu64 "\n", (uint64_t)GetMagickResourceLimit(AreaResource));
//     printf("Width:  %" PRIu64 "\n", (uint64_t)GetMagickResourceLimit(WidthResource));
//     printf("Height: %" PRIu64 "\n", (uint64_t)GetMagickResourceLimit(HeightResource));
//     printf("Thread: %" PRIu64 "\n", (uint64_t)GetMagickResourceLimit(ThreadResource));
//     printf("Time: %"   PRIu64 "\n", (uint64_t)GetMagickResourceLimit(TimeResource));
// }
import "C"

import (
	"errors"
	"fmt"
	"image"
	"runtime"
	"unsafe"

	"gocv.io/x/gocv"
)

func init() {
	C.MagickWandGenesis() // initialize ImageMagick

	const GB = uint64(1024 * 1024 * 1024)
	const unlimited = C.MagickSizeType(9223372036854775807)

	maxThreads := max(1, runtime.GOMAXPROCS(0)-1)

	C.SetMagickResourceLimit(C.MemoryResource, C.MagickSizeType(4*GB))
	C.SetMagickResourceLimit(C.MapResource, C.MagickSizeType(8*GB))
	// Unlimited disk cache.
	C.SetMagickResourceLimit(C.DiskResource, C.MagickSizeType(16*GB))
	// Maximum number of open files.
	C.SetMagickResourceLimit(C.FileResource, C.MagickSizeType(1536))
	C.SetMagickResourceLimit(C.ThreadResource, C.MagickSizeType(maxThreads))
	C.SetMagickResourceLimit(C.ThrottleResource, C.MagickSizeType(0))
	C.SetMagickResourceLimit(C.TimeResource, unlimited)

	C.PrintResourceLimits()
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
		images[i] = copyToGoMemory(&img)
	}

	return images, nil
}

type PageMats struct {
	cMats C.Mats
	Pages []*gocv.Mat
}

func pdfBytesToMats(pdf []byte, density int) (result *PageMats, err error) {

	defer func() {
		if r := recover(); r != nil {
			if result != nil {
				result.Close()
			}
			result = nil
			err = fmt.Errorf("panic recovered in pdfBytesToMats: %v", r)
		}
	}()

	if len(pdf) == 0 {
		return nil, errors.New("pdfBytesToMats: empty PDF")
	}

	var status C.PdfStatus

	cMats := C.pdf_bytes_to_mats(
		unsafe.Pointer(&pdf[0]),
		C.size_t(len(pdf)),
		C.int(density),
		&status,
	)

	if cMats.mats == nil {
		return nil, Status(status)
	}

	result = &PageMats{
		cMats: cMats,
		Pages: make([]*gocv.Mat, int(cMats.length)),
	}

	for i := 0; i < int(cMats.length); i++ {
		cMat := C.mats_get(cMats, C.size_t(i))

		if cMat == nil {
			result.Close()
			return nil, errors.New("failed to get matrix")
		}

		mat := gocv.NewMatFromCMat(cMat)
		result.Pages[i] = &mat
	}

	return result, nil
}

func (s *PageMats) Close() {
	if s == nil {
		return
	}

	C.mats_destroy(s.cMats)
	s.cMats = C.Mats{}
	s.Pages = nil
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
