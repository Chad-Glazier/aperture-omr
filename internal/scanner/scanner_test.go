package scanner

import (
	"image"
	"image/color"
	"testing"

	"gocv.io/x/gocv"
)

var black = gocv.NewScalar(0, 0, 0, 255)
var white = gocv.NewScalar(255, 255, 255, 255)

func TestBinarize(t *testing.T) {
	empty := gocv.NewMat()

	tooSmall := gocv.NewMatWithSizeFromScalar(white, 50, 50, gocv.MatTypeCV8UC3)
	defer tooSmall.Close()

	valid := gocv.NewMatWithSizeFromScalar(white, 500, 500, gocv.MatTypeCV8UC3)
	defer valid.Close()

	tests := []struct {
		name        string
		src         gocv.Mat
		expectError bool
		errContains string
	}{
		{
			name:        "Fails on empty",
			src:         empty,
			expectError: true,
			errContains: "empty image",
		},
		{
			name:        "Fails on too small",
			src:         tooSmall,
			expectError: true,
			errContains: "dimensions too small",
		},
		{
			name:        "Succeeds",
			src:         valid,
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dst := gocv.NewMat()
			defer dst.Close()

			err := binarize(tc.src, &dst)
			assertError(t, err, tc.expectError, tc.errContains)

			if !tc.expectError && dst.Channels() != 1 {
				t.Errorf("expected 1 channel (binary), but got %d", dst.Channels())
			}
		})
	}
}

func TestDeskew(t *testing.T) {
	empty := gocv.NewMat()
	defer empty.Close()

	noLinesBin := gocv.NewMatWithSizeFromScalar(black, 500, 500, gocv.MatTypeCV8UC1)
	defer noLinesBin.Close()

	validSrc := gocv.NewMatWithSizeFromScalar(white, 500, 500, gocv.MatTypeCV8UC3)
	defer validSrc.Close()

	// Create a 500x500 black image and draw a perfectly horizontal white line
	// across the middle to simulate a deskewed structural line on an exam.
	validBin := gocv.NewMatWithSizeFromScalar(black, 500, 500, gocv.MatTypeCV8UC1)
	defer validBin.Close()
	gocv.Line(&validBin, image.Pt(50, 250), image.Pt(450, 250), color.RGBA{255, 255, 255, 255}, 2)

	tests := []struct {
		name        string
		src, bin    gocv.Mat
		expectError bool
		errContains string
	}{
		{
			name:        "Fails on empty",
			src:         empty,
			bin:         empty,
			expectError: true,
			errContains: "empty image",
		},
		{
			name:        "Failds on no lines",
			src:         validSrc,
			bin:         noLinesBin,
			expectError: true,
			errContains: "could not detect skew lines",
		},
		{
			name:        "Succeeds on valid line",
			src:         validSrc,
			bin:         validBin,
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dstCol := gocv.NewMat()
			dstBin := gocv.NewMat()
			defer dstCol.Close()
			defer dstBin.Close()

			err := deskew(tc.src, tc.bin, &dstCol, &dstBin)
			assertError(t, err, tc.expectError, tc.errContains)
		})
	}
}

func TestNormalize(t *testing.T) {
	empty := gocv.NewMat()
	defer empty.Close()

	crop := gocv.NewMatWithSizeFromScalar(white, 412, 987, gocv.MatTypeCV8UC3)
	defer crop.Close()

	tests := []struct {
		name        string
		src         gocv.Mat
		expectError bool
		errContains string
	}{
		{
			name:        "Fails on empty",
			src:         empty,
			expectError: true,
			errContains: "empty image",
		},
		{
			name:        "Succeeds and Resizes",
			src:         crop,
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dst := gocv.NewMat()
			defer dst.Close()

			err := normalize(tc.src, &dst)
			assertError(t, err, tc.expectError, tc.errContains)

			if !tc.expectError {
				if dst.Cols() != Width || dst.Rows() != Height {
					t.Errorf("expected dimensions %dx%d, but got %dx%d", Width, Height, dst.Cols(), dst.Rows())
				}
			}
		})
	}
}

func TestCrop(t *testing.T) {
	empty := gocv.NewMat()

	black := gocv.NewMatWithSizeFromScalar(black, 100, 100, gocv.MatTypeCV8UC1)

	validBin := black.Clone()
	defer validBin.Close()
	gocv.Rectangle(&validBin, image.Rect(25, 25, 75, 75), color.RGBA{255, 255, 255, 255}, -1)

	validColor := gocv.NewMatWithSizeFromScalar(white, 100, 100, gocv.MatTypeCV8UC3)
	defer validColor.Close()

	tests := []struct {
		name        string
		src         gocv.Mat
		bin         gocv.Mat
		expectError bool
		errContains string
	}{
		{
			name:        "Fails on empty source",
			src:         empty,
			bin:         validBin,
			expectError: true,
			errContains: "cannot crop an empty image",
		},
		{
			name:        "Fails on empty bin",
			src:         validColor,
			bin:         empty,
			expectError: true,
			errContains: "cannot crop an empty image",
		},
		{
			name:        "Fails when no contours exist",
			src:         validColor,
			bin:         black, // this img has no contours since it is all black
			expectError: true,
			errContains: "could not detect any contours",
		},
		{
			name:        "Succeeds with valid contours",
			src:         validColor,
			bin:         validBin,
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dst := gocv.NewMatWithSizeFromScalar(white, 100, 100, gocv.MatTypeCV8UC3)
			defer dst.Close()

			err := crop(tc.src, tc.bin, &dst)
			assertError(t, err, tc.expectError, tc.errContains)

			// Asserting our output for successful case
			// Since we placed a white box at (25,25) (75,75) the output should have specific dimensions
			if dst.Empty() {
				t.Errorf("expected destination matrix to be populated, but it was empty")
			}
		})
	}
}
