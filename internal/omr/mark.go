package omr

import (
	"image"
	"image/color"
	"math"

	"gocv.io/x/gocv"
)

type MarkTemplate struct {
	Aspect       float64
	BubbleRadius float64
	Questions    [][]Question
}

type Question struct {
	Id      string
	Bubbles []Bubble
}

type Bubble struct {
	Id  string
	Pos NormalPoint
}

func (m MarkTemplate) PageCount() uint {
	return uint(len(m.Questions))
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

// Returns the index of the page that contains the identified question.
//
// If an error is returned, it will be [ErrQuestionNotDefined].
func (m MarkTemplate) QuestionPage(questionId string) (uint, error) {
	for pageIdx, questions := range m.Questions {
		for _, q := range questions {
			if q.Id == questionId {
				return uint(pageIdx), nil
			} 
		}
	}
	return 0, ErrQuestionNotDefined
}

// Returns the identied question.
//
// If an error is returned, it will be [ErrQuestionNotDefined].
func (m MarkTemplate) Question(questionId string) (Question, error) {
	for _, questions := range m.Questions {
		for _, q := range questions {
			if q.Id == questionId {
				return q, nil
			} 
		}
	}
	return Question{}, ErrQuestionNotDefined
}

// Returns the question's identified bubble.
//
// If an error is not defined, it will be [ErrBubbleNotDefined].
func (q Question) Bubble(bubbleId string) (Bubble, error) {
	for _, b := range q.Bubbles {
		if b.Id == bubbleId {
			return b, nil
		}
	}
	return Bubble{}, ErrBubbleNotDefined
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
// If an error is returned, it will be [ErrQuestionNotDefined] or 
// [ErrBubbleNotDefined].
func (m MarkTemplate) BubbleRegion(
	page Mat, 
	questionId, bubbleId string, 
) (Mat, error) {
	r := m.BubbleRadius * float64(page.Width())
	q, err := m.Question(questionId)
	if err != nil {
		return Mat{}, ErrQuestionNotDefined
	}
	b, err := q.Bubble(bubbleId) 
	if err != nil {
		return Mat{}, ErrBubbleNotDefined
	}

	var (
		x0 = int(b.Pos.X * float64(page.Width()))
		y0 = int(b.Pos.Y * float64(page.Height()))
		x1 = x0 + int(r*2)
		y1 = y0 + int(r*2)
	)

	region := newMatFromGoCV(page.m.Region(image.Rect(x0, y0, x1, y1)))
	return region, nil
}

// Returns the fill ratio (0 to 1) of the specified bubble. In order for this
// to function correctly, the pages should be binarized.
//
// The given mask will be applied to the bubble before counting the fill ratio.
// The mask should just be a circle; it will be assumed that the diameter of 
// the circle is equal to the width/height of the mask. The only reason that 
// this is passed as a parameter is so that it can be reused.
//
// If an error is returned, it will be [ErrWrongMatType], 
// [ErrQuestionNotDefined], [ErrBubbleNotDefined], or [ErrOpenCV].
func fillRatio(
	template MarkTemplate,
	mask Mat,
	pages []Mat,
	questionId,
	bubbleId string,
) (float64, error) {

	pageIdx, err := template.QuestionPage(questionId)
	if err != nil {
		return 0, ErrQuestionNotDefined
	}

	page := pages[pageIdx]
	if page.Type() != MatTypeBinary {
		return 0, ErrWrongMatType
	}

	bubble, err := template.BubbleRegion(page, questionId, bubbleId)
	if err != nil {
		return 0, ErrBubbleNotDefined
	}
	defer bubble.Close()

	masked := NewMat()
	defer masked.Close()

	err = gocv.BitwiseAnd(bubble.m, mask.m, &masked.m)
	if err != nil {
		return 0, ErrOpenCV
	}

	var (
		white = float64(gocv.CountNonZero(masked.m))
		total = math.Pi * math.Pow(float64(mask.Width()) / 2.0, 2)
		black = total - white
	)

	return black/total, nil
}
