package marker

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"

	"gocv.io/x/gocv"
)

type Bubble struct {
	Label string `json:"label"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
}

type Question struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"` // "single" (default) or "multi"
	BubbleWidth  int      `json:"bubbleWidth"`
	BubbleHeight int      `json:"bubbleHeight"`
	Options      []Bubble `json:"options"`
}

// Config holds marking-specific parameters. Nil fields fall back to defaults:
// fillThreshold: 0.5, bubbleInset: 0.75, flagThreshold: 0.5.
type Config struct {
	FillThreshold *float64 `json:"fillThreshold"`
	BubbleInset   *float64 `json:"bubbleInset"`
	// FlagThreshold is the minimum confidence below which an answer is flagged
	// for manual review. Set to 0.0 to disable confidence-based flagging.
	FlagThreshold *float64 `json:"flagThreshold"`
}

// Page holds the questions for a single exam page.
type Page struct {
	Questions []Question `json:"questions"`
}

// Template is the mark template for a single- or multi-page exam.
// Multi-page templates populate Pages; single-page templates may use the
// top-level Questions field for backward compatibility.
type Template struct {
	Config    Config     `json:"config"`
	Questions []Question `json:"questions,omitempty"` // single-page backward compat
	Pages     []Page     `json:"pages,omitempty"`     // multi-page
}

// pages returns the per-page question lists. Single-page templates that use
// the top-level Questions field are treated as a one-page template.
func (t *Template) pages() []Page {
	if len(t.Pages) > 0 {
		return t.Pages
	}
	return []Page{{Questions: t.Questions}}
}

// LoadTemplate parses a marking template from r.
func LoadTemplate(r io.Reader) (*Template, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	var tmpl Template
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return &tmpl, nil
}

type Answer struct {
	QuestionID string   `json:"questionID"`
	Selected   []string `json:"selected"`
	Confidence float64  `json:"confidence"`
	Flag       bool     `json:"flag"`
}

type Result struct {
	Answers []Answer
}

// Evaluate scores every question across all pages of the template.
// imgs must contain one binary image per page (output of the preprocessing
// pipeline), in page order. For single-page templates a one-element slice is
// expected.
func Evaluate(imgs []gocv.Mat, tmpl *Template) (*Result, error) {
	pages := tmpl.pages()
	if len(imgs) != len(pages) {
		return nil, fmt.Errorf("template has %d page(s), got %d image(s)", len(pages), len(imgs))
	}

	threshold := 0.5
	if tmpl.Config.FillThreshold != nil {
		threshold = *tmpl.Config.FillThreshold
	}
	inset := 0.75
	if tmpl.Config.BubbleInset != nil {
		inset = *tmpl.Config.BubbleInset
	}
	flagThreshold := 0.5
	if tmpl.Config.FlagThreshold != nil {
		flagThreshold = *tmpl.Config.FlagThreshold
	}

	var answers []Answer
	for i, img := range imgs {
		if img.Empty() {
			return nil, fmt.Errorf("page %d: cannot evaluate an empty image", i)
		}
		if len(pages[i].Questions) == 0 {
			return nil, fmt.Errorf("page %d: mark template contains no questions", i)
		}
		for _, q := range pages[i].Questions {
			selected, confidence := detectAnswers(img, q, threshold, inset)
			multiSelect := q.Type == "multi"
			flag := confidence < flagThreshold ||
				len(selected) == 0 ||
				(!multiSelect && len(selected) > 1)
			answers = append(answers, Answer{
				QuestionID: q.ID,
				Selected:   selected,
				Confidence: confidence,
				Flag:       flag,
			})
		}
	}

	return &Result{Answers: answers}, nil
}

func detectAnswers(
	img gocv.Mat, q Question, threshold, inset float64,
) ([]string, float64) {
	var answered []string
	var selectedFills []float64
	var highestFill float64
	var highestUnselected float64

	for _, bubble := range q.Options {
		fillRatio := bubbleFillRatio(img, bubble, q.BubbleWidth, q.BubbleHeight, inset)

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

// bubbleFillRatio returns the fraction of pixels inside the bubble's inset
// circle that are non-zero (i.e. filled) in the binary image.
// w and h are the bubble dimensions from the question; inset is the fraction
// of the bubble radius to sample (e.g. 0.75 skips the outer border ring).
// The ROI is clamped to image bounds so a bubble placed one pixel over the
// edge doesn't cause an OpenCV assertion failure.
func bubbleFillRatio(img gocv.Mat, b Bubble, w, h int, inset float64) float64 {
	x0 := max(b.X, 0)
	y0 := max(b.Y, 0)
	x1 := min(b.X+w, img.Cols())
	y1 := min(b.Y+h, img.Rows())

	if x1 <= x0 || y1 <= y0 {
		return 0.0
	}

	cw := x1 - x0
	ch := y1 - y0

	roi := img.Region(image.Rect(x0, y0, x1, y1))
	defer roi.Close()

	mask := gocv.NewMatWithSize(ch, cw, gocv.MatTypeCV8U)
	defer mask.Close()

	r := w / 2
	if h < w {
		r = h / 2
	}
	innerR := int(float64(r) * inset)
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	gocv.Circle(&mask, image.Pt(cw/2, ch/2), innerR, white, -1)

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
