/*
This package defines and implements the resources that are provided to handler
functions.
*/
package resources

import (
	"errors"
	"image"
	"io"
	"time"

	"github.com/Chad-Glazier/aperture-omr/internal/omr"
)

var (
	ErrEncodingImage    = errors.New("error encoding image data")
	ErrDecodingImage    = errors.New("error decoding image data")
	ErrDatabaseWrite    = errors.New("error writing to the database")
	ErrDatabaseRead     = errors.New("error reading from the database")
	ErrFileStorageWrite = errors.New("error writing data to file store")
	ErrFileStorageRead  = errors.New("error reading data from file store")
	ErrNotFound         = errors.New("the requested resource was not found")
	ErrSerializing      = errors.New("error serializing data")
	ErrDeserializing    = errors.New("error deserializing data")
)

// A interface that describes all resources an HTTP handler should have access
// to.
type ServerResources interface {

	// Closes all connections.
	Close() error

	// Saves a marking template and returns the new ID for it.
	//
	// If an error is returned, it will be [ErrDatabaseWrite] or
	// [ErrSeializing].
	SaveMarkingTemplate(tmpl omr.MarkTemplate) (string, error)

	// Loads a marking template and returns the new ID for it.
	//
	// If an error is returned, it will be [ErrNotFound],
	// [ErrDatabaseRead], or [ErrDeserializing].
	LoadMarkingTemplate(id string) (omr.MarkTemplate, error)

	// Deletes a marking template. Redundant calls are safe.
	DeleteMarkingTemplate(id string)

	// Saves a preprocessing template and returns the new ID for it.
	//
	// If an error is returned, it will be [ErrDecodingImage],
	// [ErrDatabaseWrite], or [ErrFileStorageWrite].
	SavePreprocessingTemplate(tmpl omr.PreprocessTemplate) (string, error)

	// Loads a preprocessing template and returns the ID for it. In the
	// returned set of matrices, the element [i][j] is the i-th oage's j-th
	// anchor.
	//
	// If an error is returned, it will be [ErrNotFound],
	// [ErrDatabaseRead], or [ErrDeserializing].
	LoadPreprocessingTemplate(id string) (omr.PreprocessTemplate, error)

	// Deletes a preprocessing template and its anchors. Redundant calls are
	// safe.
	DeletePreprocessingTemplate(id string)

	// Saves a preprocessed scan's pages.
	//
	// If an error is returned, it will be [ErrDatabaseWrite],
	// [ErrFileStorageWrite], or [ErrEncodingImage].
	SaveScan(
		pages []omr.Mat,
		templateId string,
	) (string, error)

	// Loads a preprocessed scan's binarized pages.
	//
	// If an error is returned, it will be [ErrNotFound] or [ErrDatabaseRead].
	LoadScan(scanId string) ([]omr.Mat, error)

	// Deletes a scan and its pages. Redundant calls are safe.
	DeleteScan(scanId string)

	// Deletes all scans that were created before the given time.
	DeleteAllScansFromBefore(time.Time)

	// Loads a picture from a scan. The "picture" of a scan is the version that
	// is maintained for human viewers.
	//
	// If an error is returned, it will be [ErrDatabaseRead], [ErrNotFound], or
	// [ErrFileStorageRead].
	LoadScanImage(scanId string, pageIdx uint) (image.Image, error)

	// Opens a scan picture for reading.
	//
	// If an error is returned, it will be [ErrDatabaseRead], [ErrNotFound], or
	// [ErrFileStorageRead].
	OpenScanPicture(scanId string, pageIdx uint) (io.ReadCloser, error)

	// Returns the total number of pictures stored and the number of bytes they
	// collectively occupy.
	CountPictures() (int, uint64)
	// Returns the total number of matrices stored and the number of bytes they
	// collectively occupy.
	CountMats() (int, uint64)
	// Returns the number of bytes used by the database.
	DBSize() uint64
}
