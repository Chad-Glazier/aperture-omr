package handler

import (
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"

	"ubco-team15/omr/internal/httpserver/dto"
)

const maxUploadSize = 32 * 1024 * 1024 // 32 MB

func decodeImg(r io.Reader) (image.Image, error) {
	return jpeg.Decode(r)
}

func encodeImg(w io.Writer, img image.Image) error {
	return jpeg.Encode(w, img, nil)
}

const imgType = "image/jpeg"

func sendJson(w http.ResponseWriter, v any) {
	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "error writing response", http.StatusInternalServerError)
	}
}

//
// Marking templates
//

func PostMarkingTemplate(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		defer r.Body.Close()
		jsonBuf, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(
				w, "error reading body: "+err.Error(), http.StatusBadRequest,
			)
			return
		}

		tmpl, err := dto.ParseMarkingTemplate(jsonBuf)
		if err != nil {
			http.Error(
				w, "error parsing body: "+err.Error(), http.StatusBadRequest,
			)
			return
		}

		id, err := s.SaveMarkingTemplate(tmpl)
		if err != nil {
			http.Error(
				w, "error saving template: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		sendJson(w, map[string]string{
			"templateId": id,
		})
	}
}

//
// Preprocessing templates
//

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

		// Note: It's possible for the template to be saved and then later have
		// the anchors fail. This isn't a major bug (it won't cause runtime
		// errors), but it should be fixed later.

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

		sendJson(w, map[string]string{
			"templateId": templateId,
		})
	}
}
