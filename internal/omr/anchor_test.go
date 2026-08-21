package omr

import (
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

//
// Helper functions
//

// This function takes a "guess" which is a number that may or may not be the
// square root of two. The first return value is the input; the second return
// value is a measure of the quality of the guess (greater for good guesses,
// lesser for bad ones); the last value is always nil.
func isSqrtOfTwo(guess float64) (float64, float64, error) {
	if guess*guess == 2 {
		return guess, math.Inf(+1), nil
	}
	return guess, 1 / math.Abs(guess*guess-2), nil
}

func makeDivergentFunc() func(float64) (float64, float64, error) {
	x := 0.0
	return func(f float64) (float64, float64, error) {
		x += 1.0
		return 0, x, nil
	}
}

// Draws two matrices side-by-side, then writes them to the given file (as an
// image). On failure, the test is failed.
func drawInputOutput(t *testing.T, input, output Mat, filename string) {
	l, err := MatToImage(input)
	assert.Assert(t, err == nil)
	r, err := MatToImage(output)
	assert.Assert(t, err == nil)

	var (
		lw = l.Bounds().Dx()
		lh = l.Bounds().Dy()
		rw = r.Bounds().Dx()
		rh = r.Bounds().Dy()

		padding = int(0.05 * float64(max(lw, rw, lh, rh)))
		w       = int(3*padding + lw + rw)
		h       = int(2*padding + max(lh, rh))

		out = image.NewRGBA(image.Rect(0, 0, w, h))
	)

	draw.Draw(
		out,
		image.Rect(
			padding, padding,
			padding+lw, padding+lh,
		),
		l,
		image.Point{},
		draw.Over,
	)
	draw.Draw(
		out,
		image.Rect(
			padding*2+lw, padding,
			padding*2+lw+rw, padding+rh,
		),
		r,
		image.Point{},
		draw.Over,
	)

	f, err := os.Create(filename)
	assert.Assert(t, err == nil)
	defer f.Close()

	err = png.Encode(f, out)
	assert.Assert(t, err == nil)
}

func getTestAnchorMat(t *testing.T) Mat {
	r, err := os.Open("testdata/input/anchor.jpeg")
	assert.NilError(t, err)

	ancMat, err := DecodeImageToMat(r)
	assert.NilError(t, err)

	return ancMat
}

//
// Tests
//

func TestSearchRegion(t *testing.T) {

	var mat = getTestMat(t)

	t.Run("zero region", func(t *testing.T) {
		var (
			anchor = Anchor{ Mat: NewMat() }
			region = searchRegion(mat, anchor, 0)			
		)
		assert.Assert(t, region.Width() == 0)
		assert.Assert(t, region.Height() == 0)
	})

	t.Run("zero padding region", func(t *testing.T) {
		var (
			anchor = Anchor{ Mat: getTestAnchorMat(t) }
			region = searchRegion(mat, anchor, 0)			
		)
		assert.Assert(t, region.Width() == anchor.Mat.Width())
		assert.Assert(t, region.Height() == anchor.Mat.Height())
	})

	t.Run("safe region with padding", func(t *testing.T) {
		var (
			anchor = Anchor{ 
				Mat: getTestAnchorMat(t),
				Pos: NormalPoint{ 0.45, 0.45 },
			}
			region = searchRegion(mat, anchor, 0.10)			
		)
		assert.Assert(t, region.Width() > anchor.Mat.Width())
		assert.Assert(t, region.Height() > anchor.Mat.Height())
		assert.Assert(t, region.Width() < mat.Width())
		assert.Assert(t, region.Height() < mat.Height())
	})

	t.Run("respects boundaries", func(t *testing.T) {
		var (
			anchor = Anchor{ 
				Mat: NewMat(),
				Pos: NormalPoint{ 0, 0 },
			}
			region = searchRegion(mat, anchor, 0.50)			
		)
		// We're putting the anchor in the top-left corner and asking for 50%
		// padding. We should expect that this makes it so the padded region
		// horizontally and vertically overflows on its left- and top-most 
		// sides. Since this area should be omitted from the region, we should
		// hope that the region ends up with half the width and half the height
		// of the source.
		roughlyEqual := func(a, b uint) bool { 
			if a > b { return a - b <= 3 }
			if b > a { return b - a <= 3 }
			return true
		}
		assert.Assert(t, roughlyEqual(region.Width(), mat.Width()/2))
		assert.Assert(t, roughlyEqual(region.Height(), mat.Height()/2))

		anchor.Pos.X = 1.00
		anchor.Pos.Y = 1.0
		region.Close()
		region = searchRegion(mat, anchor, 0.50)			
		assert.Assert(t, roughlyEqual(region.Width(), mat.Width()/2))
		assert.Assert(t, roughlyEqual(region.Height(), mat.Height()/2))

		region.Close()
		region = searchRegion(mat, anchor, 1.50)			
		assert.Assert(t, roughlyEqual(region.Width(), mat.Width()))
		assert.Assert(t, roughlyEqual(region.Height(), mat.Height()))
	})
}

func TestRotateAnchor(t *testing.T) {
	var tName = t.Name()

	t.Run("15deg", func(t *testing.T) {
		var outputName = "testdata/output/"+tName+"_15deg.png"

		ancMat := getTestAnchorMat(t)
		anc := Anchor{Mat: ancMat, Pos: NormalPoint{}}

		rotated, err := RotateAnchor(anc, 15.0/180.0*math.Pi)
		assert.Assert(t, err == nil)

		drawInputOutput(t, ancMat, rotated.Mat, outputName)
	})
}

func TestRefiningSearch(t *testing.T) {

	t.Run("approximates the square root of two", func(t *testing.T) {
		best, err := refiningSearch(
			1, 2,
			0,
			isSqrtOfTwo,
		)
		assert.Assert(t, err == nil)
		assert.Assert(t, math.Abs(best.candidate-math.Sqrt2) <= 0.01)
	})

	t.Run("hits max iterations error", func(t *testing.T) {
		_, err := refiningSearch(
			0, 100,
			0,
			makeDivergentFunc(),
		)
		assert.Assert(t, err == ErrMaxIterations)
	})

}
