package handler

import (
	"io"
	"net/http"

	"ubco-team15/omr/internal/httpserver/dto"
)

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

		resp := make(map[string]string, 1)
		resp["templateId"] = id
		sendJson(w, resp)

	}
}
