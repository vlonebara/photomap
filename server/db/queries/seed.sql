-- demonstration accounts cannot sign in with a password
-- name: EnsureDemoUser :exec
INSERT INTO users (name, email, password_hash) VALUES (?, ?, '!demo-login-disabled')
ON CONFLICT (email) DO NOTHING;

-- name: EnsureDemoPhoto :exec
INSERT INTO photos (id, author_id, title, description, place, longitude, latitude,
                    year, end_year, source_name, source_url, image_key, image_type,
                    image_bytes, status)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO NOTHING;

-- usable accounts are created only by the explicit demo import
-- name: EnsureDemoAccount :exec
INSERT INTO users (name, email, password_hash, role) VALUES (?, ?, ?, ?)
ON CONFLICT (email) DO NOTHING;
