package dto

//
// Scan Results (Outgoing)
//

type ScanResult struct {
	ScanIds []string `json:"scanIds"`
	Errors  []string `json:"errors"`
}

// Makes a new ScanResult struct which omits any empty strings in either of the
// given slices.
func NewScanResult(scanIds, errors []string) ScanResult {

	ids := make([]string, 0, len(scanIds))
	errs := make([]string, 0, len(errors))

	for _, scanId := range scanIds {
		if scanId != "" {
			ids = append(ids, scanId)
		}
	}
	for _, err := range errors {
		if err != "" {
			errs = append(errs, err)
		}
	}

	return ScanResult{
		ScanIds: ids,
		Errors:  errs,
	}

}
