package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// creating router and registering handlers
func NewRouter() http.Handler {
	router := chi.NewRouter()
	// returning server error if a handler panics
	router.Use(middleware.Recoverer)
	router.Get("/health", health)

	return router
}
