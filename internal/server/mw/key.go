package mw

import (
	"net/http"

	"github.com/Chad-Glazier/aperture-omr/internal/server/resources"
)

func GlobalKey(s resources.ServerResources) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			
			if authorized := s.CheckGlobalKey(r); !authorized {
				http.Error(w,
					"incorrect OMR-API-Key header",
					http.StatusUnauthorized,
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
