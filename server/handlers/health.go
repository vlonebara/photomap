package handlers

import (
	"fmt"
	"net/http"
)

// health check handler
func health(w http.ResponseWriter, r *http.Request) {
	// sending server status as json
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, `{"status":"ok"}`)
}
