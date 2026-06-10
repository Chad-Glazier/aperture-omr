package scanner

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

func binarize(src gocv.Mat, dst *gocv.Mat) error {
	if src.Empty() {
		return fmt.Errorf("cannot binarize an empty image")
	} else if src.Cols() <= 100 || src.Rows() <= 100 {
		return fmt.Errorf("image dimensions too small")
	}

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

	// Adaptive threshold block size of 11 (pixel neighborhood) and constant C of 2
	// (subtracted from the mean) effectively isolates bubble/text contrast
	// against varying paper brightness levels or lighting inconsistencies.
	gocv.AdaptiveThreshold(blur, &thresh, 255, gocv.AdaptiveThresholdMean, gocv.ThresholdBinaryInv, 11, 2)
	gocv.MorphologyEx(thresh, dst, gocv.MorphClose, kernel)

	return nil
}
