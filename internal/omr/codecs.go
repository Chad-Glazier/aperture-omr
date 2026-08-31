package omr

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"strings"

	"github.com/pierrec/lz4/v4"
	"gocv.io/x/gocv"
)

// Returns a new grayscale matrix from an RGBA image.
//
// If an error is returned, it will be [ErrDecoding].
func RgbaToMat(img *image.RGBA) (Mat, error) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	bytes := make([]byte, w*h)
	stride := img.Stride

	for x := range w {
		for y := range h {

			//
			// The luminosity formula is copied from the Go standard library:
			//
			// <https://cs.opensource.google/go/go/+/master:src/image/color/color.go;l=244>
			// <https://cs.opensource.google/go/go/+/master:src/image/color/color.go;l=30>
			//
			// Notably, we are leaving out one small computation; the rgba
			// values in the standard library are calculated as "(x << 8) | x"
			// while we are only doing "x << 8". One can prove exhaustively
			// that this will only ever change the luminosity by 1, so I'm
			// alright with forgoing it.
			//

			var (
				i = y*stride + x*4
				r = uint32(img.Pix[i]) << 8
				g = uint32(img.Pix[i+1]) << 8
				b = uint32(img.Pix[i+2]) << 8
			)

			lum := (19595*r + 38470*g + 7471*b + 1<<15) >> 24
			bytes[x+y*w] = byte(lum)
		}
	}

	m, err := gocv.NewMatFromBytes(h, w, gocv.MatTypeCV8U, bytes)
	if err != nil {
		return Mat{}, ErrOpenCV
	}

	return newMatFromGoCV(m), nil
}

