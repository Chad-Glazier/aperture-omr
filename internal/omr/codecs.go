package omr

import (
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"image"
	"io"
	"strings"

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

// Writes a matrix to the given output as an image. The encoding will match the
// given content type, which should be formatted as a MIME type. At the time of
// writing, the only supported content types are "image/jpeg" and "image/png".
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

// Converts an image encoded in base64 to a grayscale matrix. The image format
// can be any of the formats supported by OpenCV.
//
// If an error is returned, it will be [ErrDecoding].
func DecodeBase64(in string) (Mat, error) {
	buf, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		return Mat{}, ErrDecoding
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

func EncodeResultsMarkdown(result MarkResult, out io.Writer) (int, error) {

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
				fmt.Fprintf(&bubbleStr, "%s (%.0f%%)", b.Id, 100*b.Confidence)
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

	return out.Write([]byte(w.String()))
}
