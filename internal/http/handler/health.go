package handler

import (
	"log/slog"
	"net/http"
	"ubc/team15/internal/db"
)

// Writes a 200 response to indicate that the server is functioning.
func Health(w http.ResponseWriter, r *http.Request) {

	_, err := db.Connect()
	if err != nil {
		slog.Error("error connecting to database", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}
