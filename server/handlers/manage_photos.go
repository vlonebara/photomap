package handlers

import (
	"net/http"

	"github.com/vlonebara/photomap/service"
)

func (h *PhotoHandler) ListOwn(w http.ResponseWriter, r *http.Request) {
	photos, err := h.photos.ListOwn(r.Context(), viewer(r))
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, photosJSON(photos, "/api/me/photos"))
}

func (h *PhotoHandler) GetOwn(w http.ResponseWriter, r *http.Request) {
	id, ok := requestID(w, r)
	if !ok {
		return
	}

	photo, err := h.photos.GetManaged(r.Context(), id, viewer(r), false)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, photoJSON(photo, "/api/me/photos"))
}

func (h *PhotoHandler) Remove(w http.ResponseWriter, r *http.Request) {
	id, ok := requestID(w, r)
	if !ok {
		return
	}

	if err := h.photos.Remove(r.Context(), id, viewer(r)); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PhotoHandler) Queue(w http.ResponseWriter, r *http.Request) {
	photos, err := h.photos.ModerationQueue(r.Context(), viewer(r), r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, photosJSON(photos, "/api/moderation/photos"))
}

func (h *PhotoHandler) GetModerationPhoto(w http.ResponseWriter, r *http.Request) {
	id, ok := requestID(w, r)
	if !ok {
		return
	}

	photo, err := h.photos.GetManaged(r.Context(), id, viewer(r), true)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, photoJSON(photo, "/api/moderation/photos"))
}

// starting PATCH from current values lets null clear optional fields
func editableDraft(photo service.Photo) service.PhotoDraft {
	row := photo.Data
	draft := service.PhotoDraft{
		Title:       row.Title,
		Description: row.Description,
		Place:       row.Place,
		Longitude:   row.Longitude,
		Latitude:    row.Latitude,
		Year:        row.Year,
		SourceName:  textPointer(row.SourceName),
		SourceURL:   textPointer(row.SourceUrl),
	}

	if row.EndYear.Valid {
		draft.EndYear = &row.EndYear.Int64
	}

	return draft
}

func (h *PhotoHandler) Edit(w http.ResponseWriter, r *http.Request) {
	id, ok := requestID(w, r)
	if !ok {
		return
	}

	current, err := h.photos.GetManaged(r.Context(), id, viewer(r), true)
	if err != nil {
		writeError(w, err)
		return
	}

	draft := editableDraft(current)
	if !readJSON(w, r, &draft) {
		return
	}

	photo, err := h.photos.Edit(r.Context(), id, viewer(r), draft)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, photoJSON(photo, "/api/moderation/photos"))
}

func (h *PhotoHandler) Decide(w http.ResponseWriter, r *http.Request) {
	id, ok := requestID(w, r)
	if !ok {
		return
	}

	var request struct {
		Action string `json:"action"`
	}

	if !readJSON(w, r, &request) {
		return
	}

	photo, err := h.photos.Decide(r.Context(), id, viewer(r), request.Action)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, photoJSON(photo, "/api/moderation/photos"))
}
