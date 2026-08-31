//
// Notes | Updated August 22nd, 2026
//
// Although we calculate a mask for rotated anchors, we cannot use them in the
// current version of the [findAnchor] function. This is because OpenCV does
// not currently support masked template matching with the method we're relying
// on (TM_CCOEFF_NORMED). The lack of a mask means that rotated anchors will
// have added white edges. Since these rotations shouldn't be severe (roughly
// 10 degrees in the worst case) this shouldn't be a big deal. However, those
// edges *will* always be white. This creates  problems if the background color
// is not white. This is listed as a known limitation in the documentation.
//

package omr

import (
	"image"
	"image/color"
	"math"

	"gocv.io/x/gocv"
)

// x- and y-coordinates that are each in the range of 0 to 1.
type NormalPoint struct {
	X, Y float64
}

// Defines a rectangle using normalized points. See [NormalPoint].
type NormalRect struct {
	Min NormalPoint
	Max NormalPoint
}

// Anchors represent specific parts of an image that we search for on a scanned
// sheet. The preprocessing template informs us of where each anchor *should*
// be on the page. Thus, if we are given an imperfect scan, we can locate its
// anchors and compare their expected position/size/orientation to the actual
// values. With that information we can figure out how to correct the scan.
type Anchor struct {
	// The anchor. Must be binary.
	Mat Mat
	// The position of the anchor relative to the template.
	Pos NormalPoint
}

// Closes the matrix used for this anchor.
func (a Anchor) Close() {
	a.Mat.Close()
}

// Returns the anchor's coordinates in terms of the given width and height.
func (a Anchor) Coords(w, h uint) (uint, uint) {
	return uint(a.Pos.X * float64(w)), uint(a.Pos.Y * float64(h))
}

// Finds all anchors on the given page. It is assumed that the given anchors
// have already been scaled to match the page's dimensions.
//
// If an error is returned, it will be [ErrMatTypeMismatch], [ErrEmptyMat],
// [ErrOpenCV], or [ErrCannotMatchAnchor].
func FindAnchors(
	page Mat,
	anchors []Anchor,
	minConfidence float64,
	conf FindAnchorConfig,
) ([]image.Point, error) {

	//
	// Set the default configuration values.
	//

	setDefaultConfig(&conf)

	//
	// Search for the anchors.
	//

	positions := make([]image.Point, len(anchors))
	for i, anchor := range anchors {
		result, err := findAnchor(page, anchor, conf)
		if err != nil {
			return nil, err
		}
		if result.Confidence < minConfidence {
			return nil, ErrCannotMatchAnchor
		}
		positions[i] = result.Position

		// If the confidence was very high, it's likely that the orientation
		// was good. We can start with that orientation for subsequent anchors.
		if result.Confidence >= conf.MaxQuality {
			conf.InitialAngle = result.Orientation
		}
	}

	return positions, nil
}

type FindAnchorResult struct {
	// The position in the source matrix where the anchor's best match was
	// found. Importantly, this point is the *center* of the anchor's position.
	Position image.Point
	// The orientation (i.e., rotation in radians) of the anchor that yielded
	// the best match.
	Orientation float64
	// The quality (0 to 1) of the best match.
	Confidence float64
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

	// The maximum quality of an anchor match. If this value is met or
	// exceeded, the search will be terminated early (which can dramatically
	// improve performance).
	//
	// Defaults to [DefaultMaxQuality].
	MaxQuality float64
}

const (
	DefaultInitialAngle       float64 = 0.00
	DefaultAngleSearchBreadth float64 = 10.0 / 180.0 * math.Pi
	DefaultSearchAreaPadding  float64 = 0.10
	DefaultMaxQuality         float64 = 0.95
)