// Converts a matrix to an image.
//
// If an error is returned, it will be [ErrEncoding] or [ErrNoncontinuousMat].
func MatToImage(mat Mat) (image.Image, error) {
	img, err := mat.m.ToImage()
	if err != nil {
		return nil, ErrEncoding
	}
	return img, nil
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

type ImageEncoding string

const (
	ImageEncodingJpeg ImageEncoding = "image/jpeg"
	ImageEncodingPng  ImageEncoding = "image/png"
)

// Writes a matrix to the given output as an image. The encoding will match the
// given content type, which should be formatted as a MIME type. At the time of
// writing, the only supported content types are "image/jpeg" and "image/png".
//
// If an error is returned, it will be [ErrUnsupportedEncoding] or
// [ErrEncoding].
func EncodeMatToImage(w io.Writer, contentType ImageEncoding, mat Mat) (int, error) {
	var ext gocv.FileExt
	switch contentType {
	case ImageEncodingJpeg:
		ext = gocv.JPEGFileExt
	case ImageEncodingPng:
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

// Converts an image encoded in base64 to a grayscale matrix. The image format
// can be any of the formats supported by OpenCV.
//
// If an error is returned, it will be [ErrBase64Decoding] or [ErrDecoding].
func DecodeBase64(in string) (Mat, error) {
	buf, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		return Mat{}, ErrBase64Decoding
	}
	mat, err := gocv.IMDecode(buf, gocv.IMReadGrayScale)
	if err != nil {
		return Mat{}, ErrDecoding
	}
	return newMatFromGoCV(mat), nil
}

// Converts a matrix to a base64-encoded image format (PNG, specifically).
//
// If an error is returned, it will be [ErrEncoding].
func EncodeBase64(in Mat) (string, error) {
	buf, err := gocv.IMEncode(gocv.PNGFileExt, in.m)
	if err != nil {
		return "", ErrEncoding
	}
	defer buf.Close()

	str := base64.RawStdEncoding.EncodeToString(buf.GetBytes())
	return str, nil
}

// Converts marking results to CSV format and writes it to the given output.
//
// If an error is returned, it will be [Err???]
func EncodeResultsCsv(m MarkResult, out io.Writer) error {
	w := csv.NewWriter(out)
	defer w.Flush()

	err := w.Write([]string{"question id", "marked bubble", "confidence"})
	if err != nil {
		return ErrWriting
	}

	for _, page := range m.Pages {
		for _, q := range page.Questions {
			for _, b := range q.SelectedBubbles {
				err := w.Write([]string{
					q.Id,
					b.Id,
					fmt.Sprintf("%.3f", b.Confidence),
				})
				if err != nil {
					return ErrWriting
				}
			}
		}
	}

	return nil
}

// Converts marking results to a markdown table and writes it to the given
// output.
//
// If an error is returned, it will be [ErrWriting].
func EncodeResultsMarkdown(result MarkResult, out io.Writer) error {

	w := strings.Builder{}
	w.WriteString("| Question | Marked Bubbles             |\n")
	w.WriteString("|----------|----------------------------|\n")

	for _, page := range result.Pages {
		for _, q := range page.Questions {

			qIdStr := " " + q.Id
			if len(qIdStr) < 10 {
				qIdStr += strings.Repeat(" ", 10-len(qIdStr))
			}

			var bubbleStr strings.Builder
			bubbleStr.WriteString(" ")
			for i, b := range q.SelectedBubbles {
				fmt.Fprintf(&bubbleStr, "%s (%.1f%%)", b.Id, 100*b.Confidence)
				if i != len(q.SelectedBubbles)-1 {
					bubbleStr.WriteString(", ")
				}
			}
			if bubbleStr.Len() < 28 {
				bubbleStr.WriteString(strings.Repeat(" ", 28-bubbleStr.Len()))
			}

			fmt.Fprintf(
				&w,
				"|%s|%s|\n",
				qIdStr, bubbleStr.String(),
			)
		}
	}

	_, err := out.Write([]byte(w.String()))
	if err != nil {
		return ErrWriting
	}
	return nil
}

// Encodes a matrix using the "M4t" custom file format. M4t is suitable for
// persistent storage and is faster to encode/decode than traditional image
// formats. The caveat is that it only works with grayscale/binary matrices.
//
// If an error is returned, it will be [ErrEncoding], [ErrWriting], or
// [ErrWrongMatType].
func EncodeM4t(w io.Writer, mat Mat) error {

	if t := mat.Type(); t != MatTypeBinary && t != MatTypeGray {
		return ErrWrongMatType
	}

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

	if !mat.m.IsContinuous() {
		mat = Clone(mat)
		defer mat.Close()
	}

	buf, err := mat.m.DataPtrUint8()
	if err != nil {
		return ErrEncoding
	}

	header := make([]byte, 12)
	binary.Encode(header[0:4], binary.LittleEndian, int32(mat.Rows()))
	binary.Encode(header[4:8], binary.LittleEndian, int32(mat.Cols()))
	binary.Encode(header[8:12], binary.LittleEndian, int32(mat.m.Type()))

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

// Encodes a matrix using the "M4t" custom file format. See [EncodeM4t].
// The returned matrix will be [MatTypeGray].
//
// If an error is returned, it will be [ErrEncoding], [ErrReading], or
// [ErrWrongMatType].
func DecodeM4t(r io.Reader) (Mat, error) {

	header := make([]byte, 12)
	if _, err := io.ReadFull(r, header); err != nil {
		return Mat{}, ErrReading
	}
	var rows, cols, mt int32
	binary.Decode(header[0:4], binary.LittleEndian, &rows)
	binary.Decode(header[4:8], binary.LittleEndian, &cols)
	binary.Decode(header[8:12], binary.LittleEndian, &mt)

	compressedReader := lz4.NewReader(r)
	buf, err := io.ReadAll(compressedReader)
	if err != nil {
		return Mat{}, ErrDecoding
	}

	mat, err := gocv.NewMatFromBytes(
		int(rows),
		int(cols),
		gocv.MatType(mt),
		buf,
	)
	if err != nil {
		return Mat{}, ErrDecoding
	}

	return newMatFromGoCV(mat), nil
}

// Serializes a marking template in a format suitable for storage but not for
// human readers.
//
// If an error is returned, it will be [ErrEncoding].
func EncodeMarkTemplate(w io.Writer, m MarkTemplate) error {
	var (
		compressor = lz4.NewWriter(w)
		encoder    = json.NewEncoder(compressor)
	)
	defer compressor.Close()

	err := encoder.Encode(m)
	if err != nil {
		return ErrEncoding
	}
	return nil
}

// Deserializes a marking template previous encoded with [EncodeMarkTemplate].
//
// If an error is returned, it will be [ErrDecoding].
func DecodeMarkTemplate(r io.Reader) (MarkTemplate, error) {
	var (
		out          MarkTemplate
		decompressor = lz4.NewReader(r)
		decoder      = json.NewDecoder(decompressor)
	)

	err := decoder.Decode(&out)
	if err != nil {
		return MarkTemplate{}, ErrDecoding
	}
	return out, nil
}

// Serializes a preprocessing template in a format suitable for storage but not
// for human readers.
//
// If an error is returned, it will be [ErrEncoding], [ErrWriting], or
// [ErrWrongMatType].
func EncodePreprocessTemplate(w io.Writer, p PreprocessTemplate) error {

	var serialized serializedPreprocessTemplate

	serialized.Anchors = make([][]serializedAnchor, len(p.Anchors))
	for i := range p.Anchors {

		serialized.Anchors[i] = make([]serializedAnchor, len(p.Anchors[i]))
		for j := range p.Anchors[i] {

			matBytes := bytes.Buffer{}
			err := EncodeM4t(&matBytes, p.Anchors[i][j].Mat)
			if err != nil {
				return err
			}

			serialized.Anchors[i][j].MatBytes = matBytes.Bytes()
			serialized.Anchors[i][j].Pos = p.Anchors[i][j].Pos
		}
	}

	p.Anchors = nil
	serialized.Template = p

	encoder := json.NewEncoder(w)
	err := encoder.Encode(serialized)
	if err != nil {
		return ErrEncoding
	}
	return nil
}

type serializedPreprocessTemplate struct {
	Template PreprocessTemplate
	Anchors  [][]serializedAnchor
}

type serializedAnchor struct {
	MatBytes []byte
	Pos      NormalPoint
}

// Deserializes a preprocessing template encoded by [EncodePreprocessTemplate].
//
// If an error is returned, it will be [ErrEncoding], [ErrReading], or
// [ErrWrongMatType].
func DecodePreprocessTemplate(r io.Reader) (PreprocessTemplate, error) {
	var (
		serialized serializedPreprocessTemplate
		decoder    = json.NewDecoder(r)
	)

	err := decoder.Decode(&serialized)
	if err != nil {
		return PreprocessTemplate{}, ErrDecoding
	}

	out := serialized.Template

	out.Anchors = make([][]Anchor, len(serialized.Anchors))
	for i := range serialized.Anchors {

		out.Anchors[i] = make([]Anchor, len(serialized.Anchors[i]))
		for j, a := range serialized.Anchors[i] {

			mat, err := DecodeM4t(bytes.NewReader(a.MatBytes))
			if err != nil {
				return PreprocessTemplate{}, err
			}

			out.Anchors[i][j].Mat = mat
			out.Anchors[i][j].Pos = a.Pos
		}
	}

	return out, nil
}
