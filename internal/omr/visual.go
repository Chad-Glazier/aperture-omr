package omr

import (
	"image"
	"image/color"
	"io"
	"math"

	"gocv.io/x/gocv"
)

//
// In this file, we implement a few functions that help visualize the stages of
// the preprocessing/marking. These can be useful to help debug or fine-tune
// configurations.
//

// Draws the given matrices side-by-side, then writes the resulting image to
// the output (encoded as a PNG). The input matrices should be grayscale or
// binary.
//
// If an error is returned, it will be [ErrOpenCV], [ErrWrongMatType], or
// [ErrEncoding].
func VisualizeSideBySide(out io.Writer, mats ...Mat) error {
	if len(mats) == 0 {
		return nil
	}

	for i, m := range mats {
		if !m.m.IsContinuous() {
			mats[i] = Clone(mats[i])
			defer mats[i].Close()
		}
		if m.Type() != MatTypeGray && m.Type() != MatTypeBinary {
			panic("here1")
			return ErrWrongMatType
		}
	}

	var (
		padding   = 50
		x         int
		bounds    = make([]image.Rectangle, len(mats))
		maxHeight = 0
	)
	for i, mat := range mats {
		bounds[i] = image.Rect(
			x+padding,
			padding,
			x+padding+int(mat.Width()),
			padding+int(mat.Height()),
		)
		x += padding + int(mat.Width())
		maxHeight = max(maxHeight, int(mat.Height()))
	}

	dst := newMatFromGoCV(gocv.NewMatWithSize(
		maxHeight+padding*2,
		x+padding,
		gocv.MatTypeCV8U,
	))
	defer dst.Close()

	for i, mat := range mats {
		target := dst.m.Region(bounds[i])
		defer target.Close()

		err := mat.m.CopyTo(&target)
		if err != nil {
			return ErrOpenCV
		}
	}

	_, err := EncodeMatToImage(out, "image/png", dst)
	if err != nil {
		return ErrEncoding
	}

	return nil
}

// Visualizes the preprocessing step, producing an image showing three parts:
// the input, the search regions and identified locations of the images, and
// the preprocessed output.
//
// The image will be written to the given output (encoded as PNG). The input
// matrices should be grayscale or binary.
//
// If an error is returned, it will be [ErrOpenCV], [ErrWrongMatType], or
// [ErrEncoding], [ErrPageNotDefined],
func VisualizePreprocess(
	out io.Writer,
	template PreprocessTemplate,
	pages []Mat,
	pageIdx int,
) error {
	if pageIdx > len(pages) || pageIdx > int(template.PageCount()) {
		return ErrPageNotDefined
	}

	//
	// (C) Preprocessed Page
	//

	preprocessed, err := Preprocess(template, pages)
	var c Mat
	if err != nil {
		c = newMatFromGoCV(gocv.Zeros(
			1000, 1000, gocv.MatTypeCV8U,
		))
		defer c.Close()
		gocv.PutText(
			&c.m,
			"preprocessing failed",
			image.Pt(100, 400),
			gocv.FontHersheyPlain,
			2,
			color.RGBA{255, 255, 255, 255},
			3,
		)
	} else {
		defer CloseAll(preprocessed)
		c = preprocessed[pageIdx]
	}

	//
	// (A) Input Page
	//

	a := Clone(pages[pageIdx])
	defer a.Close()

	//
	// (B) Idenfied Anchors / Search Regions
	//

	anchorLocations, err := FindAnchors(
		pages[pageIdx],
		template.Anchors[pageIdx],
		template.MinAnchorConfidence,
		template.AnchorSearchConfig,
	)
	var markedAnchors Mat
	if err != nil {
		markedAnchors = Clone(pages[pageIdx])
	} else {
		markedAnchors = overlayCrosses(
			pages[pageIdx],
			anchorLocations,
			500, 500, 0.0,
		)
	}
	defer markedAnchors.Close()

	for _, a := range template.Anchors[pageIdx] {
		region, offset := searchRegion(
			markedAnchors,
			a,
			template.AnchorSearchConfig.SearchAreaPadding,
		)
		err := gocv.Rectangle(
			&markedAnchors.m,
			image.Rect(
				offset.X, offset.Y,
				offset.X+int(region.Width()),
				offset.Y+int(region.Height()),
			),
			color.RGBA{},
			5,
		)
		if err != nil {
			return ErrOpenCV
		}
	}
	b := markedAnchors

	return VisualizeSideBySide(out, a, b, c)
}

func overlayCrosses(
	mat Mat,
	centers []image.Point,
	w, h uint,
	orientation float64,
) Mat {
	copy := Clone(mat)

	angle := orientation * -1
	sin, cos := math.Sin(angle), math.Cos(angle)

	rotate := func(pt image.Point, x, y int) image.Point {
		rx := float64(x)*cos - float64(y)*sin
		ry := float64(x)*sin + float64(y)*cos

		return image.Pt(
			pt.X-int(math.Round(rx)),
			pt.Y-int(math.Round(ry)),
		)
	}

	for _, center := range centers {
		gocv.Line(
			&copy.m,
			rotate(center, 0, -int(h/2)),
			rotate(center, 0, int(h/2)),
			color.RGBA{255, 0, 0, 255},
			5,
		)
		gocv.Line(
			&copy.m,
			rotate(center, -int(w/2), 0),
			rotate(center, int(w/2), 0),
			color.RGBA{255, 0, 0, 255},
			5,
		)
	}

	return copy
}

//
func VisualizeMark(
	out io.Writer,
	template MarkTemplate,
	pages []Mat,
	pageIdx uint,
) error {
	if pageIdx > uint(len(pages)) || pageIdx > template.PageCount() {
		return ErrPageNotDefined
	}

	//
	// (A) Input
	//

	a := Clone(pages[pageIdx])
	defer a.Close()

	//
	// (B) Binarized Input
	//

	b := NewMat()
	defer b.Close()

	err := Binarize(b, a, template.Binarization)
	if err != nil {
		return err
	}

	//
	// (C) Marking Mask
	//

	mask, err := template.Mask(uint(pageIdx), int(a.Height()))
	if err != nil {
		return err
	}
	defer mask.Close()

	c := mask

	//
	// (D) Masked Input
	//

	d := NewMat()
	*d.t = MatTypeGray
	defer d.Close()

	err = gocv.BitwiseAnd(mask.m, b.m, &d.m)
	if err != nil {
		return ErrOpenCV
	}

	return VisualizeSideBySide(out, a, b, c, d)
}
