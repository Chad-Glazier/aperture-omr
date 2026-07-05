/*
This package provides the means to store files ("fs" is short for "file
storage").
*/
package fs

import (
	"errors"
	"image"

	"gocv.io/x/gocv"
)

// Represents a file store, implementing a map-like interface to save and load
// images.
type Store interface {
	// Returns true if the image exists and false otherwise.
	ImgExists(key string) bool
	// Retrieves an image from the store with the given key.
	GetImg(key string) (image.Image, error)
	// Puts an image in the store and assigns it the given key. Any
	// pre-existing image with the same key will be overwritten.
	PutImg(key string, img image.Image) error
	// Deletes an image from the store.
	DeleteImg(key string) error
	// Retrieves an image from the store and returns a copy that is cropped to
	// match the given dimensions.
	ImgSnippet(key string, x, y, width, height int) (image.Image, error)
}

type MatSaveLoader interface {
	// Saves an OpenCV matrix under the given key. If a matrix already exists
	// with the given key, it will be overwritten.
	MatSave(key string, mat *gocv.Mat) error

	// Loads an OpenCV matrix by the given key. If no matrix is associated with
	// the key, then ErrNotFound is returned.
	MatLoad(key string) (*gocv.Mat, error)
}

var (
	ErrNotFound = errors.New("the requested resource was not found")
)
