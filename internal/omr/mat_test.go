package omr

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"

	"gocv.io/x/gocv"
	"gotest.tools/v3/assert"
)

//
// Helper functions
//

func loadSampleMat(t *testing.T) Mat {
	t.Helper()

	mat := gocv.IMRead("testdata/input/sample_image.png", gocv.IMReadGrayScale)
	assert.Assert(t, !mat.Empty())

	return newMatFromGoCV(mat)
}

//
// Tests
//

func TestNewMat(t *testing.T) {
	m := NewMat()
	defer m.Close()

	assert.Assert(t, m.Empty())
	assert.Assert(t, m.Type() == MatTypeUnknown)
}

func TestMatClose(t *testing.T) {
	m := loadSampleMat(t)

	assert.Assert(t, !m.Empty())
	assert.Assert(t, !m.Closed())

	m.Close()

	assert.Assert(t, m.Closed())
	assert.Assert(t, m.Empty())
	assert.Assert(t, m.Type() == MatTypeUnknown)

	m.Close()
	assert.Assert(t, m.Empty())
	assert.Assert(t, m.Closed())
}

func TestCloseAll(t *testing.T) {
	a := loadSampleMat(t)
	b := loadSampleMat(t)
	c := loadSampleMat(t)

	CloseAll([]Mat{a, b, c})

	assert.Assert(t, a.Empty())
	assert.Assert(t, b.Empty())
	assert.Assert(t, c.Empty())

	CloseAll(nil)
	CloseAll([]Mat{})
}

func TestCloseAll2(t *testing.T) {
	a := loadSampleMat(t)
	b := loadSampleMat(t)
	c := loadSampleMat(t)

	CloseAll2([][]Mat{
		{a, b},
		{c},
		nil,
	})

	assert.Assert(t, a.Empty())
	assert.Assert(t, b.Empty())
	assert.Assert(t, c.Empty())

	CloseAll2(nil)
	CloseAll2([][]Mat{})
}

func TestClone(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		src := loadSampleMat(t)
		dst := Clone(src)
		defer Close(src, dst)

		assert.Assert(t, !dst.Empty())
		assert.Assert(t, dst.Type() == src.Type())
		assert.Assert(t, dst.Width() == src.Width())
		assert.Assert(t, dst.Height() == src.Height())
		assert.Assert(t, Equal(src, dst))

		// A clone should remain valid after the source is closed.
		src.Close()

		assert.Assert(t, !dst.Empty())
		assert.Assert(t, dst.Width() > 0)
		assert.Assert(t, dst.Height() > 0)
	})

	t.Run("empty", func(t *testing.T) {
		src := NewMat()
		dst := Clone(src)
		defer Close(src, dst)

		assert.Assert(t, dst.Empty())
		assert.Assert(t, dst.Type() == MatTypeUnknown)
	})
}

func TestScale(t *testing.T) {
	src := loadSampleMat(t)
	defer src.Close()

	t.Run("simple", func(t *testing.T) {
		dst := NewMat()
		defer dst.Close()

		err := Scale(dst, src, 0.5, 0.5)
		assert.Assert(t, err == nil)

		expectedWidth := uint(float64(src.Width()) * 0.5)
		expectedHeight := uint(float64(src.Height()) * 0.5)

		assert.Assert(t, !dst.Empty())
		assert.Assert(t, dst.Width() == expectedWidth)
		assert.Assert(t, dst.Height() == expectedHeight)
		assert.Assert(t, dst.Type() == MatTypeGray)
	})

	t.Run("on binary", func(t *testing.T) {
		binary := NewMat()
		defer binary.Close()

		err := Binarize(binary, src, sampleBinarizeConfig())
		assert.Assert(t, err == nil)
		assert.Assert(t, binary.Type() == MatTypeBinary)

		err = Scale(binary, binary, 1.5, 1.5)
		assert.Assert(t, err == nil)
		assert.Assert(t, binary.Type() == MatTypeGray)
	})

	t.Run("invalid source", func(t *testing.T) {
		src := NewMat()
		dst := NewMat()
		defer Close(src, dst)

		err := Scale(dst, src, 2, 2)
		assert.Assert(t, err != nil)
	})
}

