package handler

import (
	"encoding/base64"
	"net/http"

	"github.com/Chad-Glazier/aperture-omr/internal/scanner"
	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"

	"gocv.io/x/gocv"
)

const (
	MaxSizeMarkingTemplate       = 20 << 20 // 20 MB
	MaxSizePreprocessingTemplate = 20 << 20 // 20 MB
)

//
// Marking templates
//

func PostMarkingTemplate(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		tmpl, ok := dto.ParseJsonBody[*dto.MarkingTemplate](
			w, r, MaxSizeMarkingTemplate,
		)
		if !ok {
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

		dto.SendJson(w, dto.IdResponse{Id: id})
	}
}

func DeleteMarkingTemplate(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		q, ok := dto.ParseQuery[dto.IdQuery](w, r)
		if !ok {
			return
		}

		s.DeleteMarkingTemplate(q.Id)

		w.WriteHeader(http.StatusOK)

	}
}

//
// Preprocessing templates
//

func PostPreprocessingTemplate(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		tmpl, ok := dto.ParseJsonBody[*dto.PreprocessingTemplate](
			w, r, MaxSizePreprocessingTemplate,
		)
		if !ok {
			return
		}

		//
		// The anchor images are base64-encoded in the JSON request body. We
		// will decode them and convert them to preprocessed matrices before
		// storing them. This means that we do not need to keep their base64
		// versions inside of the JSON body when we store it.
		//

		anchorBase64 := make([][]string, len(tmpl.Pages))
		for i := range anchorBase64 {
			anchorBase64[i] = make([]string, len(tmpl.Pages[i].Anchors))
			for j := range anchorBase64[i] {
				anchorBase64[i][j] = tmpl.Pages[i].Anchors[j].Image
				tmpl.Pages[i].Anchors[j].Image = ""
			}
		}

		templateId, err := s.SavePreprocessingTemplate(tmpl)
		if err != nil {
			http.Error(
				w, "error saving template: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		binarizeConf := scanner.Config{
			BlurSize:            tmpl.Config.BlurSize,
			MorphCloseSize:      tmpl.Config.MorphCloseSize,
			MinAnchorConfidence: float32(tmpl.Config.MinAnchorConfidence),
			AdaptiveBlockSize:   tmpl.Config.AdaptiveBlockSize,
			AdaptiveC:           float32(tmpl.Config.AdaptiveC),
		}

		for i := range tmpl.Pages {
			for j := range tmpl.Pages[i].Anchors {

				buf, err := base64.StdEncoding.DecodeString(anchorBase64[i][j])
				if err != nil {
					http.Error(w,
						"error decoding base64 anchor image",
						http.StatusBadRequest,
					)
					s.DeletePreprocessingTemplate(templateId)
					return
				}

				mat, err := gocv.IMDecode(buf, gocv.IMReadGrayScale)
				if err != nil {
					http.Error(w,
						"anchor image format not recognized",
						http.StatusBadRequest,
					)
					s.DeletePreprocessingTemplate(templateId)
					return
				}
				defer mat.Close()

				err = scanner.Binarize(&mat, &mat, binarizeConf)
				if err != nil {
					http.Error(w,
						"error preprocessing anchor",
						http.StatusInternalServerError,
					)
					s.DeletePreprocessingTemplate(templateId)
					return
				}

				if err := s.SaveAnchor(mat, templateId, i, j); err != nil {
					http.Error(w,
						"error storing anchor image",
						http.StatusInternalServerError,
					)
					s.DeletePreprocessingTemplate(templateId)
					return
				}
			}
		}

		dto.SendJson(w, dto.IdResponse{Id: templateId})
	}
}

func DeletePreprocessingTemplate(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(
				w,
				"id query parameter is missing",
				http.StatusBadRequest,
			)
			return
		}

		s.DeletePreprocessingTemplate(id)

		w.WriteHeader(http.StatusOK)

	}
}
