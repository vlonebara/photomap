package handlers

import "net/http"

func (h *PhotoHandler) Star(w http.ResponseWriter, r *http.Request) {
	id, ok := requestID(w, r)
	if !ok {
		return
	}

	enabled := r.Method == http.MethodPut
	state, err := h.photos.SetStar(r.Context(), id, viewer(r), enabled)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, state)
}

func (h *PhotoHandler) Report(w http.ResponseWriter, r *http.Request) {
	id, ok := requestID(w, r)
	if !ok {
		return
	}

	if err := h.photos.Report(r.Context(), id, viewer(r)); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]bool{"reportedByMe": true})
}
