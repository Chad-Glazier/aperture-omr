package scanner

import (
	"encoding/json"
	"fmt"
	"image"
	"io"

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

type Bubble struct {
	Label  string `json:"label"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Question struct {
	ID      string   `json:"id"`
	Options []Bubble `json:"options"`
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
	Width     int        `json:"width"`
	Height    int        `json:"height"`
	Anchors   []Anchor   `json:"anchors"`
	Config    Config     `json:"config"`
	Questions []Question `json:"questions"`
}

func (t *Template) Close() {
	for i := range t.Anchors {
		t.Anchors[i].Image.Close()
	}
}

type Config struct {
	BlurSize            int     `json:"blurSize"`
	MorphCloseSize      int     `json:"morphCloseSize"`
	MinAnchorConfidence float32 `json:"minAnchorConfidence"`
}

// Runs an image through the OMR preprocessing pipeline,
// and returns the prepared image.
func Scan(r io.Reader, tmpl *Template) (*ScanData, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	img, err := gocv.IMDecode(buf, gocv.IMReadColor)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if img.Empty() {
		img.Close()
		return nil, fmt.Errorf("decoded image is empty")
	}
	if img.Cols() <= 100 || img.Rows() <= 100 {
		img.Close()
		return nil, fmt.Errorf(
			"image dimensions too small: %dx%d", img.Cols(), img.Rows())
	}

	data := &ScanData{
		Color:  img,
		Binary: gocv.NewMat(),
	}

	// The context captures any errors that occur during the pipeline
	// and exits early, instead of propagating down the pipeline further.
	ctx := &context{}
	ctx.exec(func() error { return binarize(&data.Color, &data.Binary, &tmpl.Config) })
	ctx.exec(func() error { return warp(data, data, tmpl) })

	if ctx.err != nil {
		data.Close()
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
		if anchors[i].Image.Empty() {
			return fmt.Errorf("anchor %d: image not loaded, call LoadTemplate first", i)
		}
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

func findAnchorCenter(
	binary gocv.Mat, anchor Anchor, minConfidence float32,
) (image.Point, error) {
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
		return image.Point{}, fmt.Errorf(
			"confidence %.2f below threshold %.2f", bestValue, minConfidence)
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

// LoadTemplate parses a template from r and loads anchor images using open.
// open receives each anchor's path as written in the JSON; the caller is
// responsible for resolving relative paths and expanding ~ if needed.
func LoadTemplate(r io.Reader, open func(string) (io.Reader, error)) (*Template, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	var tmpl Template
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	for i := range tmpl.Anchors {
		ar, err := open(tmpl.Anchors[i].Path)
		if err != nil {
			return nil, fmt.Errorf("anchor %d: open: %w", i, err)
		}
		tmpl.Anchors[i].Image, err = loadAnchorFromReader(ar, &tmpl.Config)
		if err != nil {
			tmpl.Close()
			return nil, fmt.Errorf("anchor %d: %w", i, err)
		}
	}
	return &tmpl, nil
}

func loadAnchorFromReader(r io.Reader, conf *Config) (gocv.Mat, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("read: %w", err)
	}
	img, err := gocv.IMDecode(data, gocv.IMReadColor)
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("decode: %w", err)
	}
	if img.Empty() {
		return gocv.Mat{}, fmt.Errorf("decoded image is empty")
	}
	if err := binarize(&img, &img, conf); err != nil {
		img.Close()
		return gocv.Mat{}, fmt.Errorf("binarize: %w", err)
	}
	return img, nil
}
