package scanner

import (
	"fmt"

	"gocv.io/x/gocv"
)

func Display(img gocv.Mat, title string) {
	window := gocv.NewWindow(title)
	defer window.Close()

	window.ResizeWindow(1000, 1414)
	window.IMShow(img)

	fmt.Printf("Press [Esc] to continue...\n")

	for {
		if window.WaitKey(0)&0xFF == 27 {
			break
		}
	}
}
