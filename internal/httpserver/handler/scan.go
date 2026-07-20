package handler

import (
	"fmt"
	"io"
	"net/http"
	"ubco-team15/omr/internal/httpserver/dto"
	"ubco-team15/omr/internal/scanner"

	"gocv.io/x/gocv"
)

func PostScan(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		//
		// Load the template.
		//

		templateId := r.URL.Query().Get("template")
		if templateId == "" {
			dto.SendError(
				w,
				http.StatusBadRequest,
				dto.ErrInvalidRequest,
				"template query parameter is missing",
			)
			return
		}

		template, err := s.LoadPreprocessingTemplate(templateId)
		if err != nil {
			dto.SendError(
				w,
				http.StatusNotFound,
				dto.ErrTemplateNotFound,
				"template "+templateId+" not found",
			)
			return
		}

		//
		// Load the template's anchors.
		//

		anchors, err := s.LoadAnchors(templateId)
		if err != nil {
			dto.SendError(
				w,
				http.StatusNotFound,
				dto.ErrMissingAnchor,
				err.Error(),
			)
			return
		}

		//
		// Read the page scans.
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

		pageCount := len(template.Pages)
		pageScans := make([]io.Reader, pageCount)

		for pageIdx := range pageCount {
			f, _, err := r.FormFile(fmt.Sprintf("page%d", pageIdx))
			if err != nil {
				dto.SendError(
					w,
					http.StatusBadRequest,
					dto.ErrPageCountMismatch,
					fmt.Sprintf(
						"expected page%d on the request "+
							"(the preprocessing template has %d pages)",
						pageIdx, pageCount,
					),
				)
				return
			}
			defer f.Close()

			pageScans[pageIdx] = f
		}

		//
		// Preprocess the scan.
		//

		tmpl, err := dto.AdaptScannerTemplate(template, anchors)
		if err != nil {
			dto.SendError(
				w,
				http.StatusInternalServerError,
				dto.ErrInternal,
				err.Error(),
			)
			return
		}
		defer tmpl.Close()

		result, err := scanner.Scan(pageScans, tmpl)
		if err != nil {
			dto.SendError(
				w,
				http.StatusInternalServerError,
				dto.ErrInternal,
				"error during preprocessing: "+err.Error(),
			)
			return
		}

		//
		// Save the results.
		//

		pageImages := make([]*gocv.Mat, pageCount)
		pagePictures := make([]*gocv.Mat, pageCount)
		for i, data := range result {
			pageImages[i] = &data.Binary
			pagePictures[i] = &data.Picture
			defer data.Close()
		}

		id, err := s.SaveScan(pageImages, pagePictures, templateId)
		if err != nil {
			dto.SendError(
				w,
				http.StatusInternalServerError,
				dto.ErrInternal,
				"error saving preprocessed scans: "+err.Error(),
			)
			return
		}

		//
		// Send the response.
		//

		dto.SendJson(w, map[string]string{
			"scanId": id,
		})
	}
}
