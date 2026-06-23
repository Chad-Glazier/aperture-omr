package marker

import (
	"fmt"
	"image"
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

func Evaluate(img gocv.Mat, questions []scanner.Question) (*Result, error) {
	if img.Empty() {
		return nil, fmt.Errorf("cannot extract from empty image")
	}
	if len(questions) == 0 {
		return nil, fmt.Errorf("cannot extract from empty template")
	}

	result := &Result{
		Answers: make([]Answer, len(questions)),
	}

	// Based on our inverted binarization, a fully filled bubble should be near 100% white.
	// A threshold of 45% is strict enough to ignore stray eraser smudges, but forgiving
	// enough for students who don't completely fill the bubble edge-to-edge.
	const threshold = 0.45

	for i, q := range questions {
		selected, confidence := detectAnswers(img, q, threshold)
		flag := confidence < 0.5 || len(selected) != 1

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

	var highestFill float64
	var nextFill float64

	for _, bubble := range q.Options {
		fillRatio := bubbleFillRatio(img, bubble)

		if fillRatio >= threshold {
			answered = append(answered, bubble.Label)
		}

		if fillRatio > highestFill {
			nextFill = highestFill
			highestFill = fillRatio
		} else if fillRatio > nextFill {
			nextFill = fillRatio
		}
	}

	var confidence float64
	if len(answered) == 0 {
		confidence = 1.0 - highestFill
	} else {
		confidence = highestFill - nextFill
	}

	confidence = math.Max(confidence, 0.0)
	confidence = math.Min(confidence, 1.0)

	return answered, confidence
}

func bubbleFillRatio(img gocv.Mat, b scanner.Bubble) float64 {
	roi := img.Region(image.Rect(b.X, b.Y, b.X+b.Width, b.Y+b.Height))
	defer roi.Close()

	filled := gocv.CountNonZero(roi)
	return float64(filled) / float64(b.Width*b.Height)
}
