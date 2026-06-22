package handler

import (
	"net/http"
	"ubco-team15/omr/internal/database"
)

// Writes a 200 response to indicate that the server is functioning.
func Health(w http.ResponseWriter, r *http.Request) {

	err := database.CheckConnection()
	if err != nil {
		http.Error(
			w,
			"error connecting to database",
			http.StatusInternalServerError,
		)
	}

	w.WriteHeader(http.StatusOK)
}
