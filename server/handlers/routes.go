package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vlonebara/photomap/db"
	"github.com/vlonebara/photomap/service"
)

type Config struct {
	MediaDir      string
	Auth          *service.AuthService
	Events        *service.Events
	SecureCookies bool
}

// creating router and registering handlers
func NewRouter(store *db.Store, config Config) http.Handler {
	router := chi.NewRouter()
	// returning server error if a handler panics
	router.Use(middleware.Recoverer)
	router.Get("/health", health)

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, service.ErrNotFound)
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{
			Code:    "method_not_allowed",
			Message: "Этот HTTP-метод недоступен для маршрута",
		})
	})

	photos := &PhotoHandler{
		photos: service.NewPhotoService(store, config.MediaDir, config.Events),
	}

	auth := &AuthHandler{
		auth:          config.Auth,
		secureCookies: config.SecureCookies,
	}

	authLimit := limitRequests(8)

	router.Route("/api", func(api chi.Router) {
		api.Use(sameOrigin)
		api.Use(auth.optionalUser)

		api.Get("/series", photos.ListSeries)
		api.Get("/series/{id}", photos.GetSeries)
		api.Get("/photos/{id}", photos.GetPhoto)
		api.Get("/photos/{id}/image", photos.image(imageAccess{}))
		api.Get("/photos/{id}/thumbnail", photos.image(imageAccess{thumbnail: true}))
		api.Get("/events", eventStream(config.Events))

		api.With(authLimit).Post("/auth/register", auth.Register)
		api.With(authLimit).Post("/auth/login", auth.Login)
		api.With(requireUser).Get("/auth/me", auth.Me)
		api.With(requireUser).Post("/auth/logout", auth.Logout)

		api.Group(func(user chi.Router) {
			user.Use(requireUser)
			user.With(limitRequests(2)).Post("/photos", photos.Upload)
			user.Put("/photos/{id}/star", photos.Star)
			user.Delete("/photos/{id}/star", photos.Star)
			user.Post("/photos/{id}/reports", photos.Report)
			user.Get("/me/photos", photos.ListOwn)
			user.Get("/me/photos/{id}", photos.GetOwn)
			user.Get("/me/photos/{id}/image", photos.image(imageAccess{private: true}))
			user.Get("/me/photos/{id}/thumbnail", photos.image(imageAccess{private: true, thumbnail: true}))
			user.Delete("/me/photos/{id}", photos.Remove)
		})

		api.Route("/moderation/photos", func(moderator chi.Router) {
			moderator.Use(moderatorOnly)
			moderator.Get("/", photos.Queue)
			moderator.Get("/{id}", photos.GetModerationPhoto)
			moderator.Get("/{id}/image", photos.image(imageAccess{private: true, moderator: true}))
			moderator.Get("/{id}/thumbnail", photos.image(imageAccess{private: true, moderator: true, thumbnail: true}))
			moderator.Patch("/{id}", photos.Edit)
			moderator.Post("/{id}/decision", photos.Decide)
		})
	})

	return router
}
