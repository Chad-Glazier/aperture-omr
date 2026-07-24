package dto

//
// Scan Results (Outgoing)
//

type ScanError struct {
	// The starting page (1-based) of the PDF where the error occurred.
	From uint32 `json:"from"`
	// The ending page (1-based; inclusive) of the PDF where the error
	// occurred.
	Thru uint32 `json:"thru"`
	// The error message. This is meant for debugging, not for end-users.
	Debug string `json:"debug"`
}

type ScanResult struct {
	ScanIds []string    `json:"scanIds"`
	Errors  []ScanError `json:"errors"`
}

// Makes a new ScanResult struct. All zero-values from the given slices will
// be omitted.
func NewScanResult(scanIds []string, errors []*ScanError) ScanResult {

	ids := make([]string, 0, len(scanIds))
	errs := make([]ScanError, 0, len(errors))

	for _, scanId := range scanIds {
		if scanId != "" {
			ids = append(ids, scanId)
		}
	}
	for _, err := range errors {
		if err != nil {
			errs = append(errs, *err)
		}
	}

	return ScanResult{
		ScanIds: ids,
		Errors:  errs,
	}

}
