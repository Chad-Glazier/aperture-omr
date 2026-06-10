package scanner

import (
	"fmt"

	"gocv.io/x/gocv"
)

func Scan(path string) (gocv.Mat, error) {
	p, err := Resolve(path)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("resolve path: %w", err)
	}

	img := gocv.IMRead(p, gocv.IMReadColor)
	if img.Empty() {
		img.Close()
		return gocv.NewMat(), fmt.Errorf("failed to read image from %q", p)
	}
	defer img.Close()

	bin := gocv.NewMat()
	defer bin.Close()

	deskewedCol := gocv.NewMat()
	defer deskewedCol.Close()

	deskewedBin := gocv.NewMat()
	defer deskewedBin.Close()

	cropped := gocv.NewMat()
	defer cropped.Close()

	normalized := gocv.NewMat()
	// defer normalized.Close()

	binarize(img, &bin)
	deskew(img, bin, &deskewedCol, &deskewedBin)
	crop(deskewedCol, deskewedBin, &cropped)
	normalize(cropped, &normalized)

	return normalized, nil
}
