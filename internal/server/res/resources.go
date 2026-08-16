/*
This package defines and implements the resources that are provided to handler
functions.
*/
package res

import (
	"image"
	"io"
	"net/http"

	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"

	"gocv.io/x/gocv"
)

// A interface that presents all resources an HTTP handler should have access
// to.
type ServerResources interface {
	// Closes all connections.
	Close() error
	// Saves a marking template and returns the new ID for it.
	SaveMarkingTemplate(tmpl *dto.MarkingTemplate) (string, error)
	// Loads a marking template and returns the new ID for it. Returns an error
	// if the template was not found.
	LoadMarkingTemplate(id string) (*dto.MarkingTemplate, error)
	// Deletes a marking template. Redundant calls are safe.
	DeleteMarkingTemplate(id string)
	// Saves a preprocessing template and returns the new ID for it.
	SavePreprocessingTemplate(tmpl *dto.PreprocessingTemplate) (string, error)
	// Loads a preprocessing template and returns the ID for it.
	LoadPreprocessingTemplate(id string) (*dto.PreprocessingTemplate, error)
	// Deletes a preprocessing template and its anchors. Redundant calls are
	// safe.
	DeletePreprocessingTemplate(id string)
	// Loads anchor matrices. To get the i-th page's j-th anchor, you would
	// index [i][j] from the returned slice.
	LoadAnchors(templateId string) ([][]gocv.Mat, error)
	// Saves the given anchor matrix.
	SaveAnchor(
		anchor gocv.Mat,
		templateId string,
		pageIdx, anchorIdx int,
	) error
	// Saves a preprocessed scan via two slices of matrices: the first
	// represents the binarized images we will use for marking, and the second
	// represents the grayscaled image we will use to make human-readable
	// snippets. The template ID refers to the preprocessing template used to
	// produce these scans.
	SaveScan(
		pages []gocv.Mat,
		pagePictures []gocv.Mat,
		templateId string,
	) (string, error)
	// Loads a preprocessed scan's binarized pages.
	LoadScan(scanId string) ([]gocv.Mat, error)
	// Deletes a scan and its pages. Redundant calls are safe.
	DeleteScan(scanId string)
	// Loads a picture from a scan. The "picture" of a scan is the version that
	// is maintained for human viewers.
	LoadScanPicture(scanId string, pageIdx uint) (image.Image, error)
	// Opens a scan picture for reading.
	OpenScanPicture(scanId string, pageIdx uint) (io.ReadCloser, error)
	// Returns the total number of pictures stored and the number of bytes they
	// collectively occupy.
	CountPictures() (int, uint64)
	// Returns the total number of matrices stored and the number of bytes they
	// collectively occupy.
	CountMats() (int, uint64)
	// Returns the number of bytes used by the database.
	DBSize() uint64
	// Returns true if and only if the request's headers include the proper
	// administrator key.
	CheckAdminKey(r *http.Request) bool
	// Sets the admin key.
	SetAdminKey(key string)
}
