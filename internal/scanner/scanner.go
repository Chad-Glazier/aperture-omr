package scanner

import (
	"fmt"
	"image"
	"math"

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
		return nil, fmt.Errorf("failed to read image from %q", p)
	}

	// The context captures any errors that occur during the pipeline
	// and exits early, instead of propagating down the pipeline further.
	ctx := &context{}
	ctx.exec(func() error { return binarize(&data.Color, &data.Binary) })
	ctx.exec(func() error { return deskew(data, data) })
	ctx.exec(func() error { return crop(data, data) })
	ctx.exec(func() error { return normalize(data, data) })

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

func deskew(src *ScanData, dst *ScanData) error {
	if src.Empty() {
		return fmt.Errorf("cannot deskew an empty image")
	}

	lines := gocv.NewMat()
	defer lines.Close()

	// Parameters:
	// 		rho=1,
	// 		theta=1 (in radians),
	// 		threshold=150,
	// 		minLineLength=150,
	// 		maxLineGap=10
	gocv.HoughLinesPWithParams(src.Binary, &lines, 1, math.Pi/180, 150, 150.0, 10.0)

	if lines.Rows() == 0 {
		return fmt.Errorf("could not detect skew lines")
	}

	var totalWeight float64
	var weightedSum float64

	for i := 0; i < lines.Rows(); i++ {
		line := lines.GetVeciAt(i, 0)
		dx := float64(line[2] - line[0])
		dy := float64(line[3] - line[1])

		angle := math.Atan2(dy, dx) * 180.0 / math.Pi
		length := math.Sqrt(dx*dx + dy*dy)

		abs := math.Abs(angle)
		if abs < 0.5 || abs > 10.0 {
			continue
		}

		weightedSum += (angle * length)
		totalWeight += length
	}

	if totalWeight == 0 {
		src.Color.CopyTo(&dst.Color)
		src.Binary.CopyTo(&dst.Binary)
		return nil
	}

	angle := weightedSum / totalWeight

	if math.Abs(angle) == 0.0 {
		src.Color.CopyTo(&dst.Color)
		src.Binary.CopyTo(&dst.Binary)
		return nil
	}

	fmt.Printf("Detected angle: %.4f degrees\n", angle)

	center := image.Pt(src.Color.Cols()/2, src.Color.Rows()/2)
	mat := gocv.GetRotationMatrix2D(center, angle, 1.0)
	defer mat.Close()

	size := image.Pt(src.Color.Cols(), src.Color.Rows())
	gocv.WarpAffine(src.Color, &dst.Color, mat, size)
	gocv.WarpAffine(src.Binary, &dst.Binary, mat, size)

	return nil
}

func crop(src *ScanData, dst *ScanData) error {
	if src.Empty() {
		return fmt.Errorf("cannot crop an empty image")
	}

	contours := gocv.FindContours(src.Binary, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	defer contours.Close()

	if contours.Size() == 0 {
		return fmt.Errorf("could not detect any contours")
	}

	x, y := src.Color.Cols(), src.Color.Rows()
	minX, minY := x, y
	maxX, maxY := 0, 0

	for i := 0; i < contours.Size(); i++ {
		contour := contours.At(i)
		area := gocv.ContourArea(contour)

		// Ignore any random scanner noise or dust
		if area < 50 {
			continue
		}

		rect := gocv.BoundingRect(contour)

		if rect.Min.X < minX {
			minX = rect.Min.X
		}
		if rect.Min.Y < minY {
			minY = rect.Min.Y
		}
		if rect.Max.X > maxX {
			maxX = rect.Max.X
		}
		if rect.Max.Y > maxY {
			maxY = rect.Max.Y
		}
	}

	// Applies 20px of padding to the bounding box to ensure
	// nothing near the boundaries get's clipped off.
	padding := 20
	minX = max(0, minX-padding)
	minY = max(0, minY-padding)
	maxX = min(x, maxX+padding)
	maxY = min(y, maxY+padding)

	rect := image.Rect(minX, minY, maxX, maxY)
	dst.Color = src.Color.Region(rect)
	dst.Binary = src.Binary.Region(rect)

	return nil
}

func normalize(src *ScanData, dst *ScanData) error {
	if src.Empty() {
		return fmt.Errorf("cannot normalize an empty image")
	}
	gocv.Resize(src.Color, &dst.Color, image.Pt(TargetWidth, TargetHeight), 0, 0, gocv.InterpolationArea)
	gocv.Resize(src.Binary, &dst.Binary, image.Pt(TargetWidth, TargetHeight), 0, 0, gocv.InterpolationArea)

	return nil
}
