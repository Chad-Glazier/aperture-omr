package omr

import (
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

type MarkTemplate struct {
	Aspect       float64
	BubbleRadius float64
	Questions    []Questions
}

type Questions map[string][]Bubble

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
		int(float64(height) * m.Aspect),
		gocv.MatTypeCV8U,
	))

	for _, q := range m.Questions[pageIdx] {
		for _, b := range q {
			var (
				r = float64(width) * m.BubbleRadius
				centerX = int(float64(width) * b.Pos.X + r)
				centerY = int(float64(height) * b.Pos.Y + r)
			)
			gocv.Circle(
				&out.m, 
				image.Pt(centerX, centerY), 
				int(r),
				color.RGBA{ 255, 255, 255, 255 },
				-1,
			)
		}
	}

	return out, nil
}
