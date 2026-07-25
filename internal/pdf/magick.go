package pdf

// #cgo pkg-config: MagickWand
// #cgo pkg-config: opencv4
//
// #include <stdlib.h>
// #include <wand/MagickWand.h>
// #include "magick.h"
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"gocv.io/x/gocv"
)

func init() {

	//
	// The ImageMagick library's default configuration is usually fine but it
	// varies across operating systems. To keep things consistent, we
	// explicitly set the resource limits here.
	//

	C.MagickWandGenesis() // initialize ImageMagick

	var maxThreads = C.MagickSizeType(max(1, runtime.GOMAXPROCS(0)-1))
	C.SetMagickResourceLimit(C.MemoryResource, C.MagickSizeType(4<<30))
	C.SetMagickResourceLimit(C.MapResource, C.MagickSizeType(8<<30))
	C.SetMagickResourceLimit(C.DiskResource, C.MagickSizeType(16<<30))
	C.SetMagickResourceLimit(C.FileResource, C.MagickSizeType(3<<9))
	C.SetMagickResourceLimit(C.ThreadResource, maxThreads)
	C.SetMagickResourceLimit(C.ThrottleResource, C.MagickSizeType(0))
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

type PageMats struct {
	cMats C.Mats
	Pages []*gocv.Mat
}

func (s *PageMats) Close() {
	if s == nil {
		return
	}

	C.mats_destroy(s.cMats)

	s.cMats = C.Mats{}
	s.Pages = nil
}

func pdfBytesToMats(
	pdf []byte, 
	density int,
) (result *PageMats, err error) {

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
