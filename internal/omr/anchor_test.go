package omr

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"testing"

	"gocv.io/x/gocv"
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
	t.Helper()

	if !input.m.IsContinuous() {
		input = Clone(input)
		defer input.Close()
	}
	if !output.m.IsContinuous() {
		output = Clone(output)
		defer output.Close()
	}

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

func getTestAnchor0Mat() (Mat, error) {

	r, err := os.Open("testdata/input/anchor_0.png")
	if err != nil {
		return Mat{}, err
	}

	ancMat, err := DecodeImageToMat(r)
	if err != nil {
		return Mat{}, err
	}

	return ancMat, nil
}

func getTestAnchor1Mat() (Mat, error) {

	r, err := os.Open("testdata/input/anchor_1.png")
	if err != nil {
		return Mat{}, err
	}

	ancMat, err := DecodeImageToMat(r)
	if err != nil {
		return Mat{}, err
	}

	return ancMat, nil
}

func getTestAnchor2Mat() (Mat, error) {

	r, err := os.Open("testdata/input/anchor_2.png")
	if err != nil {
		return Mat{}, err
	}

	ancMat, err := DecodeImageToMat(r)
	if err != nil {
		return Mat{}, err
	}

	return ancMat, nil
}

func pointsAreClose(a, b image.Point, margin int) bool {
	return math.Abs(float64(a.X-b.X)) <= float64(margin) &&
		math.Abs(float64(a.Y-b.Y)) <= float64(margin)
}

// Copies the input matrix and draws a cross over it. The copy is returned.
func overlayCross(
	mat Mat,
	center image.Point,
	w, h uint,
	orientation float64,
) Mat {
	return overlayCrosses(mat, []image.Point{center}, w, h, orientation)
}

func overlayCrosses(
	mat Mat,
	centers []image.Point,
	w, h uint,
	orientation float64,
) Mat {
	copy := Clone(mat)
	angle := orientation * -1

	sin, cos := math.Sin(angle), math.Cos(angle)

	rotate := func(pt image.Point, x, y int) image.Point {
		rx := float64(x)*cos - float64(y)*sin
		ry := float64(x)*sin + float64(y)*cos

		return image.Pt(
			pt.X-int(math.Round(rx)),
			pt.Y-int(math.Round(ry)),
		)
	}

	for _, center := range centers {
		gocv.Line(
			&copy.m,
			rotate(center, 0, -int(h/2)),
			rotate(center, 0, int(h/2)),
			color.RGBA{255, 0, 0, 255},
			5,
		)
		gocv.Line(
			&copy.m,
			rotate(center, -int(w/2), 0),
			rotate(center, int(w/2), 0),
			color.RGBA{255, 0, 0, 255},
			5,
		)
	}

	return copy
}
func rad(degrees float64) float64 {
	return degrees / 180.0 * math.Pi
}

func deg(radians float64) float64 {
	return radians / math.Pi * 180.0
}

//
// Tests
//

func TestFindAnchors(t *testing.T) {
	tName := t.Name()
	tmpl, err := getSampleTemplate()
	assert.Assert(t, err == nil)
	noisyPage, err := getNoisyPageMat()
	assert.Assert(t, err == nil)

	t.Run("near-max rotation with noise", func(t *testing.T) {
		outputName := "testdata/output/" + tName + "_maxrotation_noisy.png"

		rotatedPage := Clone(noisyPage)
		Rotate(rotatedPage, rotatedPage, rad(-9.2), color.RGBA{})

		result, err := FindAnchors(
			rotatedPage,
			tmpl.Anchors[0],
			0.95,
			&FindAnchorConfig{
				MaxQuality:        0.95,
				SearchAreaPadding: 0.15,
			},
		)
		assert.Assert(t, err == nil)

		output := overlayCrosses(
			rotatedPage,
			result,
			400, 400,
			0,
		)
		defer output.Close()

		drawInputOutput(t, rotatedPage, output, outputName)
	})
}

