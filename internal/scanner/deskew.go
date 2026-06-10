package scanner

import (
	"fmt"
	"image"
	"math"

	"gocv.io/x/gocv"
)

func deskew(src, preprocessed gocv.Mat, dst *gocv.Mat) {
	lines := gocv.NewMat()
	defer lines.Close()

	// Parameters: rho=1, theta=1 degree (in radians), threshold=100, minLineLength=100, maxLineGap=10
	gocv.HoughLinesPWithParams(preprocessed, &lines, 1, math.Pi/180, 100, 100.0, 10.0)

	var totalAngle float64
	var count int

	for i := 0; i < lines.Rows(); i++ {
		line := lines.GetVeciAt(i, 0)
		x1, y1, x2, y2 := float64(line[0]), float64(line[1]), float64(line[2]), float64(line[3])

		angle := math.Atan2(y2-y1, x2-x1) * 180.0 / math.Pi

		// Filter for lines that are roughly horizontal.
		// Printer feed errors rarely exceed +/- 5 degrees.
		if angle > -10 && angle < 10 {
			totalAngle += angle
			count++
		}
	}

	if count == 0 {
		*dst = src.Clone()
	}

	avgAngle := totalAngle / float64(count)

	if math.Abs(avgAngle) < 0.2 {
		*dst = src.Clone()
	}

	fmt.Printf("Detected angle: %.2f degrees\n", avgAngle)

	center := image.Pt(src.Cols()/2, src.Rows()/2)
	mat := gocv.GetRotationMatrix2D(center, avgAngle, 1.0)
	defer mat.Close()

	gocv.WarpAffine(src, dst, mat, image.Pt(src.Cols(), src.Rows()))
}
