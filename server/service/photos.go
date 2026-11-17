package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/vlonebara/photomap/db"
)

type PhotoService struct {
	store  *db.Store
	media  *Media
	events *Events
}

func NewPhotoService(store *db.Store, mediaDirectory string, events *Events) *PhotoService {
	return &PhotoService{store: store, media: NewMedia(mediaDirectory), events: events}
}

// reading all public cards from one consistent SQLite statement
func publicGroups(ctx context.Context, q *db.Queries) ([]Series, error) {
	rows, err := q.ListPublishedPhotos(ctx)
	if err != nil {
		return nil, err
	}

	photos := make([]Photo, 0, len(rows))
	for _, row := range rows {
		photos = append(photos, Photo{
			Data:   db.GetPublishedPhotoRow(row),
			Status: "published",
		})
	}

	return GroupPhotos(photos), nil
}

// notifying clients only after a successful commit
func (s *PhotoService) change(ctx context.Context, apply func(*db.Queries) error) error {
	var revision int64
	err := s.store.WithTx(ctx, func(q *db.Queries) error {
		if err := apply(q); err != nil {
			return err
		}

		var err error
		revision, err = q.GetMapRevision(ctx)
		return err
	})
	if err == nil {
		s.events.Publish(revision)
	}

	return err
}

// using one database snapshot for card details and current centers
func (s *PhotoService) read(ctx context.Context, apply func(*db.Queries) error) error {
	tx, err := s.store.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}

	defer tx.Rollback()
	if err := apply(s.store.Queries.WithTx(tx)); err != nil {
		return err
	}

	return tx.Commit()
}

// reading lifetime report state and current star state for a signed-in viewer
func addViewerState(ctx context.Context, q *db.Queries, photo *Photo, user *User) error {
	if user == nil {
		return nil
	}

	state, err := q.GetPhotoViewerState(ctx, db.GetPhotoViewerStateParams{PhotoID: photo.Data.ID, UserID: user.ID})
	if err != nil {
		return err
	}

	photo.StarredByMe = &state.Starred
	photo.ReportedByMe = &state.Reported
	return nil
}

func (s *PhotoService) ListSeries(ctx context.Context) ([]Series, error) {
	return publicGroups(ctx, s.store.Queries)
}

func (s *PhotoService) GetSeries(ctx context.Context, id int64, user *User) (Series, error) {
	var result Series
	err := s.read(ctx, func(q *db.Queries) error {
		groups, err := publicGroups(ctx, q)
		if err != nil {
			return err
		}

		for _, group := range groups {
			if group.ID != id {
				continue
			}

			for i := range group.Photos {
				if err := addViewerState(ctx, q, &group.Photos[i], user); err != nil {
					return err
				}
			}

			result = group
			return nil
		}

		return ErrNotFound
	})
	return result, err
}

func (s *PhotoService) GetPublishedPhoto(ctx context.Context, id int64, user *User) (Photo, error) {
	var result Photo
	err := s.read(ctx, func(q *db.Queries) error {
		row, err := q.GetPublishedPhoto(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}

		if err != nil {
			return err
		}

		groups, err := publicGroups(ctx, q)
		if err != nil {
			return err
		}

		result = Photo{Data: row, Status: "published"}
		for _, group := range groups {
			for _, photo := range group.Photos {
				if photo.Data.ID == id {
					result.SeriesID = group.ID
				}
			}
		}

		return addViewerState(ctx, q, &result, user)
	})
	return result, err
}

// public media is checked against the current status on every request
func (s *PhotoService) PublicImage(ctx context.Context, id int64, thumbnail bool) (Image, error) {
	row, err := s.store.GetPublishedPhotoImage(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Image{}, ErrNotFound
	}

	if err != nil {
		return Image{}, err
	}

	key, contentType := row.ImageKey, row.ImageType
	if thumbnail {
		photo, err := s.store.GetPhoto(ctx, id)
		if err != nil {
			return Image{}, err
		}

		if photo.Status != "published" {
			return Image{}, ErrNotFound
		}

		if photo.ThumbnailKey.Valid {
			key, contentType = photo.ThumbnailKey.String, "image/jpeg"
		}
	}

	return s.media.Open(key, contentType)
}
