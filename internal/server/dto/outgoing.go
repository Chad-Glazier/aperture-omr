package dto

import (
	"github.com/Chad-Glazier/aperture-omr/internal/marker"
	"github.com/Chad-Glazier/aperture-omr/internal/sys"
)

//
// ScanResult
//

type ScanResult struct {
	ScanIds []string    `json:"scanIds"`
	Errors  []ScanError `json:"errors"`
}

type ScanError struct {
	// The starting page (1-based) of the PDF where the error occurred.
	From uint `json:"from"`
	// The ending page (1-based; inclusive) of the PDF where the error
	// occurred.
	Thru uint `json:"thru"`
	// The error message. This is meant for debugging, not for end-users.
	Debug string `json:"debug"`
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

//
// MarkingResult
//

type MarkingResult struct {
	PagesMarked int            `json:"pagesMarked"`
	TemplateId  string         `json:"templateId"`
	Scans       []Scan         `json:"scans"`
	Errors      []MarkingError `json:"errors"`
}

type MarkingError struct {
	ScanId string `json:"scanId"`
	// The error message. This is meant for debugging, not for end-users.
	Debug string `json:"debug"`
}

type Scan struct {
	ScanId string `json:"scanId"`
	Marks  []Mark `json:"marks"`
}

type Mark struct {
	QuestionId string   `json:"questionId"`
	Flagged    bool     `json:"flagged"`
	Selected   []string `json:"selected"`
	Confidence float64  `json:"confidence"`
}

func NewMarkingResult(
	templateId string,
	pagesPerScan int,
	scans []Scan,
	errs []*MarkingError,
) MarkingResult {
	successfulScans := make([]Scan, 0, len(scans))
	errors := make([]MarkingError, 0, len(errs))

	for i := range scans {
		if errs[i] != nil {
			errors = append(errors, *errs[i])
			continue
		}
		successfulScans = append(successfulScans, scans[i])
	}

	return MarkingResult{
		PagesMarked: len(successfulScans) * pagesPerScan,
		TemplateId:  templateId,
		Scans:       successfulScans,
		Errors:      errors,
	}
}

func AdaptMark(a *marker.Answer) Mark {

	var m Mark
	m.Flagged = a.Flag
	m.QuestionId = a.QuestionID
	m.Selected = a.Selected
	copy(m.Selected, a.Selected)
	m.Confidence = a.Confidence

	return m
}

//
// ResourceUtilization
//

type ResourceUtilization struct {
	CpuHistory    []sys.CpuInfo `json:"cpuHistory"`
	MemoryHistory []sys.MemInfo `json:"memoryHistory"`
	Disk          struct {
		Usage    sys.DiskInfo `json:"usage"`
		OmrUsage struct {
			Database         uint64 `json:"database"`
			NumberOfMatrices int    `json:"numberOfMatrices"`
			Matrices         uint64 `json:"matrices"`
			NumberOfPictures int    `json:"numberOfPictures"`
			Pictures         uint64 `json:"pictures"`
			Total            uint64 `json:"total"`
		} `json:"omrUsage"`
	} `json:"disk"`
	MemoryPeak uint64 `json:"memoryPeak"`
	Uptime     uint64 `json:"uptime"`
}

//
// CpuUsageSample
//

type CpuUsageSample struct {
	Overall   float64   `json:"overall"`
	PerThread []float64 `json:"perThread"`
}

//
// DetailedCpuInfo
//

type DetailedCpuInfo struct {
	Description  string           `json:"description"`
	FrequencyMhz float64          `json:"frequencyMhz"`
	MaxThreads   int              `json:"maxThreads"`
	UsageSamples []CpuUsageSample `json:"usageSamples"`
}

//
// DetailedMemoryInfo
//

type DetailedMemoryInfo struct {
	UsageSamples []sys.MemInfo `json:"usageSamples"`
}

//
// IdResponse
//

type IdResponse struct {
	Id string `json:"id"`
}
