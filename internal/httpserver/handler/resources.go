package handler

import (
	"encoding/json"
	"image"
	"net/http"

	"ubco-team15/omr/internal/httpserver/dto"
)

//
// The ServerResources interface defines things that should be shared between
// requests. E.g., access to data/file stores.
//

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

type ServerResources interface {
	MarkingTemplateSaveLoader
	PreprocessingTemplateSaveLoader
	AnchorSaveLoader
}

//
// General helper functions.
//

func sendJson(w http.ResponseWriter, v any) {
	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "error writing response", http.StatusInternalServerError)
	}
}
