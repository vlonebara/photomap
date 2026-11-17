package service

import (
	"context"
	"database/sql"
	"errors"
	"io"

	"github.com/vlonebara/photomap/db"
)

// converting a private row without exposing storage keys
func managedPhoto(row db.GetPhotoRow) Photo {
	return Photo{
		Status: row.Status,
		Data: db.GetPublishedPhotoRow{
			ID:          row.ID,
			AuthorID:    row.AuthorID,
			AuthorName:  row.AuthorName,
			Title:       row.Title,
			Description: row.Description,
			Place:       row.Place,
			Longitude:   row.Longitude,
			Latitude:    row.Latitude,
			Year:        row.Year,
			EndYear:     row.EndYear,
			SourceName:  row.SourceName,
			SourceUrl:   row.SourceUrl,
			StarCount:   row.StarCount,
		},
	}
}

// checking ownership or moderator rights before private access
func (s *PhotoService) managed(ctx context.Context, id int64, user *User, moderator bool) (db.GetPhotoRow, error) {
	if user == nil {
		return db.GetPhotoRow{}, ErrUnauthorized
	}

	if moderator {
		if err := requireModerator(user); err != nil {
			return db.GetPhotoRow{}, err
		}
	}

	row, err := s.store.GetPhoto(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return row, ErrNotFound
	}

	if err != nil {
		return row, err
	}

	if !moderator {
		if err := requireOwner(user, row.AuthorID); err != nil {
			return row, err
		}
	}

	return row, nil
}

func (s *PhotoService) GetManaged(ctx context.Context, id int64, user *User, moderator bool) (Photo, error) {
	row, err := s.managed(ctx, id, user, moderator)
	if err != nil {
		return Photo{}, err
	}

	return managedPhoto(row), nil
}

func (s *PhotoService) ListOwn(ctx context.Context, user *User) ([]Photo, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}

	rows, err := s.store.ListOwnPhotos(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	photos := make([]Photo, 0, len(rows))
	for _, row := range rows {
		photos = append(photos, managedPhoto(db.GetPhotoRow(row)))
	}

	return photos, nil
}

// accepting an upload only after validating all metadata and image bytes
func (s *PhotoService) Upload(ctx context.Context, user *User, draft PhotoDraft, reader io.Reader) (Photo, error) {
	if user == nil {
		return Photo{}, ErrUnauthorized
	}

	if err := validateDraft(&draft); err != nil {
		return Photo{}, err
	}

	image, err := s.media.Save(reader)
	if err != nil {
		return Photo{}, err
	}

	var id int64
	err = s.change(ctx, func(q *db.Queries) error {
		var err error
		id, err = q.CreatePhoto(ctx, db.CreatePhotoParams{
			AuthorID:     user.ID,
			Title:        draft.Title,
			Description:  draft.Description,
			Place:        draft.Place,
			Longitude:    draft.Longitude,
			Latitude:     draft.Latitude,
			Year:         draft.Year,
			EndYear:      nullYear(draft.EndYear),
			SourceName:   nullText(draft.SourceName),
			SourceUrl:    nullText(draft.SourceURL),
			ImageKey:     image.Key,
			ImageType:    image.Type,
			ImageBytes:   image.Bytes,
			ThumbnailKey: sql.NullString{String: image.ThumbnailKey, Valid: true},
		})
		return err
	})
	if err != nil {
		return Photo{}, err
	}

	return s.GetManaged(ctx, id, user, false)
}

// removing public access while preserving the record, files and reactions
func (s *PhotoService) Remove(ctx context.Context, id int64, user *User) error {
	return s.change(ctx, func(q *db.Queries) error {
		row, err := q.GetPhoto(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}

		if err != nil {
			return err
		}

		if err := requireOwner(user, row.AuthorID); err != nil {
			return err
		}

		if row.Status == "removed" {
			return nil
		}

		return q.SetPhotoStatus(ctx, db.SetPhotoStatusParams{Status: "removed", ID: id})
	})
}

func (s *PhotoService) PrivateImage(ctx context.Context, id int64, user *User, moderator, thumbnail bool) (Image, error) {
	row, err := s.managed(ctx, id, user, moderator)
	if err != nil {
		return Image{}, err
	}

	key, contentType := row.ImageKey, row.ImageType
	if thumbnail && row.ThumbnailKey.Valid {
		key, contentType = row.ThumbnailKey.String, "image/jpeg"
	}

	return s.media.Open(key, contentType)
}
