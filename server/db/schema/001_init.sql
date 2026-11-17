-- initial schema copied from the existing photomap.db
-- run once for a new database before starting the server
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = FULL;
PRAGMA journal_mode = WAL;

BEGIN;

CREATE TABLE users (
    id            INTEGER PRIMARY KEY,
    name          TEXT NOT NULL CHECK (length(trim(name)) > 0),
    email          TEXT NOT NULL UNIQUE CHECK (length(trim(email)) > 0),
    password_hash TEXT NOT NULL CHECK (length(password_hash) > 0),
    role          TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'moderator')),
    created_at    INTEGER NOT NULL DEFAULT (unixepoch())
) STRICT;
CREATE TABLE sessions (
    token_hash BLOB PRIMARY KEY CHECK (length(token_hash) = 32),
    user_id    INTEGER NOT NULL REFERENCES users(id),
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    expires_at INTEGER NOT NULL CHECK (expires_at > created_at)
) STRICT, WITHOUT ROWID;
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);
CREATE TABLE photos (
    id            INTEGER PRIMARY KEY,
    author_id     INTEGER NOT NULL REFERENCES users(id),
    title         TEXT NOT NULL CHECK (length(trim(title)) > 0),
    description   TEXT NOT NULL CHECK (length(trim(description)) > 0),
    place         TEXT NOT NULL CHECK (length(trim(place)) > 0),
    longitude     REAL NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    latitude      REAL NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    year          INTEGER NOT NULL CHECK (year >= 1),
    end_year      INTEGER CHECK (end_year >= year AND end_year - year <= 50),
    source_name   TEXT,
    source_url    TEXT,
    image_key     TEXT NOT NULL UNIQUE CHECK (length(trim(image_key)) > 0),
    image_type    TEXT NOT NULL CHECK (length(trim(image_type)) > 0),
    image_bytes   INTEGER NOT NULL CHECK (image_bytes > 0),
    thumbnail_key TEXT CHECK (length(trim(thumbnail_key)) > 0),
    status        TEXT NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending', 'published', 'review', 'rejected', 'removed')),
    star_count    INTEGER NOT NULL DEFAULT 0 CHECK (star_count >= 0),
    created_at    INTEGER NOT NULL DEFAULT (unixepoch())
) STRICT;
CREATE INDEX photos_map_rank_idx
    ON photos (star_count DESC, id, longitude, latitude, year)
    WHERE status = 'published';
CREATE INDEX photos_author_feed_idx ON photos (author_id, id DESC);
CREATE INDEX photos_moderation_queue_idx ON photos (status, id);
CREATE VIEW public_photos AS
    SELECT id, author_id, title, description, place, longitude, latitude,
           year, end_year, source_name, source_url, star_count
    FROM photos
    WHERE status = 'published';
CREATE TABLE photo_stars (
    photo_id   INTEGER NOT NULL REFERENCES photos(id),
    user_id    INTEGER NOT NULL REFERENCES users(id),
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (photo_id, user_id)
) STRICT, WITHOUT ROWID;
CREATE TABLE photo_reports (
    photo_id    INTEGER NOT NULL REFERENCES photos(id),
    reporter_id INTEGER NOT NULL REFERENCES users(id),
    created_at  INTEGER NOT NULL DEFAULT (unixepoch()),
    resolved_at INTEGER CHECK (resolved_at >= created_at),
    PRIMARY KEY (photo_id, reporter_id)
) STRICT, WITHOUT ROWID;
CREATE INDEX photo_reports_active_idx
    ON photo_reports (photo_id)
    WHERE resolved_at IS NULL;
CREATE TABLE photo_moderation (
    id              INTEGER PRIMARY KEY,
    photo_id        INTEGER NOT NULL REFERENCES photos(id),
    moderator_id    INTEGER NOT NULL REFERENCES users(id),
    action          TEXT NOT NULL CHECK (action IN ('approve', 'reject', 'restore', 'unpublish')),
    previous_status TEXT NOT NULL CHECK (previous_status IN ('pending', 'published', 'review', 'rejected')),
    created_at      INTEGER NOT NULL DEFAULT (unixepoch())
) STRICT;
CREATE INDEX photo_moderation_history_idx ON photo_moderation (photo_id, id DESC);
CREATE TABLE map_state (
    id       INTEGER PRIMARY KEY CHECK (id = 1),
    revision INTEGER NOT NULL CHECK (revision >= 0)
) STRICT;
CREATE TRIGGER photos_validate_status_change
BEFORE UPDATE OF status ON photos
WHEN NEW.status <> OLD.status AND NOT (
    (OLD.status <> 'removed' AND NEW.status = 'removed')
    OR (OLD.status IN ('pending', 'review', 'rejected') AND NEW.status = 'published')
    OR (OLD.status IN ('pending', 'published', 'review') AND NEW.status = 'rejected')
    OR (OLD.status = 'published' AND NEW.status = 'review')
)
BEGIN
    SELECT RAISE(ABORT, 'invalid photo status transition');
