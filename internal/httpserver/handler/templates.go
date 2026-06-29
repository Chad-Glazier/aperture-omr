package handler

import (
	"encoding/json"
	"net/http"
)

//
// Marking Templates
//

func PostMarkingTemplate(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		
		defer r.Body.Close()
		tmpl := &MarkingTemplate{}
		
		if err := json.NewDecoder(r.Body).Decode(tmpl); err != nil {
			http.Error(
				w, "error parsing body: " + err.Error(), http.StatusBadRequest,
			)
			return
		}
		if err := tmpl.Validate(); err != nil {
			http.Error(
				w, "error validating body: " + err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		id, err := s.SaveMarkingTemplate(tmpl)
		if err != nil {
			http.Error(
				w, "error saving template: " + err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		resp := make(map[string]string, 1)
		resp["templateId"] = id
		sendJson(w, resp)

	}
}
