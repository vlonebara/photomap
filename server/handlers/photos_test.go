//go:build integration

package handlers_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestPublicationOwnershipAndRemoval(t *testing.T) {
	t.Parallel()
	f := newAPI(t)
	owner := f.register(t, "owner@example.test", false)
	other := f.register(t, "other@example.test", false)
	moderator := f.register(t, "moderator@example.test", true)
	image := testImage(t)

	response := f.upload(t, owner, nil, image, 201)
	var pending card
	decodeResponse(t, response, &pending)
	if pending.Status != "pending" {
		t.Fatalf("upload status = %q", pending.Status)
	}

	for _, suffix := range []string{"", "/image", "/thumbnail"} {
		f.request(t, "GET", photoPath(pending.ID, suffix), nil, nil, 404)
		f.request(t, "GET", photoPath(pending.ID, suffix), nil, owner, 404)
	}

	privatePath := fmt.Sprintf("/api/me/photos/%d", pending.ID)
	f.request(t, "GET", privatePath, nil, owner, 200)
	f.request(t, "GET", privatePath, nil, other, 403)
	f.request(t, "GET", privatePath+"/image", nil, other, 403)
	f.request(t, "DELETE", privatePath, nil, moderator, 403)
	privateImage := f.request(t, "GET", privatePath+"/image", nil, owner, 200)
	if !bytes.Equal(privateImage.Body.Bytes(), image) {
		t.Fatal("private image bytes changed")
	}

	f.request(t, "GET", "/api/moderation/photos", nil, other, 403)
	f.request(t, "GET", "/api/moderation/photos", nil, moderator, 200)
	f.request(t, "GET", fmt.Sprintf("/api/moderation/photos/%d", pending.ID), nil, moderator, 200)
	f.decide(t, other, pending.ID, "approve", 403)
	f.decide(t, moderator, pending.ID, "approve", 200)

	public := f.request(t, "GET", photoPath(pending.ID, ""), nil, nil, 200)
	var photo card
	decodeResponse(t, public, &photo)
	if photo.SeriesID != photo.ID || photo.StarredByMe != nil || photo.ReportedByMe != nil {
		t.Fatalf("unexpected guest card: %+v", photo)
	}

	if photo.Coordinates != [2]float64{2.341, 48.857} {
		t.Fatalf("coordinates = %v", photo.Coordinates)
	}

	for _, suffix := range []string{"/image", "/thumbnail"} {
		file := f.request(t, "GET", photoPath(photo.ID, suffix), nil, nil, 200)
		if file.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("revocable media can be cached")
		}
	}

	f.decide(t, moderator, photo.ID, "unpublish", 200)
	for _, suffix := range []string{"", "/image", "/thumbnail"} {
		f.request(t, "GET", photoPath(photo.ID, suffix), nil, nil, 404)
	}

	f.decide(t, moderator, photo.ID, "restore", 200)

	f.request(t, "DELETE", privatePath, nil, other, 403)
	f.request(t, "DELETE", privatePath, nil, owner, 204)
	f.request(t, "GET", photoPath(photo.ID, ""), nil, nil, 404)
	f.decide(t, moderator, photo.ID, "restore", 409)

	stored, err := f.store.GetPhoto(t.Context(), photo.ID)
	if err != nil {
		t.Fatal(err)
	}

	if stored.Status != "removed" {
		t.Fatalf("removed status = %q", stored.Status)
	}

	if _, err := os.Stat(filepath.Join(f.media, stored.ImageKey)); err != nil {
		t.Fatal("removed photo file was not preserved", err)
	}
}

func TestStarsReportsAndSeriesRecalculation(t *testing.T) {
	t.Parallel()
	f := newAPI(t)
	first := f.register(t, "first@example.test", false)
	second := f.register(t, "second@example.test", false)
	third := f.register(t, "third@example.test", false)
	moderator := f.register(t, "moderator@example.test", true)

	one := f.published(t, first, moderator, nil)
	two := f.published(t, second, moderator, map[string]string{"year": "1850"})
	f.request(t, "PUT", photoPath(two, "/star"), nil, nil, 401)
	f.request(t, "PUT", photoPath(two, "/star"), nil, first, 200)
	f.request(t, "PUT", photoPath(two, "/star"), nil, first, 200)

	var photo card
	decodeResponse(t, f.request(t, "GET", photoPath(one, ""), nil, nil, 200), &photo)
	if photo.SeriesID != two {
		t.Fatalf("center = %d, want %d", photo.SeriesID, two)
	}

	var series struct {
		PhotoCount int    `json:"photoCount"`
		Stars      int64  `json:"stars"`
		Photos     []card `json:"photos"`
	}

	decodeResponse(t, f.request(t, "GET", fmt.Sprintf("/api/series/%d", two), nil, nil, 200), &series)
	if series.PhotoCount != 2 || series.Stars != 1 || series.Photos[0].ID != two {
		t.Fatalf("series ordering or totals are incorrect: %+v", series)
	}

	f.request(t, "GET", fmt.Sprintf("/api/series/%d", one), nil, nil, 404)

	for index, user := range []*http.Cookie{first, second, third} {
		f.request(t, "POST", photoPath(two, "/reports"), nil, user, 201)
		if index < 2 {
			f.request(t, "GET", photoPath(two, ""), nil, nil, 200)
		}
	}

	for _, suffix := range []string{"", "/image", "/thumbnail"} {
		f.request(t, "GET", photoPath(two, suffix), nil, nil, 404)
	}

	f.request(t, "POST", photoPath(two, "/reports"), nil, first, 409)

	f.decide(t, moderator, two, "restore", 200)
	decodeResponse(t, f.request(t, "GET", photoPath(two, ""), nil, first, 200), &photo)
	if photo.ID != two || photo.Stars != 1 || photo.ReportedByMe == nil || !*photo.ReportedByMe {
		t.Fatalf("restoration lost identity or history: %+v", photo)
	}

	f.request(t, "POST", photoPath(two, "/reports"), nil, first, 409)

	var active int
	if err := f.store.DB.QueryRowContext(t.Context(), "SELECT count(*) FROM photo_reports WHERE photo_id = ? AND resolved_at IS NULL", two).Scan(&active); err != nil {
		t.Fatal(err)
	}

	if active != 0 {
		t.Fatalf("active reports after restoration = %d", active)
	}

	f.request(t, "DELETE", photoPath(two, "/star"), nil, first, 200)
	f.request(t, "DELETE", photoPath(two, "/star"), nil, first, 200)
	decodeResponse(t, f.request(t, "GET", photoPath(one, ""), nil, nil, 200), &photo)
	if photo.SeriesID != one {
		t.Fatalf("center after removing star = %d", photo.SeriesID)
	}
}

func TestConcurrentRepeatedStar(t *testing.T) {
	t.Parallel()
	f := newAPI(t)
	owner := f.register(t, "owner@example.test", false)
	moderator := f.register(t, "moderator@example.test", true)
	id := f.published(t, owner, moderator, nil)

	var requests sync.WaitGroup
	for i := 0; i < 20; i++ {
		requests.Add(1)
		go func() {
			defer requests.Done()

			request := httptest.NewRequest("PUT", photoPath(id, "/star"), nil)
			request.AddCookie(owner)
			response := httptest.NewRecorder()
			f.router.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Errorf("concurrent star: status = %d; body = %s", response.Code, response.Body.String())
			}
		}()
	}

	requests.Wait()

	photo, err := f.store.GetPhoto(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}

	if photo.StarCount != 1 {
		t.Fatalf("concurrent stars changed counter to %d", photo.StarCount)
	}
}
