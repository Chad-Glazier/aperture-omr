package handler

import (
	"fmt"
	"image"
	"io"
	"net/http"
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
			http.Error(
				w,
				"template query parameter is missing",
				http.StatusBadRequest,
			)
			return
		}
		template, err := s.LoadPreprocessingTemplate(templateId)
		if err != nil {
			http.Error(
				w,
				"template "+templateId+" not found",
				http.StatusNotFound,
			)
			return
		}

		//
		// Load the template's anchors.
		//

		anchors, err := s.LoadAnchors(templateId)
		if err != nil {
			http.Error(
				w,
				"error retrieving anchors: "+err.Error(),
				http.StatusNotFound,
			)
		}

		//
		// Read the page scans.
		//

		defer r.Body.Close()
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, "invalid multipart form", http.StatusBadRequest)
			return
		}

		pageCount := len(template.Pages)
		pageScans := make([]io.Reader, pageCount)

		for pageIdx := range pageCount {
			f, _, err := r.FormFile(fmt.Sprintf("page%d", pageIdx))
			if err != nil {
				http.Error(
					w,
					fmt.Sprintf(
						"expected page%d on the request "+
							"(the preprocessing template has %d pages)",
						pageIdx, pageCount,
					),
					http.StatusBadRequest,
				)
				return
			}
			defer f.Close()

			pageScans[pageIdx] = f
		}

		//
		// Below, we convert the data into a format that the scanner package
		// understands.
		//

		pages := make([]scanner.ScanPage, pageCount)
		for pageIdx := range pageCount {
			nAnchors := len(template.Pages[pageIdx].Anchors)
			pages[pageIdx].Anchors = make([]scanner.Anchor, nAnchors)
			for anchorIdx := range nAnchors {
				pages[pageIdx].Anchors[anchorIdx] = scanner.Anchor{
					Image: anchors[pageIdx][anchorIdx],
					ROI: image.Rectangle{
						Min: image.Point{
							X: template.Pages[pageIdx].Anchors[anchorIdx].Roi.Min.X,
							Y: template.Pages[pageIdx].Anchors[anchorIdx].Roi.Min.Y,
						},
						Max: image.Point{
							X: template.Pages[pageIdx].Anchors[anchorIdx].Roi.Max.X,
							Y: template.Pages[pageIdx].Anchors[anchorIdx].Roi.Max.Y,
						},
					},
					Center: image.Point{
						X: template.Pages[pageIdx].Anchors[anchorIdx].Center.X,
						Y: template.Pages[pageIdx].Anchors[anchorIdx].Center.Y,
					},
				}
			}
		}

		//
		// Preprocess the scan.
		//

		tmpl := &scanner.Template{
			Width:  template.Width,
			Height: template.Height,
			Config: scanner.Config{
				BlurSize:            template.Config.BlurSize,
				MorphCloseSize:      template.Config.MorphCloseSize,
				MinAnchorConfidence: float32(template.Config.MinAnchorConfidence),
				AdaptiveBlockSize:   template.Config.AdaptiveBlockSize,
				AdaptiveC:           float32(template.Config.AdaptiveC),
			},
			Pages: pages,
		}
		defer tmpl.Close()

		result, err := scanner.Scan(pageScans, tmpl)
		if err != nil {
			http.Error(
				w,
				"error during preprocessing: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		//
		// Save the results.
		//

		pageImages := make([]*gocv.Mat, pageCount)
		pageColorImages := make([]*gocv.Mat, pageCount)
		for i, data := range result {
			pageImages[i] = &data.Binary
			pageColorImages[i] = &data.Color
			defer data.Close()
		}

		id, err := s.SaveScan(pageImages, pageColorImages, templateId)
		if err != nil {
			http.Error(
				w,
				"error saving preprocessed scans: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		//
		// Send the response.
		//

		sendJson(w, map[string]string{
			"scanId": id,
		})
	}
}
