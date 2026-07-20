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
// fillThreshold: 0.25, bubbleInset: 0.75, flagThreshold: 0.5, searchRadius: 0,
// labelInset: 0.4.
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
	// LabelInset carves this fraction of the bubble radius out of the centre
	// of the alignment search's sampling mask, so a bubble's own printed
	// option label (most templates print one inside each bubble) can't pull
	// the alignment search toward itself. Only the alignment search is
	// restricted this way -- final fill measurement still samples the full
	// disk (bubbleInset), so detection sensitivity for light or
	// centre-concentrated marks is unaffected. Set to 0.0 to disable and
	// sample the full disk for alignment too, e.g. for a template that
	// doesn't print anything inside its bubbles.
	LabelInset *float64 `json:"labelInset"`
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
func Evaluate(imgs []*gocv.Mat, tmpl *Template) (*Result, error) {
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
	labelInset := 0.4
	if tmpl.Config.LabelInset != nil {
		labelInset = *tmpl.Config.LabelInset
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
			selected, confidence := detectAnswers(img, q, threshold, inset, labelInset, searchRadius)
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
	img *gocv.Mat, q Question, threshold, inset, labelInset float64, searchRadius int,
) ([]string, float64) {
	if len(q.Options) == 0 {
		return nil, 0.0
	}

	n := len(q.Options)
	w, h := q.BubbleWidth, q.BubbleHeight
	r := min(w, h) / 2

	innerR := int(float64(r) * inset)
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	// scoreMask: the full disk out to innerR -- used for the actual fill
	// measurement once alignment is settled, so a light or centre-concentrated
	// mark is still fully counted.
	scoreMask := gocv.NewMatWithSize(h, w, gocv.MatTypeCV8U)
	defer scoreMask.Close()
	gocv.Circle(&scoreMask, image.Pt(w/2, h/2), innerR, white, -1)
	scoreArea := gocv.CountNonZero(scoreMask)
	scoreMasked := gocv.NewMat()
	defer scoreMasked.Close()

	// alignMask: same disk, but with a hole punched out of the centre to
	// hide the bubble's own printed letter from the alignment search --
	// otherwise a heavily-inked letter can pull the search toward itself
	// even on an unmarked bubble. Only used to pick (dx, dy); a real mark
	// still darkens the surrounding ring either way.
	labelR := int(float64(r) * labelInset)
	alignMask := gocv.NewMatWithSize(h, w, gocv.MatTypeCV8U)
	defer alignMask.Close()
	gocv.Circle(&alignMask, image.Pt(w/2, h/2), innerR, white, -1)
	if labelR > 0 {
		gocv.Circle(&alignMask, image.Pt(w/2, h/2), labelR, color.RGBA{}, -1)
	}
	alignArea := gocv.CountNonZero(alignMask)
	alignMasked := gocv.NewMat()
	defer alignMasked.Close()

	// guardMask: a disk drawn well past the bubble's own printed ring,
	// fixed regardless of the inset config. Used only by the "is this
	// genuinely well-inked" check in the fallback below.
	//
	// scoreMask/alignMask are deliberately smaller than the ring so a real
	// mark gets measured precisely, with no surrounding print diluting it.
	// But that same smallness is a trap for a fallback that just maximises
	// raw fill: on a blank bubble, drifting the sample toward the ring
	// looks like finding ink, with nothing to stop it short of the search
	// radius. guardMask already contains the whole ring at zero shift, so
	// there's no false reward for drifting, and scoring stays untouched.
	const guardInset = 1.2
	guardInnerR := int(float64(r) * guardInset)
	guardMask := gocv.NewMatWithSize(h, w, gocv.MatTypeCV8U)
	defer guardMask.Close()
	gocv.Circle(&guardMask, image.Pt(w/2, h/2), guardInnerR, white, -1)
	guardArea := gocv.CountNonZero(guardMask)
	guardMasked := gocv.NewMat()
	defer guardMasked.Close()

	sample := func(cx, cy, boxW, boxH int, mask *gocv.Mat, area int, masked *gocv.Mat) (float64, bool) {
		if area == 0 {
			return 0.0, false
		}
		x0, y0 := cx-boxW/2, cy-boxH/2
		x1, y1 := x0+boxW, y0+boxH
		if x0 < 0 || y0 < 0 || x1 > img.Cols() || y1 > img.Rows() {
			return 0.0, false
		}
		roi := img.Region(image.Rect(x0, y0, x1, y1))
		defer roi.Close()
		gocv.BitwiseAnd(roi, *mask, masked)
		return float64(gocv.CountNonZero(*masked)) / float64(area), true
	}

	measure := func(cx, cy int) float64 {
		f, _ := sample(cx, cy, w, h, &scoreMask, scoreArea, &scoreMasked)
		return f
	}
	alignMeasure := func(cx, cy int) float64 {
		f, _ := sample(cx, cy, w, h, &alignMask, alignArea, &alignMasked)
		return f
	}
	guardMeasure := func(cx, cy int) float64 {
		f, _ := sample(cx, cy, w, h, &guardMask, guardArea, &guardMasked)
		return f
	}

	// Per-question alignment search: find the (dx, dy) offset within
	// ±searchRadius that maximises the gap between the sorted option fills
	// -- the offset where a marked bubble stands out most clearly from the
	// rest. Gap, not raw fill, because ring ink inflates every option about
	// equally as the window drifts, which raw fill would reward but gap
	// mostly cancels out.
	//
	// offBubbleFloor: a real bubble always has *some* ink (its ring, its
	// letter), so a fill this low means the sample has drifted off the
	// bubble entirely. Without this floor, that near-zero reading always
	// wins the biggest gap in the sorted list, so the search would just
	// walk to the edge of the radius chasing it instead of a real mark.
	const offBubbleFloor = 0.05

	bestDX, bestDY := 0, 0
	if searchRadius > 0 {
		sortBuf := make([]float64, n)

		gapAt := func(dx, dy int) (float64, bool) {
			for i, b := range q.Options {
				f := alignMeasure(b.X+dx, b.Y+dy)
				if f < offBubbleFloor {
					return 0, false
				}
				sortBuf[i] = f
			}
			sort.Float64s(sortBuf)
			maxGap := 0.0
			for i := 1; i < n; i++ {
				if g := sortBuf[i] - sortBuf[i-1]; g > maxGap {
					maxGap = g
				}
			}
			return maxGap, true
		}

		// Start from the centred offset's own gap, not 0.0 -- otherwise a
		// tie always goes to whichever candidate happens to be checked
		// first (the window's top-left corner), quietly dragging ambiguous
		// questions toward that corner instead of leaving them centred.
		bestGapScore, _ := gapAt(0, 0)
		for dy := -searchRadius; dy <= searchRadius; dy++ {
			for dx := -searchRadius; dx <= searchRadius; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				if gap, ok := gapAt(dx, dy); ok && gap > bestGapScore {
					bestGapScore = gap
					bestDX, bestDY = dx, dy
				}
			}
		}

		// A gap this small means fills are nearly uniform across all
		// options -- either everything's genuinely marked, or nothing is.
		// Two different fallbacks below tell those apart.
		if bestGapScore < threshold {
			// Look for a position with real ink across the board -- this is
			// what recovers a shifted all-filled row, which reads weak at
			// the template's own centre by definition (that's why it needed
			// a search at all). Uses guardMeasure, not measure/alignMeasure
			// -- those are sized for precise scoring, which is exactly what
			// makes a fill-maximising search over them drift toward the
			// ring on a blank bubble instead of stopping at the true mark.
			centeredSum := 0.0
			for _, b := range q.Options {
				centeredSum += guardMeasure(b.X, b.Y)
			}
			sumDX, sumDY := 0, 0
			bestSum := centeredSum
			for dy := -searchRadius; dy <= searchRadius; dy++ {
				for dx := -searchRadius; dx <= searchRadius; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					sum := 0.0
					for _, b := range q.Options {
						sum += guardMeasure(b.X+dx, b.Y+dy)
					}
					if sum > bestSum {
						bestSum = sum
						sumDX, sumDY = dx, dy
					}
				}
			}

			if bestSum/float64(n) > 0.5 {
				// Found a position with real ink across the board -- trust it.
				bestDX, bestDY = sumDX, sumDY
			}
			// Else: genuinely blank, nothing to align to -- leave
			// bestDX/bestDY at the template's own position.
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
	const maxLetterBaseline = 0.75
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
	} else if allFilled && q.Type != "multi" {
		// Every option looks filled on a single-choice question, so there's
		// no unselected bubble left to measure a margin against, and this is
		// itself the most ambiguous possible read.
		confidence = 0.0
	} else {
		minSelected := selectedFills[0]
		for _, f := range selectedFills[1:] {
			if f < minSelected {
				minSelected = f
			}
		}
		if allFilled {
			// A multi-select question where every option is genuinely
			// marked (e.g. a small option count with a "select all" answer)
			// has no unselected bubble to contrast against either, but
			// unlike the single-choice case above it can be a correct
			// answer, not an error. Fall back to how strongly the weakest
			// of the marked bubbles is filled.
			confidence = minSelected
		} else {
			// How cleanly the marked bubbles separate from the unmarked
			// ones, relative to how much ink a mark actually leaves.
			// An absolute margin understates confidence for genuinely
			// clean marks and for questions with more options (more
			// letterform variation among unmarked bubbles narrows the raw
			// gap without the read being any less certain). sorted[n-1] is
			// the largest adjusted fill overall, which always belongs to a
			// selected bubble.
			confidence = (minSelected - highestUnselected) / sorted[n-1]
		}
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
