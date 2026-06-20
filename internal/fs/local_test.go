package fs

import (
	"image"
	"os"
	"testing"
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

	//
	// Setup
	//

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
	srcImg, err := DecodeImg(r)
	if err != nil {
		t.Error("error decoding test image: " + err.Error())
	}

	//
	// Testing
	//

	name := "sample.tiff"

	if store.ImgExists(name) {
		t.Error("image already exists in store")
	}

	err = store.PutImg(name, srcImg)
	if err != nil {
		t.Error("error putting image: " + err.Error())
	}

	if !store.ImgExists(name) {
		t.Error("image does not exist in store after being put")
	}

	img, err := store.GetImg(name)
	if err != nil {
		t.Error("error getting image: " + err.Error())
	}

	if !imageEqual(srcImg, img) {
		t.Error("expected put image to equal source image")
	}

	snippetSize := 400
	snippet, err := store.ImgSnippet(name, 20, 20, snippetSize, snippetSize)
	if err != nil {
		t.Error("error creating snippet: " + err.Error())
	}

	bounds := snippet.Bounds()
	if bounds.Dx() != snippetSize {
		t.Errorf("expected snippet width of %d, got %d", snippetSize, bounds.Dx())
	}
	if bounds.Dy() != snippetSize {
		t.Errorf("expected snippet height of %d, got %d", snippetSize, bounds.Dy())
	}

	err = store.PutImg("snippet.tiff", snippet)
	if err != nil {
		t.Error("error putting snippet: " + err.Error())
	}

	err = store.DeleteImg(name)
	if err != nil {
		t.Error("error deleting image: " + err.Error())
	}

	if store.ImgExists(name) {
		t.Error("image exists after deletion")
	}
}