func setDefaultConfig(c *FindAnchorConfig) {
	if c.AngleSearchBreadth == 0 {
		c.AngleSearchBreadth = DefaultAngleSearchBreadth
	}
	if c.InitialAngle == 0 {
		c.InitialAngle = DefaultInitialAngle
	}
	if c.SearchAreaPadding == 0 {
		c.SearchAreaPadding = DefaultSearchAreaPadding
	}
	if c.MaxQuality == 0 {
		c.MaxQuality = DefaultMaxQuality
	}
}

// Attempts to locate an anchor on the given matrix. The return value describes
// the best match that was found.
//
// If an error is returned, it will be [ErrMatTypeMismatch], [ErrEmptyMat], or
// [ErrOpenCV].
func findAnchor(
	mat Mat,
	anchor Anchor,
	conf FindAnchorConfig,
) (FindAnchorResult, error) {
	switch { // Handle bad inputs.
	case *mat.t != *anchor.Mat.t:
		return FindAnchorResult{}, ErrMatTypeMismatch
	case mat.Empty(), anchor.Mat.Empty():
		return FindAnchorResult{}, ErrEmptyMat
	}

	setDefaultConfig(&conf)

	//
	// Search for the anchor.
	//

	searchArea, offset := searchRegion(mat, anchor, conf.SearchAreaPadding)
	defer searchArea.Close()

	best, err := bisectionSearch(
		conf.InitialAngle-conf.AngleSearchBreadth/2,
		conf.InitialAngle+conf.AngleSearchBreadth/2,
		0.5/180.0*math.Pi,
		conf.MaxQuality,
		func(angle float64) (image.Point, float64, error) {
			rotated, mask, err := RotateAnchor(anchor, angle)
			if err != nil {
				return image.Point{}, 0, ErrOpenCV
			}
			defer rotated.Close()
			defer mask.Close()

			return matchAnchor(searchArea, rotated, NewMat())
		},
	)
	if err != nil {
		return FindAnchorResult{}, err
	}

	return FindAnchorResult{
		Position: image.Pt(
			best.result.X+offset.X,
			best.result.Y+offset.Y,
		),
		Confidence:  best.quality,
		Orientation: best.candidate,
	}, nil
}

// Matches an anchor against a larger region, returning the position where the
// best match was found and the quality of that match (0 to 1). Importantly,
// the returned position represents the *center* of the match.
//
// If an error is returned, it will be [ErrOpenCV], [ErrEmptyMat], or
// [ErrMatTypeMismatch].
func matchAnchor(
	searchArea Mat,
	anchor Anchor,
	anchorMask Mat,
) (image.Point, float64, error) {
	switch {
	case searchArea.Empty() || anchor.Mat.Empty():
		return image.Point{}, 0, ErrEmptyMat
	case *searchArea.t != *anchor.Mat.t:
		return image.Point{}, 0, ErrMatTypeMismatch
	case !anchorMask.Empty():
		if anchorMask.m.Type() != gocv.MatTypeCV8U ||
			anchorMask.Width() != anchor.Mat.Width() ||
			anchorMask.Height() != anchor.Mat.Height() {
			return image.Point{}, 0, ErrInvalidMask
		}
	}

	var (
		result = gocv.NewMat()
	)
	defer result.Close()

	err := gocv.MatchTemplate(
		searchArea.m,
		anchor.Mat.m,
		&result,
		gocv.TmCcoeffNormed,
		anchorMask.m,
	)
	if err != nil {
		return image.Point{}, 0, ErrOpenCV
	}

	_, confidence, _, position := gocv.MinMaxLoc(result)
	position.X += int(anchor.Mat.Width() / 2)
	position.Y += int(anchor.Mat.Height() / 2)
	return position, float64(confidence), nil
}

