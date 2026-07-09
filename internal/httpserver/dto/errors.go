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
		"code": string(reason),
		"message": message,
	}
	if err := enc.Encode(resp); err != nil {
		panic(err)
	}
}
