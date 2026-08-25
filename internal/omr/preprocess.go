package omr

import (
	"image"
	"image/draw"
	"sync/atomic"

	"gocv.io/x/gocv"
)

type PreprocessResult struct {
	Pages []Mat
	Error error
}

type PageSet interface {
	Pages() []Mat
}

// Preprocesses page sets as they are received from the given channel, sending
// the results through the output channel. Importantly, all received matrices 
// will be closed by this function. Each input slice must have exactly as many
// pages as are expected by the template. It is also assumed that all matrices 
// are roughly the same size.
//
// The parallelism argument determines how many concurrent preprocessing
// operations will be executed at a time. If set to zero, it defaults to 1.
//
// If an error is returned, it will be [ErrCouldNotCalibrate] or [ErrOpenCV].
func PreprocessStream(
	template PreprocessingTemplate,
	parallelism uint,
	pageStream <-chan PageSet,
) (<-chan PreprocessResult, error) {
	if parallelism == 0 {
		parallelism = 1
	}

	//
	// We need to read the first set in order to scale the preprocessing
	// template for the operation.
	//

	first := <-pageStream
	firstPages := first.Pages()
	defer CloseAll(firstPages)

	if len(firstPages) == 0 {
		return nil, ErrCouldNotCalibrate
	}

	template, err := ScalePreprocessingTemplate(
		FitMethodContain,
		template,
		firstPages[0].Width(), firstPages[0].Height(),
	)
	if err != nil {
		return nil, ErrOpenCV
	}
	defer template.Close()

	firstResult := handleSet(template, first)

	//
	// Now, we can start up the threads and return the channel.
	//

	var (
		out     = make(chan PreprocessResult, parallelism)
		threads atomic.Int32
	)
	for range parallelism {
		go func() {
			threads.Add(1)

			for pageSet := range pageStream {
				out <- handleSet(template, pageSet)
			}

			if threads.Add(-1) == 0 {
				close(out)
			}
		}()
	}

	out <- firstResult
	return out, nil
}

func handleSet(template PreprocessingTemplate, set PageSet) PreprocessResult {
	pages := set.Pages()
	defer CloseAll(pages)

	result, err := Preprocess(template, pages)
	return PreprocessResult{
		Pages: result,
		Error: err,
	}
}

// Uses a preprocessing template to produce a set of correctly rotated/
// positioned matrices.
//
// If an error is returned, it will be [ErrIncompatibleTemplate],
// [ErrEmptyMat], [ErrWrongMatType], [ErrCannotMatchAnchor],
// [ErrMatTypeMismatch], [ErrEmptyMat], or [ErrOpenCV].
func Preprocess(
	template PreprocessingTemplate,
	pages []Mat,
) ([]Mat, error) {

	if len(pages) != len(template.Anchors) {
		return nil, ErrIncompatibleTemplate
	}
	for _, page := range pages {
		if page.Empty() {
			return nil, ErrEmptyMat
		}
		if page.Type() != MatTypeGray {
			return nil, ErrWrongMatType
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
				// The match failed on this page. We can try subsequent pages
				// though.
				continue
			}
			if err != nil {
				return nil, err
			}

			// the match was successful
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

	// The configuration used during the search for anchors. Leaving this empty
	// should be fine, though it can be fine-tuned to yield performance
	// improvements.
	AnchorSearchConfig *FindAnchorConfig
}

// Closes all matrices used by this template.
func (p PreprocessingTemplate) Close() {
	for i := range p.Anchors {
		for j := range p.Anchors[i] {
			p.Anchors[i][j].Close()
		}
	}
}

// Returns the number of pages expected by the template.
func (p PreprocessingTemplate) PageCount() int {
	return len(p.Anchors)
}

// Visualizes a preprocessing template as an image.
//
// If an error is returned, it will be [ErrEncoding] or [ErrIndexOutOfBounds].
func (p PreprocessingTemplate) ToImage(pageIdx uint) (image.Image, error) {
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
func (p PreprocessingTemplate) AnchorCenters() [][]image.Point {
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
	src PreprocessingTemplate,
	targetWidth, targetHeight uint,
) (PreprocessingTemplate, error) {

	targetWidth, targetHeight = FittedBounds(
		src.Width, src.Height, targetWidth, targetHeight, method,
	)
	scaled, err := scalePreprocessingTemplateTo(src, targetWidth, targetHeight)
	if err != nil {
		return PreprocessingTemplate{}, ErrOpenCV
	}

	return scaled, nil
}

// Returns a deep copy of the given template with all relevant values scaled to
// fit the given dimensions. Note that the output anchors will not be binary,
// even if the input anchors are. See [Scale].
//
// If an error is returned, it will be [ErrOpenCV].
func scalePreprocessingTemplateTo(
	src PreprocessingTemplate,
	width, height uint,
) (
	PreprocessingTemplate,
	error,
) {
	var (
		scaleX = float64(width) / float64(src.Width)
		scaleY = float64(height) / float64(src.Height)
	)

	scaled := PreprocessingTemplate{
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
				return PreprocessingTemplate{}, ErrOpenCV
			}

			scaled.Anchors[i][j] = scaledAnchor
		}
	}

	return scaled, nil
}
