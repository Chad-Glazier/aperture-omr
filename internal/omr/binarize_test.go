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

func getNoisyPageMat() (Mat, error) {
	r, err := os.Open("testdata/input/sample_page_noisy.png")
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
		var outputName = "testdata/output/" + tName + ".png"

		src, err := getSamplePageMat()
		assert.Assert(t, err == nil)
		defer src.Close()
		dst := NewMat()
		defer dst.Close()
		err = Binarize(dst, src, nil)
		assert.Assert(t, err == nil)

		out, err := os.Create(outputName)
		assert.Assert(t, err == nil)
		VisualizeSideBySide(out, src, dst)
	})

	t.Run("write noisy to output", func(t *testing.T) {
		var outputName = "testdata/output/" + tName + "_noisy.png"

		src, err := getNoisyPageMat()
		assert.Assert(t, err == nil)
		defer src.Close()
		dst := NewMat()
		defer dst.Close()
		err = Binarize(dst, src, nil)
		assert.Assert(t, err == nil)

		out, err := os.Create(outputName)
		assert.Assert(t, err == nil)
		VisualizeSideBySide(out, src, dst)
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
		Binarize(mat, mat, &BinarizeConfig{
			BlurSize:       0.0015,
			MorphCloseSize: 0.0015,
			BlockSize:      0.0255,
			AdaptiveC:      10,
		})
	}
}
