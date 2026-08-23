package omr

import (
	"image"
	"image/draw"
)

type PreprocessResult struct {
	pages []Mat
}

func Preprocess(template PreprocessingTemplate, pages []Mat) PreprocessResult {
	FindAnchors()

	return PreprocessResult{}
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

// Returns a copy of the given template, scaled to match the specified
// dimensions.
//
// If an error is returned, it will be [ErrOpenCV].
func ScalePreprocessingTemplate(
	method FitMethod,
	src PreprocessingTemplate,
	targetWidth, targetHeight uint,
) (PreprocessingTemplate, error) {

	var (
		scaleX = float64(targetWidth) / float64(src.Width)
		scaleY = float64(targetHeight) / float64(src.Height)
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
