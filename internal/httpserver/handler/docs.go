/*
This package contains handler functions for the HTTP server.
*/
package handler

import (
	"net/http"
)

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
				link.parentNode.removeChild(link)
			})
		</script>
	</body>

	</html>
	`))
}
