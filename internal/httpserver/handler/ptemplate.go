package handler

import (
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"ubco-team15/omr/internal/httpserver/dto"
)

const maxUploadSize = 30 * 1024 * 1024 // 30 MB

func decodeImg(r io.Reader) (image.Image, error) {
	return jpeg.Decode(r)
}

const imgType = "image/jpeg"

func PostPreprocessingTemplate(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		defer r.Body.Close()
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, "invalid multipart form", http.StatusBadRequest)
			return
		}

		jsonBuf := r.FormValue("template")
		tmpl, err := dto.ParsePreprocessingTemplate([]byte(jsonBuf))
		if err != nil {
			http.Error(
				w,
				"error parsing template: "+err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		templateId, err := s.SavePreprocessingTemplate(tmpl)
		if err != nil {
			http.Error(
				w, "error saving template: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		for i := range tmpl.Pages {
			for j := range tmpl.Pages[i].Anchors {
				fileKey := fmt.Sprintf("page%danchor%d", i, j)
				f, _, err := r.FormFile(fileKey)
				if err != nil {
					http.Error(
						w,
						"expected anchor image file "+fileKey,
						http.StatusBadRequest,
					)
					return
				}
				defer f.Close()

				img, err := decodeImg(f)
				if err != nil {
					http.Error(
						w,
						fileKey+
							" should be in "+
							imgType+
							" format",
						http.StatusBadRequest,
					)
					return
				}

				if err := s.SaveAnchor(img, templateId, i, j); err != nil {
					http.Error(
						w,
						fmt.Sprintf(
							"error saving anchor image page%danchor%d",
							i, j,
						),
						http.StatusInternalServerError,
					)
					return
				}
			}
		}

		resp := make(map[string]string, 1)
		resp["templateId"] = templateId
		sendJson(w, resp)
	}
}
