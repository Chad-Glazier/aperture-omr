package omr

import "gocv.io/x/gocv"

type Closer interface {
	Close() error
}

func CloseAll(c []gocv.Mat) {
	for i := range c {
		c[i].Close()
	}
}

func CloseAll2(c [][]gocv.Mat) {
	for i := range c {
		for j := range c[i] {
			c[i][j].Close()
		}
	}
}
