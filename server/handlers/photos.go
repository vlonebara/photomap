package handlers

import (
	"net/http"
	"path/filepath"

	"github.com/vlonebara/photomap/service"
)

type PhotoHandler struct {
	photos *service.PhotoService
}

type imageAccess struct {
	private   bool
	moderator bool
	thumbnail bool
}

func (h *PhotoHandler) ListSeries(w http.ResponseWriter, r *http.Request) {
	groups, err := h.photos.ListSeries(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}

	result := make([]seriesResponse, 0, len(groups))
	for _, group := range groups {
		result = append(result, seriesJSON(group))
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *PhotoHandler) GetSeries(w http.ResponseWriter, r *http.Request) {
	id, ok := requestID(w, r)
	if !ok {
		return
	}

	group, err := h.photos.GetSeries(r.Context(), id, viewer(r))
	if err != nil {
		writeError(w, err)
		return
	}

	response := seriesDetailResponse{
		seriesResponse: seriesJSON(group),
		Photos:         photosJSON(group.Photos, "/api/photos"),
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *PhotoHandler) GetPhoto(w http.ResponseWriter, r *http.Request) {
	id, ok := requestID(w, r)
	if !ok {
		return
	}

	photo, err := h.photos.GetPublishedPhoto(r.Context(), id, viewer(r))
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, photoJSON(photo, "/api/photos"))
}

// sharing file handling while keeping access checks in the service
func (h *PhotoHandler) image(access imageAccess) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := requestID(w, r)
		if !ok {
			return
		}

		var image service.Image
		var err error
		if access.private {
			image, err = h.photos.PrivateImage(r.Context(), id, viewer(r), access.moderator, access.thumbnail)
		} else {
			image, err = h.photos.PublicImage(r.Context(), id, access.thumbnail)
		}

		if err != nil {
			writeError(w, err)
			return
		}

		defer image.File.Close()

		info, err := image.File.Stat()
		if err != nil {
			writeError(w, err)
			return
		}

		w.Header().Set("Content-Type", image.Type)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Vary", "Cookie")
		http.ServeContent(w, r, filepath.Base(image.File.Name()), info.ModTime(), image.File)
	}
}
