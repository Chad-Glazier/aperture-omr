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

	preprocessed := gocv.NewMat()
	defer preprocessed.Close()

	preprocess(img, &preprocessed)

	deskewed, err := deskew(img, preprocessed)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("deskew image: %w", err)
	}

	return deskewed, nil
}
