package scanner

import (
	"fmt"
	"image"
	"math"

	"gocv.io/x/gocv"
)

func deskew(src, bin gocv.Mat, dstCol, dstBin *gocv.Mat) error {
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
	gocv.HoughLinesPWithParams(bin, &lines, 1, math.Pi/180, 150, 150.0, 10.0)

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
		src.CopyTo(dstCol)
		bin.CopyTo(dstBin)
		return nil
	}

	angle := weightedSum / totalWeight

	if math.Abs(angle) == 0.0 {
		src.CopyTo(dstCol)
		bin.CopyTo(dstBin)
		return nil
	}

	fmt.Printf("Detected angle: %.4f degrees\n", angle)

	center := image.Pt(src.Cols()/2, src.Rows()/2)
	mat := gocv.GetRotationMatrix2D(center, angle, 1.0)
	defer mat.Close()

	size := image.Pt(src.Cols(), src.Rows())
	gocv.WarpAffine(src, dstCol, mat, size)
	gocv.WarpAffine(bin, dstBin, mat, size)

	return nil
}
