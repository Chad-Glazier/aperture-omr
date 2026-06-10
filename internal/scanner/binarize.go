package scanner

import (
	"image"

	"gocv.io/x/gocv"
)

func binarize(src gocv.Mat, dst *gocv.Mat) {
	gray := gocv.NewMat()
	defer gray.Close()

	blur := gocv.NewMat()
	defer blur.Close()

	thresh := gocv.NewMat()
	defer thresh.Close()

	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer kernel.Close()

	gocv.CvtColor(src, &gray, gocv.ColorBGRToGray)
	gocv.GaussianBlur(gray, &blur, image.Pt(5, 5), 0, 0, gocv.BorderDefault)
	gocv.AdaptiveThreshold(blur, &thresh, 255, gocv.AdaptiveThresholdMean, gocv.ThresholdBinaryInv, 11, 2)
	gocv.MorphologyEx(thresh, dst, gocv.MorphClose, kernel)
}
