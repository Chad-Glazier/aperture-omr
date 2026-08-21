package omr

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

//
// Helper functions
//

func getTestMat(t *testing.T) Mat {
	t.Helper()

	r, err := os.Open("testdata/input/exam_page.jpeg")
	assert.NilError(t, err)
	defer r.Close()

	mat, err := DecodeImageToMat(r)
	assert.NilError(t, err)

	return mat
}

//
// Tests
//

func TestBinarize(t *testing.T) {

	var tName = t.Name()

	t.Run("write to output", func(t *testing.T) {
		var outputName = "testdata/output/"+tName+"_out.png"

		src := getTestMat(t)
		defer src.Close()
		dst := NewMat()
		defer dst.Close()
		err := Binarize(dst, src, nil)
		assert.Assert(t, err == nil)

		drawInputOutput(t, src, dst, outputName)
	})

	t.Run("bad inputs", func(t *testing.T) {
		empty := NewMat()
		err := Binarize(empty, empty, nil)
		assert.Assert(t, err == ErrEmptyMat)

		mat := getTestMat(t)
		*mat.t = MatTypeUnknown
		err = Binarize(mat, mat, nil)
		assert.Assert(t, err == ErrWrongMatType)
	})
}

//
// Benchmarks
//

func BenchmarkBinarize(b *testing.B) {
	r, err := os.Open("testdata/input/exam_page.jpeg")
	assert.NilError(b, err)
	defer r.Close()

	mat, err := DecodeImageToMat(r)
	assert.NilError(b, err)	
	
	for b.Loop() {
		Binarize(mat, mat, nil)
	}
}
