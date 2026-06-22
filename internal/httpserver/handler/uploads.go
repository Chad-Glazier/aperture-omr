package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"ubco-team15/omr/config"
	"ubco-team15/omr/internal/fs"
)

const maxUploadSize = 30 * 1024 * 1024 // 30 MB

func getStore() (fs.Store, error) {
	if config.TestMode() {
		return fs.NewLocalStore("data/test_uploads"), nil
	} else {
		return fs.NewS3Store()
	}
}

// Handles a request to upload an image.
func PostUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "image is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, err := fs.DecodeImg(file)
	if err != nil {
		http.Error(
			w,
			"failure decoding image; ensure its type is "+fs.ImgContentType,
			http.StatusBadRequest,
		)
		return
	}

	store, err := getStore()
	if err != nil {
		http.Error(
			w,
			"error connecting to file storage",
			http.StatusInternalServerError,
		)
	}

	id, err := fs.PutWithUUID(store, img)

	w.WriteHeader(http.StatusCreated)
	w.Header().Add("Content-Type", "application/json")
	fmt.Fprintf(w, `{ "imageId": "%s" }`, id)
}

// Retrieves an uploaded image and sends it in the response.
func GetUpload(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("imageId")
	if id == "" {
		http.Error(
			w,
			"missing imageId query parameter",
			http.StatusBadRequest,
		)
		return
	}

	store, err := getStore()
	if err != nil {
		http.Error(
			w,
			"error connecting to file storage",
			http.StatusInternalServerError,
		)
		return
	}

	img, err := store.GetImg(id)
	if err != nil {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", fs.ImgContentType)
	err = fs.EncodeImg(w, img)
	if err != nil {
		slog.Error("error sending image", "key", id, "err", err.Error())
	}
}

// Deletes an uploaded image.
func DeleteUpload(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("imageId")
	if id == "" {
		http.Error(
			w,
			"missing imageId query parameter",
			http.StatusBadRequest,
		)
		return
	}

	store, err := getStore()
	if err != nil {
		http.Error(
			w,
			"error connecting to file storage",
			http.StatusInternalServerError,
		)
		return
	}

	if !store.ImgExists(id) {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := store.DeleteImg(id); err != nil {
		http.Error(w, "image deletion failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
