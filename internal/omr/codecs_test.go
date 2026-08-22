package omr

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"os"
	"testing"

	"gocv.io/x/gocv"
	"gotest.tools/v3/assert"
)

//
// Helper functions
//

func imageToRgba(src image.Image) *image.RGBA {
	if dst, ok := src.(*image.RGBA); ok {
		return dst
	}

	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

//
// Tests
//

func TestRgbaToMat(t *testing.T) {
	var (
		inputName  = "testdata/input/sample_image.png"
		outputName = "testdata/output/" + t.Name() + "_out.png"
	)

	f, err := os.Open(inputName)
	assert.NilError(t, err)
	defer f.Close()

	img, err := png.Decode(f)
	assert.NilError(t, err)

	mat, err := RgbaToMat(imageToRgba(img))
	assert.NilError(t, err)
	assert.Assert(t, *mat.t == MatTypeGray)

	ok := gocv.IMWrite(outputName, mat.m)
	assert.Assert(t, ok)

	t.Logf("Output written to %s", outputName)
}

func TestImageCodec(t *testing.T) {
	var (
		inputName  = "testdata/input/sample_image.png"
		outputName = "testdata/output/" + t.Name() + "_out.png"
	)

	t.Run("decode from input", func(t *testing.T) {
		f, err := os.Open(inputName)
		assert.NilError(t, err)
		defer f.Close()

		mat, err := DecodeImageToMat(f)
		assert.NilError(t, err)
		assert.Assert(t, *mat.t == MatTypeGray)
	})

	t.Run("encode to output", func(t *testing.T) {
		r, err := os.Open(inputName)
		assert.NilError(t, err)
		defer r.Close()

		mat, err := DecodeImageToMat(r)
		assert.NilError(t, err)
		assert.Assert(t, *mat.t == MatTypeGray)

		w, err := os.Create(outputName)
		assert.NilError(t, err)

		_, err = EncodeMatToImage(w, "image/png", mat)
		assert.NilError(t, err)

		t.Logf("Output written to %s", outputName)
	})

	t.Run("allowed encodings", func(t *testing.T) {
		f, err := os.Open(inputName)
		assert.NilError(t, err)
		defer f.Close()

		mat, err := DecodeImageToMat(f)
		assert.NilError(t, err)
		assert.Assert(t, *mat.t == MatTypeGray)

		supported := []string{
			"image/png", "image/jpeg",
		}
		unsupported := []string{
			"image/gif", "image/tiff", "png", "jpeg", "",
		}

		for _, ct := range supported {
			_, err = EncodeMatToImage(&bytes.Buffer{}, ct, mat)
			assert.Assert(t, err == nil)
		}

		for _, ct := range unsupported {
			_, err := EncodeMatToImage(&bytes.Buffer{}, ct, mat)
			assert.Assert(t, err == ErrUnsupportedEncoding)
		}
	})
}

//
// Benchmarks
//

func BenchmarkRgbaToMat(b *testing.B) {
	const (
		inputName = "testdata/input/sample_image.png"
	)

	f, err := os.Open(inputName)
	assert.NilError(b, err)
	defer f.Close()

	img, err := png.Decode(f)
	assert.NilError(b, err)
	rgba := imageToRgba(img)

	for b.Loop() {
		RgbaToMat(rgba)
	}
}

func BenchmarkDecodeImageToMat(b *testing.B) {
	const (
		inputName = "testdata/input/sample_image.png"
	)

	buf, err := os.ReadFile(inputName)
	assert.NilError(b, err)

	for b.Loop() {
		DecodeImageToMat(bytes.NewReader(buf))
	}
}

func BenchmarkEncodeMatToImage(b *testing.B) {
	const (
		inputName = "testdata/input/sample_image.png"
	)

	buf, err := os.ReadFile(inputName)
	assert.NilError(b, err)
	mat, err := DecodeImageToMat(bytes.NewReader(buf))
	assert.NilError(b, err)

	for b.Loop() {
		EncodeMatToImage(&bytes.Buffer{}, "image/png", mat)
	}
}
