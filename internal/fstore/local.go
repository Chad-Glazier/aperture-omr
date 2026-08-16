package fstore

import (
	"image"
	"io"
	"os"
	"path/filepath"

	"gocv.io/x/gocv"
)

//
// Local ImageStore implementation.
//

type localImageStore struct {
	root string // The path to the root directory of the store.
}

func NewLocalImageStore(rootDir string) (ImageStore, error) {
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, err
	}

	return &localImageStore{root: rootDir}, nil
}

func (s *localImageStore) Get(key string) (image.Image, error) {
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

func (s *localImageStore) Set(key string, img image.Image) error {
	w, err := os.Create(filepath.Join(s.root, key))
	if err != nil {
		return err
	}
	defer w.Close()

	err = EncodeImg(w, img)
	return err
}

func (s *localImageStore) SetBytes(key string, buf []byte) error {
	return os.WriteFile(
		filepath.Join(s.root, key),
		buf,
		0755,
	)
}

func (s *localImageStore) Delete(key string) error {
	return os.Remove(filepath.Join(s.root, key))
}

func (s *localImageStore) Open(key string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(s.root, key))
}

func (s *localImageStore) Count() (int, uint64) {
	files, err := os.ReadDir(s.root)
	if err != nil {
		return 0, 0
	}

	var totalBytes uint64
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			return 0, 0
		}
		totalBytes += uint64(info.Size())
	}

	return len(files), totalBytes
}

//
// Local MatStore implementation.
//

type localMatStore struct {
	root string // The path to the root directory of the store.
}

func NewLocalMatStore(rootDir string) (MatStore, error) {
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, err
	}

	return &localMatStore{root: rootDir}, nil
}

func (s *localMatStore) Set(key string, mat gocv.Mat) error {

	w, err := os.Create(filepath.Join(s.root, key))
	if err != nil {
		return err
	}
	defer w.Close()

	return EncodeMat(w, mat)
}

func (s *localMatStore) Get(key string) (gocv.Mat, error) {

	f, err := os.Open(filepath.Join(s.root, key))
	if err != nil {
		return gocv.Mat{}, err
	}
	defer f.Close()

	return DecodeMat(f)
}

func (s *localMatStore) Delete(key string) error {
	return os.Remove(filepath.Join(s.root, key))
}

func (s *localMatStore) Count() (int, uint64) {
	files, err := os.ReadDir(s.root)
	if err != nil {
		return 0, 0
	}

	var totalBytes uint64
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			return 0, 0
		}
		totalBytes += uint64(info.Size())
	}

	return len(files), totalBytes
}
