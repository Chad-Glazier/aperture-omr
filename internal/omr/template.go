package omr

import (
	"image"
	"image/draw"
)

// x- and y-coordinates that are each in the range of 0 to 1.
type NormalCoordinate struct {
	X, Y float64
}

// Anchors represent specific parts of an image that we search for on a scanned
// sheet. The preprocessing template informs us of where each anchor *should*
// be on the page. Thus, if we are given an imperfect scan, we can locate its
// anchors and compare their expected position/size/orientation to the actual
// values. With that information we can figure out how to correct the scan.
type Anchor struct {
	// The anchor. Must be binary.
	Mat Mat
	// The x-coordinate of the anchor's top-left corner on the target scan.
	Pos NormalCoordinate
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
}

// Closes all matrices used by this template.
func (p PreprocessingTemplate) Close() {
	for i := range p.Anchors {
		for j := range p.Anchors[i] {
			p.Anchors[i][j].Close()
		}
	}
}

// Visualizes a preprocessing template as an image.
//
// If an error is returned, it will be [ErrEncoding] or [ErrIndexOutOfBounds].
func (p *PreprocessingTemplate) ToImage(pageIdx uint) (image.Image, error) {
	r := image.Rect(0, 0, int(p.Width), int(p.Height))
	img := image.NewGray(r)
	
	for _, a := range p.Anchors[pageIdx] {

		ancImg, err := MatToImage(a.Mat)
		if err != nil {
			return nil, ErrEncoding
		}
		ancRect := image.Rect(
			int(a.Pos.X * float64(p.Width)), 
			int(a.Pos.Y * float64(p.Height)), 
			int(a.Pos.X * float64(p.Width)) + int(a.Mat.Width()),
			int(a.Pos.Y * float64(p.Height)) + int(a.Mat.Height()),
		)
		draw.Draw(img, ancRect, ancImg, image.Point{}, draw.Over)
	}

	return img, nil
}

// Describes a method for fitting one rectangle within another. The options are
// named in the same way as the CSS "object-fit" property.
//
// <https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/Properties/object-fit>
type FitMethod uint

const (
	// The template will be scaled so that it completely covers the target 
	// dimensions while preserving its aspect ratio. This may cause the 
	// template to overflow.
	FitMethodCover FitMethod = iota
	// The template will be scaled so that it fits completely inside the target
	// dimensions while preserving its aspect ratio. This may cause the 
	// template to overflow.
	FitMethodContain
	// The template will be scaled to match the target dimensions exactly. This
	// may not preserve the aspect ratio.
	FitMethodFill
)

// Returns a copy of the given template, scaled to match the specified 
// dimensions.
//
// If an error is returned, it will be [ErrOpenCV].
func ScalePreprocessingTemplate(
	method FitMethod,
	src *PreprocessingTemplate, 
	targetWidth, targetHeight uint,
) (*PreprocessingTemplate, error) {

	var (
		scaleX = float64(targetWidth)/float64(src.Width)
		scaleY = float64(targetHeight)/float64(src.Height)
	)
	switch method {
	case FitMethodCover:
		if scaleY > scaleX {
			targetWidth = uint(scaleY * float64(src.Width))
		} else {
			targetHeight = uint(scaleX * float64(src.Height))
		}		
	case FitMethodContain:
		if scaleY < scaleX {
			targetWidth = uint(scaleY * float64(src.Width))
		} else {
			targetHeight = uint(scaleX * float64(src.Height))
		}
	}

	scaled, err := scalePreprocessingTemplateTo(src, targetWidth, targetHeight)
	if err != nil {
		return nil, ErrOpenCV
	}

	return scaled, nil
}

// Returns a deep copy of the given template with all relevant values scaled to
// fit the given dimensions. Note that the output anchors will not be binary, 
// even if the input anchors are. See [Scale].
//
// If an error is returned, it will be [ErrOpenCV].
func scalePreprocessingTemplateTo(
	src *PreprocessingTemplate,
	width, height uint,	
) (
	*PreprocessingTemplate, 
	error,
) {
	var (
		scaleX = float64(width)/float64(src.Width)
		scaleY = float64(height)/float64(src.Height)
	)

	scaled := PreprocessingTemplate{
		Width:  width,
		Height: height,
		MinAnchorConfidence: src.MinAnchorConfidence,
	}
	scaled.BinarizeConfig = scaleBinarizationConfig(
		src.BinarizeConfig, 
		scaleX, scaleY,
	)
	
	scaled.Anchors = make([][]Anchor, len(src.Anchors))
	for i := range src.Anchors {
		scaled.Anchors[i] = make([]Anchor, len(src.Anchors[i]))
		for j := range src.Anchors[i] {

			scaledAnchor := src.Anchors[i][j]
			m, err := Scale(src.Anchors[i][j].Mat, scaleX, scaleY)
			if err != nil {
				scaled.Close()
				return nil, ErrOpenCV
			}
			scaledAnchor.Mat = m

			scaled.Anchors[i][j] = scaledAnchor
		}
	}

	return &scaled, nil
}
