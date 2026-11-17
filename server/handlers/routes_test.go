package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vlonebara/photomap/handlers"
)

// checking health endpoint and invalid requests
func TestNewRouter(t *testing.T) {
	t.Parallel()

	// request data and expected status codes
	tests := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{name: "health", method: http.MethodGet, path: "/health", status: http.StatusOK},
		{name: "unknown route", method: http.MethodGet, path: "/missing", status: http.StatusNotFound},
		{name: "unsupported method", method: http.MethodPost, path: "/health", status: http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// sending request to router without starting a server
			response := httptest.NewRecorder()
			handlers.NewRouter(nil, handlers.Config{}).ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))

			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d", response.Code, tt.status)
			}

			// checking health response format and body
			if tt.status == http.StatusOK {
				if got := response.Header().Get("Content-Type"); got != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", got)
				}

				if got := strings.TrimSpace(response.Body.String()); got != `{"status":"ok"}` {
					t.Errorf("body = %q, want health response", got)
				}
			}
		})
	}
}
