package handler

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"ubco-team15/omr/internal/httpserver/dto"
	"ubco-team15/omr/internal/pdf"
	"ubco-team15/omr/internal/scanner"

	"gocv.io/x/gocv"
)

// The maximum allowed size for a PDF file upload.
const maxPdfSize = 200 * 1024 * 1024 // 200 MB

func examErrorStr(err error, examIdx, pagesPerExam int) string {
	return fmt.Sprintf(
		"error in exam %d (pages %d through %d): %s",
		examIdx+1,
		examIdx*pagesPerExam+1,
		(examIdx+1)*pagesPerExam,
		err.Error(),
	)
}

func PostScanPdf(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		//
		// Read and validate the body
		//

		defer r.Body.Close()
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			dto.SendError(
				w,
				http.StatusBadRequest,
				dto.ErrInvalidRequest,
				"invalid multipart form",
			)
			return
		}

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

		pdfFile, header, err := r.FormFile("pdf")
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

		if header.Size > maxPdfSize {
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

		pagesPerExam := len(pTempl.Pages)

		exams, nExams, err := pdf.RenderPageBatches(
			pdfFile,
			density,
			pagesPerExam,
			min(runtime.GOMAXPROCS(0), 8),
		)
		if err != nil {
			if err == pdf.ErrMalformedPdf {
				dto.SendError(
					w,
					http.StatusBadRequest,
					dto.ErrMalformedPdf,
					err.Error(),
				)
				return
			}
			if err == pdf.ErrPageCountMismatch {
				dto.SendError(
					w,
					http.StatusBadRequest,
					dto.ErrPageCountMismatch,
					err.Error(),
				)
				return
			}
			dto.SendError(
				w,
				http.StatusInternalServerError,
				dto.ErrInternal,
				err.Error(),
			)
			return
		}
		pdfFile.Close() // We've also deferred these Close calls, but these
		r.Body.Close()  // types ignore redundant closes.

		scanIds := make([]string, nExams)
		errorMsgs := make([]string, nExams)

		wg := sync.WaitGroup{}
		for exam := range exams {
			idx := int(exam.Index)

			if exam.Error != nil {
				scanIds[idx] = ""
				errorMsgs[idx] = examErrorStr(exam.Error, idx, pagesPerExam)
				continue
			}

			wg.Go(func() {
				defer exam.Close()

				result, err := scanner.ScanExamMats(exam.Pages, scannerTmpl)

				if err != nil {
					scanIds[idx] = ""
					errorMsgs[idx] = examErrorStr(err, idx, pagesPerExam)
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
					errorMsgs[idx] = examErrorStr(err, idx, pagesPerExam)
				}

				scanIds[idx] = scanId
			})
		}
		wg.Wait()

		//
		// Send the response.
		//

		dto.SendJson(w, dto.NewScanResult(scanIds, errorMsgs))
	}
}
