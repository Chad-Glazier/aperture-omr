package handler

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/Chad-Glazier/aperture-omr/internal/omr"
	"github.com/Chad-Glazier/aperture-omr/internal/pdf"
	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"
	"github.com/Chad-Glazier/aperture-omr/internal/server/mw"
	"github.com/Chad-Glazier/aperture-omr/internal/server/resources"
	"github.com/Chad-Glazier/aperture-omr/internal/sys"
)

const (
	MaxSizeScan = 200 << 20 // 200 MB
)

var inUse = sync.Mutex{}

func PostScanPdf(s resources.ServerResources) mw.JobHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, j mw.JobResources) {

		//
		// We only permit one PDF operation at a time.
		//

		inUse.Lock()
		defer inUse.Unlock()
		defer sys.Tidy()

		//
		// Read and validate the body
		//

		q, ok := dto.ParseQuery[dto.PostScanPdfQuery](w, r)
		if !ok {
			return
		}

		body, ok := dto.ParseBodyFile(w, r, "application/pdf", MaxSizeScan)
		if !ok {
			return
		}
		defer body.Close()

		//
		// Load the resources.
		//

		tmpl, err := s.LoadPreprocessingTemplate(q.PreprocessTemplate)
		if err != nil {
			http.Error(w,
				"no template with ID "+q.PreprocessTemplate+" was found",
				http.StatusNotFound,
			)
			return
		}
		defer tmpl.Close()

		//
		// Start the PDF rendering.
		//

		exams, nExams, err := pdf.RenderPageBlocks(
			body,
			q.Dpi,
			tmpl.PageCount(),
			0,
		)
		switch err {
		case nil:
		case pdf.ErrMalformedPdf:
			http.Error(w,
				"the given PDF is malformed",
				http.StatusBadRequest,
			)
			return
		case pdf.ErrPageCountMismatch:
			http.Error(w,
				"the number of pages in the PDF is incompatible with the "+
					"number of pages in the template",
				http.StatusBadRequest,
			)
			return
		default:
			http.Error(w,
				"unknown error while configuring the render operation",
				http.StatusInternalServerError,
			)
			return
		}

		j.SetNotes(fmt.Sprintf("preprocessing %d exams", nExams))

		//
		// Process the exam scans as they're rendered.
		//

		preprocessed, err := omr.PreprocessStream(tmpl, 4, exams)
		if err != nil {
			http.Error(w,
				"error calibrating the preprocessing pipeline. It's likely "+
					"that the first exam in the PDF was malformed",
				http.StatusBadRequest,
			)
			return
		}

		scanIds := make([]string, nExams)
		errors := make([]dto.ScanError, nExams)
		examsRendered := atomic.Int32{}
		for exam := range preprocessed {
			idx := examsRendered.Add(1) - 1
			j.SetProgress(float64(idx+1) / float64(nExams))

			if exam.Error() != nil {
				errors[idx] = dto.AdaptScanError(exam)
				continue
			}

			id, err := s.SaveScan(exam.Pages(), q.PreprocessTemplate)
			if err != nil {
				errors[idx] = dto.ScanError{
					Metadata: exam.Metadata(),
					Debug:    err.Error(),
				}
				continue
			}
			scanIds[idx] = id
		}

		j.SetProgress(1.0)
		j.SetNotes(fmt.Sprintf("preprocessed %d exams", nExams))

		//
		// Send the response.
		//

		results := dto.NewScanResult(scanIds, errors)

		switch {
		case len(results.ScanIds) == 0:
			// The PDF was well-formed, but none of the scanned exams passed
			// preprocessing.
			w.WriteHeader(http.StatusUnprocessableEntity)
			dto.SendJson(w, results)
			return
		case len(results.Errors) == 0:
			// All scans were successfully preprocessed.
			dto.SendJson(w, results)
		case len(results.Errors) != 0:
			// Some exams were preprocessed, others failed. We treat this the
			// same as the full-success case for now.
			dto.SendJson(w, results)
		}
	}
}

func DeleteScan(s resources.ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		q, ok := dto.ParseQuery[dto.IdQuery](w, r)
		if !ok {
			return
		}

		s.DeleteScan(q.Id)
		w.WriteHeader(http.StatusOK)
	}
}
