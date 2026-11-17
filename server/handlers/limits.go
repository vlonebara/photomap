package handlers

import "net/http"

// bounding expensive work while preserving the common JSON error format
func limitRequests(count int) func(http.Handler) http.Handler {
	slots := make(chan struct{}, count)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case slots <- struct{}{}:
				defer func() {
					<-slots
				}()
			default:
				w.Header().Set("Retry-After", "1")
				writeJSON(w, http.StatusServiceUnavailable, apiError{
					Code:    "busy",
					Message: "Сервер занят. Повторите запрос через секунду",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
