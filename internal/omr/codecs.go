package omr

import (
	"errors"
	"image"
	"io"

	"gocv.io/x/gocv"
)

var (
	ErrEncoding            = errors.New("error while encoding a matrix")
	ErrDecoding            = errors.New("error decoding input")
	ErrUnsupportedEncoding = errors.New("attempted to encode a matrix to an unsupported image format")
)

// Returns a new grayscale matrix from an RGBA image.
//
// If an error is returned, it will be [ErrDecoding].
func RgbaToMat(img *image.RGBA) (Mat, error) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	bytes := make([]byte, w*h)

	for x := range w {
		for y := range h {

			//
			// This luminosity formula is copied from the Go standard library.
			// <https://cs.opensource.google/go/go/+/master:src/image/color/color.go;l=244>
			//

			pixel := img.At(x, y)
			r, g, b, _ := pixel.RGBA()
			lum := (19595*r + 38470*g + 7471*b + 1<<15) >> 24
			bytes[x+y*w] = byte(lum)
		}
	}

	m, err := gocv.NewMatFromBytes(h, w, gocv.MatTypeCV8UC1, bytes)
	if err != nil {
		return Mat{}, ErrOpenCV
	}

	return newMatFromGoCV(m), nil
}

// Reads the given input as an image and converts it to a grayscale matrix.
//
// If an error is returned, it will be [ErrDecoding].
func DecodeImageToMat(r io.Reader) (Mat, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return Mat{}, ErrDecoding
	}

	mat, err := gocv.IMDecode(buf, gocv.IMReadGrayScale)
	if err != nil {
		return Mat{}, ErrDecoding
	}

	return newMatFromGoCV(mat), nil
}

// Writes a matrix to the given output as an image. The encoding will match the
// given content type. The content type should be formatted as a MIME type. At
// the time of writing, the only supported content types are "image/jpeg" and
// "image/png".
//
// If an error is returned, it will be [ErrUnsupportedEncoding] or
// [ErrEncoding].
func EncodeMatToImage(w io.Writer, contentType string, mat Mat) (int, error) {
	var ext gocv.FileExt
	switch contentType {
	case "image/jpeg":
		ext = gocv.JPEGFileExt
	case "image/png":
		ext = gocv.PNGFileExt
	default:
		return 0, ErrUnsupportedEncoding
	}

	buf, err := gocv.IMEncode(ext, mat.m)
	if err != nil {
		return 0, ErrEncoding
	}
	defer buf.Close()

	return w.Write(buf.GetBytes())
}
