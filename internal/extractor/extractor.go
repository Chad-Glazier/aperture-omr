package extractor

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

type Bubble struct {
	Label  string `json:"label"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Answer struct {
	QuestionID string   `json:"questionID"`
	Selected   []string `json:"selected"`
	Confidence float64  `json:"confidence"`
	Flag       bool     `json:"flag"`
}

type Question struct {
	ID      string   `json:"id"`
	Options []Bubble `json:"options"`
}

type Template struct {
	Questions []Question `json:"questions"`
}

type Result struct {
	Answers []Answer
}

func Extract(img gocv.Mat, tmp Template) (*Result, error) {
	if img.Empty() {
		return nil, fmt.Errorf("cannot extract from empty image")
	}

	result := &Result{
		Answers: make([]Answer, len(tmp.Questions)),
	}

	// Based on our inverted binarization, a fully filled bubble should be near 100% white.
	// A threshold of 45% is strict enough to ignore stray eraser smudges, but forgiving
	// enough for students who don't completely fill the bubble edge-to-edge.
	const threshold = 0.45

	for _, q := range tmp.Questions {
		selected := detectAnswers(img, q, threshold)

		confidence := 1.0
		flag := confidence < 0.5 || len(selected) != 1

		result.Answers = append(result.Answers, Answer{
			QuestionID: q.ID,
			Selected:   selected,
			Confidence: confidence,
			Flag:       flag,
		})
	}

	return result, nil
}

func detectAnswers(img gocv.Mat, q Question, threshold float64) []string {
	var answered []string

	for _, bubble := range q.Options {
		x := bubble.X
		y := bubble.Y
		w := bubble.Width
		h := bubble.Height

		rect := image.Rect(x, y, x+w, y+h)

		roi := img.Region(rect)
		defer roi.Close()

		filled := gocv.CountNonZero(roi)
		total := w * h

		ratio := float64(filled) / float64(total)

		if ratio >= threshold {
			answered = append(answered, bubble.Label)
		}
	}

	return answered
}
