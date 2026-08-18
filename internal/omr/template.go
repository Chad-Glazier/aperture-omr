package omr

import (
	"image"

	"gocv.io/x/gocv"
)

// Anchors represent specific parts of an image that we search for on a scanned
// sheet. The preprocessing template informs us of where each anchor *should*
// be on the page. Thus, if we are given an imperfect scan, we can locate its
// anchors and compare their expected position/size/orientation to the actual
// values. With that information we can figure out how to correct the scan.
type Anchor struct {
	// The anchor. Must be binary.
	Mat Mat
	// The x-coordinate of the anchor's top-left corner on the target scan.
	X uint 
	// The y-coordinate of the anchor's top-left corner on the target scan.
	Y uint
}

// Closes the matrix used for this anchor.
func (a Anchor) Close() {
	a.Mat.Close()
}

// A template used to inform the preprocessing steps. This template describes
// the "target" scan; i.e., the shape and size of the desired scan. The
// coordinates used for anchors are relative to the target scan.
type PreprocessingTemplate struct {
	Width  uint // The width of the target scan.
	Height uint // The height of the target scan.

	// The anchors to look for on the scans. The i-th page's j-th scan should
	// be element [i][j] of this 2-D slice.
	Anchors [][]Anchor

	// A confidence score (0.0 to 1.0) that defines how closely a suspected
	// anchor on the scan must match the actual anchor to be considered a 
	// match.
	MinAnchorConfidence float64

	// The configuration used for binarizing scans. It's important that the
	// binarization configuration is shared between the anchors and the scans.
	BinarizeConfig BinarizeConfig

	// The size of the minimum dimension used when calibrating the values in
	// this template. With this, we can scale the sizes of certain pixel
	// distances to neatly work with various resolutions.
	CalibrationSize uint
}

// Closes all matrices used by this template.
func (p PreprocessingTemplate) Close() {
	for i := range p.Anchors {
		for j := range p.Anchors[i] {
			p.Anchors[i][j].Close()
		}
	}
}

// Returns an anchor that has its dimensions is scaled by the given factors.
// The coordinates of the anchor will be shifted to preserve its center.
//
// If an error is returned, it will be [ErrOpenCV].
func ScaleAnchor(a Anchor, scaleX, scaleY float64) (Anchor, error) {
	// First, we adjust the position.
	var (
		widthChange  = (scaleX - 1) * float64(a.Mat.Width())
		heightChange = (scaleY - 1) * float64(a.Mat.Height())
	)
	a.X = uint(max(0, float64(a.X) - widthChange / 2))
	a.Y = uint(max(0, float64(a.Y) - heightChange / 2))

	// Next, we scale the matrix.
	var method gocv.InterpolationFlags
	if min(scaleX, scaleY) < 1 {
		method = gocv.InterpolationArea // Better when downsampling.
	} else {
		method = gocv.InterpolationLinear // Better when upsampling.
	}
	resized := gocv.NewMat()
	err := gocv.Resize(
		a.Mat.m, &resized,
		image.Point{},
		scaleX, scaleY,
		method,
	)
	if err != nil {
		return Anchor{}, ErrOpenCV
	}

	a.Mat = newMatFromGoCV(resized)
	return a, nil
}
