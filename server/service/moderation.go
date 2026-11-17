package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/vlonebara/photomap/db"
)

// filtering moderator lists without changing publication rules
func (s *PhotoService) ModerationQueue(ctx context.Context, user *User, statuses string) ([]Photo, error) {
	if err := requireModerator(user); err != nil {
		return nil, err
	}

	if statuses == "" {
		statuses = "pending,review"
	}

	wanted := make(map[string]bool)
	for _, status := range strings.Split(statuses, ",") {
		switch status {
		case "pending", "review", "published", "rejected", "removed":
			wanted[status] = true
		default:
			return nil, invalid("status", "Неизвестный статус фото")
		}
	}

	rows, err := s.store.ListModerationPhotos(ctx)
	if err != nil {
		return nil, err
	}

	photos := make([]Photo, 0)
	for _, row := range rows {
		if wanted[row.Status] {
			photos = append(photos, managedPhoto(db.GetPhotoRow(row)))
		}
	}

	return photos, nil
}

// editing metadata without changing the photo status
func (s *PhotoService) Edit(ctx context.Context, id int64, user *User, draft PhotoDraft) (Photo, error) {
	if err := requireModerator(user); err != nil {
		return Photo{}, err
	}

	if err := validateDraft(&draft); err != nil {
		return Photo{}, err
	}

	err := s.change(ctx, func(q *db.Queries) error {
		row, err := q.GetPhoto(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}

		if err != nil {
			return err
		}

		if row.Status == "removed" {
			return ErrConflict
		}

		return q.EditPhoto(ctx, db.EditPhotoParams{
			ID:          id,
			Title:       draft.Title,
			Description: draft.Description,
			Place:       draft.Place,
			Longitude:   draft.Longitude,
			Latitude:    draft.Latitude,
			Year:        draft.Year,
			EndYear:     nullYear(draft.EndYear),
			SourceName:  nullText(draft.SourceName),
			SourceUrl:   nullText(draft.SourceURL),
		})
	})
	if err != nil {
		return Photo{}, err
	}

	return s.GetManaged(ctx, id, user, true)
}

// applying a moderator decision and resolving reports atomically
func (s *PhotoService) Decide(ctx context.Context, id int64, user *User, action string) (Photo, error) {
	if err := requireModerator(user); err != nil {
		return Photo{}, err
	}

	err := s.change(ctx, func(q *db.Queries) error {
		photo, err := q.GetPhoto(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}

		if err != nil {
			return err
		}

		status, err := decisionStatus(photo.Status, action)
		if err != nil {
			return err
		}

		if err := q.RecordModeration(ctx, db.RecordModerationParams{PhotoID: id, ModeratorID: user.ID, Action: action, PreviousStatus: photo.Status}); err != nil {
			return err
		}

		if err := q.SetPhotoStatus(ctx, db.SetPhotoStatusParams{Status: status, ID: id}); err != nil {
			return err
		}

		return q.ResolvePhotoReports(ctx, db.ResolvePhotoReportsParams{ResolvedAt: sql.NullInt64{Int64: time.Now().Unix(), Valid: true}, PhotoID: id})
	})
	if err != nil {
		return Photo{}, err
	}

	return s.GetManaged(ctx, id, user, true)
}

// allowing only explicit transitions from the current state
func decisionStatus(current, action string) (string, error) {
	switch action {
	case "approve":
		if current == "pending" {
			return "published", nil
		}
	case "restore":
		if current == "review" || current == "rejected" {
			return "published", nil
		}
	case "reject":
		if current == "pending" || current == "review" {
			return "rejected", nil
		}
	case "unpublish":
		if current == "published" {
			return "rejected", nil
		}
	default:
		return "", invalid("action", "Неизвестное решение модератора")
	}

	return "", ErrConflict
}
