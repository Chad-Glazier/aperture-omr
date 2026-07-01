package dto

import (
	"encoding/json"
	"fmt"
)

//
// MarkingJobRequest (Incoming)
//

// Parses and validates a marking job request from JSON text.
func ParseMarkingJobRequest(jsonBuf []byte) (*MarkingJobRequest, error) {
	v := &MarkingJobRequest{}
	if err := json.Unmarshal(jsonBuf, v); err != nil {
		return nil, err
	}
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return v, nil
}

type MarkingJobRequest struct {
	TemplateId string   `json:"template"`
	ScanIds    []string `json:"scans"`
}

func (m *MarkingJobRequest) Validate() error {

	switch {
	case m.TemplateId == "":
		return fmt.Errorf("template field must be provided")
	case m.ScanIds == nil:
		return fmt.Errorf("scans must be provided")
	case len(m.ScanIds) == 0:
		return fmt.Errorf("scans must not be empty")
	}

	return nil
}

//
// MarkingResult (Outgoing)
//

type MarkingResult struct {
	PerformanceMetrics PerformanceMetrics `json:"performanceMetrics"`
	PagesMarked        int                `json:"pagesMarked"`
	TemplateId         string             `json:"templateId"`
	Scans              []Scan             `json:"scans"`
	Errors             []string           `json:"errors,omitempty"`
}

type Scan struct {
	ScanId string `json:"scanId"`
	Marks  []Mark `json:"marks"`
}

type Mark struct {
	QuestionId string   `json:"questionId"`
	Flagged    bool     `json:"flagged"`
	Selected   []string `json:"selected"`
}

type PerformanceMetrics struct {
	StartTime int64 `json:"startTime"`
	EndTime   int64 `json:"endTime"`
	DiskTime  int64 `json:"diskTime"`
	OMRTime   int64 `json:"omrTime"`
}
