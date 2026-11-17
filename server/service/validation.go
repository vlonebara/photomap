package service

import (
	"database/sql"
	"math"
	"net/url"
	"strings"
)

// normalizing optional text without inventing missing values
func optionalText(value *string) *string {
	if value == nil {
		return nil
	}

	text := strings.TrimSpace(*value)
	if text == "" {
		return nil
	}

	return &text
}

// validating the same draft for uploads and moderator edits
func validateDraft(draft *PhotoDraft) error {
	draft.Title = strings.TrimSpace(draft.Title)
	draft.Description = strings.TrimSpace(draft.Description)
	draft.Place = strings.TrimSpace(draft.Place)
	if draft.Title == "" {
		return invalid("title", "Укажите название")
	}

	if draft.Description == "" {
		return invalid("description", "Добавьте описание")
	}

	if draft.Place == "" {
		return invalid("place", "Укажите место")
	}

	if !validCoordinate(draft.Longitude, -180, 180) {
		return invalid("longitude", "Долгота должна быть от -180 до 180")
	}

	if !validCoordinate(draft.Latitude, -90, 90) {
		return invalid("latitude", "Широта должна быть от -90 до 90")
	}

	if draft.Year < 1 {
		return invalid("year", "Укажите положительный год")
	}

	if draft.EndYear != nil {
		if *draft.EndYear < draft.Year || *draft.EndYear-draft.Year > 50 {
			return invalid("endYear", "Диапазон должен быть от 0 до 50 лет")
		}
	}

	draft.SourceName = optionalText(draft.SourceName)
	draft.SourceURL = optionalText(draft.SourceURL)
	if draft.SourceURL != nil {
		link, err := url.Parse(*draft.SourceURL)
		if err != nil || link.Host == "" || (link.Scheme != "http" && link.Scheme != "https") {
			return invalid("sourceUrl", "Источник должен быть ссылкой HTTP или HTTPS")
		}
	}

	return nil
}

func validCoordinate(value, minimum, maximum float64) bool {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return false
	}

	return value >= minimum && value <= maximum
}

// converting optional values for generated SQLite queries
func nullText(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}

	return sql.NullString{String: *value, Valid: true}
}

func nullYear(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}

	return sql.NullInt64{Int64: *value, Valid: true}
}