func TestFindAnchor(t *testing.T) {
	tName := t.Name()
	tmpl, err := getSampleTemplate()
	assert.Assert(t, err == nil)
	page, err := getSamplePageMat()
	assert.Assert(t, err == nil)
	noisyPage, err := getNoisyPageMat()
	assert.Assert(t, err == nil)

	t.Run("unrotated", func(t *testing.T) {
		outputName := "testdata/output/" + tName + "_unrotated.png"

		result, err := findAnchor(
			page,
			tmpl.Anchors[0][0],
			FindAnchorConfig{
				MaxQuality: 0.99,
			},
		)
		assert.Assert(t, err == nil)
		assert.Assert(t, result.Confidence >= 0.99)
		assert.Assert(t, math.Abs(result.Orientation) <= rad(1))

		output := overlayCross(
			page,
			result.Position,
			500, 500,
			result.Orientation,
		)
		defer output.Close()

		drawInputOutput(t, page, output, outputName)
	})

	t.Run("unrotated with noise", func(t *testing.T) {
		outputName := "testdata/output/" + tName + "_unrotated_noisy.png"

		result, err := findAnchor(
			page,
			tmpl.Anchors[0][0],
			FindAnchorConfig{
				MaxQuality: 0.95,
			},
		)
		assert.Assert(t, err == nil)
		assert.Assert(t, result.Confidence >= 0.95)

		output := overlayCross(
			noisyPage,
			result.Position,
			500, 500,
			result.Orientation,
		)
		defer output.Close()

		drawInputOutput(t, noisyPage, output, outputName)
	})

	t.Run("near-max rotation with noise", func(t *testing.T) {
		outputName := "testdata/output/" + tName + "_maxrotation_noisy.png"

		rotatedPage := Clone(noisyPage)
		Rotate(rotatedPage, rotatedPage, rad(-9.2), color.RGBA{})

		result, err := findAnchor(
			rotatedPage,
			tmpl.Anchors[0][0],
			FindAnchorConfig{
				MaxQuality: 0.95,
			},
		)
		assert.Assert(t, err == nil)
		assert.Assert(t, result.Confidence >= 0.95)

		output := overlayCross(
			rotatedPage,
			result.Position,
			500, 500,
			result.Orientation,
		)
		defer output.Close()

		drawInputOutput(t, rotatedPage, output, outputName)
	})

	t.Run("various angles", func(t *testing.T) {
		maxOffBy := 0.0
		for angle := rad(-10.0); angle <= rad(10.0); angle += rad(3) {

			rotatedPage := NewMat()
			Rotate(rotatedPage, page, angle, color.RGBA{})

			result, err := findAnchor(
				rotatedPage,
				tmpl.Anchors[0][0],
				FindAnchorConfig{
					InitialAngle:       0,
					AngleSearchBreadth: rad(20),
					SearchAreaPadding:  0.125,
					Granularity:        7,
					MaxQuality:         0.99,
				},
			)
			offBy := deg(math.Abs(result.Orientation - angle))
			maxOffBy = max(maxOffBy, offBy)
			t.Logf(
				"true angle: %.2f\u00b0"+
					"\tdetected angle: %.2f\u00b0"+
					"\terror: %.5f\u00b0",
				deg(angle),
				deg(result.Orientation),
				offBy,
			)
			assert.Assert(t, err == nil)
			assert.Assert(t, result.Confidence >= 0.99)
		}
		t.Logf("maximum angle error: %.2f\u00b0", maxOffBy)
	})

}

func TestMatchAnchor(t *testing.T) {

	t.Run("bad inputs", func(t *testing.T) {
		tmpl, err := getSampleTemplate()
		assert.Assert(t, err == nil)
		defer tmpl.Close()

		_, _, err = matchAnchor(NewMat(), tmpl.Anchors[0][0], NewMat())
		assert.Assert(t, err == ErrEmptyMat)

		page, err := getSamplePageMat()
		assert.Assert(t, err == nil)
		defer page.Close()

		*tmpl.Anchors[0][0].Mat.t = MatTypeUnknown
		_, _, err = matchAnchor(page, tmpl.Anchors[0][0], NewMat())
		assert.Assert(t, err == ErrMatTypeMismatch)
	})

	t.Run("match anchor 0 on full page", func(t *testing.T) {
		tmpl, err := getSampleTemplate()
		assert.Assert(t, err == nil)
		defer tmpl.Close()
		page, err := getSamplePageMat()
		assert.Assert(t, err == nil)
		defer page.Close()

		anc := tmpl.Anchors[0][0]
		expectedCenter := image.Pt(
			int(float64(tmpl.Width)*anc.Pos.X)+int(anc.Mat.Width()/2),
			int(float64(tmpl.Height)*anc.Pos.Y)+int(anc.Mat.Height()/2),
		)

		// This is a practically ideal sample, so we should expect the match to
		// be very close and have a very high quality score.
		center, quality, err := matchAnchor(page, anc, NewMat())
		t.Log(NewMat().Empty())
		t.Log(err)
		assert.Assert(t, err == nil)
		assert.Assert(t, quality >= 0.95)
		assert.Assert(t, pointsAreClose(center, expectedCenter, 3))
	})

	t.Run("match anchor 0 on bad region", func(t *testing.T) {
		tmpl, err := getSampleTemplate()
		assert.Assert(t, err == nil)
		defer tmpl.Close()
		page, err := getSamplePageMat()
		assert.Assert(t, err == nil)
		defer page.Close()

		anc := tmpl.Anchors[0][0]

		// We deliberately cut the region to omit the anchor. We expect the
		// match quality to be low.
		anc.Pos = NormalPoint{0, 0}
		searchArea, _ := searchRegion(page, anc, 0.10)
		_, quality, err := matchAnchor(searchArea, anc, NewMat())
		assert.Assert(t, err == nil)
		assert.Assert(t, quality <= 0.50)
	})
}

