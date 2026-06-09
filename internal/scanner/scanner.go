package scanner

import (
	"gocv.io/x/gocv"
)

func Scan(path string) {
	img := gocv.IMRead(path, gocv.IMReadColor)
	defer img.Close()

	window := gocv.NewWindow("Image!")
	defer window.Close()

	window.ResizeWindow(1000, 1414)
	window.IMShow(img)

	for {
		if window.WaitKey(0)&0xFF == 27 {
			break
		}
	}
}
