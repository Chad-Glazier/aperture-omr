package omr

import (
	"image/png"
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

//
// Helpers
//

func getSampleTemplate(t *testing.T) *PreprocessingTemplate {
	t.Helper()

	tmpl := PreprocessingTemplate{}
	tmpl.Width = 1200
	tmpl.Height = 1700
	tmpl.BinarizeConfig = BinarizeConfig{
		BlurSize:       3,
		MorphCloseSize: 3,
		BlockSize:      51,
		AdaptiveC:      10,
	}
	tmpl.MinAnchorConfidence = 0.50
	tmpl.Width = 1200
	tmpl.Height = 1700

	r, err := os.Open("testdata/input/anchor.jpeg")
	assert.NilError(t, err)

	ancMat, err := DecodeImageToMat(r)
	assert.NilError(t, err)

	tmpl.Anchors = make([][]Anchor, 2)
	tmpl.Anchors[0] = make([]Anchor, 3)
	tmpl.Anchors[1] = make([]Anchor, 3)
	tmpl.Anchors[0][0] = Anchor{
		Mat: ancMat,
		Pos: NormalCoordinate{
			X:   float64(24)/float64(tmpl.Width),
			Y:   float64(24)/float64(tmpl.Height),			
		},
	}
	tmpl.Anchors[0][1] = Anchor{
		Mat: Clone(ancMat),
		Pos: NormalCoordinate{
			X:   float64(1152)/float64(tmpl.Width),
			Y:   float64(24)/float64(tmpl.Height),	
		},
	}
	tmpl.Anchors[0][2] = Anchor{
		Mat: Clone(ancMat),
		Pos: NormalCoordinate{
			X:   float64(24)/float64(tmpl.Width),
			Y:   float64(1652)/float64(tmpl.Height),	
		},
	}
	tmpl.Anchors[1][0] = Anchor{
		Mat: ancMat,
		Pos: NormalCoordinate{
			X:   float64(24)/float64(tmpl.Width),
			Y:   float64(24)/float64(tmpl.Height),	
		},
	}
	tmpl.Anchors[1][1] = Anchor{
		Mat: Clone(ancMat),
		Pos: NormalCoordinate{
			X:   float64(1152)/float64(tmpl.Width),
			Y:   float64(24)/float64(tmpl.Height),	
		},
	}
	tmpl.Anchors[1][2] = Anchor{
		Mat: Clone(ancMat),
		Pos: NormalCoordinate{
			X:   float64(1152)/float64(tmpl.Width),
			Y:   float64(1652)/float64(tmpl.Height),	
		},
	}

	return &tmpl
}

//
// Tests
//

func TestPreprocessingTemplate(t *testing.T) {

	tmpl := getSampleTemplate(t)
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
			2000, 2000,
		)
		assert.NilError(t, err)
		defer scaledUp.Close()

		// Since we're using "cover" and targetting 2000x2000, we expect that 
		// the larger dimension is greater than or equal to 2000 and the 
		// smaller is exactly equal to 2000. 
		assert.Assert(t, min(scaledUp.Width, scaledUp.Height) == 2000)
		assert.Assert(t, max(scaledUp.Width, scaledUp.Height) >= 2000)

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
