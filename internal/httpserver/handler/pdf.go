package handler

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"ubco-team15/omr/internal/httpserver/dto"
	"ubco-team15/omr/internal/pdf"
	"ubco-team15/omr/internal/scanner"

	"gocv.io/x/gocv"
)

// The maximum allowed size for a PDF file upload.
const maxPdfSize = 200 * 1024 * 1024     // 200 MB
const maxFileMemSize = 100 * 1024 * 1024 // 100 MB

func PostScanPdf(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		
		//
		// Read and validate the body
		//

		defer debug.FreeOSMemory()
		defer r.Body.Close()
		
		r.Body = http.MaxBytesReader(w, r.Body, maxPdfSize)
		if err := r.ParseMultipartForm(maxFileMemSize); err != nil {

			var mbe *http.MaxBytesError
			if errors.As(err, &mbe) {
				dto.SendError(
					w,
					http.StatusRequestEntityTooLarge,
					dto.ErrContentTooLarge,
					fmt.Sprintf(
						"the attached pdf is larger than %.1fMB",
						float64(maxPdfSize)/(1024.0*1024.0),
					),
				)
				return
			}

			dto.SendError(
				w,
				http.StatusBadRequest,
				dto.ErrInvalidRequest,
				"invalid multipart form",
			)
			return
		}
		defer r.MultipartForm.RemoveAll()

		pTemplId := r.FormValue("preprocessingTemplate")
		if pTemplId == "" {
			dto.SendError(
				w,
				http.StatusBadRequest,
				dto.ErrInvalidRequest,
				"preprocessingTemplate is required",
			)
			return
		}

		dpiStr := r.FormValue("dpi")
		density := 300
		if dpiStr != "" {
			dpi, err := strconv.Atoi(dpiStr)
			if err != nil || dpi <= 0 {
				dto.SendError(
					w,
					http.StatusBadRequest,
					dto.ErrInvalidRequest,
					"dpi must be a positive integer",
				)
				return
			}
			density = dpi
		}

		pdfFile, _, err := r.FormFile("pdf")
		if err != nil {
			dto.SendError(
				w,
				http.StatusBadRequest,
				dto.ErrInvalidRequest,
				err.Error(),
			)
			return
		}
		defer pdfFile.Close()

		//
		// Load the resources.
		//

		pTempl, err := s.LoadPreprocessingTemplate(pTemplId)
		if err != nil {
			dto.SendError(
				w,
				http.StatusNotFound,
				dto.ErrTemplateNotFound,
				err.Error(),
			)
			return
		}

		anchors, err := s.LoadAnchors(pTemplId)
		if err != nil {
			dto.SendError(
				w,
				http.StatusNotFound,
				dto.ErrMissingAnchor,
				err.Error(),
			)
			return
		}
		for i := range anchors {
			for j := range anchors[i] {
				defer anchors[i][j].Close()
			}
		}

		scannerTmpl, err := dto.AdaptScannerTemplate(pTempl, anchors)
		if err != nil {
			dto.SendError(
				w,
				http.StatusInternalServerError,
				dto.ErrInternal,
				err.Error(),
			)
			return
		}

		//
		// Start the PDF rendering.
		//

		pagesPerExam := len(pTempl.Pages)

		exams, nExams, err := pdf.RenderPageBatches(
			pdfFile,
			density,
			pagesPerExam,
			min(runtime.GOMAXPROCS(0), 8),
		)
		switch err {
		case nil:
			break
		case pdf.ErrMalformedPdf:
			dto.SendError(
				w,
				http.StatusBadRequest,
				dto.ErrMalformedPdf,
				err.Error(),
			)
			return
		case pdf.ErrPageCountMismatch:
			dto.SendError(
				w,
				http.StatusBadRequest,
				dto.ErrPageCountMismatch,
				err.Error(),
			)
			return
		default:
			dto.SendError(
				w,
				http.StatusInternalServerError,
				dto.ErrInternal,
				err.Error(),
			)
			return
		}

		// Clean up the incoming file resources. These calls have already been
		// deferred, but that won't lead to any panics (the standard library
		// generally keeps close operations idempotent).
		r.MultipartForm.RemoveAll()
		pdfFile.Close()
		r.Body.Close()

		//
		// Process the exam scans as they're rendered.
		//

		scanIds := make([]string, nExams)
		errorMsgs := make([]*dto.ScanError, nExams)

		wg := sync.WaitGroup{}
		for exam := range exams {
			idx := int(exam.Index)

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
				defer exam.Close()

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
				defer result.Close()

				pictures := make([]*gocv.Mat, pagesPerExam)
				binarized := make([]*gocv.Mat, pagesPerExam)
				for i := range result.Pages {
					binarized[i] = &result.Pages[i].Binary
					pictures[i] = &result.Pages[i].Picture
				}

				scanId, err := s.SaveScan(binarized, pictures, pTemplId)
				if err != nil {
					scanIds[idx] = ""
					errorMsgs[idx] = &dto.ScanError{
						From:  exam.From,
						Thru:  exam.Thru,
						Debug: err.Error(),
					}
				}

				scanIds[idx] = scanId
			})
		}
		wg.Wait()

		//
		// Tidy up and send the response.
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
