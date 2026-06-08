package handler

import (
	"log/slog"
	"net/http"
	"ubco-team15/omr/internal/database"
)

// Writes a 200 response to indicate that the server is functioning.
func Health(w http.ResponseWriter, r *http.Request) {

	err := database.CheckConnection()
	if err != nil {
		slog.Error("error connecting to database", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}
