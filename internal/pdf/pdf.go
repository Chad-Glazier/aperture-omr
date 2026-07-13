/*
This package wraps the ImageMagick library to convert PDFs into other formats.
*/
package pdf

import (
	"errors"
	"io"
	"os"

	"gocv.io/x/gocv"
)

var (
	ErrMalformedPdf = errors.New("the given file does not form a PDF")
)

// Interprets the reader as a stream of bytes representing a PDF file. The
// output is a slice of grayscale OpenCV matrices representing the PDF's pages
// in their original order.
//
// Returns ErrMalformedPdf if the given file was not a proper PDF.
func RenderPageMats(r io.Reader) ([]*gocv.Mat, error) {

	//
	// Create a temporariy file and write the incoming bytes to it.
	//

	f, err := os.CreateTemp("", "*.pdf")
	if err != nil {
		return nil, err
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	if _, err := io.Copy(f, r); err != nil {
		return nil, err
	}

	//
	// Get the page images and convert them to matrices.
	//

	images, err := pdfToGrayPages(f.Name())
	if err != nil {
		if err == ErrMalformedPdf {
			return nil, err
		}
		return nil, err
	}

	if len(images) == 0 {
		return nil, ErrMalformedPdf
	}

	mats := make([]*gocv.Mat, len(images))
	for i := range images {
		mat, err := gocv.ImageGrayToMatGray(images[i])
		if err != nil {
			for j := range i {
				mats[j].Close()
			}
			return nil, err
		}
		mats[i] = &mat
	}
	
	return mats, nil
}
