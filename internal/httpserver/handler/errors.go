package handler

import (
	"encoding/json"
	"net/http"
)

// Error codes returned in JSON error bodies, letting callers distinguish a
// scan-quality problem (ask for a rescan) from a request or server problem.
const (
	ErrCodeInvalidRequest    = "invalid_request"
	ErrCodeTemplateNotFound  = "template_not_found"
	ErrCodeScanNotFound      = "scan_not_found"
	ErrCodePageCountMismatch = "page_count_mismatch"
	ErrCodeLowScanQuality    = "low_scan_quality"
	ErrCodeInternal          = "internal_error"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeError sends a structured JSON error body instead of the plain-text
// http.Error, so backend clients can classify failures (e.g. a skewed scan
// vs. a template config problem) instead of pattern-matching response text.
func writeError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorBody{Code: code, Message: message})
}
