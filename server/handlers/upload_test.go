//go:build integration

package handlers_test

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestUploadValidation(t *testing.T) {
	t.Parallel()
	f := newAPI(t)
	owner := f.register(t, "owner@example.test", false)
	image := testImage(t)

	cases := []struct {
		name   string
		fields map[string]string
		image  []byte
		status int
	}{
		{name: "invalid longitude", fields: map[string]string{"longitude": "181"}, image: image, status: 422},
		{name: "not a finite coordinate", fields: map[string]string{"latitude": "NaN"}, image: image, status: 422},
		{name: "missing year", fields: map[string]string{"year": ""}, image: image, status: 422},
		{name: "reversed range", fields: map[string]string{"endYear": "1899"}, image: image, status: 422},
		{name: "range longer than 50", fields: map[string]string{"endYear": "1951"}, image: image, status: 422},
		{name: "script source", fields: map[string]string{"sourceUrl": "javascript:alert(1)"}, image: image, status: 422},
		{name: "fake image", image: []byte("<script>not an image</script>"), status: 422},
		{name: "file too large", image: bytes.Repeat([]byte("x"), (10<<20)+1), status: 413},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			f.upload(t, owner, test.fields, test.image, test.status)
		})
	}

	var count int
	if err := f.store.DB.QueryRowContext(t.Context(), "SELECT count(*) FROM photos").Scan(&count); err != nil {
		t.Fatal(err)
	}

	if count != 0 {
		t.Fatalf("invalid uploads created %d records", count)
	}

	f.request(t, "POST", "/api/photos", map[string]string{"title": "not multipart"}, owner, 422)
	f.upload(t, nil, nil, image, 401)
	f.upload(t, owner, map[string]string{"endYear": "1950"}, image, 201)
}

func TestPrivateMediaCannotEscapeItsDirectory(t *testing.T) {
	t.Parallel()
	f := newAPI(t)
	owner := f.register(t, "owner@example.test", false)
	moderator := f.register(t, "moderator@example.test", true)
	id := f.published(t, owner, moderator, nil)

	outside := filepath.Join(t.TempDir(), "outside.png")
	if err := os.WriteFile(outside, testImage(t), 0o600); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(f.media, "outside-link.png")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"../outside.png", outside, "outside-link.png"} {
		t.Run(key, func(t *testing.T) {
			_, err := f.store.DB.ExecContext(t.Context(), "UPDATE photos SET image_key = ? WHERE id = ?", key, id)
			if err != nil {
				t.Fatal(err)
			}

			f.request(t, "GET", photoPath(id, "/image"), nil, nil, http.StatusNotFound)
		})
	}

	f.request(t, "GET", "/media/outside-link.png", nil, nil, 404)
	f.request(t, "GET", fmt.Sprintf("/api/photos/%d/image/../../photomap.db", id), nil, nil, 404)
}

func TestModerationPatchIsValidatedAndKeepsStatus(t *testing.T) {
	t.Parallel()
	f := newAPI(t)
	owner := f.register(t, "owner@example.test", false)
	moderator := f.register(t, "moderator@example.test", true)
	id := f.published(t, owner, moderator, map[string]string{"endYear": "1910"})
	path := fmt.Sprintf("/api/moderation/photos/%d", id)

	f.request(t, "PATCH", path, map[string]any{"endYear": 1899}, moderator, 422)
	f.request(t, "PATCH", path, map[string]any{"status": "removed"}, moderator, 400)
	f.request(t, "PATCH", path, map[string]any{"title": "Исправленное название", "endYear": nil}, moderator, 200)

	photo, err := f.store.GetPhoto(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}

	if photo.Status != "published" || photo.Title != "Исправленное название" || photo.EndYear.Valid {
		t.Fatalf("partial correction changed the wrong fields: %+v", photo)
	}
}
