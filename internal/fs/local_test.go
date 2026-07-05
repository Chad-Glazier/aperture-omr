package fs

import (
	"image"
	"os"
	"testing"

	"gocv.io/x/gocv"
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

func TestLocalStoreImg(t *testing.T) {

	//
	// Setup
	//

	store := NewLocalStore(t.TempDir())

	r, err := os.Open("./testdata/sample_image.png")
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

	name := "sample.png"

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

	err = store.PutImg("snippet.png", snippet)
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

//
// Test the MatSaveLoader implementation.
//

func TestLocalMatSaveLoader(t *testing.T) {
	s := NewLocalStore(t.TempDir())

	mat := gocv.IMRead("testdata/sample_image.png", gocv.IMReadGrayScale)
	if mat.Empty() {
		t.Fatal("failed to read source image")
	}

	s.MatSave("testkey", &mat)

	loaded, err := s.MatLoad("testkey")
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Cols() != mat.Cols() {
		t.Fatalf("expected %d columns, got %d", mat.Cols(), loaded.Cols())
	}

	if loaded.Rows() != mat.Rows() {
		t.Fatalf("expected %d rows, got %d", mat.Rows(), loaded.Rows())
	}

	if loaded.Type() != mat.Type() {
		t.Fatalf("expected matrix type %d, got %d", mat.Type(), loaded.Type())
	}

	for i := range mat.Rows() {
		for j := range mat.Cols() {
			for k := range mat.Channels() {
				if mat.GetUCharAt3(i, j, k) != loaded.GetUCharAt3(i, j, k) {
					t.Fatalf(
						"expected (%d,%d,%d) to have %d, got %d",
						i, j, k,
						mat.GetUCharAt3(i, j, k),
						loaded.GetUCharAt3(i, j, k),
					)
				}
			}
		}
	}
}
