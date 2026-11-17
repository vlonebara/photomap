package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/vlonebara/photomap/db"
)

type StarState struct {
	Stars       int64 `json:"stars"`
	StarredByMe bool  `json:"starredByMe"`
}

// adding and removing a star through the existing counter triggers
func (s *PhotoService) SetStar(ctx context.Context, id int64, user *User, enabled bool) (StarState, error) {
	if user == nil {
		return StarState{}, ErrUnauthorized
	}

	var result StarState
	err := s.change(ctx, func(q *db.Queries) error {
		if _, err := q.GetPublishedPhoto(ctx, id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}

			return err
		}

		var err error
		if enabled {
			err = q.AddPhotoStar(ctx, db.AddPhotoStarParams{PhotoID: id, UserID: user.ID})
		} else {
			err = q.RemovePhotoStar(ctx, db.RemovePhotoStarParams{PhotoID: id, UserID: user.ID})
		}

		if err != nil {
			return err
		}

		photo, err := q.GetPublishedPhoto(ctx, id)
		if err != nil {
			return err
		}

		result = StarState{Stars: photo.StarCount, StarredByMe: enabled}
		return nil
	})
	return result, err
}

// a report remains unique even after it has been resolved
func (s *PhotoService) Report(ctx context.Context, id int64, user *User) error {
	if user == nil {
		return ErrUnauthorized
	}

	return s.change(ctx, func(q *db.Queries) error {
		state, err := q.GetPhotoViewerState(ctx, db.GetPhotoViewerStateParams{PhotoID: id, UserID: user.ID})
		if err != nil {
			return err
		}

		if state.Reported {
			return ErrAlreadyReported
		}

		if _, err := q.GetPublishedPhoto(ctx, id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}

			return err
		}

		return q.AddPhotoReport(ctx, db.AddPhotoReportParams{PhotoID: id, ReporterID: user.ID})
	})
}
