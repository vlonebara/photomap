package demo

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vlonebara/photomap/db"
)

func seedPhoto(ctx context.Context, q *db.Queries, photo archivePhoto, author int64, directory, status string) (bool, error) {
	_, existingError := q.GetPhoto(ctx, photo.ID)
	if existingError != nil && !errors.Is(existingError, sql.ErrNoRows) {
		return false, existingError
	}

	file, err := os.Stat(filepath.Join(directory, photo.ImageKey))
	if err != nil {
		return false, err
	}

	var endYear sql.NullInt64
	if photo.EndYear != nil {
		endYear = sql.NullInt64{Int64: *photo.EndYear, Valid: true}
	}

	err = q.EnsureDemoPhoto(ctx, db.EnsureDemoPhotoParams{
		ID:          photo.ID,
		AuthorID:    author,
		Title:       photo.Title,
		Description: photo.Description,
		Place:       photo.Place,
		Longitude:   photo.Coordinates[0],
		Latitude:    photo.Coordinates[1],
		Year:        photo.Year,
		EndYear:     endYear,
		SourceName:  sql.NullString{String: photo.SourceName, Valid: photo.SourceName != ""},
		SourceUrl:   sql.NullString{String: photo.SourceURL, Valid: photo.SourceURL != ""},
		ImageKey:    photo.ImageKey,
		ImageType:   "image/jpeg",
		ImageBytes:  file.Size(),
		Status:      status,
	})
	if err != nil {
		return false, err
	}

	stored, err := q.GetPhoto(ctx, photo.ID)
	if err != nil {
		return false, err
	}

	if stored.AuthorID != author || stored.ImageKey != photo.ImageKey {
		return false, fmt.Errorf("photo ID %d is already in use", photo.ID)
	}

	return errors.Is(existingError, sql.ErrNoRows), nil
}

// keeping demo counters consistent with actual photo_stars rows
func seedArchivePhoto(ctx context.Context, q *db.Queries, photo archivePhoto, author int64, directory string) error {
	if _, err := seedPhoto(ctx, q, photo, author, directory, "published"); err != nil {
		return err
	}

	for number := int64(1); number <= photo.Stars; number++ {
		email := fmt.Sprintf("star-%02d@demo.photomap.invalid", number)
		name := fmt.Sprintf("Демо-голос %02d", number)
		userID, err := disabledUser(ctx, q, name, email)
		if err != nil {
			return err
		}

		if err := q.AddPhotoStar(ctx, db.AddPhotoStarParams{PhotoID: photo.ID, UserID: userID}); err != nil {
			return err
		}
	}

	return nil
}

// copying a fixture without deleting or replacing the original
func copyImage(directory, source, target string) error {
	data, err := os.ReadFile(filepath.Join(directory, source))
	if err != nil {
		return err
	}

	path := filepath.Join(directory, target)
	existing, err := os.ReadFile(path)
	if err == nil {
		if !bytes.Equal(existing, data) {
			return fmt.Errorf("demo image already exists with different content: %s", target)
		}

		return nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}

	_, writeError := file.Write(data)
	return errors.Join(writeError, file.Close())
}
