package handler

import (
	"io"
	"net/http"
	"strconv"
	"ubco-team15/omr/internal/fs"
)

func GetImage(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		scanId := r.URL.Query().Get("scan")
		if scanId == "" {
			http.Error(
				w,
				"scan query parameter is missing",
				http.StatusBadRequest,
			)
			return
		}
		pageParam := r.URL.Query().Get("page")
		if pageParam == "" {
			http.Error(
				w,
				"page query parameter is missing",
				http.StatusBadRequest,
			)
			return
		}
		pageIdx, err := strconv.Atoi(pageParam)
		if err != nil {
			http.Error(
				w,
				"page query parameter must be an integer",
				http.StatusBadRequest,
			)
			return
		}

		img, err := s.OpenScanPicture(scanId, pageIdx)
		if err != nil {
			http.Error(
				w,
				"error retrieving scan image: "+err.Error(),
				http.StatusNotFound,
			)
			return
		}
		defer img.Close()

		w.Header().Add("Content-Type", fs.ImgContentType)
		if _, err := io.Copy(w, img); err != nil {
			http.Error(
				w,
				"error writing image to response: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}
	}
}
