package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"errors"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vlonebara/photomap/db"
	"golang.org/x/crypto/bcrypt"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const SessionDuration = 24 * time.Hour

type AuthService struct {
	store     *db.Store
	secret    []byte
	dummyHash []byte
}

type Session struct {
	User    User
	Token   string
	Expires time.Time
}

// requiring a private signing key before authentication is enabled
func NewAuthService(store *db.Store, secret string) (*AuthService, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET must contain at least 32 bytes")
	}

	dummyHash, err := bcrypt.GenerateFromPassword([]byte(rand.Text()), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return &AuthService{
		store:     store,
		secret:    []byte(secret),
		dummyHash: dummyHash,
	}, nil
}

// checking email without accepting display names or malformed addresses
func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 254 {
		return "", invalid("email", "Введите корректный email")
	}

	return email, nil
}

// registration and initial session are committed together
func (s *AuthService) Register(ctx context.Context, name, email, password string) (Session, error) {
	var result Session
	email, err := normalizeEmail(email)
	if err != nil {
		return result, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return result, invalid("name", "Укажите имя")
	}

	if utf8.RuneCountInString(password) < 8 || len(password) > 72 {
		return result, invalid("password", "Пароль должен содержать от 8 символов и не более 72 байт")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return result, err
	}

	err = s.store.WithTx(ctx, func(q *db.Queries) error {
		user, err := q.CreateUser(ctx, db.CreateUserParams{
			Name:         name,
			Email:        email,
			PasswordHash: string(hash),
		})
		if err != nil {
			var databaseError *sqlite.Error
			if errors.As(err, &databaseError) && databaseError.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
				return ErrEmailExists
			}

			return err
		}

		result, err = s.newSession(ctx, q, User{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role})
		return err
	})
	return result, err
}

// using the same password check cost for unknown and disabled accounts
func (s *AuthService) Login(ctx context.Context, email, password string) (Session, error) {
	var result Session
	email, err := normalizeEmail(email)
	if err != nil {
		return result, ErrCredentials
	}

	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}

	hash := s.dummyHash
	if err == nil && strings.HasPrefix(user.PasswordHash, "$2") {
		hash = []byte(user.PasswordHash)
	}

	passwordError := bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err != nil || passwordError != nil || !strings.HasPrefix(user.PasswordHash, "$2") {
		return result, ErrCredentials
	}

	err = s.store.WithTx(ctx, func(q *db.Queries) error {
		var sessionError error
		result, sessionError = s.newSession(ctx, q, User{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role})
		return sessionError
	})
	return result, err
}

// signing a JWT and storing only its SHA-256 hash
func (s *AuthService) newSession(ctx context.Context, q *db.Queries, user User) (Session, error) {
	now := time.Now().UTC().Truncate(time.Second)
	expires := now.Add(SessionDuration)

	claims := jwt.RegisteredClaims{
		Issuer:    "photomap",
		Subject:   strconv.FormatInt(user.ID, 10),
		ID:        rand.Text(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expires),
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return Session{}, err
	}

	hash := sha256.Sum256([]byte(token))
	err = q.CreateSession(ctx, db.CreateSessionParams{
		TokenHash: hash[:],
		UserID:    user.ID,
		CreatedAt: now.Unix(),
		ExpiresAt: expires.Unix(),
	})

	return Session{User: user, Token: token, Expires: expires}, err
}

// checking signature, expiration and a live session before loading the role
func (s *AuthService) Authenticate(ctx context.Context, value string) (User, error) {
	claims := &jwt.RegisteredClaims{}
	key := func(token *jwt.Token) (any, error) {
		return s.secret, nil
	}

	token, err := jwt.ParseWithClaims(
		value,
		claims,
		key,
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer("photomap"),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil || !token.Valid {
		return User{}, ErrUnauthorized
	}

	hash := sha256.Sum256([]byte(value))
	user, err := s.store.GetSessionUser(ctx, db.GetSessionUserParams{TokenHash: hash[:], ExpiresAt: time.Now().Unix()})
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUnauthorized
	}

	if err != nil {
		return User{}, err
	}

	if claims.Subject != strconv.FormatInt(user.ID, 10) {
		return User{}, ErrUnauthorized
	}

	return User{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role}, nil
}

// revoking the current session without keeping a usable token
func (s *AuthService) Logout(ctx context.Context, value string) error {
	hash := sha256.Sum256([]byte(value))
	return s.store.WithTx(ctx, func(q *db.Queries) error { return q.DeleteSession(ctx, hash[:]) })
}

// moderator permissions are checked using the database identity
func requireModerator(user *User) error {
	if user == nil {
		return ErrUnauthorized
	}

	if user.Role != "moderator" {
		return ErrForbidden
	}

	return nil
}

// ownership never comes from a request field
func requireOwner(user *User, authorID int64) error {
	if user == nil {
		return ErrUnauthorized
	}

	if user.ID != authorID {
		return ErrForbidden
	}

	return nil
}
