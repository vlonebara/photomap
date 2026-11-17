package handlers

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/vlonebara/photomap/service"
)

type photoResponse struct {
	ID           int64      `json:"id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Place        string     `json:"place"`
	Coordinates  [2]float64 `json:"coordinates"`
	Year         int64      `json:"year"`
	EndYear      *int64     `json:"endYear,omitempty"`
	SourceName   *string    `json:"sourceName,omitempty"`
	SourceURL    *string    `json:"sourceUrl,omitempty"`
	ImageURL     string     `json:"imageUrl"`
	ThumbnailURL string     `json:"thumbnailUrl"`
	AuthorID     string     `json:"authorId"`
	AuthorName   string     `json:"authorName"`
	Stars        int64      `json:"stars"`
	SeriesID     int64      `json:"seriesId,omitempty"`
	Status       string     `json:"status,omitempty"`
	StarredByMe  *bool      `json:"starredByMe,omitempty"`
	ReportedByMe *bool      `json:"reportedByMe,omitempty"`
}

type seriesResponse struct {
	ID           int64      `json:"id"`
	Coordinates  [2]float64 `json:"coordinates"`
	Title        string     `json:"title"`
	Place        string     `json:"place"`
	PhotoCount   int        `json:"photoCount"`
	Stars        int64      `json:"stars"`
	ThumbnailURL string     `json:"thumbnailUrl"`
}

type seriesDetailResponse struct {
	seriesResponse
	Photos []photoResponse `json:"photos"`
}

func textPointer(value sql.NullString) *string {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}

	return &value.String
}

func photoJSON(photo service.Photo, prefix string) photoResponse {
	row := photo.Data
	response := photoResponse{
		ID:           row.ID,
		Title:        row.Title,
		Description:  row.Description,
		Place:        row.Place,
		Coordinates:  [2]float64{row.Longitude, row.Latitude},
		Year:         row.Year,
		SourceName:   textPointer(row.SourceName),
		SourceURL:    textPointer(row.SourceUrl),
		ImageURL:     fmt.Sprintf("%s/%d/image", prefix, row.ID),
		ThumbnailURL: fmt.Sprintf("%s/%d/thumbnail", prefix, row.ID),
		AuthorID:     fmt.Sprint(row.AuthorID),
		AuthorName:   row.AuthorName,
		Stars:        row.StarCount,
		SeriesID:     photo.SeriesID,
		StarredByMe:  photo.StarredByMe,
		ReportedByMe: photo.ReportedByMe,
	}

	if row.EndYear.Valid {
		response.EndYear = &row.EndYear.Int64
	}

	if prefix != "/api/photos" {
		response.Status = photo.Status
	}

	return response
}

func photosJSON(photos []service.Photo, prefix string) []photoResponse {
	result := make([]photoResponse, 0, len(photos))
	for _, photo := range photos {
		result = append(result, photoJSON(photo, prefix))
	}

	return result
}

func seriesJSON(series service.Series) seriesResponse {
	return seriesResponse{
		ID:           series.ID,
		Coordinates:  series.Coordinates,
		Title:        series.Title,
		Place:        series.Place,
		PhotoCount:   len(series.Photos),
		Stars:        series.Stars,
		ThumbnailURL: fmt.Sprintf("/api/photos/%d/thumbnail", series.ID),
	}
}
