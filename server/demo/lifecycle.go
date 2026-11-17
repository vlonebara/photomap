package demo

import (
	"context"
	"fmt"

	"github.com/vlonebara/photomap/db"
)

// adding visible examples for the author cabinet and moderation queue
func seedLifecycle(ctx context.Context, q *db.Queries, photos []archivePhoto, users []int64, directory string) error {
	statuses := []string{"published", "pending", "review", "rejected", "removed"}
	for index, status := range statuses {
		photo := photos[index]
		source := photo.ImageKey
		photo.ID = int64(109 + index)
		photo.Stars = 0
		photo.ImageKey = fmt.Sprintf("demo-%d-%s", photo.ID, source)
		photo.Description += " Демонстрационная запись для проверки статуса " + status + "."

		if err := copyImage(directory, source, photo.ImageKey); err != nil {
			return err
		}

		initialStatus := "pending"
		if status == "review" {
			initialStatus = "published"
		}

		created, err := seedPhoto(ctx, q, photo, users[1], directory, initialStatus)
		if err != nil {
			return err
		}

		if !created || status == "pending" {
			continue
		}

		if status == "review" {
			for _, userID := range users[1:] {
				if err := q.AddPhotoReport(ctx, db.AddPhotoReportParams{PhotoID: photo.ID, ReporterID: userID}); err != nil {
					return err
				}
			}

			continue
		}

		if status != "removed" {
			action := "reject"
			if status == "published" {
				action = "approve"
			}

			err := q.RecordModeration(ctx, db.RecordModerationParams{
				PhotoID:        photo.ID,
				ModeratorID:    users[0],
				Action:         action,
				PreviousStatus: "pending",
			})
			if err != nil {
				return err
			}
		}

		if err := q.SetPhotoStatus(ctx, db.SetPhotoStatusParams{Status: status, ID: photo.ID}); err != nil {
			return err
		}
	}

	return nil
}