func TestSearchRegion(t *testing.T) {

	mat, err := getSamplePageMat()
	assert.Assert(t, err == nil)
	defer mat.Close()

	t.Run("zero region", func(t *testing.T) {
		ancMat, err := getTestAnchor0Mat()
		assert.Assert(t, err == nil)
		defer ancMat.Close()

		var (
			anchor    = Anchor{Mat: NewMat()}
			region, _ = searchRegion(mat, anchor, 0)
		)
		assert.Assert(t, region.Width() == 0)
		assert.Assert(t, region.Height() == 0)
	})

	t.Run("zero padding region", func(t *testing.T) {
		ancMat, err := getTestAnchor0Mat()
		assert.Assert(t, err == nil)
		defer ancMat.Close()

		var (
			anchor    = Anchor{Mat: ancMat}
			region, _ = searchRegion(mat, anchor, 0)
		)
		assert.Assert(t, region.Width() == anchor.Mat.Width())
		assert.Assert(t, region.Height() == anchor.Mat.Height())
	})

	t.Run("safe region with padding", func(t *testing.T) {
		ancMat, err := getTestAnchor0Mat()
		assert.Assert(t, err == nil)
		defer ancMat.Close()

		var (
			anchor = Anchor{
				Mat: ancMat,
				Pos: NormalPoint{0.45, 0.45},
			}
			region, _ = searchRegion(mat, anchor, 0.10)
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
				Pos: NormalPoint{0, 0},
			}
			region, _ = searchRegion(mat, anchor, 0.50)
		)
		// We're putting the anchor in the top-left corner and asking for 50%
		// padding. We should expect that this makes it so the padded region
		// horizontally and vertically overflows on its left- and top-most
		// sides. Since this area should be omitted from the region, we should
		// hope that the region ends up with half the width and half the height
		// of the source.
		roughlyEqual := func(a, b uint) bool {
			if a > b {
				return a-b <= 3
			}
			if b > a {
				return b-a <= 3
			}
			return true
		}
		assert.Assert(t, roughlyEqual(region.Width(), mat.Width()/2))
		assert.Assert(t, roughlyEqual(region.Height(), mat.Height()/2))

		// Now we do the same test but in the bottom-right corner.
		anchor.Pos.X = 1.00
		anchor.Pos.Y = 1.0
		region.Close()
		region, _ = searchRegion(mat, anchor, 0.50)
		assert.Assert(t, roughlyEqual(region.Width(), mat.Width()/2))
		assert.Assert(t, roughlyEqual(region.Height(), mat.Height()/2))

		// Finally, we try to add padding larger than the region and confirm
		// that it's still bounded.
		region.Close()
		region, _ = searchRegion(mat, anchor, 1.50)
		assert.Assert(t, region.Width() == mat.Width())
		assert.Assert(t, region.Height() == mat.Height())
	})
}

