package handler

import "net/http"

func GetSnippet(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		//
		// Parse the request information.
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
		scanId := r.URL.Query().Get("scan")
		if scanId == "" {
			http.Error(
				w,
				"scan query parameter is missing",
				http.StatusBadRequest,
			)
			return
		}
		questionId := r.URL.Query().Get("question")
		if questionId == "" {
			http.Error(
				w,
				"question query parameter is missing",
				http.StatusBadRequest,
			)
			return
		}

		//
		// Get the snippet.
		//

		img, err := s.LoadSnippet(scanId, templateId, questionId)
		if err != nil {
			http.Error(
				w,
				"error retrieving image: "+err.Error(),
				http.StatusNotFound,
			)
			return
		}

		//
		// Send the snippet.
		//

		w.Header().Add("Content-Type", imgType)
		if err := encodeImg(w, img); err != nil {
			http.Error(
				w,
				"error writing image to response: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}
	}
}
