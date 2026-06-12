package fs

import (
	"image"
	"os"
	"testing"

	"golang.org/x/image/tiff"
)

func imageEqual(a, b image.Image) bool {
	bounds := a.Bounds()

	if !bounds.Eq(b.Bounds()) {
		return false
	}

	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			if a.At(x, y) != b.At(x, y) {
				return false
			}
		}
	}

	return true
}

func TestLocalStore(t *testing.T) {
	root, err := os.MkdirTemp(".", "omr_test_fs_*")
	if err != nil {
		t.Errorf("error creating temporary directory")
	}
	defer os.RemoveAll(root)

	store := NewLocalStore(root)

	r, err := os.Open("./testdata/sample_image.tiff")
	if err != nil {
		t.Error("error reading test image: " + err.Error())
	}
	srcImg, err := tiff.Decode(r)
	if err != nil {
		t.Error("error decoding test image: " + err.Error())
	}

	//
	// Testing the methods.
	//

	name := "sample.tiff"

	err = store.PutImg(name, srcImg)
	if err != nil {
		t.Error("error putting image: " + err.Error())
	}

	img, err := store.GetImg(name)
	if err != nil {
		t.Error("error getting image: " + err.Error())
	}

	if !imageEqual(srcImg, img) {
		t.Error("expected put image to equal source image")
	}

	snippet, err := store.GetImgSnippet(name, 20, 20, 100, 100)
	bounds := snippet.Bounds()
	if bounds.Dx() != 100 {
		t.Errorf("expected snippet width of %d, got %d", 100, bounds.Dx())
	}
	if bounds.Dy() != 100 {
		t.Errorf("expected snippet height of %d, got %d", 100, bounds.Dy())
	}
}
