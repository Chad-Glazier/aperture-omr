package handler

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/Chad-Glazier/aperture-omr/internal/omr"
	"github.com/Chad-Glazier/aperture-omr/internal/pdf"
	"github.com/Chad-Glazier/aperture-omr/internal/scanner"
	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"
	"github.com/Chad-Glazier/aperture-omr/internal/server/mw"
	"github.com/Chad-Glazier/aperture-omr/internal/sys"

	"gocv.io/x/gocv"
)

var inUse = sync.Mutex{}

func PostScanPdf(s ServerResources) mw.JobHandlerFunc {
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

		body, ok := dto.ParseBodyFile(w, r, "application/pdf", 200<<20)
		if !ok {
			return
		}
		defer body.Close()

		//
		// Load the resources.
		//

		pTempl, err := s.LoadPreprocessingTemplate(q.PreprocessingTemplate)
		if err != nil {
			http.Error(w,
				"no template with ID "+q.PreprocessingTemplate+" was found",
				http.StatusNotFound,
			)
			return
		}

		anchors, err := s.LoadAnchors(q.PreprocessingTemplate)
		if err != nil {
			s.DeletePreprocessingTemplate(q.PreprocessingTemplate)
			http.Error(w,
				"template "+q.PreprocessingTemplate+" is corrupted",
				http.StatusInternalServerError,
			)
			return
		}
		defer omr.CloseAll2(anchors)

		scannerTmpl, err := dto.AdaptScannerTemplate(pTempl, anchors)
		if err != nil {
			panic(err)
		}

		//
		// Start the PDF rendering.
		//

		pagesPerExam := uint(len(pTempl.Pages))

		exams, nExams, err := pdf.RenderPageBlocks(
			body,
			q.Dpi,
			pagesPerExam,
			0,
		)
		switch err {
		case nil:
			break
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

		j.SetNotes(fmt.Sprintf("rendering %d pages", nExams*uint(pagesPerExam)))

		// Clean up the incoming file resources. These calls have already been
		// deferred, but that won't lead to any panics (the standard library
		// generally keeps close operations idempotent).
		r.MultipartForm.RemoveAll()
		body.Close()
		r.Body.Close()

		//
		// Process the exam scans as they're rendered.
		//

		scanIds := make([]string, nExams)
		errorMsgs := make([]*dto.ScanError, nExams)

		examsRendered := atomic.Uint32{}
		examIdx := atomic.Int32{}
		semaphore := make(chan struct{}, 4)

		wg := sync.WaitGroup{}
		for exam := range exams {
			idx := examIdx.Add(1) - 1

			if exam.Error != nil {
				scanIds[idx] = ""
				errorMsgs[idx] = &dto.ScanError{
					From:  exam.From,
					Thru:  exam.Thru,
					Debug: exam.Error.Error(),
				}
				continue
			}

			wg.Go(func() {
				semaphore <- struct{}{}
				defer exam.Close()
				defer func() {
					<-semaphore
					rendered := examsRendered.Add(1)
					j.SetProgress(float64(rendered)/float64(nExams))
				}()

				result, err := scanner.ScanExamMats(exam.Pages, scannerTmpl)
				if err != nil {
					scanIds[idx] = ""
					errorMsgs[idx] = &dto.ScanError{
						From:  exam.From,
						Thru:  exam.Thru,
						Debug: err.Error(),
					}
					return
				}
				exam.Close()
				defer result.Close()

				pictures := make([]gocv.Mat, pagesPerExam)
				binarized := make([]gocv.Mat, pagesPerExam)
				for i := range result.Pages {
					binarized[i] = result.Pages[i].Binary
					pictures[i] = result.Pages[i].Picture
				}

				scanId, err := s.SaveScan(
					binarized, 
					pictures, 
					q.PreprocessingTemplate,
				)
				if err != nil {
					scanIds[idx] = ""
					errorMsgs[idx] = &dto.ScanError{
						From:  exam.From,
						Thru:  exam.Thru,
						Debug: err.Error(),
					}
					return
				}

				scanIds[idx] = scanId
			})
		}
		wg.Wait()

		//
		// Send the response.
		//

		results := dto.NewScanResult(scanIds, errorMsgs)

		switch {
		case len(results.ScanIds) == 0:
			// The PDF was well-formed, but none of the scanned exams passed
			// preprocessing.
			w.WriteHeader(http.StatusUnprocessableEntity)
			dto.SendJson(w, results.Errors)
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

func DeleteScans(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		//
		// Validate the request
		//

		scanIds, ok := dto.ParseJsonBody[dto.ScanDeleteRequest](w, r, 1<<20)
		if !ok {
			return
		}

		//
		// Delete the scans and send back 200, whether or not the scans are
		// actually present.
		//

		for _, scanId := range scanIds {
			s.DeleteScan(scanId)
		}

		w.WriteHeader(http.StatusOK)

	}
}
