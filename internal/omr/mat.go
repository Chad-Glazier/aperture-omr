package omr

import (
	"errors"
	"image"
	"image/color"
	"math"

	"gocv.io/x/gocv"
)

var (
	ErrEncoding            = errors.New("error while encoding a matrix")
	ErrDecoding            = errors.New("error decoding input")
	ErrUnsupportedEncoding = errors.New("attempted to encode a matrix to an unsupported image format")
	ErrWrongMatType        = errors.New("a matrix operand was not of the correct type")
	ErrEmptyMat            = errors.New("a matrix operand was empty when it was not allowed to be")
	ErrOpenCV              = errors.New("opencv returned an unexpected error")
	ErrIndexOutOfBounds    = errors.New("index out of bounds")
	ErrCannotContain       = errors.New("attempted to fit a larger object inside of a smaller container")
	ErrCannotMatchAnchor   = errors.New("could not locate the anchor on the given matrix")
	ErrMaxIterations       = errors.New("a maximum iterations bound was met")
)

type MatType int

const (
	MatTypeUnknown MatType = iota

	// The same as [MatTypeGray], except it has been binarized. I.e., each of
	// the bytes are either 255 or 0. OpenCV does not treat "binarized" as a
	// distinct type, so this is still represented by 8-bit unsigned integers
	// on a single channel.
	//
	// See [gocv.MatTypeCV8UC1].
	MatTypeBinary

	// 8-bit unsigned integers on a single-channel.
	//
	// See [gocv.MatTypeCV8UC1].
	MatTypeGray
)

// Represents an OpenCV matrix. Do not instantiate this with a literal.
//
// Underneath the hood, this struct is small and only holds pointers. It's
// recommended that users exclusively pass it around as a value rather than a
// reference.
type Mat struct {
	m      gocv.Mat
	closed *bool
	t      *MatType
}

func NewMat() Mat {
	var (
		closed = false
		t      MatType
	)
	return Mat{
		m:      gocv.NewMat(),
		closed: &closed,
		t:      &t,
	}
}

// Creates a new matrix from a gocv matrix.
func newMatFromGoCV(mat gocv.Mat) Mat {
	var (
		closed = false
		t      MatType
		m      = Mat{m: mat, closed: &closed, t: &t}
	)
	if mat.Type() == gocv.MatTypeCV8UC1 {
		*m.t = MatTypeGray
	}
	return m
}

// Closes a matrix. This is always safe to call, even if the matrix hasn't been
// properly initialized.
func (m Mat) Close() {
	if m.closed == nil {
		// The matrix is uninitialized.
		return
	}

	if *m.closed {
		return
	}

	m.m.Close()
	*m.closed = true
}

// Closes all matrices in the given slice.
func CloseAll(c []Mat) {
	for i := range c {
		c[i].Close()
	}
}

// Closes all matrices in the given 2-D slice.
func CloseAll2(c [][]Mat) {
	for i := range c {
		for j := range c[i] {
			c[i][j].Close()
		}
	}
}

// Returns true if the matrix is empty. Note that all closed matrices are also
// empty.
func (m Mat) Empty() bool {
	if *m.closed {
		return true
	}

	return m.m.Empty()
}

func (m Mat) Rows() uint {
	return uint(m.m.Rows())
}

func (m Mat) Height() uint {
	return uint(m.m.Rows())
}

func (m Mat) Cols() uint {
	return uint(m.m.Cols())
}

func (m Mat) Width() uint {
	return uint(m.m.Cols())
}

// Returns an identical deep copy of the given matrix.
func Clone(m Mat) Mat {
	var (
		closed = *m.closed
		t      = *m.t
	)
	return Mat{
		m:      m.m.Clone(),
		closed: &closed,
		t:      &t,
	}
}

// Returns a clone of the given matrix, scaled by the given factors.
//
// Due to the interpolation used for scaling matrices, the returned matrix will
// never by binary. If the input matrix is [MatTypeBinary], the output will be
// [MatTypeGray].
//
// If an error is returned, it will be [ErrOpenCV].
func Scale(dst, src Mat, scaleX, scaleY float64) error {

	var method gocv.InterpolationFlags
	if min(scaleX, scaleY) < 1 {
		// Best quality images when upsampling.
		method = gocv.InterpolationArea
	} else {
		// Visually best when upsampling. However, [gocv.InterpolationLinear]
		// is faster and we could use that instead to maximize the speed of
		// this function.
		method = gocv.InterpolationCubic
	}

	err := gocv.Resize(
		src.m, &dst.m,
		image.Point{},
		scaleX, scaleY,
		method,
	)
	if err != nil {
		return ErrOpenCV
	}

	if *src.t == MatTypeBinary {
		*dst.t = MatTypeGray
	} else {
		*dst.t = *src.t
	}

	return nil
}

func (m Mat) Region() image.Rectangle {
	return image.Rect(
		0, 0,
		int(m.Width()),
		int(m.Height()),
	)
}

// Rotates a matrix by the given angle in radians.
//
// If an error is returned, it will be [ErrOpenCV].
func Rotate(dst Mat, src Mat, angle float64) error {
	var (
		w = int(src.Width())
		h = int(src.Height())

		newW = float64(h)*math.Abs(math.Sin(angle)) +
			float64(w)*math.Abs(math.Cos(angle))
		newH = float64(h)*math.Abs(math.Cos(angle)) +
			float64(w)*math.Abs(math.Sin(angle))
	)

	rotation := gocv.GetRotationMatrix2D(
		image.Pt(int(w/2), int(h/2)),
		angle/math.Pi*180,
		1.0,
	)
	defer rotation.Close()

	rotation.SetDoubleAt(0, 2,
		rotation.GetDoubleAt(0, 2)+(newW-float64(w))/2,
	)
	rotation.SetDoubleAt(1, 2,
		rotation.GetDoubleAt(1, 2)+(newH-float64(h))/2,
	)

	err := gocv.WarpAffineWithParams(
		src.m, &dst.m,
		rotation,
		image.Pt(int(newW), int(newH)),
		gocv.InterpolationArea,
		gocv.BorderConstant,
		color.RGBA{ 255, 255, 255, 255 },
	)
	if err != nil {
		return ErrOpenCV
	}

	if *src.t == MatTypeBinary {
		*dst.t = MatTypeGray
	} else {
		*dst.t = *src.t
	}

	return nil
}
