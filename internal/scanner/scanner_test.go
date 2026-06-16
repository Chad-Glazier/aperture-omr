package scanner

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"gocv.io/x/gocv"
)

var black = gocv.NewScalar(0, 0, 0, 255)
var white = gocv.NewScalar(255, 255, 255, 255)

func assertError(t *testing.T, err error, expectError bool, errContains string) {
	t.Helper() // Tells Go test runner to report failures at the caller's line number

	if expectError {
		if err == nil {
			t.Fatalf("expected an error containing %q, but got nil", errContains)
		}
		if !strings.Contains(err.Error(), errContains) {
			t.Errorf("expected error to contain %q, but got %q", errContains, err.Error())
		}
		return
	}

	if err != nil {
		t.Fatalf("did not expect an error, but got: %v", err)
	}
}

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

			err := binarize(&tc.src, &dst)
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

	validCol := gocv.NewMatWithSizeFromScalar(white, 500, 500, gocv.MatTypeCV8UC3)
	defer validCol.Close()

	// Create a 500x500 black image and draw a perfectly horizontal white line
	// across the middle to simulate a deskewed structural line on an exam.
	validBin := gocv.NewMatWithSizeFromScalar(black, 500, 500, gocv.MatTypeCV8UC1)
	defer validBin.Close()
	gocv.Line(&validBin, image.Pt(50, 250), image.Pt(450, 250), color.RGBA{255, 255, 255, 255}, 2)

	tests := []struct {
		name        string
		src         ScanData
		expectError bool
		errContains string
	}{
		{
			name: "Fails on empty",
			src: ScanData{
				Color:  empty,
				Binary: empty,
			},
			expectError: true,
			errContains: "empty image",
		},
		{
			name: "Failds on no lines",
			src: ScanData{
				Color:  validCol,
				Binary: noLinesBin,
			},
			expectError: true,
			errContains: "could not detect skew lines",
		},
		{
			name: "Succeeds on valid line",
			src: ScanData{
				Color:  validCol,
				Binary: validBin,
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dst := ScanData{
				Color:  gocv.NewMat(),
				Binary: gocv.NewMat(),
			}

			err := deskew(&tc.src, &dst)
			assertError(t, err, tc.expectError, tc.errContains)
		})
	}
}

func TestNormalize(t *testing.T) {
	empty := gocv.NewMat()
	defer empty.Close()

	crop := gocv.NewMatWithSizeFromScalar(white, 412, 987, gocv.MatTypeCV8UC3)
	defer crop.Close()

	cropBin := gocv.NewMat()
	gocv.CvtColor(crop, &cropBin, gocv.ColorBGRToGray)
	defer cropBin.Close()

	tests := []struct {
		name        string
		src         ScanData
		expectError bool
		errContains string
	}{
		{
			name: "Fails on empty",
			src: ScanData{
				Color:  empty,
				Binary: empty,
			},
			expectError: true,
			errContains: "empty image",
		},
		{
			name: "Succeeds and Resizes",
			src: ScanData{
				Color:  crop,
				Binary: cropBin,
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dst := ScanData{
				Color:  gocv.NewMat(),
				Binary: gocv.NewMat(),
			}

			err := normalize(&tc.src, &dst)
			assertError(t, err, tc.expectError, tc.errContains)

			if !tc.expectError {
				if dst.Color.Cols() != TargetWidth || dst.Color.Rows() != TargetHeight {
					t.Errorf("expected dimensions %dx%d, but got %dx%d", TargetWidth, TargetHeight, dst.Color.Cols(), dst.Color.Rows())
				}
			}
		})
	}
}

func TestCrop(t *testing.T) {
	empty := gocv.NewMat()
	defer empty.Close()

	black := gocv.NewMatWithSizeFromScalar(black, 100, 100, gocv.MatTypeCV8UC1)
	defer black.Close()

	validBin := black.Clone()
	defer validBin.Close()
	gocv.Rectangle(&validBin, image.Rect(25, 25, 75, 75), color.RGBA{255, 255, 255, 255}, -1)

	validColor := gocv.NewMatWithSizeFromScalar(white, 100, 100, gocv.MatTypeCV8UC3)
	defer validColor.Close()

	tests := []struct {
		name        string
		src         ScanData
		expectError bool
		errContains string
	}{
		{
			name: "Fails on empty source",
			src: ScanData{
				Color:  empty,
				Binary: validBin,
			},
			expectError: true,
			errContains: "cannot crop an empty image",
		},
		{
			name: "Fails on empty bin",
			src: ScanData{
				Color:  validColor,
				Binary: empty,
			},
			expectError: true,
			errContains: "cannot crop an empty image",
		},
		{
			name: "Fails when no contours exist",
			src: ScanData{
				Color:  validColor,
				Binary: black,
			}, // this img has no contours since it is all black
			expectError: true,
			errContains: "could not detect any contours",
		},
		{
			name: "Succeeds with valid contours",
			src: ScanData{
				Color:  validColor,
				Binary: validBin,
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dst := ScanData{
				Color:  gocv.NewMat(),
				Binary: gocv.NewMat(),
			}
			defer dst.Close()

			err := crop(&tc.src, &dst)
			assertError(t, err, tc.expectError, tc.errContains)

			// Asserting our output for successful case
			// Since we placed a white box at (25,25) (75,75) the output should have specific dimensions
			if !tc.expectError && dst.Empty() {
				t.Errorf("expected destination matrix to be populated, but it was empty")
			}
		})
	}
}
