package fstore

import (
	"encoding/binary"
	"image"
	"image/jpeg"
	"io"

	"github.com/pierrec/lz4/v4"
	"gocv.io/x/gocv"
)

//
// This file contains encoding information, wrapping any encoding/decoding
// of images so that the consumer of the package doesn't need to care about
// which image format we're using.
//

// The MIME type of the image encoding used by this package. This might be
// useful if, for example, you want to pipe an image through an HTTP response
// and you want to set the right Content-Type header.
const ImgContentType = "image/jpeg"

const ImgFileExt = ".jpg"
const OpenCVImgExt gocv.FileExt = gocv.JPEGFileExt

// Encodes an image and writes it to the given destination. The encoding
// method matches the format specified by ImgContentType.
func EncodeImg(w io.Writer, img image.Image) error {
	return jpeg.Encode(w, img, nil)
}

// Decodes an image and returns it. The decoding method matches the image
// format specified by ImgContentType.
func DecodeImg(r io.Reader) (image.Image, error) {
	return jpeg.Decode(r)
}

func EncodeMat(w io.Writer, mat gocv.Mat) error {

	//
	// Rather than storing OpenCV matrices as images, which requires
	// inefficient encoding/decoding, we can store them a bit more neatly by
	// just using the underlying byte buffer that OpenCV maintains. The file
	// format we use is described below.
	//
	// OpenCV matrices have the following data that needs to be stored in order
	// to recreate them:
	// - the rows and columns in the matrix,
	// - the matrix type flag, and
	// - the bytes that store the data.
	// In order to store this data, we will write it into a binary file that
	// includes the dimensions and matrix type as part of a header.
	//
	//        [int32][int32][int32][bytes...]
	//         │      │      │      └─ the bytes buffer for the matrix
	//         │      │      └──────── the matrix type flag
	//         │      └─────────────── the number of columns
	//         └────────────────────── the number of rows
	//
	// The integers are stored in little endian format and the bytes buffer is
	// compressed with the LZ4 algorithm.
	//

	buf, err := mat.DataPtrUint8()
	if err != nil {
		return err
	}

	header := make([]byte, 12)
	binary.Encode(header[0:4], binary.LittleEndian, int32(mat.Rows()))
	binary.Encode(header[4:8], binary.LittleEndian, int32(mat.Cols()))
	binary.Encode(header[8:12], binary.LittleEndian, int32(mat.Type()))

	if _, err := w.Write(header); err != nil {
		return err
	}

	compressedWriter := lz4.NewWriter(w)
	defer compressedWriter.Close()

	if _, err := compressedWriter.Write(buf); err != nil {
		return err
	}

	return nil
}

func DecodeMat(r io.Reader) (gocv.Mat, error) {

	header := make([]byte, 12)
	if _, err := io.ReadFull(r, header); err != nil {
		return gocv.Mat{}, err
	}
	var rows, cols, mt int32
	binary.Decode(header[0:4], binary.LittleEndian, &rows)
	binary.Decode(header[4:8], binary.LittleEndian, &cols)
	binary.Decode(header[8:12], binary.LittleEndian, &mt)

	compressedReader := lz4.NewReader(r)
	buf, err := io.ReadAll(compressedReader)
	if err != nil {
		return gocv.Mat{}, err
	}

	mat, err := gocv.NewMatFromBytes(
		int(rows),
		int(cols),
		gocv.MatType(mt),
		buf,
	)
	if err != nil {
		return gocv.Mat{}, err
	}

	return mat, nil
}
