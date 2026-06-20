package scanner

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"
	"ubco-team15/omr/internal/utils"

	"gocv.io/x/gocv"
)

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
	Image  gocv.Mat        `json:"-"`
	Path   string          `json:"path"`
	ROI    image.Rectangle `json:"roi"`
	Center image.Point     `json:"center"`
}

func (a *Anchor) Close() {
	a.Image.Close()
}

type Template struct {
	Width   int      `json:"width"`
	Height  int      `json:"height"`
	Anchors []Anchor `json:"anchors"`
	Config  Config   `json:"config"`
	// Dir resolved at load time, anchors paths are relative to this directory
	Dir string `json:"-"`
}

type Config struct {
	BlurSize            int     `json:"blurSize"`
	MorphCloseSize      int     `json:"morphCloseSize"`
	MinAnchorConfidence float32 `json:"minAnchorConfidence"`
}

// Runs an image through the OMR preprocessing pipeline,
// and returns the prepared image.
func Scan(img *gocv.Mat, tmpl *Template) (*ScanData, error) {
	data := &ScanData{
		Color:  img.Clone(),
		Binary: gocv.NewMat(),
	}

	// The context captures any errors that occur during the pipeline
	// and exits early, instead of propagating down the pipeline further.
	ctx := &context{}
	ctx.exec(func() error { return binarize(&data.Color, &data.Binary, &tmpl.Config) })
	ctx.exec(func() error { return warp(data, data, tmpl) })

	if ctx.err != nil {
		return nil, fmt.Errorf("preprocessing pipeline failed: %w", ctx.err)
	}

	return data, nil
}

func binarize(src, dst *gocv.Mat, conf *Config) error {
	if src.Empty() {
		return fmt.Errorf("cannot binarize an empty image")
	}
	if conf.BlurSize%2 == 0 {
		return fmt.Errorf("blurSize must be odd, got %d", conf.BlurSize)
	}

	gray := gocv.NewMat()
	defer gray.Close()

	blur := gocv.NewMat()
	defer blur.Close()

	thresh := gocv.NewMat()
	defer thresh.Close()

	kernelSize := image.Pt(conf.MorphCloseSize, conf.MorphCloseSize)
	kernel := gocv.GetStructuringElement(gocv.MorphRect, kernelSize)
	defer kernel.Close()

	gocv.CvtColor(*src, &gray, gocv.ColorBGRToGray)
	blurSize := image.Pt(conf.BlurSize, conf.BlurSize)
	gocv.GaussianBlur(gray, &blur, blurSize, 0, 0, gocv.BorderDefault)

	// Replaces adaptive thresholding because binary threshold better preserves
	// pencil marks within bubbles
	gocv.Threshold(blur, &thresh, 0, 255, gocv.ThresholdBinaryInv|gocv.ThresholdOtsu)
	gocv.MorphologyEx(thresh, dst, gocv.MorphClose, kernel)

	return nil
}

func warp(src, dst *ScanData, tmpl *Template) error {
	if src.Empty() {
		return fmt.Errorf("cannot warp an empty image")
	}

	anchors := tmpl.Anchors
	if len(anchors) != 3 {
		return fmt.Errorf("warping requires exactly 3 anchors, provided %d", len(anchors))
	}

	for i := range anchors {
		var err error
		anchors[i].Image, err = loadAnchorImage(anchors[i].Path, tmpl)
		if err != nil {
			return fmt.Errorf("anchor %d: load image: %w", i, err)
		}
		defer anchors[i].Image.Close()
	}

	srcPts := make([]image.Point, 3)
	dstPts := make([]image.Point, 3)

	srcSize := image.Pt(src.Binary.Cols(), src.Binary.Rows())
	targetSize := image.Pt(tmpl.Width, tmpl.Height)

	for i := range 3 {
		anchor := anchors[i]
		anchor.ROI = scaleROI(anchor.ROI, srcSize, targetSize)

		pt, err := findAnchorCenter(src.Binary, anchor, tmpl.Config.MinAnchorConfidence)
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

	gocv.WarpAffine(src.Color, &dst.Color, transform, targetSize)
	gocv.WarpAffine(src.Binary, &dst.Binary, transform, targetSize)

	return nil
}

func findAnchorCenter(binary gocv.Mat, anchor Anchor, minConfidence float32) (image.Point, error) {
	roi := binary.Region(anchor.ROI)
	defer roi.Close()

	mask := gocv.NewMat()
	defer mask.Close()

	size := image.Pt(anchor.Image.Cols(), anchor.Image.Rows())

	var bestValue float32
	var bestLocation image.Point
	center := image.Pt(size.X/2, size.Y/2)

	for angle := -5.0; angle <= 5.0; angle += 0.5 {
		matrix := gocv.GetRotationMatrix2D(center, angle, 1.0)
		rotated := gocv.NewMat()

		gocv.WarpAffine(anchor.Image, &rotated, matrix, size)
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

	if bestValue < minConfidence {
		return image.Point{}, fmt.Errorf("confidence %.2f below threshold %.2f", bestValue, minConfidence)
	}

	return image.Pt(
		anchor.ROI.Min.X+bestLocation.X+size.X/2,
		anchor.ROI.Min.Y+bestLocation.Y+size.Y/2,
	), nil
}

// ROI coordinates are defined relative to the target image so that they
// can remain resolution independent. This scales their dimensions relative
// to the source image's dimensions before searching.
func scaleROI(roi image.Rectangle, src, target image.Point) image.Rectangle {
	sx := float64(src.X) / float64(target.X)
	sy := float64(src.Y) / float64(target.Y)
	return image.Rect(
		int(float64(roi.Min.X)*sx), int(float64(roi.Min.Y)*sy),
		int(float64(roi.Max.X)*sx), int(float64(roi.Max.Y)*sy),
	)
}

func loadAnchorImage(path string, tmpl *Template) (gocv.Mat, error) {
	if tmpl.Dir != "" && !filepath.IsAbs(path) && !strings.HasPrefix(path, "~") {
		path = filepath.Join(tmpl.Dir, path)
	}
	path, err := utils.Resolve(path)
	if err != nil {
		return gocv.Mat{}, err
	}
	anchor := gocv.IMRead(path, gocv.IMReadColor)
	if anchor.Empty() {
		return gocv.Mat{}, fmt.Errorf("failed to read image from %q", path)
	}

	if err := binarize(&anchor, &anchor, &tmpl.Config); err != nil {
		anchor.Close()
		return gocv.Mat{}, fmt.Errorf("binarize: %w", err)
	}

	return anchor, nil
}
