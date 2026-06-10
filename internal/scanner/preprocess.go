package scanner

import (
	"image"

	"gocv.io/x/gocv"
)

func preprocess(src gocv.Mat, dst *gocv.Mat) {
	gray := gocv.NewMat()
	defer gray.Close()

	blur := gocv.NewMat()
	defer blur.Close()

	gocv.CvtColor(src, &gray, gocv.ColorBGRToGray)
	gocv.GaussianBlur(gray, &blur, image.Pt(5, 5), 0, 0, gocv.BorderDefault)
	gocv.AdaptiveThreshold(blur, dst, 255, gocv.AdaptiveThresholdMean, gocv.ThresholdBinaryInv, 11, 2)
}
