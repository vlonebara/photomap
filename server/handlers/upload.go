package handlers

import (
	"errors"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vlonebara/photomap/service"
)

// reading multipart metadata before accepting the file
func uploadDraft(r *http.Request) (service.PhotoDraft, error) {
	var draft service.PhotoDraft
	draft.Title = r.FormValue("title")
	draft.Description = r.FormValue("description")
	draft.Place = r.FormValue("place")

	longitude, err := strconv.ParseFloat(r.FormValue("longitude"), 64)
	if err != nil {
		return draft, fieldError("longitude", "Укажите долготу")
	}

	latitude, err := strconv.ParseFloat(r.FormValue("latitude"), 64)
	if err != nil {
		return draft, fieldError("latitude", "Укажите широту")
	}

	year, err := strconv.ParseInt(r.FormValue("year"), 10, 64)
	if err != nil {
		return draft, fieldError("year", "Укажите год")
	}

	draft.Longitude = longitude
	draft.Latitude = latitude
	draft.Year = year
	if value := strings.TrimSpace(r.FormValue("endYear")); value != "" {
		endYear, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return draft, fieldError("endYear", "Проверьте конечный год")
		}

		draft.EndYear = &endYear
	}

	if value := strings.TrimSpace(r.FormValue("sourceName")); value != "" {
		draft.SourceName = &value
	}

	if value := strings.TrimSpace(r.FormValue("sourceUrl")); value != "" {
		draft.SourceURL = &value
	}

	return draft, nil
}

func fieldError(field, message string) error {
	return &service.ValidationError{
		Message: message,
		Fields:  map[string]string{field: message},
	}
}

func (h *PhotoHandler) Upload(w http.ResponseWriter, r *http.Request) {
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(60 * time.Second))
	r.Body = http.MaxBytesReader(w, r.Body, service.MaxImageBytes+(1<<20))
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) || errors.Is(err, multipart.ErrMessageTooLarge) {
			writeError(w, service.ErrImageTooLarge)
		} else {
			writeError(w, fieldError("image", "Ожидается multipart/form-data с изображением"))
		}

		return
	}

	defer r.MultipartForm.RemoveAll()

	if len(r.MultipartForm.File["image"]) != 1 {
		writeError(w, fieldError("image", "Добавьте одно изображение"))
		return
	}

	draft, err := uploadDraft(r)
	if err != nil {
		writeError(w, err)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		writeError(w, fieldError("image", "Добавьте файл изображения"))
		return
	}

	defer file.Close()

	photo, err := h.photos.Upload(r.Context(), viewer(r), draft, file)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, photoJSON(photo, "/api/me/photos"))
}
