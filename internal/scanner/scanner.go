package scanner

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"

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
	Image  *gocv.Mat        `json:"-"`
	Path   string          `json:"path"`
	ROI    image.Rectangle `json:"roi"`
	Center image.Point     `json:"center"`
}

func (a *Anchor) Close() {
	a.Image.Close()
}

// ScanPage holds the anchor markers for a single exam page.
type ScanPage struct {
	Anchors []Anchor `json:"anchors"`
}

// Template is the scan template for a multi-page exam. Each entry in Pages
// describes one physical page with its own set of anchor markers. Single-page
// exams use a Pages array with one entry.
type Template struct {
	Width  int        `json:"width"`
	Height int        `json:"height"`
	Pages  []ScanPage `json:"pages"`
	Config Config     `json:"config"`
}

func (t *Template) Close() {
	for pi := range t.Pages {
		for i := range t.Pages[pi].Anchors {
			t.Pages[pi].Anchors[i].Image.Close()
		}
	}
}

type Config struct {
	BlurSize            int     `json:"blurSize"`
	MorphCloseSize      int     `json:"morphCloseSize"`
	MinAnchorConfidence float32 `json:"minAnchorConfidence"`
	// AdaptiveBlockSize is the neighbourhood size for adaptive thresholding
	// (must be odd). Defaults to 91, which works well for 200–300 DPI scans.
	AdaptiveBlockSize int `json:"adaptiveBlockSize"`
	// AdaptiveC is subtracted from the local mean; negative values make the
	// threshold more lenient and are needed to catch light pencil marks.
	AdaptiveC float32 `json:"adaptiveC"`
}

// Scan runs each reader through the OMR preprocessing pipeline using the
// corresponding page's anchors from tmpl. The number of readers must match
// len(tmpl.Pages). Each returned ScanData must be closed by the caller.
func Scan(readers []io.Reader, tmpl *Template) ([]*ScanData, error) {
	n := len(tmpl.Pages)
	if len(readers) != n {
		return nil, fmt.Errorf("template has %d page(s), got %d image(s)", n, len(readers))
	}
	results := make([]*ScanData, n)
	for i, r := range readers {
		data, err := scanPage(r, tmpl, i)
		if err != nil {
			for j := range i {
				results[j].Close()
			}
			return nil, fmt.Errorf("page %d: %w", i, err)
		}
		results[i] = data
	}
	return results, nil
}

// scanPage runs a single reader through the preprocessing pipeline using the
// anchors for page idx.
func scanPage(r io.Reader, tmpl *Template, idx int) (*ScanData, error) {
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
		return nil, fmt.Errorf("image dimensions too small: %dx%d", img.Cols(), img.Rows())
	}

	data := &ScanData{
		Color:  img,
		Binary: gocv.NewMat(),
	}

	ctx := &context{}
	ctx.exec(func() error { return Binarize(&data.Color, &data.Binary, &tmpl.Config) })
	ctx.exec(func() error { return warp(data, data, tmpl.Pages[idx].Anchors, tmpl.Width, tmpl.Height, tmpl.Config) })

	if ctx.err != nil {
		data.Close()
		return nil, fmt.Errorf("preprocessing pipeline failed: %w", ctx.err)
	}

	return data, nil
}

func Binarize(src, dst *gocv.Mat, conf *Config) error {
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

	blockSize := conf.AdaptiveBlockSize
	if blockSize == 0 {
		blockSize = 91
	}
	adaptiveC := conf.AdaptiveC
	if adaptiveC == 0 {
		adaptiveC = float32(-15)
	}
	// Adaptive threshold compares each pixel to its local neighbourhood mean,
	// so light pencil marks (which global Otsu misses) are reliably detected.
	gocv.AdaptiveThreshold(blur, &thresh, 255, gocv.AdaptiveThresholdGaussian, gocv.ThresholdBinaryInv, blockSize, adaptiveC)
	gocv.MorphologyEx(thresh, dst, gocv.MorphClose, kernel)

	return nil
}

