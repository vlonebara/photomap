-- public photo card with its author
-- name: GetPublishedPhoto :one
SELECT p.id, p.author_id, p.title, p.description, p.place,
       p.longitude, p.latitude, p.year, p.end_year,
       p.source_name, p.source_url, p.star_count, u.name AS author_name
FROM public_photos AS p
JOIN users AS u ON u.id = p.author_id
WHERE p.id = ?;

-- image metadata is available only while the photo is published
-- name: GetPublishedPhotoImage :one
SELECT image_key, image_type, image_bytes
FROM photos
WHERE id = ? AND status = 'published';

-- ranked public points for selecting current series centers
-- name: ListPublishedPhotoPoints :many
SELECT id, longitude, latitude, star_count
FROM photos
WHERE status = 'published'
ORDER BY star_count DESC, id;
