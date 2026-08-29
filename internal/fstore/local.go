package fstore

import (
	"image"
	"io"
	"os"
	"path/filepath"

	"github.com/Chad-Glazier/aperture-omr/internal/omr"
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
		return nil, ErrNotFound
	}
	defer f.Close()

	img, err := DecodeImg(f)
	if err != nil {
		return nil, ErrDecoding
	}

	return img, nil
}

func (s *localImageStore) Set(key string, img image.Image) error {
	w, err := os.Create(filepath.Join(s.root, key))
	if err != nil {
		return ErrCreatingFile
	}
	defer w.Close()

	if err = EncodeImg(w, img); err != nil {
		return ErrEncoding
	}

	return nil
}

func (s *localImageStore) SetBytes(key string, buf []byte) error {
	err := os.WriteFile(
		filepath.Join(s.root, key),
		buf,
		0755,
	)
	if err != nil {
		return ErrCreatingFile
	}

	return nil
}

func (s *localImageStore) Delete(key string) {
	os.Remove(filepath.Join(s.root, key))
}

func (s *localImageStore) Open(key string) (io.ReadCloser, error) {
	r, err := os.Open(filepath.Join(s.root, key))
	if err != nil {
		return nil, ErrNotFound
	}
	return r, nil
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

func (s *localMatStore) Set(key string, mat omr.Mat) error {
	w, err := os.Create(filepath.Join(s.root, key))
	if err != nil {
		return ErrCreatingFile
	}
	defer w.Close()

	if err := EncodeMat(w, mat); err != nil {
		return ErrEncoding
	}

	return nil
}

func (s *localMatStore) Get(key string) (omr.Mat, error) {
	f, err := os.Open(filepath.Join(s.root, key))
	if err != nil {
		return omr.Mat{}, ErrNotFound
	}
	defer f.Close()

	mat, err := DecodeMat(f)
	if err != nil {
		return omr.Mat{}, ErrDecoding
	}

	return mat, nil
}

func (s *localMatStore) Delete(key string) {
	os.Remove(filepath.Join(s.root, key))
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
