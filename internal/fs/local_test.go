package fs

import (
	"os"
	"testing"

	"gocv.io/x/gocv"
)

//
// Tests
//

func TestLocalImageStore(t *testing.T) {

	store, err := NewLocalImageStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	r, err := os.Open("./testdata/sample_image.jpg")
	if err != nil {
		t.Error("error reading test image: " + err.Error())
	}
	defer r.Close()

	srcImg, err := DecodeImg(r)
	if err != nil {
		t.Error("error decoding test image: " + err.Error())
	}

	name := "sample.jpg"

	if img, _ := store.Get(name); img != nil {
		t.Error("image already exists in store")
	}

	err = store.Set(name, srcImg)
	if err != nil {
		t.Error("error putting image: " + err.Error())
	}

	if _, err := store.Get(name); err != nil {
		t.Error("error getting image: " + err.Error())
	}

	err = store.Delete(name)
	if err != nil {
		t.Error("error deleting image: " + err.Error())
	}

	if img, _ := store.Get(name); img != nil {
		t.Error("image exists after deletion")
	}
}

func TestLocalMatStore(t *testing.T) {

	s, err := NewLocalMatStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	mat := gocv.IMRead("testdata/sample_image.jpg", gocv.IMReadGrayScale)
	if mat.Empty() {
		t.Fatal("failed to read source image")
	}
	defer mat.Close()

	s.Set("testkey", &mat)

	loaded, err := s.Get("testkey")
	if err != nil {
		t.Fatal(err)
	}
	defer loaded.Close()

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

//
// Benchmarks
//

func BenchmarkLocalMatLoad(b *testing.B) {
	s, err := NewLocalMatStore(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}

	mat := gocv.IMRead("testdata/sample_image.png", gocv.IMReadGrayScale)
	if mat.Empty() {
		b.Fatal("failed to read source image")
	}

	s.Set("testkey", &mat)

	for b.Loop() {
		mat, _ := s.Get("testkey")	
		mat.Close()	
	}	
}

func BenchmarkLocalMatSave(b *testing.B) {
	s, err := NewLocalMatStore(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}

	mat := gocv.IMRead("testdata/sample_image.png", gocv.IMReadGrayScale)
	if mat.Empty() {
		b.Fatal("failed to read source image")
	}
	defer mat.Close()

	for b.Loop() {
		s.Set("testkey", &mat)	
	}
}
