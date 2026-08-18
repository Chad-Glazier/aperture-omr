package omr

import (
	"gocv.io/x/gocv"
)

type MatType int

const (
	MatTypeUnknown MatType = iota

	// The same as [MatGray], except it has been binarized. I.e., each of the
	// bytes are either 255 or 0. OpenCV does not treat "binarized" as a
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

// Creates a new matrix from a gocv matrix.
func newMatFromGoCV(mat gocv.Mat) Mat {
	var (
		closed = false
		t      MatType
	)
	if mat.Type() == gocv.MatTypeCV8UC1 {
		t = MatTypeGray
	}
	return Mat{
		m:      mat,
		closed: &closed,
		t:      &t,
	}
}

// Closes a matrix. Redundant calls are safe.
func (m Mat) Close() {
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

// Returns the size of the smaller dimension of this matrix. E.g., if this
// matrix has more rows than columns, this function will return the number of
// columns.
func (m Mat) MinDim() uint {
	return min(m.Width(), m.Height()) 
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
