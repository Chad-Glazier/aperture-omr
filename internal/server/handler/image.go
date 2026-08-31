package handler

import (
	"io"
	"net/http"

	"github.com/Chad-Glazier/aperture-omr/internal/fstore"
	"github.com/Chad-Glazier/aperture-omr/internal/server/dto"
	"github.com/Chad-Glazier/aperture-omr/internal/server/resources"
)

//
// Send an scan's page image.
//

func GetImage(s resources.ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		q, ok := dto.ParseQuery[dto.GetImageQuery](w, r)
		if !ok {
			return
		}

		img, err := s.OpenScanPicture(q.Scan, q.Page)
		if err != nil {
			http.Error(w,
				"scan page not found",
				http.StatusNotFound,
			)
			return
		}
		defer img.Close()

		w.Header().Add("Content-Type", fstore.ImgContentType)
		if _, err := io.Copy(w, img); err != nil {
			http.Error(w,
				"error writing image to response",
				http.StatusInternalServerError,
			)
			return
		}
	}
}
