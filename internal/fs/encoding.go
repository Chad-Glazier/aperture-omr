package fs

//
// This file contains encoding information, wrapping any encoding/decoding
// of images so that the consumer of the package doesn't need to care about
// which image format we're using.
//

import (
	"image"
	"io"

	"image/png"
)

// The MIME type of the image encoding used by this package. This might be
// useful if, for example, you want to pipe an image through an HTTP response
// and you want to set the right Content-Type header.
const ImgContentType = "image/png"

const ImgFileExt = ".png"

// Encodes an image and writes it to the given destination. The encoding
// method matches the format specified by ImgContentType.
func EncodeImg(w io.Writer, img image.Image) error {
	return png.Encode(w, img)
}

// Decodes an image and returns it. The decoding method matches the image
// format specified by ImgContentType.
func DecodeImg(r io.Reader) (image.Image, error) {
	return png.Decode(r)
}
