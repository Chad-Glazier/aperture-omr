package omr

import (
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
		BlurSize: 3,
		MorphCloseSize: 3,
		BlockSize: 91,
		AdaptiveC: -5,
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
		X: 24,
		Y: 24,
	}
	tmpl.Anchors[0][1] = Anchor{
		Mat: Clone(ancMat),
		X: 1152,
		Y: 24,
	}
	tmpl.Anchors[0][2] = Anchor{
		Mat: Clone(ancMat),
		X: 24,
		Y: 1652,
	}
	tmpl.Anchors[1][0] = Anchor{
		Mat: ancMat,
		X: 24,
		Y: 24,
	}
	tmpl.Anchors[1][1] = Anchor{
		Mat: Clone(ancMat),
		X: 1152,
		Y: 24,
	}
	tmpl.Anchors[1][2] = Anchor{
		Mat: Clone(ancMat),
		X: 1152,
		Y: 1652,
	}

	return &tmpl
}


//
// Tests
// 


