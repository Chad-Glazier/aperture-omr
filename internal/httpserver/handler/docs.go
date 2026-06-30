/*
This package contains handler functions for the HTTP server.
*/
package handler

import (
	"net/http"
)

// Note: in the production build, we can just embed the YAML file instead of
// serving it from the filesystem. I'm leaving it like this for now so that I
// can update the spec without having to restart the server.

// Sends the OpenAPI specification in YAML format.
func OpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-yaml")
	http.ServeFile(w, r, "./api/openapi.yaml")
}

// Sends an HTML page that documents the API based on the OpenAPI
// specification.
func DocsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
	<!DOCTYPE html>

	<html>

	<head>
		<title>OMR Service</title>
	</head>

	<body>
		<elements-api apiDescriptionUrl="/openapi.yaml" layout="sidebar" router="memory" tryItCredentialsPolicy="include" />
		<script src="https://unpkg.com/@stoplight/elements/web-components.min.js"></script>
		<link rel="stylesheet" href="https://unpkg.com/@stoplight/elements/styles.min.css" />
		<style>
			body>elements-api>div {
				height: 100vh !important;
			}
		</style>
		<script>
			// Remove the ugly logo. Sorry.
			window.addEventListener("DOMContentLoaded", () => {
				let link = document.querySelector("a.sl-flex.sl-items-center.sl-px-4.sl-py-3.sl-border-t")
				link.lastChild.innerHTML = "Powered by Swalalala"
				link.href = "https://youtu.be/2Ax-TWSceLs?si=gvZhjODbIhNw-dSf"
				setInterval(() => {
					let link = document.querySelector("a.sl-flex.sl-items-center.sl-px-4.sl-py-3.sl-border-t")
					link.lastChild.innerHTML = "Powered by Swalla-la-la"
					link.href = "https://youtu.be/2Ax-TWSceLs?si=gvZhjODbIhNw-dSf"
				}, 200)
			})
		</script>
	</body>

	</html>
	`))
}

// Writes a 200 response to indicate that the server is functioning.
func Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

