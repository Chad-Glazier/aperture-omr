/*
This package provides the means to store files ("fstore" is short for "file
storage").
*/
package fstore

import (
	"errors"
	"image"
	"io"

	"github.com/Chad-Glazier/aperture-omr/internal/omr"
)

var (
	ErrNotFound     = errors.New("fstore: the requested resource was not found")
	ErrEncoding     = errors.New("fstore: error encoding data")
	ErrDecoding     = errors.New("fstore: error decoding data")
	ErrCreatingFile = errors.New("fstore: the underlying system failed to create a new file")
)

// Represents an image store, implementing a map-like interface to save and
// load images to persistent storage.
type ImageStore interface {

	// Retrieves an image from the store with the given key.
	//
	// If an error is returned, it will be [ErrNotFound] or [ErrDecoding].
	Get(key string) (image.Image, error)

	// Puts an image in the store and assigns it the given key. Any
	// pre-existing image with the same key will be overwritten.
	//
	// If an error is returned, it will be [ErrEncoding] or [ErrCreatingFile].
	Set(key string, img image.Image) error

	// Deletes an image from the store.
	Delete(key string)

	// Counts the number of stored images and the total number of bytes they
	// occupy.
	Count() (int, uint64)

	// Works the same as Set, except it stores the raw bytes and assumes that
	// they represent an image that is correctly encoded with the format
	// specified by [ImgContentType]. It is strongly recommended that image
	// encoding is verified in some way before calling this function.
	//
	// If an error is returned, it will be [ErrCreatingFile].
	SetBytes(key string, buf []byte) error

	// Opens an existing image for reading. The encoding of the image should
	// match [ImgContentType].
	//
	// If an error is returned, it will be [ErrNotFound].
	Open(key string) (io.ReadCloser, error)

	// Creates a new image and returns a writer for it. If the key was already
	// associated with an image, it will be overwritten.
	//
	// If an error is returned, it will be [ErrCreatingFile].
	Create(key string) (io.WriteCloser, error)
}

// Represents an OpenCV matrix store, implementing a map-like interface to
// save and load matrices to/from persistent storage.
type MatStore interface {

	// Loads an OpenCV matrix by the given key.
	//
	// If an error is returned, it will be [ErrNotFound] or [ErrDecoding].
	Get(key string) (omr.Mat, error)

	// Saves an OpenCV matrix under the given key. If a matrix already exists
	// with the given key, it will be overwritten.
	//
	// If an error is returned, it will be [ErrEncoding] or [ErrCreatingFile].
	Set(key string, mat omr.Mat) error

	// Deletes a stored matrix.
	Delete(key string)

	// Counts the number of stored matrices and the total number of bytes they
	// occupy.
	Count() (int, uint64)
}