// Creates a region within a larger matrix. The region will be centered on the
// given anchor and will be at least as large as it. Padding will be added to
// the region as a percentage (0 to 1) of the source matrix's respective
// dimension. The returned region will be restricted so that it never goes
// beyond the source matrix's bounds.
//
// The second return value is the top-left corner of the region in terms of the
// source's coordinates.
func searchRegion(
	mat Mat,
	anchor Anchor,
	padding float64,
) (Mat, image.Point) {
	padding = max(0, padding)
	var (
		w = float64(mat.Width())
		h = float64(mat.Height())

		x0 = int(w * (anchor.Pos.X - padding))
		y0 = int(h * (anchor.Pos.Y - padding))
		x1 = int(w*(anchor.Pos.X+padding) + float64(anchor.Mat.Width()))
		y1 = int(h*(anchor.Pos.Y+padding) + float64(anchor.Mat.Height()))
	)

	x0 = max(x0, 0)
	y0 = max(y0, 0)
	x1 = min(x1, int(mat.Width()))
	y1 = min(y1, int(mat.Height()))

	region := mat.m.Region(image.Rect(x0, y0, x1, y1))
	regionMat := newMatFromGoCV(region)
	*regionMat.t = *mat.t
	return regionMat, image.Pt(x0, y0)
}

// Rotates an anchor by the given angle (in radians) and returns the result.
// Since the rotated anchor may have larger dimensions than the source, a mask
// is also returned that will cut out the added empty space.
//
// If an error is returned, it will be [ErrOpenCV].
func RotateAnchor(src Anchor, angle float64) (Anchor, Mat, error) {
	rotated := NewMat()
	err := Rotate(rotated, src.Mat, angle, color.RGBA{255, 255, 255, 255})
	if err != nil {
		rotated.Close()
		return Anchor{}, Mat{}, ErrOpenCV
	}

	mask := newMatFromGoCV(gocv.NewMatWithSize(
		src.Mat.m.Rows(), src.Mat.m.Cols(),
		gocv.MatTypeCV8U,
	))
	err = Rotate(mask, mask, angle, color.RGBA{255, 255, 255, 255})
	if err != nil {
		mask.Close()
		rotated.Close()
		return Anchor{}, Mat{}, ErrOpenCV
	}
	gocv.Threshold(mask.m, &mask.m, 127, 255, gocv.ThresholdBinaryInv)

	src.Mat = rotated
	return src, mask, nil
}

// Conducts a search on the set of values in between a and b (inclusive).
// The objective function should return three values: the first is some result,
// the second is the evaluated quality of that result, and the last is an
// error. The "quality" value is the one used to guide the search. If an error
// is ever returned from the objective function, the search will terminate
// early and return it. The function may also terminate early if the maximum
// quality is met or exceeded.
//
// If an error is returned, it will be [ErrMaxIterations].
func bisectionSearch[T any](
	a, b float64,
	epsilon float64,
	maxQuality float64,
	objective func(float64) (T, float64, error),
) (bisectionSearchResult[T], error) {

	lo, hi := min(a, b), max(a, b)

	r, q, err := objective((hi - lo) / 2.0)
	if err != nil {
		return bisectionSearchResult[T]{}, err
	}

	best := bisectionSearchResult[T]{
		result:    r,
		quality:   q,
		candidate: (hi - lo) / 2.0,
	}

	if q > maxQuality {
		return best, nil
	}

	for range 1 << 20 {

		for _, candidate := range []float64{lo, hi} {

			r, q, err := objective(candidate)
			if err != nil {
				return bisectionSearchResult[T]{}, err
			}

			result := bisectionSearchResult[T]{
				result:    r,
				quality:   q,
				candidate: candidate,
			}

			if result.quality > best.quality {
				best = result
			}

			if best.quality > maxQuality {
				return best, nil
			}
		}

		if hi-lo < epsilon {
			return best, nil
		}

		breadth := (hi - lo) / 2.0
		lo = best.candidate - breadth/2.0
		hi = best.candidate + breadth/2.0
	}

	return bisectionSearchResult[T]{}, ErrMaxIterations
}

type bisectionSearchResult[T any] struct {
	result    T       // The best result found in the search.
	candidate float64 // The candidate (i.e., input) that yielded this result.
	quality   float64 // The quality of the result.
}
