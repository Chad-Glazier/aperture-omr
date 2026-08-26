package omr

import (
	"fmt"
	"image/color"
	"testing"

	"gocv.io/x/gocv"
	"gotest.tools/v3/assert"
)

//
// Helper functions
//

func getSampleMarkTemplate() MarkTemplate {
	tmpl := MarkTemplate{
		Aspect:       1952.0 / 2496.0,
		BubbleRadius: 52.0 / 2 / 1952.0,
		Questions:    make([]Questions, 0),
	}
	tmpl.Questions = append(tmpl.Questions, make(Questions))

	bubbles := func(firstX, firstY float64) []Bubble {
		out := make([]Bubble, 5)
		for i, id := range []string{"A", "B", "C", "D", "E" } {
			out[i].Id = id
			out[i].Pos.X = float64(i) * (69.0 / 1952.0) + firstX
			out[i].Pos.Y = firstY
		}
		return out
	}

	bubbleCol := func(firstX, firstY float64, n uint) [][]Bubble {
		out := make([][]Bubble, n)
		for i := range n {
			y := firstY + float64(i) * (80.0 / 2496.0)
			out[i] = bubbles(firstX, y)
		}
		return out
	}

	questions := bubbleCol(418.0 / 1952.0, 512.0 / 2496.0, 10)
	for i, q := range questions {
		qId := fmt.Sprintf("Q%d", i+1)
		tmpl.Questions[0][qId] = q
	}

	return tmpl
}

//
// Tests
//

func TestMarkTemplateImage(t *testing.T) {

	tName := t.Name()
	page, err := getSamplePageMat()
	assert.Assert(t, err == nil)
	defer page.Close()

	t.Run("draw template", func(t *testing.T) {
		output := "testdata/output/" + tName + ".png"

		tmpl := getSampleMarkTemplate()
		mask, err := tmpl.Mask(0, 2496)
		assert.Assert(t, err == nil)
		defer mask.Close()

		gocv.BitwiseAnd(page.m, mask.m, &mask.m)

		drawInputOutput(t, page, mask, output)
	})

	t.Run("draw template over preprocessed image", func(t *testing.T) {
		output := "testdata/output/" + tName + "_preprocessed.png"

		page, err := getNoisyPageMat()
		assert.Assert(t, err == nil)
		defer page.Close()

		err = Rotate(page, page, rad(5), color.RGBA{})
		assert.Assert(t, err == nil)

		pTmpl, err := getSampleTemplate()
		assert.Assert(t, err == nil)

		preprocessed, err := Preprocess(pTmpl, []Mat{ page } )
		assert.Assert(t, err == nil)

		tmpl := getSampleMarkTemplate()
		mask, err := tmpl.Mask(0, int(preprocessed[0].Height()))
		assert.Assert(t, err == nil)
		defer mask.Close()

		gocv.BitwiseAnd(preprocessed[0].m, mask.m, &mask.m)

		drawInputOutput(t, page, mask, output)
	})

}
