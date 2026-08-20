package omr

import (
	"image"
	"math"

	"gocv.io/x/gocv"
)

type FindAnchorResult struct {
	Position    image.Point
	Orientation float64
	Confidence  float64
}
 
type FindAnchorConfig struct {
	// The initial orientation of the anchor.
	//
	// Defaults to [DefaultInitialAngle].
	InitialAngle float64

	// The breadth of the angles that should be searched. E.g., if the angle
	// breadth is π/2 and the initial angle is 0, then the search will
	// consider angles in the interval [-π/4, π/4].
	//
	// Defaults to [DefaultAngleSearchBreadth].
	AngleSearchBreadth float64

	// The initial padding on the search area. See [searchRegion].
	//
	// Defaults to [DefaultSearchAreaPadding].
	SearchAreaPadding float64

	// The search will be terminated early if the maximum confidence is met.
	//
	// Defaults to [DefaultMaxConfidence].
	MaxConfidence float64
}

const (
	DefaultInitialAngle       float64 = 0.00
	DefaultAngleSearchBreadth float64 = math.Pi / 12
	DefaultSearchAreaPadding  float64 = 0.10
	DefaultMaxConfidence      float64 = 0.95
)

// Attempts to locate an anchor on the given matrix. The return value describes
// the best match that was found.
//
// If an error is returned, it will be [ErrWrongMatType], [ErrEmptyMat], or 
// [ErrOpenCV].
func FindAnchor(
	mat Mat,
	anchor Anchor,
	minConfidence float64,
	conf *FindAnchorConfig,
) (FindAnchorResult, error) {
	switch { // Handle bad inputs.
	case *mat.t != MatTypeBinary, *anchor.Mat.t != MatTypeBinary:
		return FindAnchorResult{}, ErrWrongMatType
	case mat.Empty(), anchor.Mat.Empty():
		return FindAnchorResult{}, ErrEmptyMat
	}

	//
	// Set the default configuration values.
	//

	var c FindAnchorConfig
	if conf == nil {
		c = FindAnchorConfig{}
	} else {
		c = *conf
	}

	if c.AngleSearchBreadth == 0 {
		c.AngleSearchBreadth = DefaultAngleSearchBreadth
	}
	if c.InitialAngle == 0 {
		c.InitialAngle = DefaultInitialAngle
	}
	if c.SearchAreaPadding == 0 {
		c.SearchAreaPadding = DefaultSearchAreaPadding
	}
	if c.MaxConfidence == 0 {
		c.MaxConfidence = DefaultMaxConfidence
	}

	//
	// We run an iterative refining search here:
	//
	// 1) We start by picking a specific middle point and a breadth.
	//
	// 2) We select a few equidistant points in that search area and
	//    iterate over all of them.
	//
	// 3) We take note of the one with the highest value and restart the search
	//    with it as the new middle point, except that breadth has been shrunk
	//    by some factor (the "refining factor").
	//
	// We could use a more sophisticated convex optimization method, but since
	// we can't really guarantee the convexity of the objective function I
	// doubt that the risk of falling into a local maximum is worth the
	// marginal performance boost.
	//

	const (
		refiningIterations   = 3
		refiningFactor       = 2.0
		anglesPerIteration   = 4
	)

	var (
		bestConfidence float64
		bestLocation   image.Point

		searchArea = searchRegion(mat, anchor, 0.10)
		middle     = initialMiddle
		breadth    = initialBreadth
		angles     = [anglesPerIteration]float64{}
	)
	defer searchArea.Close()

	for range refiningIterations {

		delta := breadth / float64(anglesPerIteration-1)
		lo := middle - breadth/2
		for i := range angles {
			angles[i] = lo + float64(i)*delta
		}
		bestAngle := middle

		for _, angle := range angles {

			rotated, err := RotateAnchor(anchor, angle)
			if err != nil {
				return image.Point{}, ErrOpenCV
			}
			defer rotated.Close()

			location, confidence, err := matchAnchor(searchArea, rotated)
			if err != nil {
				return image.Point{}, ErrOpenCV
			}

			if confidence > bestConfidence {
				bestConfidence = confidence
				bestLocation = location
				bestAngle = angle

				if confidence > earlyBreakConfidence {
					break
				}
			}
		}

		middle = bestAngle
		breadth /= refiningFactor
	}

	return bestLocation, nil
}

// Matches an anchor against a larger region, returning the position where the
// 
//
// If an error is returned, it will be [ErrOpenCV].
func matchAnchor(
	searchArea Mat,
	anchor Anchor,
) (image.Point, float64, error) {
	var (
		mask   = gocv.NewMat()
		result = gocv.NewMat()
	)
	defer mask.Close()
	defer result.Close()

	err := gocv.MatchTemplate(
		searchArea.m,
		anchor.Mat.m,
		&result,
		gocv.TmCcoeffNormed,
		mask,
	)
	if err != nil {
		return image.Point{}, 0, ErrOpenCV
	}

	_, confidence, _, position := gocv.MinMaxLoc(result)
	return position, float64(confidence), nil
}

// Creates a region within a larger matrix. The region will be centered on the
// given anchor and will be at least as large as it. Padding will be added to
// the region as a percentage (0 to 1) of the source matrix's respective
// dimension. The returned region will be restricted so that it never goes
// beyond the source matrix's bounds.
func searchRegion(
	mat Mat,
	anchor Anchor,
	padding float64,
) Mat {
	var (
		w = float64(mat.Width())
		h = float64(mat.Height())

		x0 = int(w * (anchor.Pos.X - padding))
		y0 = int(h * (anchor.Pos.Y - padding))
		x1 = int(w*(anchor.Pos.X+padding) + float64(anchor.Mat.Width()))
		y1 = int(w*(anchor.Pos.Y+padding) + float64(anchor.Mat.Height()))
	)

	x0 = max(x0, 0)
	y0 = max(y0, 0)
	x1 = min(x1, int(mat.Width()))
	y1 = min(y1, int(mat.Height()))

	region := mat.m.Region(image.Rect(x0, y0, x1, y1))
	return newMatFromGoCV(region)
}

// Rotates an anchor by the given angle in radians and returns the result.
//
// If an error is returned, it will be [ErrOpenCV].
func RotateAnchor(src Anchor, angle float64) (Anchor, error) {
	rotated := NewMat()
	err := Rotate(rotated, src.Mat, angle)
	if err != nil {
		rotated.Close()
		return Anchor{}, ErrOpenCV
	}

	src.Mat = rotated
	return src, nil
}
