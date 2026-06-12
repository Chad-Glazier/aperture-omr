/*
This package provides the means to store files ("fs" is short for "file
storage"). It
*/
package fs

import (
	"image"
)

// Represents a file store, implementing a map-like interface to save and load
// images.
type Store interface {
	// Retrieves an image from the store with the given key.
	GetImg(key string) (image.Image, error)
	// Puts an image in the store and assigns it the given key.
	PutImg(key string, img image.Image) error
	// Retrieves an image from the store and returns a copy that is cropped to
	// match the given dimensions.
	GetImgSnippet(key string, x, y, width, height int) (image.Image, error)
}

func Connect() Store {
	return NewLocalStore("./tmp_store")
}
