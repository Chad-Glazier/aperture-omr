package handler

import (
	"net/http"
)

// Writes a 200 response to indicate that the server is functioning.
func Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
