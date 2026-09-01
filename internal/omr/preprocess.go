package omr

import (
	"image"
	"image/draw"
	"math"

	"gocv.io/x/gocv"
)

const TolerableAspectRatioDifference = 0.10

// Uses a preprocessing template to produce a set of correctly rotated/
// positioned matrices. The input matrices will be mutated, but only by
// reordering them or flipping them to match the correct form.
//
// If an error is returned, it will be [ErrIncompatiblePageCount],
// [ErrEmptyMat], [ErrWrongMatType], [ErrCannotMatchAnchor],
// [ErrMatTypeMismatch], [ErrEmptyMat], [ErrOpenCV], or
// [ErrIncompatibleAspect].
func Preprocess(
	template PreprocessTemplate,
	pages []Mat,
) ([]Mat, error) {

	if len(pages) != len(template.Anchors) {
		return nil, ErrIncompatiblePageCount
	}
	for _, page := range pages {
		if page.Empty() {
			return nil, ErrEmptyMat
		}
		if page.Type() != MatTypeGray {
			return nil, ErrWrongMatType
		}
		if !template.AspectRatioIsTolerable(page) {
			return nil, ErrIncompatibleAspect
		}
	}

	//
	// In order to correct the scale/orientation/position of the input pages,
	// we will take the following steps:
	//
	// 1) We locate the anchors on each page. We want this to tolerate out-of-
	//    order pages, so shuffling the order of our page slice may be
	//    necessary.
	//
	// 2) Using the expected positions of each anchor and the actual positions,
	//    we use a robust method to compute the transformation that would
	//    convert the input page to the desired target (as described by the
	//    template).
	//    <https://docs.opencv.org/4.12.0/d9/d0c/group__calib3d.html#ga27865b1d26bac9ce91efaee83e94d4dd>
	//
	// 3) We apply the transformation to the input matrix.
	//    <https://docs.opencv.org/4.12.0/da/d54/group__imgproc__transform.html#ga0203d9ee5fcd28d40dbc4a1ea4451983>
	//

	// (1) Match anchors.
	trueAnchorLocations := make([][]image.Point, len(pages))

outer:
	for i, anchors := range template.Anchors {
		for j := i; j < len(pages); j++ {
			locations, err := FindAnchors(
				pages[j],
				anchors,
				template.MinAnchorConfidence,
				template.AnchorSearchConfig,
			)
			if err == ErrCannotMatchAnchor {
				// Try the inverted page.
				gocv.Flip(pages[j].m, &pages[j].m, -1)
				locations, err = FindAnchors(
					pages[j],
					anchors,
					template.MinAnchorConfidence,
					template.AnchorSearchConfig,
				)

				if err == ErrCannotMatchAnchor {
					// The match failed on this page. We flip the page back and
					// then continue to try subsequent pages.
					gocv.Flip(pages[j].m, &pages[j].m, -1)
					continue
				}
			}
			if err != nil {
				return nil, err
			}

			// The match was successful.
			pages[i], pages[j] = pages[j], pages[i]
			trueAnchorLocations[i] = locations
			continue outer
		}

		// No match was found for the anchors.
		return nil, ErrCannotMatchAnchor
	}
	expectedAnchorLocations := template.AnchorCenters()

	// (2) Estimate the transformations for each page.
	transformations := make([]gocv.Mat, len(pages))
	for i := range pages {
		transformations[i] = gocv.EstimateAffine2D(
			pointsToPoint2f(trueAnchorLocations[i]),
			pointsToPoint2f(expectedAnchorLocations[i]),
		)
		defer transformations[i].Close()

		if transformations[i].Empty() {
			return nil, ErrOpenCV
		}
	}

	// (3) Apply the transformations
	out := make([]Mat, len(pages))
	for i, page := range pages {
		transformed := gocv.NewMat()
		err := gocv.WarpAffine(
			page.m, &transformed,
			transformations[i],
			image.Pt(int(template.Width), int(template.Height)),
		)
		if err != nil {
			CloseAll(out)
			return nil, ErrOpenCV
		}

		out[i] = newMatFromGoCV(transformed)
	}

	return out, nil
}

// A template used to inform the preprocessing steps. This template describes
// the "target" scan; i.e., the shape and size of the desired scan. The
// coordinates used for anchors are relative to the target scan.
type PreprocessTemplate struct {
	Width  uint // The width of the target scan.
	Height uint // The height of the target scan.

	// The anchors to look for on the scans. The i-th page's j-th scan should
	// be element [i][j] of this 2-D slice.
	Anchors [][]Anchor

	// A confidence score (0.0 to 1.0) that defines how closely a suspected
	// anchor on the scan must match the actual anchor to be considered a
	// match.
	MinAnchorConfidence float64

	// The configuration used during the search for anchors. Leaving this empty
	// should be fine, though it can be fine-tuned to yield performance
	// improvements.
	AnchorSearchConfig FindAnchorConfig
}