func TestFittedBounds(t *testing.T) {
	tt := []struct {
		name           string
		width          uint
		height         uint
		targetWidth    uint
		targetHeight   uint
		method         FitMethod
		expectedWidth  uint
		expectedHeight uint
	}{
		{
			name:           "fill",
			width:          100,
			height:         50,
			targetWidth:    200,
			targetHeight:   200,
			method:         FitMethodFill,
			expectedWidth:  200,
			expectedHeight: 200,
		},
		{
			name:           "contain landscape",
			width:          100,
			height:         50,
			targetWidth:    200,
			targetHeight:   200,
			method:         FitMethodContain,
			expectedWidth:  200,
			expectedHeight: 100,
		},
		{
			name:           "contain portrait",
			width:          50,
			height:         100,
			targetWidth:    200,
			targetHeight:   200,
			method:         FitMethodContain,
			expectedWidth:  100,
			expectedHeight: 200,
		},
		{
			name:           "cover landscape",
			width:          100,
			height:         50,
			targetWidth:    200,
			targetHeight:   200,
			method:         FitMethodCover,
			expectedWidth:  400,
			expectedHeight: 200,
		},
		{
			name:           "cover portrait",
			width:          50,
			height:         100,
			targetWidth:    200,
			targetHeight:   200,
			method:         FitMethodCover,
			expectedWidth:  200,
			expectedHeight: 400,
		},
		{
			name:           "same aspect ratio",
			width:          100,
			height:         50,
			targetWidth:    200,
			targetHeight:   100,
			method:         FitMethodContain,
			expectedWidth:  200,
			expectedHeight: 100,
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			width, height := FittedBounds(
				test.width,
				test.height,
				test.targetWidth,
				test.targetHeight,
				test.method,
			)

			assert.Assert(t, width == test.expectedWidth)
			assert.Assert(t, height == test.expectedHeight)
		})
	}
}

func TestScaleTo(t *testing.T) {
	src := loadSampleMat(t)
	defer src.Close()

	tt := []struct {
		name   string
		fit    FitMethod
		width  uint
		height uint
	}{
		{
			name:   "fill",
			fit:    FitMethodFill,
			width:  100,
			height: 200,
		},
		{
			name:   "contain",
			fit:    FitMethodContain,
			width:  100,
			height: 200,
		},
		{
			name:   "cover",
			fit:    FitMethodCover,
			width:  100,
			height: 200,
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			dst := NewMat()
			defer dst.Close()

			err := ScaleTo(dst, src, test.width, test.height, test.fit)
			assert.Assert(t, err == nil)

			switch test.fit {
			case FitMethodFill:
				assert.Assert(t, dst.Width() == test.width)
				assert.Assert(t, dst.Height() == test.height)

			case FitMethodContain:
				assert.Assert(t, dst.Width() <= test.width)
				assert.Assert(t, dst.Height() <= test.height)

			case FitMethodCover:
				assert.Assert(t, dst.Width() >= test.width)
				assert.Assert(t, dst.Height() >= test.height)
			}

			assert.Assert(t, !dst.Empty())
			assert.Assert(t, dst.Type() == MatTypeGray)
		})
	}
}

func TestRegionEmpty(t *testing.T) {
	m := NewMat()
	defer m.Close()

	assert.Assert(t, m.Region() == image.Rectangle{})
}

func TestRotate(t *testing.T) {
	src := loadSampleMat(t)
	defer src.Close()

	t.Run("simple", func(t *testing.T) {
		dst := NewMat()
		defer dst.Close()

		err := Rotate(dst, src, math.Pi/4, color.RGBA{255, 255, 255, 255})
		assert.Assert(t, err == nil)

		assert.Assert(t, !dst.Empty())
		assert.Assert(t, dst.Type() == MatTypeGray)

		// The rotation should increase the dimensions.
		assert.Assert(t, dst.Width() >= src.Width())
		assert.Assert(t, dst.Height() >= src.Height())
	})

	t.Run("no angle", func(t *testing.T) {
		dst := NewMat()
		defer dst.Close()

		err := Rotate(dst, src, 0, color.RGBA{255, 255, 255, 255})
		assert.Assert(t, err == nil)
		assert.Assert(t, dst.Width() == src.Width())
		assert.Assert(t, dst.Height() == src.Height())
	})
}

