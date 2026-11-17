//go:build integration

package handlers_test

import (
	"bytes"
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthenticationAndRoles(t *testing.T) {
	t.Parallel()
	f := newAPI(t)
	f.request(t, "GET", "/api/auth/me", nil, nil, 401)
	f.request(t, "GET", "/api/me/photos", nil, nil, 401)
	f.request(t, "GET", "/api/moderation/photos", nil, nil, 401)
	f.request(t, "GET", "/api/series", nil, nil, 200)

	cookie := f.register(t, "alice@example.test", false)
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
		t.Fatalf("unsafe session cookie: %+v", cookie)
	}

	response := f.request(t, "GET", "/api/auth/me", nil, cookie, 200)
	if strings.Contains(response.Body.String(), cookie.Value) || strings.Contains(response.Body.String(), "password") {
		t.Fatal("authentication response exposes credentials")
	}

	f.request(t, "GET", "/api/moderation/photos", nil, cookie, 403)

	// the database must contain the token hash, never the JWT itself
	var storedHash []byte
	if err := f.store.DB.QueryRowContext(t.Context(), "SELECT token_hash FROM sessions").Scan(&storedHash); err != nil {
		t.Fatal(err)
	}

	expected := sha256.Sum256([]byte(cookie.Value))
	if !bytes.Equal(storedHash, expected[:]) {
		t.Fatal("session does not contain the SHA-256 token hash")
	}

	f.request(t, "POST", "/api/auth/login", map[string]string{
		"email":    "alice@example.test",
		"password": "wrong-password",
	}, nil, 401)

	f.request(t, "POST", "/api/auth/register", map[string]string{
		"name":     "Duplicate",
		"email":    "ALICE@example.test",
		"password": testPassword,
	}, nil, 409)

	// registration cannot choose a moderator role
	f.request(t, "POST", "/api/auth/register", map[string]string{
		"name":     "Attacker",
		"email":    "attacker@example.test",
		"password": testPassword,
		"role":     "moderator",
	}, nil, 400)

	forged := *cookie
	forged.Value += "tampered"
	f.request(t, "GET", "/api/auth/me", nil, &forged, 401)

	// changing a role in the database takes effect without replacing the JWT
	if _, err := f.store.DB.ExecContext(t.Context(), "UPDATE users SET role = 'moderator' WHERE email = ?", "alice@example.test"); err != nil {
		t.Fatal(err)
	}

	f.request(t, "GET", "/api/moderation/photos", nil, cookie, 200)

	f.request(t, "POST", "/api/auth/logout", nil, cookie, 204)
	f.request(t, "GET", "/api/auth/me", nil, cookie, 401)

	login := f.request(t, "POST", "/api/auth/login", map[string]string{
		"email":    "alice@example.test",
		"password": testPassword,
	}, nil, 200)
	f.request(t, "GET", "/api/auth/me", nil, login.Result().Cookies()[0], 200)
}

func TestExpiredSessionsAndJWTAlgorithms(t *testing.T) {
	t.Parallel()
	f := newAPI(t)
	cookie := f.register(t, "session@example.test", false)
	user, err := f.store.GetUserByEmail(t.Context(), "session@example.test")
	if err != nil {
		t.Fatal(err)
	}

	// an expired database session cannot be rescued by a valid JWT signature
	_, err = f.store.DB.ExecContext(t.Context(), "UPDATE sessions SET created_at = ?, expires_at = ?", time.Now().Unix()-100, time.Now().Unix()-1)
	if err != nil {
		t.Fatal(err)
	}

	f.request(t, "GET", "/api/auth/me", nil, cookie, 401)

	claims := jwt.RegisteredClaims{
		Issuer:    "photomap",
		Subject:   "1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}

	for _, method := range []jwt.SigningMethod{jwt.SigningMethodHS512, jwt.SigningMethodNone} {
		key := any([]byte(testSecret))
		if method == jwt.SigningMethodNone {
			key = jwt.UnsafeAllowNoneSignatureType
		}

		token, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}

		hash := sha256.Sum256([]byte(token))
		_, err = f.store.DB.ExecContext(t.Context(), "INSERT INTO sessions (token_hash, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)", hash[:], user.ID, time.Now().Unix(), time.Now().Unix()+3600)
		if err != nil {
			t.Fatal(err)
		}

		badCookie := &http.Cookie{Name: "session_token", Value: token}
		f.request(t, "GET", "/api/auth/me", nil, badCookie, 401)
	}
}

func TestCrossSiteWritesAreRejected(t *testing.T) {
	t.Parallel()
	f := newAPI(t)
	cookie := f.register(t, "csrf@example.test", false)
	request, err := http.NewRequest("POST", "/api/auth/logout", nil)
	if err != nil {
		t.Fatal(err)
	}

	request.Header.Set("Origin", "https://another-site.example")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	f.router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("cross-site logout: status = %d", response.Code)
	}

	f.request(t, "GET", "/api/auth/me", nil, cookie, 200)
}
