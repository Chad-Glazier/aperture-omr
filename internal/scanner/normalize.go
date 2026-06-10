package scanner

import (
	"image"

	"gocv.io/x/gocv"
)

const Width = 1200
const Height = 1700

func normalize(src gocv.Mat, dst *gocv.Mat) {
	gocv.Resize(src, dst, image.Pt(Width, Height), 0, 0, gocv.InterpolationArea)
}