func TestRotateWithoutResizing(t *testing.T) {
	src := loadSampleMat(t)
	defer src.Close()

	t.Run("simple", func(t *testing.T) {
		dst := NewMat()
		defer dst.Close()

		err := RotateWithoutResizing(
			dst,
			src,
			math.Pi/4,
			color.RGBA{255, 255, 255, 255},
		)
		assert.Assert(t, err == nil)
		assert.Assert(t, !dst.Empty())
		assert.Assert(t, dst.Type() == MatTypeGray)
		assert.Assert(t, dst.Width() == src.Width())
		assert.Assert(t, dst.Height() == src.Height())
	})

	t.Run("zero angle", func(t *testing.T) {
		dst := NewMat()
		defer dst.Close()

		err := RotateWithoutResizing(
			dst,
			src,
			0,
			color.RGBA{255, 255, 255, 255},
		)
		assert.Assert(t, err == nil)
		assert.Assert(t, dst.Width() == src.Width())
		assert.Assert(t, dst.Height() == src.Height())
		assert.Assert(t, Equal(src, dst))
	})
}

func TestEqual(t *testing.T) {
	a := loadSampleMat(t)
	b := Clone(a)
	defer Close(a, b)

	t.Run("simple", func(t *testing.T) {
		assert.Assert(t, Equal(a, b))
		assert.Assert(t, Equal(a, a))
	})

	t.Run("different matrices", func(t *testing.T) {
		a := gocv.NewMatWithSize(20, 20, gocv.MatTypeCV8U)
		b := gocv.NewMatWithSize(20, 20, gocv.MatTypeCV8U)

		a.SetUCharAt(0, 0, 255)

		ma := newMatFromGoCV(a)
		mb := newMatFromGoCV(b)
		defer Close(ma, mb)

		assert.Assert(t, !Equal(ma, mb))
	})

	t.Run("different dimensions", func(t *testing.T) {
		aCv := gocv.NewMatWithSize(10, 10, gocv.MatTypeCV8U)
		bCv := gocv.NewMatWithSize(20, 20, gocv.MatTypeCV8U)

		a := newMatFromGoCV(aCv)
		b := newMatFromGoCV(bCv)
		defer Close(a, b)

		assert.Assert(t, !Equal(a, b))

		a.Close()
		b.Close()
	})

	t.Run("empty", func(t *testing.T) {
		a := NewMat()
		b := NewMat()

		assert.Assert(t, !Equal(a, b))
	})

	t.Run("closed", func(t *testing.T) {
		a := loadSampleMat(t)
		b := Clone(a)

		a.Close()
		b.Close()

		assert.Assert(t, !Equal(a, b))
	})
}

func TestScaleDimensions(t *testing.T) {
	src := loadSampleMat(t)
	defer src.Close()

	tt := []struct {
		scaleX float64
		scaleY float64
	}{
		{0.5, 0.5},
		{0.5, 1.5},
		{1.5, 0.5},
		{2, 2},
	}

	for _, test := range tt {
		name := fmt.Sprintf("%.0fx%.0fy", test.scaleX, test.scaleY)
		t.Run(name, func(t *testing.T) {
			dst := NewMat()
			defer dst.Close()

			err := Scale(dst, src, test.scaleX, test.scaleY)
			assert.Assert(t, err == nil)

			expectedWidth := uint(float64(src.Width()) * test.scaleX)
			expectedHeight := uint(float64(src.Height()) * test.scaleY)

			assert.Assert(t, dst.Width() == expectedWidth)
			assert.Assert(t, dst.Height() == expectedHeight)
		})
	}
}
