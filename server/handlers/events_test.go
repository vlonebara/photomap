//go:build integration

package handlers_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vlonebara/photomap/service"
)

func openEvents(t *testing.T, server *httptest.Server) *bufio.Reader {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	request, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/events", nil)
	if err != nil {
		cancel()
		t.Fatal(err)
	}

	request.Header.Set("Last-Event-ID", "0")

	response, err := server.Client().Do(request)
	if err != nil {
		cancel()
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "text/event-stream" {
		response.Body.Close()
		cancel()
		t.Fatalf("SSE response: %d", response.StatusCode)
	}

	t.Cleanup(func() {
		cancel()
		response.Body.Close()
	})

	return bufio.NewReader(response.Body)
}

func nextEvent(t *testing.T, reader *bufio.Reader) service.MapEvent {
	t.Helper()

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal("waiting for SSE:", err)
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		var event service.MapEvent
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
			t.Fatal(err)
		}

		return event
	}
}

func TestTwoClientsReceiveCommittedChangesAndReconnect(t *testing.T) {
	t.Parallel()
	f := newAPI(t)
	owner := f.register(t, "owner@example.test", false)
	moderator := f.register(t, "moderator@example.test", true)
	response := f.upload(t, owner, nil, testImage(t), 201)
	var photo card
	decodeResponse(t, response, &photo)

	server := httptest.NewServer(f.router)
	t.Cleanup(server.Close)
	first := openEvents(t, server)
	second := openEvents(t, server)
	if nextEvent(t, first).Revision != 0 || nextEvent(t, second).Revision != 0 {
		t.Fatal("pending photo changed the public revision")
	}

	f.decide(t, moderator, photo.ID, "approve", 200)
	approved := nextEvent(t, first)
	if nextEvent(t, second).Revision != approved.Revision {
		t.Fatal("clients observed different publication revisions")
	}

	committed, err := f.store.GetMapRevision(t.Context())
	if err != nil || committed != approved.Revision {
		t.Fatalf("event preceded the database commit: %d, %v", committed, err)
	}

	f.request(t, "GET", photoPath(photo.ID, ""), nil, nil, 200)

	f.request(t, "PUT", photoPath(photo.ID, "/star"), nil, owner, 200)
	starred := nextEvent(t, first)
	if starred.Revision <= approved.Revision || nextEvent(t, second).Revision != starred.Revision {
		t.Fatal("star change did not reach both clients")
	}

	// a reconnect receives the latest revision even with a stale Last-Event-ID
	reconnected := openEvents(t, server)
	if nextEvent(t, reconnected).Revision != starred.Revision {
		t.Fatal("reconnected client missed the current revision")
	}

	f.decide(t, moderator, photo.ID, "unpublish", 200)
	hidden := nextEvent(t, first)
	if hidden.Revision <= starred.Revision || nextEvent(t, second).Revision != hidden.Revision {
		t.Fatal("hiding did not reach both clients")
	}

	f.request(t, "GET", photoPath(photo.ID, "/image"), nil, nil, 404)

	f.events.Close()
}

func TestFailedModerationRollsBackHistoryStatusAndNotifications(t *testing.T) {
	t.Parallel()
	f := newAPI(t)
	first := f.register(t, "first@example.test", false)
	second := f.register(t, "second@example.test", false)
	third := f.register(t, "third@example.test", false)
	moderator := f.register(t, "moderator@example.test", true)
	id := f.published(t, first, moderator, nil)

	for _, user := range []*http.Cookie{first, second, third} {
		f.request(t, "POST", photoPath(id, "/reports"), nil, user, 201)
	}

	revision, err := f.store.GetMapRevision(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	// failing the last statement must also undo the preceding status and history writes
	_, err = f.store.DB.ExecContext(t.Context(), `
		CREATE TRIGGER fail_resolution BEFORE UPDATE OF resolved_at ON photo_reports
		BEGIN SELECT RAISE(ABORT, 'test resolution failure'); END;
	`)
	if err != nil {
		t.Fatal(err)
	}

	updates, unsubscribe := f.events.Subscribe()
	defer unsubscribe()
	<-updates
	f.decide(t, moderator, id, "restore", 500)

	stored, err := f.store.GetPhoto(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}

	if stored.Status != "review" {
		t.Fatalf("failed decision changed status to %s", stored.Status)
	}

	var history, active int
	if err := f.store.DB.QueryRowContext(t.Context(), "SELECT count(*) FROM photo_moderation WHERE photo_id = ?", id).Scan(&history); err != nil {
		t.Fatal(err)
	}

	if err := f.store.DB.QueryRowContext(t.Context(), "SELECT count(*) FROM photo_reports WHERE photo_id = ? AND resolved_at IS NULL", id).Scan(&active); err != nil {
		t.Fatal(err)
	}

	after, err := f.store.GetMapRevision(t.Context())
	if err != nil || after != revision || history != 1 || active != 3 {
		t.Fatalf("rollback failed: revision %d/%d, history %d, reports %d, error %v", revision, after, history, active, err)
	}

	select {
	case event := <-updates:
		t.Fatalf("failed transaction emitted event %+v", event)
	default:
	}
}
