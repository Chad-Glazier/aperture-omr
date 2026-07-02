/*
This package provides the means to store files ("fs" is short for "file
storage").
*/
package fs

import (
	"image"

	"github.com/google/uuid"
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

// Adds an image to the store with a newly-generated UUID which is then
// returned.
func PutWithUUID(store Store, img image.Image) (string, error) {
	id := uuid.New().String()
	err := store.PutImg(id, img)
	return id, err
}
