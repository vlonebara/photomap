-- registration always creates an ordinary user
-- name: CreateUser :one
INSERT INTO users (name, email, password_hash)
VALUES (?, ?, ?)
RETURNING id, name, email, role;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, created_at, expires_at)
VALUES (?, ?, ?, ?);

-- current role comes from the database, not from JWT claims
-- name: GetSessionUser :one
SELECT u.id, u.name, u.email, u.role
FROM sessions AS s JOIN users AS u ON u.id = s.user_id
WHERE s.token_hash = ? AND s.expires_at > ?;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = ?;
