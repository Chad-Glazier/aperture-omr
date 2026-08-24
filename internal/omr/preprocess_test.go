package omr

import (
	"fmt"
	"image/color"
	"image/png"
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

//
// Helpers
//

func getSampleTemplate() (PreprocessingTemplate, error) {

	tmpl := PreprocessingTemplate{}
	tmpl.Width = 1952
	tmpl.Height = 2496
	tmpl.MinAnchorConfidence = 0.50

	ancMat0, err := getTestAnchor0Mat()
	if err != nil {
		return PreprocessingTemplate{}, err
	}

	ancMat1, err := getTestAnchor1Mat()
	if err != nil {
		return PreprocessingTemplate{}, err
	}

	ancMat2, err := getTestAnchor2Mat()
	if err != nil {
		return PreprocessingTemplate{}, err
	}

	tmpl.Anchors = make([][]Anchor, 1)
	tmpl.Anchors[0] = make([]Anchor, 3)
	tmpl.Anchors[0][0] = Anchor{
		Mat: ancMat0,
		Pos: NormalPoint{
			X: float64(1680) / float64(tmpl.Width),
			Y: float64(1710) / float64(tmpl.Height),
		},
	}
	tmpl.Anchors[0][1] = Anchor{
		Mat: ancMat1,
		Pos: NormalPoint{
			X: float64(164) / float64(tmpl.Width),
			Y: float64(144) / float64(tmpl.Height),
		},
	}
	tmpl.Anchors[0][2] = Anchor{
		Mat: ancMat2,
		Pos: NormalPoint{
			X: float64(164) / float64(tmpl.Width),
			Y: float64(2260) / float64(tmpl.Height),
		},
	}

	tmpl.AnchorSearchConfig = &FindAnchorConfig{
		MaxQuality: 0.85,
		Granularity: 5,
	}

	return tmpl, nil
}

//
// Tests
//

func TestPreprocess(t *testing.T) {
	tmpl, err := getSampleTemplate()
	assert.Assert(t, err == nil)
	defer tmpl.Close()
	
	noisyPage, err := getNoisyPageMat()
	assert.Assert(t, err == nil)
	defer noisyPage.Close()

	page, err := getSamplePageMat()
	assert.Assert(t, err == nil)
	defer page.Close()

	tName := t.Name()

	t.Run("preprocess ideal input", func(t *testing.T) {
		var output = "testdata/output/" + tName + "_ideal.png"

		result, err := Preprocess(tmpl, []Mat{ page })
		assert.Assert(t, err == nil)

		drawInputOutput(t, page, result[0], output)
	})

	t.Run("preprocess noisy and rotated", func(t *testing.T) {
		rotated := NewMat()
		defer rotated.Close()

		for _, degrees := range []float64{ 1.5, -5 } {

			t.Logf("trying %.1f\u00b0 rotation", degrees)

			var output = fmt.Sprintf(
				"testdata/output/%s_%.0fdeg_noisy.png",
				tName, degrees,
			)

			RotateWithoutResizing(
				rotated, 
				noisyPage, 
				rad(degrees), 
				color.RGBA{},
			)

			result, err := Preprocess(tmpl, []Mat{ rotated })
			assert.Assert(t, err == nil)

			drawInputOutput(t, rotated, result[0], output)
		}
	})
}

func TestPreprocessingTemplate(t *testing.T) {

	tmpl, err := getSampleTemplate()
	assert.Assert(t, err == nil)
	defer tmpl.Close()

	tName := t.Name()

	t.Run("draw sample template", func(t *testing.T) {
		var output = "testdata/output/" + tName + "_unscaled_out.png"

		w, err := os.Create(output)
		assert.NilError(t, err)
		defer w.Close()

		page0, err := tmpl.ToImage(0)
		assert.NilError(t, err)

		err = png.Encode(w, page0)
		assert.NilError(t, err)

		t.Log("output written to " + output)
	})

	t.Run("draw scaled up cover", func(t *testing.T) {
		var output = "testdata/output/" + tName + "_scaleup_out.png"

		w, err := os.Create(output)
		assert.NilError(t, err)
		defer w.Close()

		scaledUp, err := ScalePreprocessingTemplate(
			FitMethodCover,
			tmpl,
			3000, 3000,
		)
		assert.NilError(t, err)
		defer scaledUp.Close()

		// Since we're using "cover" and targetting 2000x2000, we expect that
		// the larger dimension is greater than or equal to 2000 and the
		// smaller is exactly equal to 2000.
		assert.Assert(t, min(scaledUp.Width, scaledUp.Height) == 3000)
		assert.Assert(t, max(scaledUp.Width, scaledUp.Height) >= 3000)

		page0, err := scaledUp.ToImage(0)
		assert.NilError(t, err)

		err = png.Encode(w, page0)
		assert.NilError(t, err)

		t.Log("output written to " + output)
	})

	t.Run("draw scaled down contain", func(t *testing.T) {
		var output = "testdata/output/" + tName + "_scaledown_out.png"

		w, err := os.Create(output)
		assert.NilError(t, err)
		defer w.Close()

		scaledDown, err := ScalePreprocessingTemplate(
			FitMethodContain,
			tmpl,
			1000, 1000,
		)
		assert.NilError(t, err)
		defer scaledDown.Close()

		// Since we're using "contain" and targetting 1000x1000, we expect that
		// the larger dimension is exactly 1000 and the smaller is less than or
		// equal to 1000.
		assert.Assert(t, max(scaledDown.Width, scaledDown.Height) == 1000)
		assert.Assert(t, min(scaledDown.Width, scaledDown.Height) <= 1000)

		page0, err := scaledDown.ToImage(0)
		assert.NilError(t, err)

		err = png.Encode(w, page0)
		assert.NilError(t, err)

		t.Log("output written to " + output)
	})

	t.Run("draw scaled down fill", func(t *testing.T) {
		var output = "testdata/output/" + tName + "_scaledown_fill_out.png"

		w, err := os.Create(output)
		assert.NilError(t, err)
		defer w.Close()

		scaledDown, err := ScalePreprocessingTemplate(
			FitMethodFill,
			tmpl,
			800, 800,
		)
		assert.NilError(t, err)
		defer scaledDown.Close()

		// Since we're using "fill" and targetting 800x800, we expect that both
		// dimensions are exactly 800.
		assert.Assert(t, scaledDown.Height == 800)
		assert.Assert(t, scaledDown.Width == 800)

		page0, err := scaledDown.ToImage(0)
		assert.NilError(t, err)

		err = png.Encode(w, page0)
		assert.NilError(t, err)

		t.Log("output written to " + output)
	})
}

//
// Benchmarks
//

func BenchmarkPreprocess(b *testing.B) {
	tmpl, err := getSampleTemplate()
	assert.Assert(b, err == nil)
	defer tmpl.Close()
	
	noisyPage, err := getNoisyPageMat()
	assert.Assert(b, err == nil)
	defer noisyPage.Close()

	page, err := getSamplePageMat()
	assert.Assert(b, err == nil)
	defer page.Close()

	rotated := NewMat()
	defer rotated.Close()
	Rotate(rotated, noisyPage, rad(3), color.RGBA{})

	b.Run("preprocess ideal", func(b *testing.B) {
		for b.Loop() {
			Preprocess(tmpl, []Mat{ page })
		}
	})

	b.Run("preprocess noisy rotated", func(b *testing.B) {
		for b.Loop() {
			Preprocess(tmpl, []Mat{ rotated })
		}
	})
}
