package scanner

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

const TargetWidth = 1200
const TargetHeight = 1700

type ScanData struct {
	Color  gocv.Mat
	Binary gocv.Mat
}

func (d *ScanData) Close() {
	d.Color.Close()
	d.Binary.Close()
}

func (d *ScanData) Empty() bool {
	return d.Color.Empty() || d.Binary.Empty()
}

type context struct {
	err error
}

func (ctx *context) exec(op func() error) {
	if ctx.err != nil {
		return
	}
	ctx.err = op()
}

type Anchor struct {
	Template gocv.Mat
	Center   image.Point
	ROI      image.Rectangle
}

func (a *Anchor) Close() {
	a.Template.Close()
}

// Reads an image from the provided file path, runs it through the
// OMR preprocessing pipeline, and returns the prepared image.
func Scan(path string) (*ScanData, error) {
	data := &ScanData{
		Color:  gocv.NewMat(),
		Binary: gocv.NewMat(),
	}

	data.Color = gocv.IMRead(path, gocv.IMReadColor)
	if data.Color.Empty() {
		data.Close()
		return nil, fmt.Errorf("failed to read image from %q", path)
	}

	// The context captures any errors that occur during the pipeline
	// and exits early, instead of propagating down the pipeline further.
	ctx := &context{}
	ctx.exec(func() error { return binarize(&data.Color, &data.Binary) })

	if ctx.err != nil {
		return nil, fmt.Errorf("preprocessing pipeline failed: %w", ctx.err)
	}

	return data, nil
}

func binarize(src *gocv.Mat, dst *gocv.Mat) error {
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

	gocv.CvtColor(*src, &gray, gocv.ColorBGRToGray)
	gocv.GaussianBlur(gray, &blur, image.Pt(5, 5), 0, 0, gocv.BorderDefault)

	// Replaces adaptive thresholding because binary threshold better preserves
	// pencil marks within bubbles
	gocv.Threshold(blur, &thresh, 0, 255, gocv.ThresholdBinaryInv|gocv.ThresholdOtsu)
	gocv.MorphologyEx(thresh, dst, gocv.MorphClose, kernel)

	return nil
}
