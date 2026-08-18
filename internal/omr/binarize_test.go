package omr

import (
	"os"
	"testing"

	"gocv.io/x/gocv"
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
	var (
		outputName = "testdata/output/"+t.Name()+"_out.png"
	)

	t.Run("write to output", func(t *testing.T) {

		mat := getTestMat(t)
		Binarize(mat, mat, nil)

		w, err := os.Create(outputName)
		assert.NilError(t, err)
		defer w.Close()

		EncodeMatToImage(w, "image/png", mat)
	})

	t.Run("bad inputs", func(t *testing.T) {
		empty := newMatFromGoCV(gocv.NewMat())
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
