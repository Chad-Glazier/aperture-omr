/*
This package provides the means to store files ("fs" is short for "file
storage").
*/
package fs

import (
	"image"
)

// Represents a file store, implementing a map-like interface to save and load
// images.
type Store interface {
	// Returns true if the image exists and false otherwise.
	ImgExists(key string) bool
	// Retrieves an image from the store with the given key.
	GetImg(key string) (image.Image, error)
	// Puts an image in the store and assigns it the given key. Any pre-
	// existing with the same key will be overwritten.
	PutImg(key string, img image.Image) error
	// Deletes an image from the store.
	DeleteImg(key string) error
	// Retrieves an image from the store and returns a copy that is cropped to
	// match the given dimensions.
	ImgSnippet(key string, x, y, width, height int) (image.Image, error)
	// // Opens an image as a byte stream. The encoding matches the type specified
	// // by ImgContentType and can be decoded with Decode.
	// ImgReader(key string) (io.ReadCloser, error)
	// // Opens a writer to create a new image associated with the key. If the key
	// // is already associated with an image, then an error will be returned.
	// ImgWriter(key string) (io.WriteCloser, error)
}
