package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vlonebara/photomap/service"
)

type apiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// sending one JSON response without caching private or revocable data
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Vary", "Cookie")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("Write JSON: %v", err)
	}
}

// keeping internal errors out of public responses
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	response := apiError{Code: "internal_error", Message: "Не удалось выполнить запрос"}
	var validation *service.ValidationError
	var tooLarge *http.MaxBytesError

	switch {
	case errors.As(err, &validation):
		status = http.StatusUnprocessableEntity
		response = apiError{Code: "invalid_fields", Message: validation.Message, Fields: validation.Fields}
	case errors.As(err, &tooLarge), errors.Is(err, service.ErrImageTooLarge):
		status = http.StatusRequestEntityTooLarge
		response = apiError{Code: "file_too_large", Message: "Файл должен быть не больше 10 МБ"}
	case errors.Is(err, service.ErrUnauthorized), errors.Is(err, service.ErrCredentials):
		status = http.StatusUnauthorized
		response = apiError{Code: "unauthorized", Message: "Необходим вход"}
		if errors.Is(err, service.ErrCredentials) {
			response = apiError{Code: "invalid_credentials", Message: "Неверный email или пароль"}
		}
	case errors.Is(err, service.ErrForbidden):
		status = http.StatusForbidden
		response = apiError{Code: "forbidden", Message: "Недостаточно прав"}
	case errors.Is(err, service.ErrNotFound):
		status = http.StatusNotFound
		response = apiError{Code: "not_found", Message: "Фото или серия недоступны"}
	case errors.Is(err, service.ErrAlreadyReported):
		status = http.StatusConflict
		response = apiError{Code: "already_reported", Message: "Вы уже жаловались на это фото"}
	case errors.Is(err, service.ErrEmailExists):
		status = http.StatusConflict
		response = apiError{Code: "email_exists", Message: "Этот email уже зарегистрирован"}
	case errors.Is(err, service.ErrConflict):
		status = http.StatusConflict
		response = apiError{Code: "invalid_state", Message: "Действие недоступно в текущем статусе"}
	}

	if status == http.StatusInternalServerError {
		log.Printf("API request: %v", err)
	}

	writeJSON(w, status, response)
}

// accepting one JSON object and rejecting unknown fields
func readJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(10 * time.Second))
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(value); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Code: "invalid_json", Message: "Проверьте JSON запроса"})
		return false
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, apiError{Code: "invalid_json", Message: "Ожидается один JSON-объект"})
		return false
	}

	return true
}

// parsing the same positive identifier for photo and series routes
func requestID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, service.ErrNotFound)
		return 0, false
	}

	return id, true
}
