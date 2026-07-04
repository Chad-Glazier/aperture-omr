package fs

//
// This file implements the Store interface by using local storage.
//

import (
	"image"
	"image/draw"
	"os"
	"path/filepath"
)

type localStore struct {
	// The root directory of the storage.
	root string
}

// Opens the named root directory as a local file store.
func NewLocalStore(root string) Store {
	return &localStore{
		root: root,
	}
}

func (s *localStore) ImgExists(key string) bool {
	_, err := os.Stat(filepath.Join(s.root, key))
	return err == nil
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
	if err := os.MkdirAll(s.root, 0755); err != nil {
		return err
	}

	w, err := os.Create(filepath.Join(s.root, key))
	if err != nil {
		return err
	}
	defer w.Close()

	err = EncodeImg(w, img)
	return err
}

func (s *localStore) DeleteImg(key string) error {
	return os.Remove(filepath.Join(s.root, key))
}

func (s *localStore) ImgSnippet(
	key string,
	x, y, width, height int,
) (image.Image, error) {
	img, err := s.GetImg(key)
	if err != nil {
		return nil, err
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
