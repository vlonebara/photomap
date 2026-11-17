//go:build integration

package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/vlonebara/photomap/db"
	"github.com/vlonebara/photomap/handlers"
	"github.com/vlonebara/photomap/service"
)

const testPassword = "TestPassword123!"
const testSecret = "test-signing-key-with-at-least-32-bytes"

type apiFixture struct {
	router http.Handler
	store  *db.Store
	events *service.Events
	media  string
}

type card struct {
	ID           int64      `json:"id"`
	Title        string     `json:"title"`
	Status       string     `json:"status"`
	Stars        int64      `json:"stars"`
	SeriesID     int64      `json:"seriesId"`
	Coordinates  [2]float64 `json:"coordinates"`
	ImageURL     string     `json:"imageUrl"`
	StarredByMe  *bool      `json:"starredByMe"`
	ReportedByMe *bool      `json:"reportedByMe"`
}

// every API test receives an isolated database and media directory
func newAPI(t *testing.T) *apiFixture {
	t.Helper()

	directory := t.TempDir()
	path := filepath.Join(directory, "photomap.db")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	connection, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { connection.Close() })

	schema, err := os.ReadFile("../db/schema/001_init.sql")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := connection.ExecContext(t.Context(), string(schema)); err != nil {
		t.Fatal(err)
	}

	store := db.NewStore(connection)
	auth, err := service.NewAuthService(store, testSecret)
	if err != nil {
		t.Fatal(err)
	}

	events := service.NewEvents(t.Context(), 0)
	t.Cleanup(events.Close)
	media := filepath.Join(directory, "media")
	router := handlers.NewRouter(store, handlers.Config{
		MediaDir: media,
		Auth:     auth,
		Events:   events,
	})

	return &apiFixture{router: router, store: store, events: events, media: media}
}

func (f *apiFixture) request(t *testing.T, method, path string, body any, cookie *http.Cookie, status int) *httptest.ResponseRecorder {
	t.Helper()

	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}

	request := httptest.NewRequest(method, path, bytes.NewReader(data)).WithContext(t.Context())
	request.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		request.AddCookie(cookie)
	}

	response := httptest.NewRecorder()
	f.router.ServeHTTP(response, request)
	if response.Code != status {
		t.Fatalf("%s %s: status = %d, want %d; body = %s", method, path, response.Code, status, response.Body.String())
	}

	return response
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, value any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), value); err != nil {
		t.Fatal(err)
	}
}

func (f *apiFixture) register(t *testing.T, email string, moderator bool) *http.Cookie {
	t.Helper()

	response := f.request(t, "POST", "/api/auth/register", map[string]string{
		"name":     email,
		"email":    email,
		"password": testPassword,
	}, nil, http.StatusCreated)

	if moderator {
		_, err := f.store.DB.ExecContext(t.Context(), "UPDATE users SET role = 'moderator' WHERE email = ?", email)
		if err != nil {
			t.Fatal(err)
		}
	}

	return response.Result().Cookies()[0]
}

func testImage(t *testing.T) []byte {
	t.Helper()
	var imageBytes bytes.Buffer
	if err := png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 20, 15))); err != nil {
		t.Fatal(err)
	}

	return imageBytes.Bytes()
}

func (f *apiFixture) upload(t *testing.T, cookie *http.Cookie, fields map[string]string, imageBytes []byte, status int) *httptest.ResponseRecorder {
	t.Helper()

	values := map[string]string{
		"title":       "Исторический снимок",
		"description": "Проверка загрузки",
		"place":       "Париж",
		"longitude":   "2.341",
		"latitude":    "48.857",
		"year":        "1900",
	}

	for name, value := range fields {
		values[name] = value
	}

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for name, value := range values {
		if err := form.WriteField(name, value); err != nil {
			t.Fatal(err)
		}
	}

	file, err := form.CreateFormFile("image", "../../untrusted.png")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := file.Write(imageBytes); err != nil {
		t.Fatal(err)
	}

	if err := form.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest("POST", "/api/photos", &body).WithContext(t.Context())
	request.Header.Set("Content-Type", form.FormDataContentType())
	if cookie != nil {
		request.AddCookie(cookie)
	}

	response := httptest.NewRecorder()
	f.router.ServeHTTP(response, request)
	if response.Code != status {
		t.Fatalf("upload: status = %d, want %d; body = %s", response.Code, status, response.Body.String())
	}

	return response
}

func (f *apiFixture) decide(t *testing.T, cookie *http.Cookie, id int64, action string, status int) {
	t.Helper()
	path := fmt.Sprintf("/api/moderation/photos/%d/decision", id)
	f.request(t, "POST", path, map[string]string{"action": action}, cookie, status)
}

func (f *apiFixture) published(t *testing.T, owner, moderator *http.Cookie, fields map[string]string) int64 {
	t.Helper()
	response := f.upload(t, owner, fields, testImage(t), http.StatusCreated)
	var photo card
	decodeResponse(t, response, &photo)
	f.decide(t, moderator, photo.ID, "approve", http.StatusOK)
	return photo.ID
}

func photoPath(id int64, suffix string) string {
	return fmt.Sprintf("/api/photos/%d%s", id, suffix)
}
