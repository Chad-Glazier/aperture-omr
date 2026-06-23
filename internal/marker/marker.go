package marker

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"ubco-team15/omr/internal/scanner"

	"gocv.io/x/gocv"
)

type Answer struct {
	QuestionID string   `json:"questionID"`
	Selected   []string `json:"selected"`
	Confidence float64  `json:"confidence"`
	Flag       bool     `json:"flag"`
}

type Result struct {
	Answers []Answer
}

func Evaluate(img gocv.Mat, questions []scanner.Question, threshold float64) (*Result, error) {
	if img.Empty() {
		return nil, fmt.Errorf("cannot extract from empty image")
	}
	if len(questions) == 0 {
		return nil, fmt.Errorf("cannot extract from empty template")
	}

	result := &Result{
		Answers: make([]Answer, len(questions)),
	}

	// Fill ratio is measured against the inner 75% of the bubble radius (see bubbleFillRatio),
	// so the printed border ring is excluded.
	for i, q := range questions {
		selected, confidence := detectAnswers(img, q, threshold)
		flag := confidence < 0.5 || len(selected) == 0

		result.Answers[i] = Answer{
			QuestionID: q.ID,
			Selected:   selected,
			Confidence: confidence,
			Flag:       flag,
		}
	}

	return result, nil
}

func detectAnswers(img gocv.Mat, q scanner.Question, threshold float64) ([]string, float64) {
	var answered []string
	var selectedFills []float64
	var highestFill float64
	var highestUnselected float64

	for _, bubble := range q.Options {
		fillRatio := bubbleFillRatio(img, bubble)

		if fillRatio > highestFill {
			highestFill = fillRatio
		}

		if fillRatio >= threshold {
			answered = append(answered, bubble.Label)
			selectedFills = append(selectedFills, fillRatio)
		} else if fillRatio > highestUnselected {
			highestUnselected = fillRatio
		}
	}

	var confidence float64
	if len(answered) == 0 {
		confidence = 1.0 - highestFill
	} else {
		// Weakest selected minus strongest unselected: measures how clearly the
		// marked bubbles are separated from the unmarked ones regardless of count.
		minSelected := selectedFills[0]
		for _, f := range selectedFills[1:] {
			if f < minSelected {
				minSelected = f
			}
		}
		confidence = minSelected - highestUnselected
	}

	confidence = math.Max(confidence, 0.0)
	confidence = math.Min(confidence, 1.0)

	return answered, confidence
}

func bubbleFillRatio(img gocv.Mat, b scanner.Bubble) float64 {
	roi := img.Region(image.Rect(b.X, b.Y, b.X+b.Width, b.Y+b.Height))
	defer roi.Close()

	// Sample only the inner 75% of the bubble radius so the printed border ring
	// is excluded entirely. Empty bubbles read near 0; marked ones read high.
	mask := gocv.NewMatWithSize(b.Height, b.Width, gocv.MatTypeCV8U)
	defer mask.Close()

	r := b.Width / 2
	if b.Height < b.Width {
		r = b.Height / 2
	}
	innerR := int(float64(r) * 0.75)
	gocv.Circle(&mask, image.Pt(b.Width/2, b.Height/2), innerR, color.RGBA{255, 255, 255, 255}, -1)

	masked := gocv.NewMat()
	defer masked.Close()
	gocv.BitwiseAnd(roi, mask, &masked)

	circleArea := gocv.CountNonZero(mask)
	filled := gocv.CountNonZero(masked)

	if circleArea == 0 {
		return 0.0
	}

	return float64(filled) / float64(circleArea)
}
