package service

import (
	"sort"

	"github.com/golang/geo/s2"
)

const seriesRadius = 100.0
const earthRadius = 6371000.0

// measuring the same spherical distance as the existing frontend
func photoDistance(a, b Photo) float64 {
	from := s2.LatLngFromDegrees(a.Data.Latitude, a.Data.Longitude)
	to := s2.LatLngFromDegrees(b.Data.Latitude, b.Data.Longitude)

	return from.Distance(to).Radians() * earthRadius
}

// selecting the nearest center with a stable ID tie break
func nearestCenter(centers []Photo, photo Photo) int {
	nearest := -1
	distance := seriesRadius

	for i, center := range centers {
		metres := photoDistance(photo, center)
		if metres > distance {
			continue
		}

		if nearest >= 0 && metres == distance && center.Data.ID > centers[nearest].Data.ID {
			continue
		}

		nearest = i
		distance = metres
	}

	return nearest
}

// grouping published photos by ranked centers within 100 metres
func GroupPhotos(photos []Photo) []Series {
	ranked := append([]Photo(nil), photos...)
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Data.StarCount == ranked[j].Data.StarCount {
			return ranked[i].Data.ID < ranked[j].Data.ID
		}

		return ranked[i].Data.StarCount > ranked[j].Data.StarCount
	})

	// choosing centers before assigning photos to them
	centers := make([]Photo, 0)
	groups := make([]Series, 0)
	for _, photo := range ranked {
		if nearestCenter(centers, photo) >= 0 {
			continue
		}

		centers = append(centers, photo)
		groups = append(groups, Series{
			ID:          photo.Data.ID,
			Coordinates: [2]float64{photo.Data.Longitude, photo.Data.Latitude},
			Title:       photo.Data.Place,
			Place:       photo.Data.Place,
			Photos:      make([]Photo, 0),
		})
	}

	// assigning every photo to its nearest chosen center
	for _, photo := range ranked {
		index := nearestCenter(centers, photo)
		photo.SeriesID = groups[index].ID
		groups[index].Photos = append(groups[index].Photos, photo)
		groups[index].Stars += photo.Data.StarCount
	}

	// ordering the timeline independently of popularity
	for i := range groups {
		items := groups[i].Photos
		sort.Slice(items, func(a, b int) bool {
			if items[a].Data.Year == items[b].Data.Year {
				return items[a].Data.ID < items[b].Data.ID
			}

			return items[a].Data.Year < items[b].Data.Year
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Stars == groups[j].Stars {
			return groups[i].ID < groups[j].ID
		}

		return groups[i].Stars > groups[j].Stars
	})

	return groups
}
