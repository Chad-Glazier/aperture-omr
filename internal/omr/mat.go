package omr

import (
	"errors"
	"image"
	"image/color"
	"math"
	"slices"

	"gocv.io/x/gocv"
)

var (
	ErrEncoding             = errors.New("error while encoding a matrix")
	ErrDecoding             = errors.New("error decoding input")
	ErrUnsupportedEncoding  = errors.New("attempted to encode a matrix to an unsupported image format")
	ErrWrongMatType         = errors.New("a matrix operand was not of the correct type")
	ErrEmptyMat             = errors.New("a matrix operand was empty when it was not allowed to be")
	ErrOpenCV               = errors.New("opencv returned an unexpected error")
	ErrIndexOutOfBounds     = errors.New("index out of bounds")
	ErrCannotContain        = errors.New("attempted to fit a larger object inside of a smaller container")
	ErrCannotMatchAnchor    = errors.New("could not locate the anchor on the given matrix")
	ErrMaxIterations        = errors.New("a maximum iterations bound was met")
	ErrNoncontinuousMat     = errors.New("attempted to use a non-continuous matrix for an operation that requires a continuous one")
	ErrMatTypeMismatch      = errors.New("attempted to perform an invalid operation on two matrices of different types")
	ErrInvalidMask          = errors.New("the given bit mask is invalid")
	ErrIncompatibleTemplate = errors.New("the template expects a different number of pages than was given")
	ErrCouldNotCalibrate    = errors.New("the pipeline could not be calibrated; this is likely because the first input was malformed")
	ErrIncompatibleAspect   = errors.New("the aspect ratio of the template radically varies from that of the input")
	ErrQuestionNotDefined   = errors.New("the identified question is not defined on the marking template")
	ErrBubbleNotDefined     = errors.New("the identified bubble is not defined on the marking template")
)

type MatType int

const (
	MatTypeUnknown MatType = iota

	// The same as [MatTypeGray], except it has been binarized. I.e., each of
	// the bytes are either 255 or 0. OpenCV does not treat "binarized" as a
	// distinct type, so this is still represented by 8-bit unsigned integers
	// on a single channel.
	//
	// See [gocv.MatTypeCV8U].
	MatTypeBinary

	// 8-bit unsigned integers on a single channel.
	//
	// See [gocv.MatTypeCV8U].
	MatTypeGray
)

// Represents an OpenCV matrix. Do not instantiate this with a literal.
//
// Underneath the hood, this struct is small and only holds pointers. It's
// recommended that users exclusively pass it around as a value rather than a
// reference.
type Mat struct {
	m      gocv.Mat
	t      *MatType
	closed *bool
}

func NewMat() Mat {
	t := MatTypeUnknown
	closed := false
	return Mat{
		m:      gocv.NewMat(),
		t:      &t,
		closed: &closed,
	}
}

// Creates a new matrix from a gocv matrix.
func newMatFromGoCV(mat gocv.Mat) Mat {
	var (
		t = MatTypeUnknown
		c = false
		m = Mat{m: mat, t: &t, closed: &c}
	)
	if mat.Type() == gocv.MatTypeCV8U || mat.Type() == gocv.MatTypeCV8UC1 {
		*m.t = MatTypeGray
	}
	return m
}

func (m Mat) Closed() bool {
	return m.closed == nil || *m.closed
}

// Closes a matrix. This is always safe to call, even if the matrix hasn't been
// properly initialized.
func (m Mat) Close() {
	if m.Closed() {
		return
	}

	m.m.Close()
	*m.closed = true
}