END;
CREATE TRIGGER photo_stars_count_insert
AFTER INSERT ON photo_stars
BEGIN
    UPDATE photos SET star_count = star_count + 1 WHERE id = NEW.photo_id;
END;
CREATE TRIGGER photo_stars_count_delete
AFTER DELETE ON photo_stars
BEGIN
    UPDATE photos SET star_count = star_count - 1 WHERE id = OLD.photo_id;
END;
CREATE TRIGGER photo_stars_keep_identity
BEFORE UPDATE OF photo_id, user_id ON photo_stars
WHEN NEW.photo_id <> OLD.photo_id OR NEW.user_id <> OLD.user_id
BEGIN
    SELECT RAISE(ABORT, 'star identity cannot change');
END;
CREATE TRIGGER photo_reports_require_published
BEFORE INSERT ON photo_reports
WHEN NEW.resolved_at IS NULL
    AND (SELECT status FROM photos WHERE id = NEW.photo_id) IS NOT 'published'
BEGIN
    SELECT RAISE(ABORT, 'active reports require a published photo');
END;
CREATE TRIGGER photo_reports_hide_after_third
AFTER INSERT ON photo_reports
WHEN NEW.resolved_at IS NULL AND (
    SELECT count(*) FROM photo_reports
    WHERE photo_id = NEW.photo_id AND resolved_at IS NULL
) >= 3
BEGIN
    UPDATE photos SET status = 'review' WHERE id = NEW.photo_id;
END;
CREATE TRIGGER photo_reports_keep_history
BEFORE UPDATE ON photo_reports
WHEN NEW.photo_id IS NOT OLD.photo_id OR NEW.reporter_id IS NOT OLD.reporter_id
    OR NEW.created_at IS NOT OLD.created_at
    OR (OLD.resolved_at IS NOT NULL AND NEW.resolved_at IS NOT OLD.resolved_at)
BEGIN
    SELECT RAISE(ABORT, 'only an active report can be resolved');
END;
CREATE TRIGGER photos_revision_insert
AFTER INSERT ON photos
WHEN NEW.status = 'published'
BEGIN
    UPDATE map_state SET revision = revision + 1 WHERE id = 1;
END;
CREATE TRIGGER photos_revision_update
AFTER UPDATE ON photos
WHEN (OLD.status = 'published' OR NEW.status = 'published') AND (
    OLD.status IS NOT NEW.status OR OLD.id IS NOT NEW.id
    OR OLD.author_id IS NOT NEW.author_id
    OR OLD.title IS NOT NEW.title OR OLD.description IS NOT NEW.description
    OR OLD.place IS NOT NEW.place
    OR OLD.longitude IS NOT NEW.longitude OR OLD.latitude IS NOT NEW.latitude
    OR OLD.year IS NOT NEW.year OR OLD.end_year IS NOT NEW.end_year
    OR OLD.source_name IS NOT NEW.source_name OR OLD.source_url IS NOT NEW.source_url
    OR OLD.image_key IS NOT NEW.image_key OR OLD.thumbnail_key IS NOT NEW.thumbnail_key
    OR OLD.image_type IS NOT NEW.image_type OR OLD.image_bytes IS NOT NEW.image_bytes
    OR OLD.star_count IS NOT NEW.star_count
)
BEGIN
    UPDATE map_state SET revision = revision + 1 WHERE id = 1;
END;
CREATE TRIGGER photos_revision_delete
AFTER DELETE ON photos
WHEN OLD.status = 'published'
BEGIN
    UPDATE map_state SET revision = revision + 1 WHERE id = 1;
END;

-- initial public map revision and database version
INSERT INTO map_state (id, revision) VALUES (1, 0);
PRAGMA user_version = 1;

COMMIT;

