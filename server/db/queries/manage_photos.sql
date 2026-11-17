-- name: ListPublishedPhotos :many
SELECT p.id, p.author_id, p.title, p.description, p.place,
       p.longitude, p.latitude, p.year, p.end_year,
       p.source_name, p.source_url, p.star_count, u.name AS author_name
FROM public_photos AS p JOIN users AS u ON u.id = p.author_id
ORDER BY p.star_count DESC, p.id;

-- name: GetPhoto :one
SELECT p.*, u.name AS author_name
FROM photos AS p JOIN users AS u ON u.id = p.author_id
WHERE p.id = ?;

-- name: ListOwnPhotos :many
SELECT p.*, u.name AS author_name
FROM photos AS p JOIN users AS u ON u.id = p.author_id
WHERE p.author_id = ? AND p.status <> 'removed' ORDER BY p.id DESC;

-- name: ListModerationPhotos :many
SELECT p.*, u.name AS author_name
FROM photos AS p JOIN users AS u ON u.id = p.author_id
ORDER BY p.id DESC;

-- new uploads are never published immediately
-- name: CreatePhoto :one
INSERT INTO photos (author_id, title, description, place, longitude, latitude,
                    year, end_year, source_name, source_url,
                    image_key, image_type, image_bytes, thumbnail_key)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: EditPhoto :exec
UPDATE photos SET title = ?, description = ?, place = ?, longitude = ?, latitude = ?,
                  year = ?, end_year = ?, source_name = ?, source_url = ?
WHERE id = ?;

-- name: SetPhotoStatus :exec
UPDATE photos SET status = ? WHERE id = ?;

-- name: RecordModeration :exec
INSERT INTO photo_moderation (photo_id, moderator_id, action, previous_status)
VALUES (?, ?, ?, ?);

-- report history is preserved after a moderation decision
-- name: ResolvePhotoReports :exec
UPDATE photo_reports SET resolved_at = ? WHERE photo_id = ? AND resolved_at IS NULL;

-- name: GetMapRevision :one
SELECT revision FROM map_state WHERE id = 1;