func Close(m ...Mat) {
	CloseAll(m)
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

// Returns the type of the matrix. Closed matrices always have the type
// [MatTypeUnknown].
func (m Mat) Type() MatType {
	if m.t == nil || m.Closed() {
		return MatTypeUnknown
	}

	return *m.t
}

// Returns true if the matrix is empty. All closed matrices are also considered
// empty.
func (m Mat) Empty() bool {
	if m.Closed() {
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

// Returns the aspect ratio (width : height) of the matrix.
func (m Mat) Aspect() float64 {
	return float64(m.Width()) / float64(m.Height())
}

// Returns an identical deep copy of the given matrix.
func Clone(m Mat) Mat {
	t := *m.t
	c := *m.closed
	return Mat{
		m:      m.m.Clone(),
		t:      &t,
		closed: &c,
	}
}

// Scales a matrix by some factors.
//
// Due to the interpolation used for scaling matrices, the returned matrix will
// never by binary. If the input matrix is [MatTypeBinary], the output will be
// [MatTypeGray].
//
// If an error is returned, it will be [ErrOpenCV].
func Scale(dst, src Mat, scaleX, scaleY float64) error {
	return ScaleTo(
		dst, src,
		uint(scaleX*float64(src.Width())),
		uint(scaleY*float64(src.Height())),
		FitMethodFill,
	)
}

// Describes a method for fitting one rectangle within another. The options are
// named in the same way as the CSS "object-fit" property.
//
// <https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/Properties/object-fit>
type FitMethod uint

const (
	// The image will be scaled so that it completely covers the target
	// dimensions while preserving its aspect ratio. This may cause the
	// image to overflow.
	FitMethodCover FitMethod = iota
	// The image will be scaled so that it fits completely inside the target
	// dimensions while preserving its aspect ratio. This may cause the
	// image to overflow.
	FitMethodContain
	// The image will be scaled to match the target dimensions exactly. This
	// may not preserve the aspect ratio.
	FitMethodFill
)

// Given the current width and height of a box and the width and height of some
// target, this function returns the adjusted width and height that match the
// given fitting method.
//
// For example, suppose you have one box A that you want resized to take up the
// maximum available space in B. If you want to make sure that A fits within B
// perfectly, you would use [FitMethodFill]. However, this may ruin A's aspect
// ratio. If you can't tolerate this, you could use [FitMethodContain] instead.
// If you instead want the minimum size A has to be in order to completely
// cover B, while preserving the aspect ratio, you would use [FitMethodCover].
func FittedBounds(
	width, height, targetWidth, targetHeight uint,
	fittingMethod FitMethod,
) (uint, uint) {
	var (
		scaleX = float64(targetWidth) / float64(width)
		scaleY = float64(targetHeight) / float64(height)
	)
	switch fittingMethod {
	case FitMethodCover:
		if scaleY > scaleX {
			targetWidth = uint(scaleY * float64(width))
		} else {
			targetHeight = uint(scaleX * float64(height))
		}
	case FitMethodContain:
		if scaleY < scaleX {
			targetWidth = uint(scaleY * float64(width))
		} else {
			targetHeight = uint(scaleX * float64(height))
		}
	}
	return targetWidth, targetHeight
}

// Returns a clone of the given matrix, scaled to fit the given dimensions. The
// fitting method determines how the matrix should be resized when the desired
// aspect ratio is different from the initial one.
//
// Due to the interpolation used for scaling matrices, the returned matrix will
// never by binary. If the input matrix is [MatTypeBinary], the output will be
// [MatTypeGray].
//
// If an error is returned, it will be [ErrOpenCV].
func ScaleTo(dst, src Mat, w, h uint, fit FitMethod) error {

	w, h = FittedBounds(src.Width(), src.Height(), w, h, fit)

	var method gocv.InterpolationFlags
	if w < src.Width() || h < src.Height() {
		// Best quality images when downsampling.
		method = gocv.InterpolationArea
	} else {
		// Visually best when upsampling. However, [gocv.InterpolationLinear]
		// is faster and we could use that instead to maximize the speed of
		// this function.
		method = gocv.InterpolationCubic
	}

	err := gocv.Resize(
		src.m, &dst.m,
		image.Pt(int(w), int(h)),
		0, 0,
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

// Returns the bounds of the matrix. The minimum bound will always be (0, 0).
func (m Mat) Region() image.Rectangle {
	return image.Rect(
		0, 0,
		int(m.Width()),
		int(m.Height()),
	)
}

// Rotates a matrix by the given angle in radians. The background color will be
// used to fill in any empty space.
//
// If an error is returned, it will be [ErrOpenCV].
func Rotate(dst, src Mat, angle float64, bg color.RGBA) error {
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
		bg,
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

// Rotates a matrix by the given angle in radians. The background color will be
// used to fill in any empty space. Unlike with [Rotate], the matrix will
// preserve its original dimensions, even if that means clipping corners out.
//
// If an error is returned, it will be [ErrOpenCV].
func RotateWithoutResizing(dst, src Mat, angle float64, bg color.RGBA) error {

	rotation := gocv.GetRotationMatrix2D(
		image.Pt(
			int(src.Width()/2),
			int(src.Height()/2),
		),
		angle/math.Pi*180,
		1.0,
	)
	defer rotation.Close()

	err := gocv.WarpAffineWithParams(
		src.m, &dst.m,
		rotation,
		image.Pt(
			int(src.Width()),
			int(src.Height()),
		),
		gocv.InterpolationArea,
		gocv.BorderConstant,
		bg,
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

// Checks matrices for deep equality. Empty matrices are always treated as
// having no equal.
func Equal(mats ...Mat) bool {
	if len(mats) == 0 {
		return true
	}

	// Check for empty matrices.
	for _, mat := range mats {
		if mat.Empty() {
			return false
		}
	}

	// Check whether all matrices have the same underlying pointer.
	ptr := mats[0].m.Ptr()
	ptrsEqual := true
	for _, mat := range mats[1:] {
		if mat.m.Ptr() != ptr {
			ptrsEqual = false
			break
		}
	}
	if ptrsEqual {
		return true
	}

	// Confirm that the the types are all equal.
	t := mats[0].Type()
	for _, mat := range mats[1:] {
		if t != mat.Type() {
			return false
		}
	}

	// Confirm that the dimensions are all equal.
	w, h := mats[0].Width(), mats[0].Height()
	for _, mat := range mats[1:] {
		if mat.Width() != w || mat.Height() != h {
			return false
		}
	}

	// Lastly, try checking the underlying bytes.
	var prev []byte
	for _, mat := range mats {
		var current []byte

		// [gocv.Mat.IsContinuous] is much faster than copying the bytes into
		// go-owned memory, but it errs if (and only if) the matrix is not
		// continuous.
		if mat.m.IsContinuous() {
			current, _ = mat.m.DataPtrUint8()
		} else {
			current = mat.m.ToBytes()
		}

		if prev == nil {
			prev = current
			continue
		}

		if !slices.Equal(current, prev) {
			return false
		}
	}

	return true
}
