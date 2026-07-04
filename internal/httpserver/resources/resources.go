package resources

import (
	"image"

	"ubco-team15/omr/internal/httpserver/dto"
)

// A union interface that presents all resources that an HTTP handler should 
// have access to.
type ServerResources interface {
	MarkingTemplateSaveLoader
	PreprocessingTemplateSaveLoader
	AnchorSaveLoader
	ScanSaveLoader
	SnippetLoader
}

type MarkingTemplateSaveLoader interface {
	// Saves a marking template and returns the new ID for it.
	SaveMarkingTemplate(tmpl *dto.MarkingTemplate) (string, error)

	// Loads a marking template and returns the new ID for it.
	LoadMarkingTemplate(id string) (*dto.MarkingTemplate, error)
}

type PreprocessingTemplateSaveLoader interface {
	// Saves a preprocessing template and returns the new ID for it.
	SavePreprocessingTemplate(tmpl *dto.PreprocessingTemplate) (string, error)

	// Loads a preprocessing template and returns the ID for it.
	LoadPreprocessingTemplate(id string) (*dto.PreprocessingTemplate, error)
}

type AnchorSaveLoader interface {
	// Saves an anchor image and returns the new ID for it.
	SaveAnchor(anchor image.Image, templateId string, pageIdx, anchorIdx int) error

	// Loads an anchor image.
	LoadAnchor(templateId string, pageIdx, anchorIdx int) (image.Image, error)
}

type ScanSaveLoader interface {
	// Saves a preprocessed scan via a slice of images where each image
	// represents a page. The preprocessing template ID is also included for
	// the sake of debugging. Returns an ID for the scan.
	SaveScan(
		pages []image.Image,
		colorPages []image.Image,
		templateId string,
	) (string, error)

	// Loads a preprocessed scan's pages.
	LoadScan(scanId string) ([]image.Image, error)

	// Loads a preprocessed scan's color pages.
	LoadColorScan(scanId string) ([]image.Image, error)
}

type SnippetLoader interface {
	// Loads an image snippet that isolates a question on the given scan.
	LoadSnippet(scanId, templateId, questionId string) (image.Image, error)
}