func warp(src, dst *ScanData, anchors []Anchor, width, height int, conf Config) error {
	if src.Empty() {
		return fmt.Errorf("cannot warp an empty image")
	}
	if len(anchors) < 3 {
		return fmt.Errorf("warping requires at least 3 anchors, provided %d", len(anchors))
	}

	for i := range anchors {
		if anchors[i].Image.Empty() {
			return fmt.Errorf("anchor %d: image not loaded, call LoadTemplate first", i)
		}
	}

	n := len(anchors)
	srcPts := make([]image.Point, n)
	dstPts := make([]image.Point, n)

	srcSize := image.Pt(src.Binary.Cols(), src.Binary.Rows())
	targetSize := image.Pt(width, height)

	for i := range n {
		anchor := anchors[i]
		anchor.ROI = scaleROI(anchor.ROI, srcSize, targetSize)

		// Scale the anchor template to the size it occupies in the scanned
		// image before running matchTemplate — template matching is not
		// scale-invariant, so a 24px PNG matched against a ~51px printed anchor
		// (at 300 DPI) produces near-zero confidence scores.
		scaled := scaleAnchorTemplate(anchor.Image, srcSize, targetSize)
		defer scaled.Close()
		anchor.Image = scaled

		pt, err := findAnchorCenter(src.Binary, anchor, conf.MinAnchorConfidence)
		if err != nil {
			return fmt.Errorf("anchor %d: %w", i, err)
		}
		log.Printf("anchor %d: found at %v → dst %v", i, pt, anchors[i].Center)

		srcPts[i] = pt
		dstPts[i] = anchors[i].Center
	}

	srcPts2f := make([]gocv.Point2f, n)
	dstPts2f := make([]gocv.Point2f, n)
	for i := range n {
		srcPts2f[i] = gocv.Point2f{X: float32(srcPts[i].X), Y: float32(srcPts[i].Y)}
		dstPts2f[i] = gocv.Point2f{X: float32(dstPts[i].X), Y: float32(dstPts[i].Y)}
	}
	srcVec := gocv.NewPoint2fVectorFromPoints(srcPts2f)
	defer srcVec.Close()
	dstVec := gocv.NewPoint2fVectorFromPoints(dstPts2f)
	defer dstVec.Close()

	warped := ScanData{
		Color:  gocv.NewMat(),
		Binary: gocv.NewMat(),
	}

	// EstimateAffine2D fits a least-squares affine from 3+ point pairs, so
	// additional anchors reduce sensitivity to individual matching errors.
	transform := gocv.EstimateAffine2D(srcVec, dstVec)
	defer transform.Close()
	if transform.Empty() {
		return fmt.Errorf("could not estimate affine transform from anchor points")
	}

	gocv.WarpAffine(src.Color, &warped.Color, transform, targetSize)
	gocv.WarpAffine(src.Binary, &warped.Binary, transform, targetSize)

	// Bilinear interpolation produces intermediate gray values at edges; re-snap
	// to strict 0/255 so CountNonZero only counts genuinely filled pixels.
	gocv.Threshold(warped.Binary, &warped.Binary, 128, 255, gocv.ThresholdBinary)

	// Since we are modifying dst in-place,
	// we must close the old mats before overwriting.
	dst.Close()
	dst.Color = warped.Color
	dst.Binary = warped.Binary

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

		gocv.WarpAffine(*anchor.Image, &rotated, matrix, size)
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

// scaleAnchorTemplate resizes the anchor PNG to the size it occupies in the
// scanned image (src), given the template coordinate space (target). The caller
// must Close the returned Mat.
func scaleAnchorTemplate(tmpl *gocv.Mat, src, target image.Point) *gocv.Mat {
	newW := int(float64(tmpl.Cols())*float64(src.X)/float64(target.X) + 0.5)
	newH := int(float64(tmpl.Rows())*float64(src.Y)/float64(target.Y) + 0.5)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}
	scaled := gocv.NewMat()
	gocv.Resize(*tmpl, &scaled, image.Pt(newW, newH), 0, 0, gocv.InterpolationLinear)
	return &scaled
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
// Both single-page (top-level anchors) and multi-page (pages array) formats
// are supported.
func LoadTemplate(
	r io.Reader, open func(string) (io.ReadCloser, error),
) (*Template, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	var tmpl Template
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	loadAnchors := func(anchors []Anchor, label string) error {
		for i := range anchors {
			ar, err := open(anchors[i].Path)
			if err != nil {
				return fmt.Errorf("%s anchor %d: open: %w", label, i, err)
			}
			anchors[i].Image, err = loadAnchorFromReader(ar, &tmpl.Config)
			ar.Close()
			if err != nil {
				return fmt.Errorf("%s anchor %d: %w", label, i, err)
			}
		}
		return nil
	}

	for pi := range tmpl.Pages {
		if err := loadAnchors(tmpl.Pages[pi].Anchors, fmt.Sprintf("page %d", pi)); err != nil {
			tmpl.Close()
			return nil, err
		}
	}

	return &tmpl, nil
}

func loadAnchorFromReader(r io.Reader, conf *Config) (*gocv.Mat, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	img, err := gocv.IMDecode(data, gocv.IMReadColor)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if img.Empty() {
		return nil, fmt.Errorf("decoded image is empty")
	}

	// Cap adaptive block size to below the anchor's smallest dimension.
	// OpenCV degrades to a near-global threshold when blockSize ≥ min(w,h),
	// producing binary stroke widths that differ from those in the full-scan
	// binary and reduce template-match precision.
	anchorConf := *conf
	minDim := img.Cols()
	if img.Rows() < minDim {
		minDim = img.Rows()
	}
	if anchorConf.AdaptiveBlockSize >= minDim {
		bs := minDim - 1
		if bs%2 == 0 {
			bs--
		}
		if bs < 3 {
			bs = 3
		}
		anchorConf.AdaptiveBlockSize = bs
	}

	if err := Binarize(&img, &img, &anchorConf); err != nil {
		img.Close()
		return nil, fmt.Errorf("binarize: %w", err)
	}
	return &img, nil
}
