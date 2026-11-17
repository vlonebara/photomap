package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/vlonebara/photomap/service"
)

const sessionCookie = "session_token"

type viewerKey struct{}

type AuthHandler struct {
	auth          *service.AuthService
	secureCookies bool
}

type authRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func userJSON(user service.User) userResponse {
	return userResponse{
		ID:    strconv.FormatInt(user.ID, 10),
		Name:  user.Name,
		Email: user.Email,
		Role:  user.Role,
	}
}

func viewer(r *http.Request) *service.User {
	user, _ := r.Context().Value(viewerKey{}).(*service.User)
	return user
}

// public requests remain available when a cookie has expired
func (h *AuthHandler) optionalUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil || cookie.Value == "" {
			next.ServeHTTP(w, r)
			return
		}

		user, err := h.auth.Authenticate(r.Context(), cookie.Value)
		if errors.Is(err, service.ErrUnauthorized) {
			next.ServeHTTP(w, r)
			return
		}

		if err != nil {
			writeError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), viewerKey{}, &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if viewer(r) == nil {
			writeError(w, service.ErrUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func moderatorOnly(next http.Handler) http.Handler {
	return requireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if viewer(r).Role != "moderator" {
			writeError(w, service.ErrForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}))
}

// rejecting cross-site writes before cookie authentication is used
func sameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			writeError(w, service.ErrForbidden)
			return
		}

		if origin := r.Header.Get("Origin"); origin != "" {
			parsed, err := url.Parse(origin)
			if err != nil || parsed.Host != r.Host || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				writeError(w, service.ErrForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var request authRequest
	if !readJSON(w, r, &request) {
		return
	}

	session, err := h.auth.Register(r.Context(), request.Name, request.Email, request.Password)
	if err != nil {
		writeError(w, err)
		return
	}

	h.setCookie(w, session)
	writeJSON(w, http.StatusCreated, userJSON(session.User))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request authRequest
	if !readJSON(w, r, &request) {
		return
	}

	session, err := h.auth.Login(r.Context(), request.Email, request.Password)
	if err != nil {
		writeError(w, err)
		return
	}

	h.setCookie(w, session)
	writeJSON(w, http.StatusOK, userJSON(session.User))
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, userJSON(*viewer(r)))
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookie)
	if err == nil {
		if err := h.auth.Logout(r.Context(), cookie.Value); err != nil {
			writeError(w, err)
			return
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
	})
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

// the JWT is never returned in JSON or exposed to browser scripts
func (h *AuthHandler) setCookie(w http.ResponseWriter, session service.Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(service.SessionDuration.Seconds()),
		Expires:  session.Expires,
	})
}
