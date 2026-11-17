package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	router.Get("/health", health)

	return router
}
