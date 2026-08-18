package omr

import (
	"errors"
	"image"

	"gocv.io/x/gocv"
)

var (
	ErrWrongMatType = errors.New("a matrix operand was not of the correct type")
	ErrEmptyMat     = errors.New("a matrix operand was empty when it was not allowed to be")
	ErrOpenCV       = errors.New("opencv returned an unexpected error")
)

type BinarizeConfig struct {

	// The Gaussian kernel size, used for blurring. Must be positive and odd.
	//
	// Defaults to [DefaultBlurSize].
	//
	// <https://docs.opencv.org/4.12.0/d4/d86/group__imgproc__filter.html#gae8bdcd9154ed5ca3cbc1766d960f45c1>
	BlurSize uint

	// The kernel size for the morphological closing operation.
	//
	// Defaults to [DefaultMorphCloseSize].
	//
	// <https://docs.opencv.org/4.12.0/d4/d86/group__imgproc__filter.html#ga67493776e3ad1a3df63883829375201f>
	MorphCloseSize uint

	// The size of a pixel neighborhood that is used to calculate a threshold
	// value for each pixel. Must be positive and odd.
	//
	// Defaults to [DefaultBlockSize].
	//
	// <https://docs.opencv.org/4.12.0/d7/d1b/group__imgproc__misc.html#ga72b913f352e4a1b1b397736707afcde3?>
	BlockSize uint

	// A constant that is subtracted from each pixel's threshold value. Making
	// this positive will cause the thresholding to ignore lighter values,
	// while making it negative will make them more sensitive to them. Since
	// pencil marks can be very light, it's recommended that you keep it
	// negative.
	//
	// Defaults to [DefaultAdaptiveC].
	//
	// <https://docs.opencv.org/4.12.0/d7/d1b/group__imgproc__misc.html#ga72b913f352e4a1b1b397736707afcde3>
	AdaptiveC float32
}

const (
	DefaultBlurSize       = 3
	DefaultMorphCloseSize = 3
	DefaultBlockSize      = 91
	DefaultAdaptiveC      = -5
)

// Binarizes a grayscale matrix.
//
// If an error is returned, it will be [ErrWrongMatType], [ErrEmptyMat], or
// [ErrOpenCV]. If it's [ErrOpenCV], then the destination matrix may have
// corrupted values from a partial operation and should be closed.
func Binarize(dst Mat, src Mat, conf *BinarizeConfig) error {
	switch { // Handle bad inputs.
	case *src.t != MatTypeGray:
		return ErrWrongMatType
	case src.Empty():
		return ErrEmptyMat
	}

	// We copy the configuration value so that we can silently adjust it as
	// needed, without risking race conditions. The only reason we're taking a
	// pointer instead of the value is so that the caller can neatly pass [nil]
	// if they want to use the default configuration.
	var c BinarizeConfig
	if conf == nil {
		c = BinarizeConfig{}
	} else {
		c = *conf
	}

	// Set default configuration values.
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

	// Certain values must be odd. We could return an error when they are not,
	// but I prefer silently adjusting them.
	if c.BlurSize%2 == 0 {
		c.BlurSize++
	}

	// If the block size is larger than the minimum dimension of the matrix,
	// extra work will be done for no good reason. Thus we cap it.
	c.BlockSize = min(c.BlockSize, src.Height(), src.Width())
	if c.BlockSize%2 == 0 {
		c.BlockSize++
	}

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
			image.Pt(int(c.MorphCloseSize), int(c.MorphCloseSize)),
		)
	)
	defer kernel.Close()

	err := gocv.GaussianBlur(
		src.m, &dst.m,
		image.Pt(int(c.BlurSize), int(c.BlurSize)),
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
		gocv.ThresholdBinaryInv,
		int(c.BlockSize),
		c.AdaptiveC,
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

// Scales the pixel dimensions in the configuration by the given multiplier.
// This is useful if, for example, you've configured the values for scans at
// one specific resolution but you want it to behave properly with other
// resolutions as well.
func ScaleConfig(c BinarizeConfig, multiplier float64) BinarizeConfig {
	c.BlurSize = uint(multiplier * float64(c.BlurSize))
	c.MorphCloseSize = uint(multiplier * float64(c.MorphCloseSize))
	c.BlockSize = uint(multiplier * float64(c.BlockSize))

	// Make sure the values that have to be odd are, in fact, odd.
	if c.BlurSize%2 == 0 {
		c.BlurSize++
	}
	if c.BlockSize%2 == 0 {
		c.BlockSize++
	}

	return c
}
