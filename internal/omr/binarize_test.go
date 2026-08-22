package omr

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

//
// Helper functions
//

func getSamplePageMat() (Mat, error) {

	r, err := os.Open("testdata/input/sample_page.png")
	if err != nil {
		return Mat{}, err
	}
	defer r.Close()

	mat, err := DecodeImageToMat(r)
	if err != nil {
		return Mat{}, err
	}

	return mat, nil
}

//
// Tests
//

func TestBinarize(t *testing.T) {

	var tName = t.Name()

	t.Run("write to output", func(t *testing.T) {
		var outputName = "testdata/output/"+tName+"_out.png"

		src, err := getSamplePageMat()
		assert.Assert(t, err == nil)
		defer src.Close()
		dst := NewMat()
		defer dst.Close()
		err = Binarize(dst, src, nil)
		assert.Assert(t, err == nil)

		drawInputOutput(t, src, dst, outputName)
	})

	t.Run("bad inputs", func(t *testing.T) {
		empty := NewMat()
		err := Binarize(empty, empty, nil)
		assert.Assert(t, err == ErrEmptyMat)

		mat, err := getSamplePageMat()
		assert.Assert(t, err == nil)
		*mat.t = MatTypeUnknown
		err = Binarize(mat, mat, nil)
		assert.Assert(t, err == ErrWrongMatType)
	})
}

//
// Benchmarks
//

func BenchmarkBinarize(b *testing.B) {
	r, err := os.Open("testdata/input/sample_page.png")
	assert.NilError(b, err)
	defer r.Close()

	mat, err := DecodeImageToMat(r)
	assert.NilError(b, err)	
	
	for b.Loop() {
		Binarize(mat, mat, nil)
	}
}
