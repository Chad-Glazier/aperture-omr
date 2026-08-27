package omr

import (
	"image"
	"image/color"
	"math"

	"gocv.io/x/gocv"
)

type MarkResult struct {
	Pages []PageResult
}

type PageResult struct {
	Questions []QuestionResult
}

type QuestionResult struct {
	SelectedBubbles []string
}

// Marks the given pages. The input pages will be closed by this function.
func Mark(template MarkTemplate, pages []Mat) (MarkResult, error) {
	defer CloseAll(pages)

	for _, page := range pages {
		err := Binarize(page, page, &template.BinarizeConfig)
		if err != nil {
			return MarkResult{}, err
		}
	}

	fillRatios, err := template.RawFillRatios(pages)
	if err != nil {
		return MarkResult{}, err
	}
	NormalizeFillRatios(fillRatios)

	out := MarkResult{}
	out.Pages = make([]PageResult, len(fillRatios))
	for i := range uint(len(fillRatios)) {

		questions := make([]QuestionResult, len(fillRatios[i]))
		for j := range uint(len(fillRatios[i])) {
			
			selected := make([]string, 0)
			for k := range uint(len(fillRatios[i][j])) {

				if fillRatios[i][j][k] > template.MinimumConfidence {
					selected = append(selected, template.Bubble(i, j, k).Id)
				}
			}

			questions[j].SelectedBubbles = selected
		}

		out.Pages[i].Questions = questions
	}

	return out, nil
}

type MarkTemplate struct {
	Aspect            float64
	BubbleRadius      float64
	Questions         [][]Question
	MinimumConfidence float64
	BinarizeConfig    BinarizeConfig
}

type Question struct {
	Id      string
	Bubbles []Bubble
}

type Bubble struct {
	Id  string
	Pos NormalPoint
}

// Returns true if and only if the given matrix's aspect ratio is close to the
// template's (the maximum difference is set by
// [TolerableAspectRatioDifference]).
func (m MarkTemplate) AspectRatioIsTolerable(mat Mat) bool {
	return math.Abs(m.Aspect-mat.Aspect()) <= TolerableAspectRatioDifference
}

func (m MarkTemplate) PageCount() uint {
	return uint(len(m.Questions))
}

// This function does not do any bounds checking; passing invalid indices will
// cause a panic.
func (m MarkTemplate) Page(idx uint) []Question {
	return m.Questions[idx]
}

func (m MarkTemplate) QuestionCount(pageIdx uint) uint {
	return uint(len(m.Questions[pageIdx]))
}

// This function does not do any bounds checking; passing invalid indices will
// cause a panic.
func (m MarkTemplate) Question(pageIdx, questionIdx uint) Question {
	return m.Questions[pageIdx][questionIdx]
}

func (m MarkTemplate) BubbleCount(pageIdx, questionIdx uint) uint {
	return uint(len(m.Questions[pageIdx][questionIdx].Bubbles))
}

// This function does not do any bounds checking; passing invalid indices will
// cause a panic.
func (m MarkTemplate) Bubble(pageIdx, questionIdx, bubbleIdx uint) Bubble {
	return m.Questions[pageIdx][questionIdx].Bubbles[bubbleIdx]
}

// Computes the radius of a bubble in terms of pixels by scaling with the given
// page width.
func (m MarkTemplate) BubblePixelRadius(pageWidth uint) uint {
	return uint(m.BubbleRadius * float64(pageWidth))
}

// Creates a binary mask that represents the entire marking template.
//
// If an error is returned, it will be [ErrEncoding] or [ErrIndexOutOfBounds].
func (m MarkTemplate) Mask(pageIdx uint, height int) (Mat, error) {
	if m.PageCount() <= pageIdx {
		return Mat{}, ErrIndexOutOfBounds
	}

	width := int(m.Aspect * float64(height))
	out := newMatFromGoCV(gocv.NewMatWithSize(
		height,
		int(float64(height)*m.Aspect),
		gocv.MatTypeCV8U,
	))

	for _, q := range m.Questions[pageIdx] {
		for _, b := range q.Bubbles {
			var (
				r       = float64(width) * m.BubbleRadius
				centerX = int(float64(width)*b.Pos.X + r)
				centerY = int(float64(height)*b.Pos.Y + r)
			)
			gocv.Circle(
				&out.m,
				image.Pt(centerX, centerY),
				int(r),
				color.RGBA{255, 255, 255, 255},
				int(gocv.Filled),
			)
		}
	}

	return out, nil
}

