package handler

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"ubco-team15/omr/internal/fs"
	"ubco-team15/omr/internal/scanner"

	"gocv.io/x/gocv"
)

//
// NOTE: Some parts of this function are kind of hacky. I plan to fix this by
// slightly refactoring the interface of the `scanner` package in a couple ways.
// Also, ignore the excessive image encoding/decoding. I'm going to fix those
// quirks later on.
//

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

		pageCount := len(template.Pages)
		anchors := make([][]gocv.Mat, pageCount)
		for pageIdx := range template.Pages {
			for anchorIdx := range template.Pages[pageIdx].Anchors {

				anchor, err := s.LoadAnchor(templateId, pageIdx, anchorIdx)
				if err != nil {
					http.Error(
						w,
						fmt.Sprintf(
							"error loading anchor page%danchor%d for %s",
							pageIdx, anchorIdx, templateId,
						),
						http.StatusInternalServerError,
					)
					return
				}

				buf := bytes.Buffer{}
				if err := fs.EncodeImg(&buf, anchor); err != nil {
					http.Error(
						w,
						fmt.Sprintf(
							"error loading anchor page%danchor%d for %s",
							pageIdx, anchorIdx, templateId,
						),
						http.StatusInternalServerError,
					)
					return
				}

				mat, err := gocv.IMDecode(buf.Bytes(), gocv.IMReadColor)
				if err != nil {
					http.Error(
						w,
						fmt.Sprintf(
							"error decoding anchor page%danchor%d for %s",
							pageIdx, anchorIdx, templateId,
						),
						http.StatusInternalServerError,
					)
					return
				}

				scanner.Binarize(&mat, &mat, &scanner.Config{
					BlurSize:            template.Config.BlurSize,
					MorphCloseSize:      template.Config.MorphCloseSize,
					MinAnchorConfidence: float32(template.Config.MinAnchorConfidence),
				})

				anchors[pageIdx] = append(anchors[pageIdx], mat)
			}
		}

		//
		// Read the page scans.
		//

		defer r.Body.Close()
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, "invalid multipart form", http.StatusBadRequest)
			return
		}

		pageScans := make([]io.Reader, pageCount)
		for pageIdx := range template.Pages {
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

		pageImages := make([]image.Image, pageCount)
		pageColorImages := make([]image.Image, pageCount)
		for i, data := range result {
			buf, err := gocv.IMEncode(gocv.PNGFileExt, data.Binary)
			if err != nil {
				http.Error(
					w,
					"error encoding image: "+err.Error(),
					http.StatusInternalServerError,
				)
				return
			}
			defer buf.Close()

			r := bytes.NewReader(buf.GetBytes())
			img, err := png.Decode(r)
			if err != nil {
				http.Error(
					w,
					"error encoding image: "+err.Error(),
					http.StatusInternalServerError,
				)
				return
			}

			pageImages[i] = img

			buf, err = gocv.IMEncode(gocv.PNGFileExt, data.Color)
			if err != nil {
				http.Error(
					w,
					"error encoding image: "+err.Error(),
					http.StatusInternalServerError,
				)
				return
			}
			defer buf.Close()

			r = bytes.NewReader(buf.GetBytes())
			img, err = png.Decode(r)
			if err != nil {
				http.Error(
					w,
					"error encoding image: "+err.Error(),
					http.StatusInternalServerError,
				)
				return
			}

			pageColorImages[i] = img
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
