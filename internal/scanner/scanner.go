package scanner

import (
	"fmt"

	"gocv.io/x/gocv"
)

type context struct {
	err error
}

func (ctx *context) exec(op func() error) {
	if ctx.err != nil {
		return
	}
	ctx.err = op()
}

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

	ctx := &context{}

	ctx.exec(func() error { return binarize(img, &bin) })
	ctx.exec(func() error { return deskew(img, bin, &deskewedCol, &deskewedBin) })
	ctx.exec(func() error { return crop(deskewedCol, deskewedBin, &cropped) })
	ctx.exec(func() error { return normalize(cropped, &normalized) })

	if ctx.err != nil {
		return gocv.NewMat(), fmt.Errorf("preprocessing pipeline failed: %w", ctx.err)
	}

	return normalized, nil
}
