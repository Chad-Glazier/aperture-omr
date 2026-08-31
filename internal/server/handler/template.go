package handler

import (
	"net/http"

	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"
	"github.com/Chad-Glazier/aperture-omr/internal/server/resources"
)

const (
	MaxSizeMarkingTemplate       = 20 << 20 // 20 MB
	MaxSizePreprocessingTemplate = 20 << 20 // 20 MB
)

//
// Marking templates
//

func PostMarkingTemplate(s resources.ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		
		tmpl, ok := dto.ParseJsonBody[dto.MarkTemplate](
			w, r, MaxSizeMarkingTemplate,
		)
		if !ok {
			return
		}

		id, err := s.SaveMarkingTemplate(tmpl.Adapt())
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

func DeleteMarkingTemplate(s resources.ServerResources) http.HandlerFunc {
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

func PostPreprocessingTemplate(s resources.ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, ok := dto.ParseJsonBody[dto.PreprocessTemplate](
			w, r, MaxSizePreprocessingTemplate,
		)
		if !ok {
			return
		}

		omrTmpl, err := tmpl.Adapt()
		if err != nil {
			http.Error(
				w, "error encoding template: "+err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		templateId, err := s.SavePreprocessingTemplate(omrTmpl)
		if err != nil {
			http.Error(
				w, "error saving template: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		dto.SendJson(w, dto.IdResponse{Id: templateId})
	}
}

func DeletePreprocessingTemplate(s resources.ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q, ok := dto.ParseQuery[dto.IdQuery](w, r)
		if !ok {
			return
		}

		s.DeletePreprocessingTemplate(q.Id)
		w.WriteHeader(http.StatusOK)
	}
}
