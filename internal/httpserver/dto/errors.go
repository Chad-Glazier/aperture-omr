package dto

import (
	"encoding/json"
	"net/http"
)

type ErrReason string

const (
	ErrInvalidRequest    ErrReason = "invalid_request"
	ErrTemplateNotFound  ErrReason = "template_not_found"
	ErrScanNotFound      ErrReason = "scan_not_found"
	ErrPageCountMismatch ErrReason = "page_count_mismatch"
	ErrLowScanQuality    ErrReason = "low_scan_quality"
	ErrPageOutOfOrder    ErrReason = "page_out_of_order"
	ErrInternal          ErrReason = "internal_error"
	ErrMissingAnchor     ErrReason = "missing_anchor"
	ErrMalformedPdf      ErrReason = "malformed_pdf"
	ErrContentTooLarge   ErrReason = "content_too_large"
)

// Sends a JSON error response with a classifier (reason) so that the client
// can distinguish between different types of errors.
func SendError(
	w http.ResponseWriter,
	status int,
	reason ErrReason,
	message string,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	resp := map[string]string{
		// I'd like to change this to "reason" in the future.
		"code":    string(reason),
		"message": message,
	}
	if err := enc.Encode(resp); err != nil {
		panic(err)
	}
}

// Sends a JSON body.
func SendJson(w http.ResponseWriter, v any) {
	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "error writing response", http.StatusInternalServerError)
	}
}
