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

	closing := gocv.NewMat()
	defer closing.Close()

	gocv.CvtColor(src, &gray, gocv.ColorBGRToGray)
	gocv.GaussianBlur(gray, &blur, image.Pt(5, 5), 0, 0, gocv.BorderDefault)

	// Replaces adaptive thresholding because binary threshold better preserves
	// pencil marks within bubbles
	gocv.Threshold(blur, &thresh, 0, 255, gocv.ThresholdBinaryInv|gocv.ThresholdOtsu)
	gocv.MorphologyEx(thresh, &closing, gocv.MorphClose, kernel)

	// Added edge detection which improves line detection during deskewing
	gocv.Canny(closing, dst, 50, 150)

	return nil
}
