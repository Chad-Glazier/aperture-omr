package marker

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"sort"
	"strings"

	"gocv.io/x/gocv"
)

var debugAlign = os.Getenv("OMR_DEBUG_ALIGN") != ""

// debugf prints alignment diagnostics to stderr when OMR_DEBUG_ALIGN is set,
// and does nothing otherwise.
func debugf(format string, args ...any) {
	if debugAlign {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}

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
	// LabelInset cuts a hole out of the centre of the alignment search's
	// sampling mask, sized as a fraction of the bubble radius. This stops a
	// bubble's own printed option label from pulling the alignment search
	// toward itself. Only alignment uses this mask; the final fill
	// measurement still samples the full disk (bubbleInset), so detection
	// sensitivity is unaffected. Set to 0.0 to sample the full disk for
	// alignment too, for templates that don't print anything inside bubbles.
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

	// scoreMask is the full disk out to innerR. It measures the actual fill
	// once alignment is settled, so a light or centre-concentrated mark
	// still counts in full.
	scoreMask := gocv.NewMatWithSize(h, w, gocv.MatTypeCV8U)
	defer scoreMask.Close()
	gocv.Circle(&scoreMask, image.Pt(w/2, h/2), innerR, white, -1)
	scoreArea := gocv.CountNonZero(scoreMask)
	scoreMasked := gocv.NewMat()
	defer scoreMasked.Close()

	// alignMask is the same disk with a hole punched out of the centre,
	// hiding the bubble's own printed letter from the alignment search.
	// Otherwise a heavily-inked letter can pull the search toward itself on
	// an unmarked bubble. This mask only picks (dx, dy); a real mark still
	// darkens the surrounding ring either way.
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

	// guardMask is a disk drawn well past the bubble's own printed ring,
	// fixed regardless of the inset config. It's only used by the "is this
	// genuinely well-inked" check in the fallback below.
	//
	// scoreMask and alignMask are deliberately smaller than the ring, so a
	// real mark is measured precisely without surrounding print diluting it.
	// But that smallness is a trap for a fallback that just maximises raw
	// fill: on a blank bubble, drifting the sample toward the ring looks
	// like finding ink, with nothing to stop it short of the search radius.
	// guardMask already contains the whole ring at zero shift, so drifting
	// gets no false reward, and scoring itself stays untouched.
	const guardInset = 1.2
	guardInnerR := int(float64(r) * guardInset)
	guardMask := gocv.NewMatWithSize(h, w, gocv.MatTypeCV8U)
	defer guardMask.Close()
	gocv.Circle(&guardMask, image.Pt(w/2, h/2), guardInnerR, white, -1)
	guardArea := gocv.CountNonZero(guardMask)
	guardMasked := gocv.NewMat()
	defer guardMasked.Close()

	sample := func(cx, cy int, mask *gocv.Mat, area int, masked *gocv.Mat) (float64, bool) {
		if area == 0 {
			return 0.0, false
		}
		x0, y0 := cx-w/2, cy-h/2
		x1, y1 := x0+w, y0+h
		if x0 < 0 || y0 < 0 || x1 > img.Cols() || y1 > img.Rows() {
			return 0.0, false
		}
		roi := img.Region(image.Rect(x0, y0, x1, y1))
		defer roi.Close()
		gocv.BitwiseAnd(roi, *mask, masked)
		return float64(gocv.CountNonZero(*masked)) / float64(area), true
	}

	measure := func(cx, cy int) float64 {
		f, _ := sample(cx, cy, &scoreMask, scoreArea, &scoreMasked)
		return f
	}
	alignMeasure := func(cx, cy int) float64 {
		f, _ := sample(cx, cy, &alignMask, alignArea, &alignMasked)
		return f
	}
	guardMeasure := func(cx, cy int) float64 {
		f, _ := sample(cx, cy, &guardMask, guardArea, &guardMasked)
		return f
	}

	// Per-question alignment search: find the (dx, dy) offset within
	// ±searchRadius that maximises the gap between the sorted option fills.
	// That's the offset where a marked bubble stands out most clearly from
	// the rest. We use gap, not raw fill, because ring ink inflates every
	// option about equally as the window drifts. Raw fill would reward that
	// drift; gap mostly cancels it out.
	//
	// offBubbleFloor exists because a real bubble always has some ink, from
	// its ring or its letter. A fill this low means the sample has drifted
	// off the bubble entirely. Without this floor, that near-zero reading
	// always wins the biggest gap, so the search would just walk to the
	// edge of the radius chasing it instead of a real mark.
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

		// Start from the centred offset's own gap, not 0.0. Otherwise ties
		// always go to whichever candidate is checked first (the window's
		// top-left corner), dragging ambiguous questions toward that corner
		// instead of leaving them centred.
		//
		// The centred offset can itself fail the off-bubble floor. This is
		// common on a wide row: alignMask hides each bubble's printed
		// label, so a thin blank glyph's ring ink alone can dip below the
		// floor even when nothing's wrong. haveBest tracks whether we have
		// a real candidate yet. If centred fails, the loop below takes the
		// first valid candidate as its starting point instead of defaulting
		// to 0, which used to let any candidate with gap > 0 win by default.
		bestGapScore, haveBest := gapAt(0, 0)
		debugf("  [%s] centered gapAt(0,0)=%.4f ok=%v\n", q.ID, bestGapScore, haveBest)
		for dy := -searchRadius; dy <= searchRadius; dy++ {
			for dx := -searchRadius; dx <= searchRadius; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				if gap, ok := gapAt(dx, dy); ok && (!haveBest || gap > bestGapScore) {
					bestGapScore = gap
					bestDX, bestDY = dx, dy
					haveBest = true
				}
			}
		}
		debugf("  [%s] chosen bestDX,bestDY=(%d,%d) bestGapScore=%.4f\n", q.ID, bestDX, bestDY, bestGapScore)

		// A gap this small means fills are nearly uniform across all
		// options. Either everything's genuinely marked, or nothing is. Two
		// different checks below tell those apart.
		if bestGapScore < threshold {
			// Look for a position with real ink across the board. This is
			// what recovers a shifted all-filled row, which reads weak at
			// the template's own centre by definition (that's why it needed
			// a search at all). We use guardMeasure here, not measure or
			// alignMeasure: those are sized for precise scoring, which is
			// exactly what makes a fill-maximising search drift toward the
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
				// Found a position with real ink across the board. Trust it.
				bestDX, bestDY = sumDX, sumDY
			} else {
				// Genuinely blank, or a single ordinary mark, not
				// all-filled. Nothing above confidently identifies a real
				// shift, so stay at the template's own position instead of
				// keeping whatever offset the primary gap search landed on.
				// That search only needed a gap above the floor to move off
				// centre, not one above threshold. On a wide mostly-blank
				// row, some noisy off-centre offset commonly clears the
				// floor without being the true alignment. This stops that
				// noise from silently relocating the sample window for the
				// fill measurement below.
				bestDX, bestDY = 0, 0
			}
		}
	}

	// Measure raw fill ratios at the chosen offset.
	fills := make([]float64, n)
	for i, bubble := range q.Options {
		fills[i] = measure(bubble.X+bestDX, bubble.Y+bestDY)
	}
	if debugAlign {
		var raw strings.Builder
		fmt.Fprintf(&raw, "  [%s] raw fills:", q.ID)
		for i, f := range fills {
			fmt.Fprintf(&raw, " %s=%.3f", q.Options[i].Label, f)
		}
		fmt.Fprintln(os.Stderr, raw.String())
	}

	// Baseline is the row's background unmarked ink level. We subtract it
	// below so printed labels inside empty bubbles read close to zero.
	//
	// The simplest estimate, the row's literal minimum fill, is a bad one:
	// it always chases whichever glyph happens to be thinnest, so it gets
	// more extreme as the option count grows. A 26-letter name row always
	// includes the single thinnest glyph in the alphabet, while a 5-option
	// question only ever draws from 5 letters. That alone suppresses
	// confidence on wide rows, regardless of which letter is actually
	// marked.
	//
	// Instead we find the row's own largest gap in raw fill, the same
	// split the selection logic below finds independently, just computed
	// here on raw fills before baseline is subtracted (a uniform shift
	// can't move where the biggest gap falls). Everything at or below that
	// split is the row's unmarked cluster, and its mean is the baseline.
	// This correctly excludes however many bubbles are actually marked,
	// and uses every unmarked sample in the row instead of chasing a
	// single extreme one.
	sortedRaw := make([]float64, n)
	copy(sortedRaw, fills)
	sort.Float64s(sortedRaw)

	maxGapRaw := 0.0
	splitAtRaw := sortedRaw[n-1]
	for i := 1; i < n; i++ {
		if g := sortedRaw[i] - sortedRaw[i-1]; g > maxGapRaw {
			maxGapRaw = g
			splitAtRaw = sortedRaw[i-1]
		}
	}

	baselineSum, baselineCount := 0.0, 0
	for _, f := range fills {
		if f <= splitAtRaw {
			baselineSum += f
			baselineCount++
		}
	}
	baseline := baselineSum / float64(baselineCount)

	// allFilled is true when even the row's unmarked cluster looks
	// substantially filled: every bubble is probably marked, so we skip
	// normalization and fall back to absolute detection below.
	//
	// This looks similar to checking maxGapRaw < threshold (no clean
	// split), but it's not the same thing and isn't redundant with it. A
	// pure gap check can't tell a genuinely all-filled row apart from a
	// flat, low-ink row where the sample just landed off target. Replacing
	// this with a gap check alone broke exactly that case in testing.
	// There's a known, separate edge case this doesn't catch either: a
	// heavy print font can leave the unmarked cluster's mean under 0.75
	// while its top is still too close to a real mark for the gap check to
	// trust, and nothing gets selected at all. Fixing that needs its own
	// signal, not a reuse of this check or of threshold.
	const maxLetterBaseline = 0.75
	allFilled := baseline > maxLetterBaseline
	if allFilled {
		baseline = 0.0
	}

	adjusted := make([]float64, n)
	for i, f := range fills {
		adjusted[i] = f - baseline
	}

	// Gap-based selection: everything above the row's largest gap is
	// "selected". threshold is the minimum gap required for a selection to
	// count. Gaps smaller than this are just noise from ink variation
	// between letter shapes (a "W" carries roughly 4x the ink of an "I"
	// across a 26-option row).
	//
	// We don't need to re-sort and re-scan adjusted here. Subtracting a
	// constant from every value can't change any pairwise gap, so
	// maxGapRaw and splitAtRaw, shifted by baseline, already describe the
	// same split found above. That holds in the allFilled case too, since
	// baseline is forced to 0.0 there and adjusted becomes fills exactly.
	//
	// When allFilled is set there's no meaningful gap to trust, so
	// selection falls back to an absolute 0.5 cutoff instead.
	maxGap := maxGapRaw
	splitAt := splitAtRaw - baseline

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
		minSelected := selectedFills[0]
		for _, f := range selectedFills[1:] {
			if f < minSelected {
				minSelected = f
			}
		}
		if allFilled {
			// No unselected bubble is left to measure a margin against.
			// That limits what confidence can mean here, but it doesn't
			// mean the read itself is uncertain: every option reading as
			// filled is often the clearest possible observation about the
			// page. Evaluate already flags any single-select answer with
			// more than one selection, regardless of this value, so
			// forcing it to 0 here would only be misleading, not useful.
			// Report how strongly the weakest marked bubble is filled,
			// same as a multi-select row where "everything marked" can
			// genuinely be correct.
			confidence = minSelected
		} else {
			// How cleanly the marked bubbles separate from the unmarked
			// ones, relative to how much ink a real mark leaves. A plain
			// margin understates confidence for clean marks on questions
			// with more options, since letterform variation among
			// unmarked bubbles narrows the raw gap without the read being
			// any less certain. sortedRaw[n-1] minus baseline is the
			// largest adjusted fill overall, which always belongs to a
			// selected bubble.
			confidence = (minSelected - highestUnselected) / (sortedRaw[n-1] - baseline)
		}
	}

	confidence = math.Max(confidence, 0.0)
	confidence = math.Min(confidence, 1.0)

	return answered, confidence
}
