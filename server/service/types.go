package service

import "github.com/vlonebara/photomap/db"

// identity loaded from the database for the current request
type User struct {
	ID    int64
	Name  string
	Email string
	Role  string
}

// public card data and optional personal reaction flags
type Photo struct {
	Data         db.GetPublishedPhotoRow
	SeriesID     int64
	Status       string
	StarredByMe  *bool
	ReportedByMe *bool
}

type Series struct {
	ID          int64
	Coordinates [2]float64
	Title       string
	Place       string
	Stars       int64
	Photos      []Photo
}

// editable fields used by uploads and moderator corrections
type PhotoDraft struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Place       string  `json:"place"`
	Longitude   float64 `json:"longitude"`
	Latitude    float64 `json:"latitude"`
	Year        int64   `json:"year"`
	EndYear     *int64  `json:"endYear"`
	SourceName  *string `json:"sourceName"`
	SourceURL   *string `json:"sourceUrl"`
}
