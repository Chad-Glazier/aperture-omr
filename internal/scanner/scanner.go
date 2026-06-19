package scanner

import (
	"fmt"
	"image"
	"ubco-team15/omr/internal/utils"

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

const MinAnchorConfidence = 0.5

// Placeholder for the time being
var anchors = []Anchor{
	{
		Template: newTemplate("~/Downloads/new_template_footer.jpg"),
		Center:   image.Pt(596, 1592),
		ROI:      image.Rect(76, 1453, 1124, 1700),
	},
	{
		Template: newTemplate("~/Downloads/new_template_logo.jpg"),
		Center:   image.Pt(938, 161),
		ROI:      image.Rect(759, 0, 1200, 415),
	},
	{
		Template: newTemplate("~/Downloads/new_template_info.jpg"),
		Center:   image.Pt(161, 214),
		ROI:      image.Rect(0, 125, 341, 374),
	},
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
	if data.Color.Cols() <= 100 || data.Color.Rows() <= 100 {
		data.Close()
		return nil, fmt.Errorf("image dimensions too small: %dx%d", data.Color.Cols(), data.Color.Rows())
	}

	// The context captures any errors that occur during the pipeline
	// and exits early, instead of propagating down the pipeline further.
	ctx := &context{}
	ctx.exec(func() error { return binarize(&data.Color, &data.Binary) })
	ctx.exec(func() error { return warp(data, data, anchors) })

	if ctx.err != nil {
		return nil, fmt.Errorf("preprocessing pipeline failed: %w", ctx.err)
	}

	return data, nil
}

func binarize(src, dst *gocv.Mat) error {
	if src.Empty() {
		return fmt.Errorf("cannot binarize an empty image")
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

func warp(src, dst *ScanData, anchors []Anchor) error {
	if src.Empty() {
		return fmt.Errorf("cannot warp an empty image")
	}
	if len(anchors) != 3 {
		return fmt.Errorf("warping requires exactly 3 anchors, provided %d", len(anchors))
	}

	srcPts := make([]image.Point, 3)
	dstPts := make([]image.Point, 3)

	srcSize := image.Pt(src.Binary.Cols(), src.Binary.Rows())
	targetSize := image.Pt(TargetWidth, TargetHeight)

	for i := range 3 {
		anchor := anchors[i]
		anchor.ROI = scaleROI(anchor.ROI, srcSize, targetSize)

		pt, err := findAnchorCenter(src.Binary, anchor)
		if err != nil {
			return fmt.Errorf("anchor %d: %w", i, err)
		}

		srcPts[i] = pt
		dstPts[i] = anchors[i].Center
	}

	srcVec := gocv.NewPointVectorFromPoints(srcPts)
	defer srcVec.Close()
	dstVec := gocv.NewPointVectorFromPoints(dstPts)
	defer dstVec.Close()

	transform := gocv.GetAffineTransform(srcVec, dstVec)
	defer transform.Close()

	size := image.Pt(TargetWidth, TargetHeight)
	gocv.WarpAffine(src.Color, &dst.Color, transform, size)
	gocv.WarpAffine(src.Binary, &dst.Binary, transform, size)

	return nil
}

func findAnchorCenter(binary gocv.Mat, anchor Anchor) (image.Point, error) {
	roi := binary.Region(anchor.ROI)
	defer roi.Close()

	mask := gocv.NewMat()
	defer mask.Close()

	size := image.Pt(anchor.Template.Cols(), anchor.Template.Rows())

	var bestValue float32
	var bestLocation image.Point
	center := image.Pt(size.X/2, size.Y/2)

	for angle := -5.0; angle <= 5.0; angle += 0.5 {
		matrix := gocv.GetRotationMatrix2D(center, angle, 1.0)
		rotated := gocv.NewMat()

		gocv.WarpAffine(anchor.Template, &rotated, matrix, size)
		matrix.Close()

		result := gocv.NewMat()
		gocv.MatchTemplate(roi, rotated, &result, gocv.TmCcoeffNormed, mask)
		_, value, _, location := gocv.MinMaxLoc(result)

		result.Close()
		rotated.Close()

		if value > bestValue {
			bestValue = value
			bestLocation = location
		}
	}

	if bestValue < MinAnchorConfidence {
		return image.Point{}, fmt.Errorf("confidence %.2f below threshold %.2f", bestValue, MinAnchorConfidence)
	}

	return image.Pt(
		anchor.ROI.Min.X+bestLocation.X+size.X/2,
		anchor.ROI.Min.Y+bestLocation.Y+size.Y/2,
	), nil
}

func newTemplate(path string) gocv.Mat {
	path, _ = utils.Resolve(path)
	template := gocv.IMRead(path, gocv.IMReadColor)
	binarize(&template, &template)
	return template
}

func scaleROI(roi image.Rectangle, src, target image.Point) image.Rectangle {
	sx := float64(src.X) / float64(target.X)
	sy := float64(src.Y) / float64(target.Y)
	return image.Rect(
		int(float64(roi.Min.X)*sx), int(float64(roi.Min.Y)*sy),
		int(float64(roi.Max.X)*sx), int(float64(roi.Max.Y)*sy),
	)
}