// Closes all matrices used by this template.
func (p PreprocessTemplate) Close() {
	for i := range p.Anchors {
		for j := range p.Anchors[i] {
			p.Anchors[i][j].Close()
		}
	}
}

// Returns true if and only if the given matrix's aspect ratio is close to the
// template's (the maximum difference is set by
// [TolerableAspectRatioDifference]).
func (p PreprocessTemplate) AspectRatioIsTolerable(m Mat) bool {
	return math.Abs(m.Aspect()-p.Aspect()) <= TolerableAspectRatioDifference
}

// Returns the number of pages expected by the template.
func (p PreprocessTemplate) PageCount() uint {
	return uint(len(p.Anchors))
}

// Returns the aspect ratio (width : height) of the template.
func (p PreprocessTemplate) Aspect() float64 {
	return float64(p.Width) / float64(p.Height)
}

// Visualizes a preprocessing template as an image.
//
// If an error is returned, it will be [ErrEncoding] or [ErrIndexOutOfBounds].
func (p PreprocessTemplate) Image(pageIdx uint) (image.Image, error) {

	r := image.Rect(0, 0, int(p.Width), int(p.Height))
	img := image.NewGray(r)

	for _, a := range p.Anchors[pageIdx] {

		ancImg, err := MatToImage(a.Mat)
		if err != nil {
			return nil, ErrEncoding
		}
		ancRect := image.Rect(
			int(a.Pos.X*float64(p.Width)),
			int(a.Pos.Y*float64(p.Height)),
			int(a.Pos.X*float64(p.Width))+int(a.Mat.Width()),
			int(a.Pos.Y*float64(p.Height))+int(a.Mat.Height()),
		)
		draw.Draw(img, ancRect, ancImg, image.Point{}, draw.Over)
	}

	return img, nil
}

// Returns the anchor locations in terms of the template's coordinates. The
// returned positions mark the centers of the anchors, not their top-left
// corners.
func (p PreprocessTemplate) AnchorCenters() [][]image.Point {
	out := make([][]image.Point, len(p.Anchors))
	for i := range p.Anchors {
		out[i] = make([]image.Point, len(p.Anchors[i]))
		for j, a := range p.Anchors[i] {
			out[i][j] = image.Pt(
				int(a.Pos.X*float64(p.Width)+float64(a.Mat.Width()/2)),
				int(a.Pos.Y*float64(p.Height)+float64(a.Mat.Height()/2)),
			)
		}
	}
	return out
}

// Converts a set of [image.Point] to [gocv.Point2f].
func pointsToPoint2f(points []image.Point) gocv.Point2fVector {
	pts := make([]gocv.Point2f, len(points))
	for i, p := range points {
		pts[i] = gocv.NewPoint2f(float32(p.X), float32(p.Y))
	}
	return gocv.NewPoint2fVectorFromPoints(pts)
}

// Returns a copy of the given template, scaled to match the specified
// dimensions.
//
// If an error is returned, it will be [ErrOpenCV].
func ScalePreprocessingTemplate(
	method FitMethod,
	src PreprocessTemplate,
	targetWidth, targetHeight uint,
) (PreprocessTemplate, error) {

	targetWidth, targetHeight = FittedBounds(
		src.Width, src.Height, targetWidth, targetHeight, method,
	)
	scaled, err := scalePreprocessingTemplateTo(src, targetWidth, targetHeight)
	if err != nil {
		return PreprocessTemplate{}, ErrOpenCV
	}

	return scaled, nil
}

// Returns a deep copy of the given template with all relevant values scaled to
// fit the given dimensions. Note that the output anchors will not be binary,
// even if the input anchors are. See [Scale].
//
// If an error is returned, it will be [ErrOpenCV].
func scalePreprocessingTemplateTo(
	src PreprocessTemplate,
	width, height uint,
) (
	PreprocessTemplate,
	error,
) {
	var (
		scaleX = float64(width) / float64(src.Width)
		scaleY = float64(height) / float64(src.Height)
	)

	scaled := PreprocessTemplate{
		Width:               width,
		Height:              height,
		MinAnchorConfidence: src.MinAnchorConfidence,
		AnchorSearchConfig:  src.AnchorSearchConfig,
	}

	scaled.Anchors = make([][]Anchor, len(src.Anchors))
	for i := range src.Anchors {
		scaled.Anchors[i] = make([]Anchor, len(src.Anchors[i]))
		for j := range src.Anchors[i] {

			scaledAnchor := src.Anchors[i][j]
			scaledAnchor.Mat = NewMat()
			err := Scale(
				scaledAnchor.Mat,
				src.Anchors[i][j].Mat,
				scaleX, scaleY,
			)
			if err != nil {
				scaled.Close()
				return PreprocessTemplate{}, ErrOpenCV
			}

			scaled.Anchors[i][j] = scaledAnchor
		}
	}

	return scaled, nil
}
