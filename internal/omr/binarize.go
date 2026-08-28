package omr

import (
	"image"

	"gocv.io/x/gocv"
)

type BinarizeConfig struct {
	// The Gaussian kernel size, used for blurring.  Represented as a
	// proportion of the matrix's width.
	//
	// Defaults to [DefaultBlurSize].
	//
	// <https://docs.opencv.org/4.12.0/d4/d86/group__imgproc__filter.html#gae8bdcd9154ed5ca3cbc1766d960f45c1>
	BlurSize float64

	// The kernel size for the morphological closing operation. Represented as
	// a proportion of the matrix's width.
	//
	// Defaults to [DefaultMorphCloseSize].
	//
	// <https://docs.opencv.org/4.12.0/d4/d86/group__imgproc__filter.html#ga67493776e3ad1a3df63883829375201f>
	MorphCloseSize float64

	// The size of a pixel neighborhood that is used to calculate a threshold
	// value for each pixel. Represented as a proportion of the matrix's width.
	//
	// Defaults to [DefaultBlockSize].
	//
	// <https://docs.opencv.org/4.12.0/d7/d1b/group__imgproc__misc.html#ga72b913f352e4a1b1b397736707afcde3?>
	BlockSize float64

	// A constant that is subtracted from each pixel's threshold value.
	//
	// Defaults to [DefaultAdaptiveC].
	//
	// <https://docs.opencv.org/4.12.0/d7/d1b/group__imgproc__misc.html#ga72b913f352e4a1b1b397736707afcde3>
	AdaptiveC float64
}

const (
	DefaultBlurSize       float64 = 0.0015
	DefaultMorphCloseSize float64 = 0.0015
	DefaultBlockSize      float64 = 0.0255
	DefaultAdaptiveC      float64 = 10.000
)

// Binarizes a grayscale matrix.
//
// If an error is returned, it will be [ErrWrongMatType], [ErrEmptyMat], or
// [ErrOpenCV]. If it's [ErrOpenCV], then the destination matrix may have
// corrupted values from a partial operation and should be closed.
func Binarize(dst Mat, src Mat, conf *BinarizeConfig) error {
	switch { // Handle bad inputs.
	case src.Empty():
		return ErrEmptyMat
	case src.Type() != MatTypeGray:
		return ErrWrongMatType
	}

	//
	// Set the default configuration values.
	//

	var c BinarizeConfig
	if conf == nil {
		c = BinarizeConfig{}
	} else {
		c = *conf
	}

	if c.BlurSize == 0 {
		c.BlurSize = DefaultBlurSize
	}
	if c.MorphCloseSize == 0 {
		c.MorphCloseSize = DefaultMorphCloseSize
	}
	if c.BlockSize == 0 {
		c.BlockSize = DefaultBlockSize
	}
	if c.AdaptiveC == 0 {
		c.AdaptiveC = DefaultAdaptiveC
	}

	var (
		w              = src.Width()
		blurSize       = mpyOdd(c.BlurSize, w)
		morphCloseSize = mpyOdd(c.MorphCloseSize, w)
		blockSize      = mpyOdd(c.BlockSize, w)
		adaptiveC      = c.AdaptiveC
	)

	//
	// Now that the configured values are set, we begin the operation. We use
	// three steps:
	//
	// 1) Blur the image.
	//    <https://docs.opencv.org/4.12.0/da/dc5/group__gapi__filters.html#gaaca00b81d171421032917e53751ac427>
	//
	// 2) Use adaptive thresholding to binarize the image.
	//    <https://docs.opencv.org/4.12.0/d7/d1b/group__imgproc__misc.html#ga72b913f352e4a1b1b397736707afcde3>
	//
	// 3) Use a morphological close to close small gaps and make the lines
	//    nicer.
	//    <https://docs.opencv.org/4.12.0/d4/d86/group__imgproc__filter.html#ga67493776e3ad1a3df63883829375201f>
	//

	var (
		kernel = gocv.GetStructuringElement(
			gocv.MorphRect,
			image.Pt(morphCloseSize, morphCloseSize),
		)
	)
	defer kernel.Close()

	err := gocv.GaussianBlur(
		src.m, &dst.m,
		image.Pt(blurSize, blurSize),
		0, 0,
		gocv.BorderDefault,
	)
	if err != nil {
		return ErrOpenCV
	}

	err = gocv.AdaptiveThreshold(
		dst.m, &dst.m,
		255,
		gocv.AdaptiveThresholdMean,
		gocv.ThresholdBinary,
		blockSize,
		float32(adaptiveC),
	)
	if err != nil {
		return ErrOpenCV
	}

	err = gocv.MorphologyEx(
		dst.m, &dst.m,
		gocv.MorphClose,
		kernel,
	)
	if err != nil {
		return ErrOpenCV
	}

	*dst.t = MatTypeBinary
	return nil
}

// Multiplies the two given numbers but always returns a positive, odd integer.
func mpyOdd(a float64, b uint) int {
	i := int(a * float64(b))
	if i < 1 {
		return 1
	}
	if i%2 == 0 {
		return i + 1
	}
	return i
}