func TestRotateAnchor(t *testing.T) {
	var tName = t.Name()

	t.Run("15deg", func(t *testing.T) {
		var outputName = "testdata/output/" + tName + "_15deg.png"
		var maskOutName = "testdata/output/" + tName + "_mask_15deg.png"

		ancMat, err := getTestAnchor0Mat()
		assert.Assert(t, err == nil)
		defer ancMat.Close()

		anc := Anchor{Mat: ancMat, Pos: NormalPoint{}}

		rotated, mask, err := RotateAnchor(anc, 15.0/180.0*math.Pi)
		assert.Assert(t, err == nil)

		drawInputOutput(t, ancMat, rotated.Mat, outputName)
		drawInputOutput(t, ancMat, mask, maskOutName)
	})

	t.Run("-15deg", func(t *testing.T) {
		var outputName = "testdata/output/" + tName + "_-15deg.png"

		ancMat, err := getTestAnchor0Mat()
		assert.Assert(t, err == nil)
		defer ancMat.Close()

		anc := Anchor{Mat: ancMat, Pos: NormalPoint{}}

		rotated, mask, err := RotateAnchor(anc, -15.0/180.0*math.Pi)
		gocv.BitwiseAnd(rotated.Mat.m, mask.m, &rotated.Mat.m)
		assert.Assert(t, err == nil)

		drawInputOutput(t, ancMat, rotated.Mat, outputName)
	})
}

func TestRefiningSearch(t *testing.T) {

	t.Run("approximates the square root of two", func(t *testing.T) {
		best, err := refiningSearch(
			1, 2,
			0,
			11,
			math.Inf(+1),
			isSqrtOfTwo,
		)
		assert.Assert(t, err == nil)
		assert.Assert(t, math.Abs(best.candidate-math.Sqrt2) <= 0.01)
	})

	t.Run("hits max iterations error", func(t *testing.T) {
		_, err := refiningSearch(
			0, 100,
			0,
			3,
			math.Inf(+1),
			makeDivergentFunc(),
		)
		assert.Assert(t, err == ErrMaxIterations)
	})
}

//
// Benchmarks
//

func BenchmarkFindAnchors(b *testing.B) {
	tmpl, err := getSampleTemplate()
	assert.Assert(b, err == nil)
	page, err := getSamplePageMat()
	assert.Assert(b, err == nil)
	rotated := Clone(page)
	err = Rotate(rotated, rotated, rad(-1.23), color.RGBA{})
	assert.Assert(b, err == nil)

	// This configuration is wide enough to support +/- 10 degree rotations.
	// In practice, most scanning machines will have a much better skews which
	// will allow for tighter searches (which are much, much faster). We're
	// just benchmarking the worst case.
	conf := FindAnchorConfig{
		InitialAngle:       0,
		AngleSearchBreadth: rad(20),
		SearchAreaPadding:  0.15,
		Granularity:        7,
		MaxQuality:         0.95,
	}

	b.Run("unrotated with good initial guess", func(b *testing.B) {
		for b.Loop() {
			FindAnchors(rotated, tmpl.Anchors[0], 0.95, &conf)
		}
	})

	b.Run("rotated with bad initial guess", func(b *testing.B) {
		for b.Loop() {
			FindAnchors(rotated, tmpl.Anchors[0], 0.95, &conf)
		}
	})

	b.Run("rotated with good initial guess", func(b *testing.B) {
		conf := conf
		conf.InitialAngle = rad(-1.23)
		for b.Loop() {
			FindAnchors(rotated, tmpl.Anchors[0], 0.95, &conf)
		}
	})
}

func BenchmarkFindAnchor(b *testing.B) {

	tmpl, err := getSampleTemplate()
	assert.Assert(b, err == nil)
	page, err := getSamplePageMat()
	assert.Assert(b, err == nil)
	rotated := Clone(page)
	err = Rotate(rotated, rotated, rad(-1.23), color.RGBA{})
	assert.Assert(b, err == nil)

	// This configuration is wide enough to support +/- 10 degree rotations.
	// In practice, most scanning machines will have a much better skews which
	// will allow for tighter searches (which are much, much faster). We're
	// just benchmarking the worst case.
	conf := FindAnchorConfig{
		InitialAngle:       0,
		AngleSearchBreadth: rad(20),
		SearchAreaPadding:  0.15,
		Granularity:        7,
		MaxQuality:         0.99,
	}

	b.Run("unrotated with good initial guess", func(b *testing.B) {
		for b.Loop() {
			findAnchor(page, tmpl.Anchors[0][0], conf)
		}
	})

	b.Run("rotated with bad initial guess", func(b *testing.B) {
		for b.Loop() {
			findAnchor(rotated, tmpl.Anchors[0][0], conf)
		}
	})

	b.Run("rotated with good initial guess", func(b *testing.B) {
		conf := conf
		conf.InitialAngle = rad(-1.23)
		for b.Loop() {
			findAnchor(rotated, tmpl.Anchors[0][0], conf)
		}
	})
}
