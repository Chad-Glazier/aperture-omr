/*
This package contains handler functions for the HTTP server.
*/
package handler

import "net/http"

// Sends the OpenAPI specification in YAML format.
func OpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-yaml")
	http.ServeFile(w, r, "./api/openapi.yaml")
}

// Sends an HTML page that documents the API based on the OpenAPI
// specification.
func DocsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	http.ServeFile(w, r, "./api/docs.html")
}
