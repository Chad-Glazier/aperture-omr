package dto

import (
	"errors"

	"github.com/Chad-Glazier/aperture-omr/internal/omr"
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
	// May contain metadata about the scan that was attached during a
	// preprocessing pipeline.
	Metadata map[string]string `json:"metadata"`
	// The error message. This is meant for debugging, not for end-users.
	Debug string `json:"debug"`
}

func AdaptScanError(pages omr.PageSet) ScanError {
	err := pages.Error()
	if pages.Error() == nil {
		err = errors.New("unspecified error")
	}

	return ScanError{
		Metadata: pages.Metadata(),
		Debug:    err.Error(),
	}
}

// Creates a new [ScanResult] struct that omits all zero-valued elements of the
// input slices.
func NewScanResult(scanIds []string, errors []ScanError) ScanResult {
	
	filteredScanIds := make([]string, 0, len(scanIds))
	for _, s := range scanIds {
		if s != "" {
			filteredScanIds = append(filteredScanIds, s)
		}
	}

	filteredErrors := make([]ScanError, 0, len(errors))
	for _, e := range errors {
		if e.Metadata != nil || e.Debug != "" {
			filteredErrors = append(filteredErrors, e)
		}
	}

	return ScanResult{
		ScanIds: filteredScanIds,
		Errors: filteredErrors,
	}
}

//
// MarkingResult
//

type MarkResult struct {
	Results []ScanMarks    `json:"results"`
	Errors  []MarkingError `json:"errors"`
}

type ScanMarks struct {
	ScanId    string    `json:"scanId"`
	Questions Questions `json:"questions"`
}

// Question ID |-> marked bubbles
type Questions map[string]MarkedBubbles

// Marked bubble ID |-> confidence
type MarkedBubbles map[string]float64

type MarkingError struct {
	ScanId string `json:"scanId"`
	Debug  string `json:"debug"`
}

// Converts an [omr.MarkResult] struct to an equivalent [ScanMarks]
// representation.
func AdaptScanMarks(scanId string, marks omr.MarkResult) ScanMarks {
	var out ScanMarks

	out.ScanId = scanId
	out.Questions = make(Questions)
	for _, p := range marks.Pages {
		for _, q := range p.Questions {
			out.Questions[q.Id] = make(MarkedBubbles)
			for _, b := range q.SelectedBubbles {
				out.Questions[q.Id][b.Id] = b.Confidence
			}
		}
	}

	return out
}

func AdaptMarkingError(scanId string, err error) MarkingError {
	return MarkingError{
		ScanId: scanId,
		Debug:  err.Error(),
	}
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
