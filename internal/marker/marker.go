package marker

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"sort"
	"strings"

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
// fillThreshold: 0.25, bubbleInset: 0.75, flagThreshold: 0.5, searchRadius: 0.
type Config struct {
	FillThreshold *float64 `json:"fillThreshold"`
	BubbleInset   *float64 `json:"bubbleInset"`
	// FlagThreshold is the minimum confidence below which an answer is flagged
	// for manual review. Set to 0.0 to disable confidence-based flagging.
	FlagThreshold *float64 `json:"flagThreshold"`
	// SearchRadius is a per-axis pixel search window applied around each
	// bubble's template position. The fill ratio returned is the maximum found
	// across all (2*r+1)² candidate centres. Use 3–5 to tolerate small
	// printer/scanner misalignment without changing the warp pipeline.
	SearchRadius *int `json:"searchRadius"`
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

	threshold := 0.25
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
	searchRadius := 0
	if tmpl.Config.SearchRadius != nil {
		searchRadius = *tmpl.Config.SearchRadius
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
			selected, confidence := detectAnswers(img, q, threshold, inset, searchRadius)
			multiSelect := q.Type == "multi"
			flag := confidence < flagThreshold ||
				(len(selected) == 0 && strings.HasPrefix(q.ID, "Q")) ||
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
	img gocv.Mat, q Question, threshold, inset float64, searchRadius int,
) ([]string, float64) {
	if len(q.Options) == 0 {
		return nil, 0.0
	}

	n := len(q.Options)
	w, h := q.BubbleWidth, q.BubbleHeight

	// Pre-build the inset circle mask once for this question.
	r := w / 2
	if h < w {
		r = h / 2
	}
	innerR := int(float64(r) * inset)
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	qMask := gocv.NewMatWithSize(h, w, gocv.MatTypeCV8U)
	defer qMask.Close()
	gocv.Circle(&qMask, image.Pt(w/2, h/2), innerR, white, -1)
	circleArea := gocv.CountNonZero(qMask)
	masked := gocv.NewMat()
	defer masked.Close()

	measure := func(cx, cy int) float64 {
		if circleArea == 0 {
			return 0.0
		}
		x0, y0 := cx-w/2, cy-h/2
		x1, y1 := x0+w, y0+h
		if x0 < 0 || y0 < 0 || x1 > img.Cols() || y1 > img.Rows() {
			return 0.0
		}
		roi := img.Region(image.Rect(x0, y0, x1, y1))
		defer roi.Close()
		gocv.BitwiseAnd(roi, qMask, &masked)
		return float64(gocv.CountNonZero(masked)) / float64(circleArea)
	}

	// Per-question alignment search: find the single (dx, dy) offset within
	// ±searchRadius that maximises the largest gap between consecutive sorted
	// fills across all options. This directly optimises for detection
	// discrimination — the offset where selected bubbles stand out most
	// clearly from empty ones. Maximising raw peak fill instead can select
	// offsets where printed ring pixels inflate all bubbles equally, shrinking
	// the gap and producing spurious multi-selections. Using the gap metric
	// avoids those offsets because uniform inflation across all options leaves
	// the gap unchanged or smaller.
	bestDX, bestDY := 0, 0
	if searchRadius > 0 {
		sortBuf := make([]float64, n)

		bestGapScore := 0.0
		for dy := -searchRadius; dy <= searchRadius; dy++ {
			for dx := -searchRadius; dx <= searchRadius; dx++ {
				for i, b := range q.Options {
					sortBuf[i] = measure(b.X+dx, b.Y+dy)
				}
				sort.Float64s(sortBuf)
				maxGap := 0.0
				for i := 1; i < n; i++ {
					if g := sortBuf[i] - sortBuf[i-1]; g > maxGap {
						maxGap = g
					}
				}
				if maxGap > bestGapScore {
					bestGapScore = maxGap
					bestDX, bestDY = dx, dy
				}
			}
		}

		// If the gap found is below the fill threshold, the discrimination is
		// too small to be meaningful — fills are nearly uniform across all
		// options. This happens when all bubbles are selected (or all empty).
		// In this regime the gap search chases noise and may land on a position
		// where accidental pixel variation looks like a gap but the absolute
		// fills are poor. Fall back instead to the offset that maximises the
		// total fill across all options: for the all-selected case this centres
		// the window on the best available position so allFilled can trigger;
		// for the all-empty case fills stay near zero regardless of offset.
		if bestGapScore < threshold {
			bestSum := -1.0
			for dy := -searchRadius; dy <= searchRadius; dy++ {
				for dx := -searchRadius; dx <= searchRadius; dx++ {
					sum := 0.0
					for _, b := range q.Options {
						sum += measure(b.X+dx, b.Y+dy)
					}
					if sum > bestSum {
						bestSum = sum
						bestDX, bestDY = dx, dy
					}
				}
			}
		}
	}

	// Measure raw fill ratios at the chosen offset and find the per-question minimum.
	fills := make([]float64, n)
	baseline := 1.0
	for i, bubble := range q.Options {
		fills[i] = measure(bubble.X+bestDX, bubble.Y+bestDY)
		if fills[i] < baseline {
			baseline = fills[i]
		}
	}

	// Subtract the per-question minimum so that printed labels inside empty
	// bubbles register near zero. Guard: if the emptiest bubble already looks
	// filled (baseline > 0.6), all bubbles are probably marked — skip
	// normalization so their raw fills are preserved for absolute detection.
	const maxLetterBaseline = 0.6
	allFilled := baseline > maxLetterBaseline
	if allFilled {
		baseline = 0.0
	}

	adjusted := make([]float64, n)
	for i, f := range fills {
		adjusted[i] = f - baseline
	}

	// Gap-based selection: sort the adjusted fills and find the largest gap
	// between consecutive values. Everything above that gap is "selected".
	// threshold is the minimum gap required for a selection to count — gaps
	// smaller than this are noise from ink variation between letter shapes
	// (e.g. "W" has ~4× the ink of "I" across a 26-option row).
	//
	// Exception: when allFilled is set, there is no baseline ink offset to
	// subtract and no meaningful gap to find, so we fall back to an absolute
	// 0.5 cutoff to detect the (unusual) all-bubbles-marked case.
	sorted := make([]float64, n)
	copy(sorted, adjusted)
	sort.Float64s(sorted)

	maxGap := 0.0
	splitAt := sorted[n-1] // default: gap at the very top — nothing selected
	for i := 1; i < n; i++ {
		if gap := sorted[i] - sorted[i-1]; gap > maxGap {
			maxGap = gap
			splitAt = sorted[i-1]
		}
	}

	var answered []string
	var selectedFills []float64
	var highestFill float64
	var highestUnselected float64

	for i, bubble := range q.Options {
		adj := adjusted[i]
		if adj > highestFill {
			highestFill = adj
		}
		gapSelected := !allFilled && maxGap >= threshold && adj > splitAt
		absSelected := allFilled && adj >= 0.5
		if gapSelected || absSelected {
			answered = append(answered, bubble.Label)
			selectedFills = append(selectedFills, adj)
		} else if adj > highestUnselected {
			highestUnselected = adj
		}
	}

	var confidence float64
	if len(answered) == 0 {
		// How confidently blank: 1.0 when all adjusted fills are identical.
		confidence = 1.0 - highestFill
	} else {
		// Weakest selected minus strongest unselected: measures how cleanly the
		// marked bubbles separate from the unmarked ones.
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
// The ROI is skipped (returns 0) if the window would fall outside the image,
// so an off-edge bubble doesn't cause an OpenCV assertion failure.
func bubbleFillRatio(img gocv.Mat, b Bubble, w, h int, inset float64) float64 {
	x0 := b.X - w/2
	y0 := b.Y - h/2
	x1 := x0 + w
	y1 := y0 + h
	if x0 < 0 || y0 < 0 || x1 > img.Cols() || y1 > img.Rows() {
		return 0.0
	}

	r := w / 2
	if h < w {
		r = h / 2
	}
	innerR := int(float64(r) * inset)

	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	mask := gocv.NewMatWithSize(h, w, gocv.MatTypeCV8U)
	defer mask.Close()
	gocv.Circle(&mask, image.Pt(w/2, h/2), innerR, white, -1)
	circleArea := gocv.CountNonZero(mask)
	if circleArea == 0 {
		return 0.0
	}

	roi := img.Region(image.Rect(x0, y0, x1, y1))
	defer roi.Close()
	masked := gocv.NewMat()
	defer masked.Close()
	gocv.BitwiseAnd(roi, mask, &masked)
	return float64(gocv.CountNonZero(masked)) / float64(circleArea)
}
