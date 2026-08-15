/*
This package implements the Data Transfer Objects. That is, the structured data
we expect to send to/from the client. For each of these types, we include
functions for serialization, deserialization, and validation where appropriate.
*/
package dto

import (
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"strings"
)

// Sends a JSON body.
func SendJson(w http.ResponseWriter, v any) {
	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "error writing response", http.StatusInternalServerError)
	}
}

// If the request header indicates that it can handle compressed data in one of
// the recognized formats, then we will send the given data as JSON in that
// format. Otherwise, the uncompressed JSON will be sent.
//
// At the time of writing, supported formats include gzip and deflate.
func SendCompressedJson(w http.ResponseWriter, r *http.Request, v any) error {
	w.Header().Add("Content-Type", "application/json")

	switch {
	case strings.Contains(r.Header.Get("Accept-Encoding"), "deflate"):
		w.Header().Add("Content-Encoding", "deflate")

		compressor, _ := flate.NewWriter(w, 1)
		defer compressor.Close()

		encoder := json.NewEncoder(compressor)
		return encoder.Encode(v)

	case strings.Contains(r.Header.Get("Accept-Encoding"), "gzip"):
		w.Header().Add("Content-Encoding", "gzip")

		compressor := gzip.NewWriter(w)
		defer compressor.Close()

		encoder := json.NewEncoder(compressor)
		return encoder.Encode(v)

	default:
		encoder := json.NewEncoder(w)
		return encoder.Encode(v)
	}
}
