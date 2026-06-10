package scanner

import (
	"image"

	"gocv.io/x/gocv"
)

func crop(src, bin gocv.Mat, dst *gocv.Mat) {
	contours := gocv.FindContours(bin, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	defer contours.Close()

	x, y := src.Cols(), src.Rows()
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

	padding := 20
	minX = max(0, minX-padding)
	minY = max(0, minY-padding)
	maxX = min(x, maxX+padding)
	maxY = min(y, maxY+padding)

	rect := image.Rect(minX, minY, maxX, maxY)
	*dst = src.Region(rect)
}