// Creates a (filled) circle mask.
func circleMask(radius uint) Mat {
	mask := newMatFromGoCV(gocv.NewMatWithSize(
		int(radius*2),
		int(radius*2),
		gocv.MatTypeCV8U,
	))
	gocv.Circle(
		&mask.m,
		image.Pt(int(radius), int(radius)),
		int(radius),
		color.RGBA{255, 255, 255, 255},
		-1,
	)

	return mask
}

// Returns a matrix that only includes the specified bubble.
//
// This function does not do any bounds checking; passing invalid indices will
// cause a panic.
func (m MarkTemplate) BubbleRegion(
	page Mat,
	pageIdx, questionIdx, bubbleIdx uint,
) Mat {
	r := m.BubblePixelRadius(page.Width())
	b := m.Bubble(pageIdx, questionIdx, bubbleIdx)

	var (
		x0 = int(b.Pos.X * float64(page.Width()))
		y0 = int(b.Pos.Y * float64(page.Height()))
		x1 = x0 + int(r*2)
		y1 = y0 + int(r*2)
	)

	region := newMatFromGoCV(page.m.Region(image.Rect(x0, y0, x1, y1)))
	return region
}

// Normalizes the given fill ratios in place. After normalization, the minimum
// fill ratio will be 0 and the maximum will be 1.
func NormalizeFillRatios(values [][][]float64) {
	var minVal, maxVal float64
	for i := range values {
		for j := range values[i] {
			for _, val := range values[i][j] {
				minVal = min(minVal, val)
				maxVal = max(maxVal, val)
			}
		}
	}

	valueRange := maxVal - minVal
	for i := range values {
		for j := range values[i] {
			for k, val := range values[i][j] {
				values[i][j][k] = (val - minVal) / valueRange
			}
		}
	}
}

// Calculates the fill ratios of each bubble. The i-th page's j-th question's
// k-th bubble will have its fill ratio stored at the element indexed [i][j][k]
// in the returned slice. Note that the input pages will be mutated.
//
// The input matrices should all be binary and have a similar aspect ratio to
// the template.
//
// If an error is returned, it will be [ErrIncompatiblePageCount],
// [ErrIncompatibleAspect], [ErrWrongMatType], or [ErrOpenCV].
func (m MarkTemplate) RawFillRatios(pages []Mat) ([][][]float64, error) {
	if len(pages) != int(m.PageCount()) {
		return nil, ErrIncompatiblePageCount
	}

	mask := circleMask(m.BubblePixelRadius(pages[0].Width()))

	out := make([][][]float64, m.PageCount())
	for i := range uint(len(pages)) {
		if !m.AspectRatioIsTolerable(pages[i]) {
			return nil, ErrIncompatibleAspect
		}
		if pages[i].Type() != MatTypeBinary {
			return nil, ErrWrongMatType
		}

		out[i] = make([][]float64, m.QuestionCount(i))
		for j := range uint(len(m.Questions[i])) {

			out[i][j] = make([]float64, m.BubbleCount(i, j))
			for k := range uint(len(m.Questions[i][j].Bubbles)) {

				fr, err := fillRatio(m, mask, pages, i, j, k)
				if err != nil {
					return nil, err
				}
				out[i][j][k] = fr
			}
		}
	}

	return out, nil
}

// Returns the fill ratio (0 to 1) of the specified bubble. In order for this
// to function correctly, the pages should be binarized. Note that the input
// pages will be mutated.
//
// The given mask will be applied to the bubble before counting the fill ratio.
// The mask should just be a circle; it will be assumed that the diameter of
// the circle is equal to the width/height of the mask. The only reason that
// this is passed as a parameter is so that it can be reused.
//
// If an error is returned, it will be [ErrOpenCV]. This function does not do
// any bounds checking; passing invalid indices will cause a panic.
func fillRatio(
	template MarkTemplate,
	mask Mat,
	pages []Mat,
	pageIdx,
	questionIdx,
	bubbleIdx uint,
) (float64, error) {

	bubble := template.BubbleRegion(
		pages[pageIdx],
		pageIdx,
		questionIdx,
		bubbleIdx,
	)
	defer bubble.Close()

	err := gocv.BitwiseAnd(bubble.m, mask.m, &bubble.m)
	if err != nil {
		return 0, ErrOpenCV
	}

	var (
		white = float64(gocv.CountNonZero(bubble.m))
		total = math.Pi * math.Pow(float64(mask.Width())/2.0, 2)
		black = total - white
	)

	return black / total, nil
}
