package extractor

import (
	"fmt"
	"image"
	"math"
	"os"
	"strings"
	"text/tabwriter"

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
	Options    []Bubble `json:"options"`
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
	if len(tmp.Questions) == 0 {
		return nil, fmt.Errorf("cannot extract from empty template")
	}

	result := &Result{
		Answers: make([]Answer, len(tmp.Questions)),
	}

	// Based on our inverted binarization, a fully filled bubble should be near 100% white.
	// A threshold of 45% is strict enough to ignore stray eraser smudges, but forgiving
	// enough for students who don't completely fill the bubble edge-to-edge.
	const threshold = 0.45

	for i, q := range tmp.Questions {
		selected, confidence := detectAnswers(img, q, threshold)
		flag := confidence < 0.5 || len(selected) != 1

		result.Answers[i] = Answer{
			QuestionID: q.ID,
			Options:    q.Options,
			Selected:   selected,
			Confidence: confidence,
			Flag:       flag,
		}
	}

	print(Result{
		Answers: result.Answers[:12],
	})

	return result, nil
}

func detectAnswers(img gocv.Mat, q Question, threshold float64) ([]string, float64) {
	var answered []string

	var highestFill float64 = 0.0
	var nextFill float64 = 0.0

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

		fillRatio := float64(filled) / float64(total)

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

func print(r Result) {
	fmt.Println("\n======================================================")
	fmt.Println("             OMR BUBBLE EXTRACTION REPORT             ")
	fmt.Println("======================================================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)

	fmt.Fprintln(w, "QUESTION\tSELECTED\tCONFIDENCE\tMANUAL REVIEW FLAG")
	fmt.Fprintln(w, "--------\t--------\t----------\t------------------")

	for _, ans := range r.Answers {
		fmt.Fprintf(w, "%s\t%s\t%.2f\t%t\n",
			ans.QuestionID,
			strings.Join(ans.Selected, ", "),
			ans.Confidence,
			ans.Flag,
		)
	}

	w.Flush()
	fmt.Println("======================================================")
}
