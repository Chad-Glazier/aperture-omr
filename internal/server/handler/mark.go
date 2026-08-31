package handler

import (
	"fmt"
	"net/http"

	"github.com/Chad-Glazier/aperture-omr/internal/omr"
	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"
	"github.com/Chad-Glazier/aperture-omr/internal/server/mw"
	"github.com/Chad-Glazier/aperture-omr/internal/server/resources"
	"github.com/Chad-Glazier/aperture-omr/internal/sys"
)

const (
	MaxSizeMarksRequest = 1 << 20 // 1 MB
)

func RequestMarks(s resources.ServerResources) mw.JobHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, j mw.JobResources) {

		defer sys.Tidy()
		defer r.Body.Close()

		//
		// Parse the request.
		//

		body, ok := dto.ParseJsonBody[dto.MarkingJobRequest](
			w, r, MaxSizeMarksRequest,
		)
		if !ok {
			return
		}

		//
		// Fetch the required resources.
		//

		tmpl, err := s.LoadMarkingTemplate(body.TemplateId)
		if err != nil {
			http.Error(w,
				"template ID not recognized",
				http.StatusNotFound,
			)
			return
		}

		//
		// We stream the scans and mark them sequentially. Marking is quite 
		// fast; parallelization would induce more overhead than it's worth.
		//

		j.SetNotes(fmt.Sprintf("marking %d exams", len(body.ScanIds)))

		var out dto.MarkResult

		for i, scanId := range body.ScanIds {
			pages, err := s.LoadScan(scanId)
			if err != nil {
				markErr := dto.AdaptMarkingError(scanId, err)
				out.Errors = append(out.Errors, markErr)
				continue
			}

			marks, err := omr.Mark(tmpl, pages)
			if err != nil {
				markErr := dto.AdaptMarkingError(scanId, err)
				out.Errors = append(out.Errors, markErr)
				continue
			}

			result := dto.AdaptScanMarks(scanId, marks)
			out.Results = append(out.Results, result)

			j.SetProgress(float64(i+1)/float64(len(body.ScanIds)))
		}

		j.SetProgress(1.0)
		j.SetNotes(fmt.Sprintf(
			"marked %d/%d exams", 
			len(out.Results), 
			len(body.ScanIds),
		))

		switch {
		case len(out.Results) == 0:
			// Every scan in the batch failed to mark.
			w.WriteHeader(http.StatusUnprocessableEntity)
			dto.SendCompressedJson(w, r, out)
		default:
			dto.SendCompressedJson(w, r, out)
		}
	}
}
