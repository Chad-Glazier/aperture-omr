package mw

import (
	"crypto/sha512"
	"net/http"
)

type KeyHolder interface {
	// Returns true if and only if the request's headers include the proper
	// administrator key.
	CheckAdminKey(r *http.Request) bool
	// Sets the admin key.
	SetAdminKey(key string)
	// Returns true if and only if the request's headers include the proper
	// global key.
	CheckGlobalKey(r *http.Request) bool
	// Sets the global API key.
	SetGlobalKey(key string)
}

func GlobalKey(s KeyHolder) func(next http.Handler) http.Handler {
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

func AdminKey(s KeyHolder) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			
			if authorized := s.CheckAdminKey(r); !authorized {
				http.Error(w,
					"incorrect OMR-Admin-Key header",
					http.StatusUnauthorized,
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

//
// Keyholder Implementation
//

type keyholder struct {
	adminKey       [64]byte
	globalKey      [64]byte
	globalKeyIsSet bool
}

func NewKeyholder() KeyHolder {
	return &keyholder{}
}

func (h *keyholder) SetAdminKey(key string) {
	h.adminKey = sha512.Sum512([]byte(key))
}

func (h *keyholder) CheckAdminKey(r *http.Request) bool {
	k := r.Header.Get("OMR-Admin-Key")
	if k == "" {
		return false
	}

	return h.adminKey == sha512.Sum512([]byte(k))
}

func (h *keyholder) SetGlobalKey(key string) {
	h.globalKey = sha512.Sum512([]byte(key))
	h.globalKeyIsSet = true
}

func (h *keyholder) CheckGlobalKey(r *http.Request) bool {

	if !h.globalKeyIsSet {
		return true
	}

	k := r.Header.Get("OMR-API-Key")
	if k == "" {
		return false
	}

	return h.globalKey == sha512.Sum512([]byte(k))
}
