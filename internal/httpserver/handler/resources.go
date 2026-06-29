package handler

import (
	"encoding/json"
	"net/http"

	"ubco-team15/omr/internal/httpserver/dto"
)

//
// The ServerResources interface defines things that should be shared between
// requests. E.g., access to data/file stores.
//

type ServerResources interface {
	// Saves a marking template and returns the new ID for it or an error if
	// the operation failed.
	SaveMarkingTemplate(tmpl *dto.MarkingTemplate) (string, error)
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
