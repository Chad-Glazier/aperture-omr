package handler

import (
	"encoding/json"
	"fmt"
	"image"
	"net/http"
	"ubco-team15/omr/internal/fs"
)

// An uploaded preprocessing template.
// TODO: Settle on a template shape.
type PreprocessTemplateDto struct {
	Width   int `json:"width"`
	Height  int `json:"height"`
	Anchors []struct {
		Center struct {
			X int `json:"x"`
			Y int `json:"y"`
		} `json:"center"`
		ROI struct {
			Min struct {
				X int `json:"x"`
				Y int `json:"y"`
			} `json:"min"`
			Max struct {
				X int `json:"x"`
				Y int `json:"y"`
			} `json:"max"`
		} `json:"roi"`
	} `json:"anchors"`
	Config struct {
		BlurSize            int     `json:"blurSize"`
		MorphCloseSize      int     `json:"morphCloseSize"`
		MinAnchorConfidence float64 `json:"minAnchorConfidence"`
	} `json:"config"`
}

func PostPreprocessTemplate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	anchors := make([]image.Image, 3)
	for i := range 3 {
		name := fmt.Sprintf("anchor%d", i)
		file, _, err := r.FormFile(name)
		if err != nil {
			http.Error(w, name+" is required", http.StatusBadRequest)
			return
		}
		defer file.Close()

		img, err := fs.DecodeImg(file)
		if err != nil {
			http.Error(
				w,
				fmt.Sprintf(
					"failure decoding %s; ensure its type is %s",
					name, fs.ImgContentType,
				),
				http.StatusBadRequest,
			)
			return
		}

		anchors[i] = img
	}

	templateJson := r.FormValue("template")
	if templateJson == "" {
		http.Error(w, "template field required", http.StatusBadRequest)
		return
	}

	template := PreprocessTemplateDto{}
	if err := json.Unmarshal([]byte(templateJson), &template); err != nil {
		http.Error(
			w,
			"error parsing template JSON data, "+
				"ensure it matches the specified format",
			http.StatusBadRequest,
		)
		return
	}

	// TODO: Validate/store the template and its anchors.
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
