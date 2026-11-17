-- triggers maintain the counter; repeated stars do nothing
-- name: AddPhotoStar :exec
INSERT INTO photo_stars (photo_id, user_id) VALUES (?, ?)
ON CONFLICT (photo_id, user_id) DO NOTHING;

-- name: RemovePhotoStar :exec
DELETE FROM photo_stars WHERE photo_id = ? AND user_id = ?;

-- resolved reports still count as already reported
-- name: GetPhotoViewerState :one
SELECT EXISTS(SELECT 1 FROM photo_stars AS s
              WHERE s.photo_id = sqlc.arg(photo_id) AND s.user_id = sqlc.arg(user_id)) AS starred,
       EXISTS(SELECT 1 FROM photo_reports AS r
              WHERE r.photo_id = sqlc.arg(photo_id) AND r.reporter_id = sqlc.arg(user_id)) AS reported;

-- the third active report hides the photo through the existing trigger
-- name: AddPhotoReport :exec
INSERT INTO photo_reports (photo_id, reporter_id) VALUES (?, ?);
