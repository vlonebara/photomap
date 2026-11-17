package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vlonebara/photomap/service"
)

// streaming public revisions and releasing connections on shutdown
func eventStream(events *service.Events) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unavailable", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		updates, unsubscribe := events.Subscribe()
		defer unsubscribe()

		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()

		for {
			var message string

			select {
			case <-r.Context().Done():
				return
			case event, open := <-updates:
				if !open {
					return
				}

				data, err := json.Marshal(event)
				if err != nil {
					return
				}

				message = fmt.Sprintf("id: %d\nevent: map.updated\ndata: %s\n\n", event.Revision, data)
			case <-heartbeat.C:
				message = ": keep-alive\n\n"
			}

			controller := http.NewResponseController(w)
			_ = controller.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, err := fmt.Fprint(w, message); err != nil {
				return
			}

			flusher.Flush()
		}
	}
}
