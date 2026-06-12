package fs

//
// This file implements the Store interface by using local storage.
//

import (
	"errors"
	"image"
	"image/draw"
	"os"
	"path/filepath"
)

type localStore struct {
	root string
}

// Opens the named root directory as a local file store.
func NewLocalStore(root string) Store {
	return &localStore{
		root: root,
	}
}

func (s *localStore) GetImg(key string) (image.Image, error) {
	f, err := os.Open(filepath.Join(s.root, key))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := DecodeImg(f)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func (s *localStore) PutImg(key string, img image.Image) error {
	fullPath := filepath.Join(s.root, key)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	w, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer w.Close()

	err = EncodeImg(w, img)
	return err
}

func (s *localStore) GetImgSnippet(
	path string,
	x, y, width, height int,
) (image.Image, error) {
	img, err := s.GetImg(path)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()

	if x < bounds.Min.X ||
		y < bounds.Min.Y ||
		x+width > bounds.Max.X ||
		y+height > bounds.Max.Y {
		return nil, errors.New("requested region outside image bounds")
	}

	rect := image.Rect(0, 0, width, height)

	cropped := image.NewRGBA(rect)

	draw.Draw(
		cropped,
		rect,
		img,
		image.Point{X: x, Y: y},
		draw.Src,
	)

	return cropped, nil
}
