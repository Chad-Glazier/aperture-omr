package omr

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
	"reflect"
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
			_, err = EncodeMatToImage(&bytes.Buffer{}, ImageEncoding(ct), mat)
			assert.Assert(t, err == nil)
		}

		for _, ct := range unsupported {
			_, err := EncodeMatToImage(&bytes.Buffer{}, ImageEncoding(ct), mat)
			assert.Assert(t, err == ErrUnsupportedEncoding)
		}
	})
}

func TestBase64(t *testing.T) {
	tName := t.Name()

	t.Run("decode", func(t *testing.T) {
		r, err := os.Open("testdata/input/sample_image.png")
		assert.Assert(t, err == nil)
		defer r.Close()

		buf, err := io.ReadAll(r)
		assert.Assert(t, err == nil)

		r.Seek(0, io.SeekStart)
		inMat, err := DecodeImageToMat(r)
		assert.Assert(t, err == nil)
		defer inMat.Close()

		encoded := base64.StdEncoding.EncodeToString(buf)
		outMat, err := DecodeBase64(encoded)
		assert.Assert(t, err == nil)
		defer outMat.Close()

		drawInputOutput(t, inMat, outMat,
			"testdata/output/"+tName+".png",
		)
	})

	t.Run("codec is stable", func(t *testing.T) {
		r, err := os.Open("testdata/input/sample_image.png")
		assert.Assert(t, err == nil)
		defer r.Close()

		mat1, err := DecodeImageToMat(r)
		assert.Assert(t, err == nil)
		defer mat1.Close()

		str, err := EncodeBase64(mat1)
		assert.Assert(t, err == nil)

		mat2, err := DecodeBase64(str)
		assert.Assert(t, err == nil)
		defer mat2.Close()

		assert.Assert(t, Equal(mat1, mat2))
	})
}

func TestEncodeDecodeM4t(t *testing.T) {
	t.Run("decodes the same matrix as other codecs", func(t *testing.T) {
		r, err := os.Open("testdata/input/sample_image.png")
		assert.Assert(t, err == nil)
		defer r.Close()

		mat1, err := DecodeImageToMat(r)
		assert.Assert(t, err == nil)
		defer mat1.Close()

		encoded := bytes.Buffer{}
		err = EncodeM4t(&encoded, mat1)
		assert.Assert(t, err == nil)

		mat2, err := DecodeM4t(&encoded)
		assert.Assert(t, err == nil)
		defer mat2.Close()

		assert.Assert(t, Equal(mat1, mat2))
	})
}

func TestEncodeDecodeMarkTemplate(t *testing.T) {
	t.Run("codec is stable", func(t *testing.T) {
		original := getSampleMarkTemplate()
		buf := bytes.Buffer{}

		err := EncodeMarkTemplate(&buf, original)
		assert.Assert(t, err == nil)
		t.Log("stored size", buf.Len())

		decoded, err := DecodeMarkTemplate(&buf)
		assert.Assert(t, err == nil)

		assert.Assert(t, reflect.DeepEqual(decoded, original))
	})
}

func TestEncodeDecodePreprocessTemplate(t *testing.T) {
	t.Run("codec is stable", func(t *testing.T) {
		original, err := getSampleTemplate()
		assert.Assert(t, err == nil)

		buf := bytes.Buffer{}
		err = EncodePreprocessTemplate(&buf, original)
		assert.Assert(t, err == nil)
		t.Log("stored size", buf.Len())

		decoded, err := DecodePreprocessTemplate(&buf)
		assert.Assert(t, err == nil)

		assert.Assert(t, reflect.DeepEqual(
			decoded.AnchorSearchConfig,
			original.AnchorSearchConfig,
		))
		for i := range original.Anchors {
			for j := range original.Anchors[i] {
				a, b := original.Anchors[i][j], decoded.Anchors[i][j]
				assert.Assert(t, Equal(a.Mat, b.Mat))
				assert.Assert(t, a.Pos == b.Pos)
			}
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

func BenchmarkM4t(b *testing.B) {
	const (
		inputName = "testdata/input/sample_image.png"
	)

	buf, err := os.ReadFile(inputName)
	assert.NilError(b, err)
	mat, err := DecodeImageToMat(bytes.NewReader(buf))
	assert.NilError(b, err)

	b.Run("encode", func(b *testing.B) {
		for b.Loop() {
			EncodeM4t(&bytes.Buffer{}, mat)
		}
	})

	encoded := bytes.Buffer{}
	err = EncodeM4t(&encoded, mat)
	assert.Assert(b, err == nil)

	b.Run("decode", func(b *testing.B) {
		for b.Loop() {
			DecodeM4t(bytes.NewReader(encoded.Bytes()))
		}
	})

}

func BenchmarkMarkTemplateCodec(b *testing.B) {
	var (
		original = getSampleMarkTemplate()
		encoded  = bytes.Buffer{}
	)
	EncodeMarkTemplate(&encoded, original)

	b.Run("encode to new buffer", func(b *testing.B) {
		EncodeMarkTemplate(&bytes.Buffer{}, original)
	})

	b.Run("decode", func(b *testing.B) {
		DecodeMarkTemplate(bytes.NewReader(encoded.Bytes()))
	})
}

func BenchmarkPreprocessTemplateCodec(b *testing.B) {
	original, err := getSampleTemplate()
	assert.Assert(b, err == nil)

	encoded := bytes.Buffer{}
	EncodePreprocessTemplate(&encoded, original)

	b.Run("encode to new buffer", func(b *testing.B) {
		EncodePreprocessTemplate(&bytes.Buffer{}, original)
	})

	b.Run("decode", func(b *testing.B) {
		DecodePreprocessTemplate(bytes.NewReader(encoded.Bytes()))
	})
}
