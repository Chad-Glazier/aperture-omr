package handler

import (
	"fmt"
	"net/http"
	"ubco-team15/omr/internal/httpserver/dto"
	"ubco-team15/omr/internal/pdf"
	"ubco-team15/omr/internal/scanner"

	"gocv.io/x/gocv"
)

// The minimum size across any given dimension (length or width) for a PDF
// page.
const minDim = 100

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

		pages, err := pdf.RenderPageMats(pdfFile)
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
		default:
			dto.SendError(
				w,
				http.StatusInternalServerError,
				dto.ErrInternal,
				err.Error(),
			)
			return
		}
		for i := range pages {
			defer pages[i].Close()
		}

		if len(pages)%len(pTempl.Pages) != 0 {
			dto.SendError(
				w,
				http.StatusBadRequest,
				dto.ErrPageCountMismatch,
				fmt.Sprintf(
					"the PDF page count (%d) must be divisible by the "+
						"template's expected page count (%d)",
					len(pages), len(pTempl.Pages),
				),
			)
			return
		}

		for i := range pages {
			if pages[i].Rows() < minDim || pages[i].Cols() < minDim {
				dto.SendError(
					w,
					http.StatusBadRequest,
					dto.ErrMalformedPdf,
					fmt.Sprintf(
						"PDF pages must all be at least %d pixels wide "+
							"and tall",
						minDim,
					),
				)
				return
			}
		}

		//
		// Preprocess the scan.
		//

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

		results, err := scanner.ScanMats(pages, scannerTmpl)
		if err != nil {
			dto.SendError(
				w,
				http.StatusInternalServerError,
				dto.ErrInternal,
				"error during preprocessing: "+err.Error(),
			)
			return
		}
		for i := range results {
			defer results[i].Close()
		}

		//
		// Save the results.
		//

		savedScanIds := make([]string, len(results))

		for i := range results {

			pages := make([]*gocv.Mat, len(results[i].Pages))
			pagePictures := make([]*gocv.Mat, len(results[i].Pages))

			for j := range pages {
				pages[j] = &results[i].Pages[j].Binary
				pagePictures[j] = &results[i].Pages[j].Picture
			}

			id, err := s.SaveScan(pages, pagePictures, pTemplId)
			if err != nil {
				dto.SendError(
					w,
					http.StatusInternalServerError,
					dto.ErrInternal,
					"error saving preprocessed scans: "+err.Error(),
				)
				return
			}

			savedScanIds[i] = id

		}

		//
		// Send the response.
		//

		sendJson(w, savedScanIds)
	}
}
