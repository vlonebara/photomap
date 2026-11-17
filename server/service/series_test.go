package service_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/vlonebara/photomap/db"
	"github.com/vlonebara/photomap/service"
)

func point(id, stars, year int64, metres float64) service.Photo {
	longitude := metres / 6371000 * 180 / math.Pi
	return service.Photo{Data: db.GetPublishedPhotoRow{
		ID:        id,
		StarCount: stars,
		Year:      year,
		Longitude: longitude,
	}}
}

func TestSeriesRules(t *testing.T) {
	t.Parallel()

	photos := []service.Photo{
		point(20, 3, 1900, -80),
		point(10, 2, 1900, 80),
		point(30, 0, 1850, 0),
	}

	groups := service.GroupPhotos(photos)
	if len(groups) != 2 {
		t.Fatalf("center count = %d", len(groups))
	}

	var selected int64
	for _, group := range groups {
		for _, photo := range group.Photos {
			if photo.Data.ID == 30 {
				selected = photo.SeriesID
				if group.Photos[0].Data.ID != 30 {
					t.Fatal("photos are not ordered by year")
				}
			}
		}
	}

	if selected != 10 {
		t.Fatalf("equal-distance center = %d, want smaller ID 10", selected)
	}
}

func TestSeriesRadiusAndRanking(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		metres float64
		count  int
	}{
		{name: "inside", metres: 99.999, count: 1},
		{name: "inclusive boundary", metres: 100, count: 1},
		{name: "outside", metres: 100.001, count: 2},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			groups := service.GroupPhotos([]service.Photo{point(2, 0, 1900, test.metres), point(1, 0, 1900, 0)})
			if len(groups) != test.count || groups[0].ID != 1 {
				t.Fatalf("unexpected centers: %+v", groups)
			}
		})
	}
}

// measuring the actual grouping cost with a populated district
func BenchmarkGroupPhotos(b *testing.B) {
	for _, count := range []int{100, 1000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			photos := make([]service.Photo, count)
			for index := range photos {
				photos[index] = point(int64(index+1), int64(index%30), 1900, float64(index)*2)
			}

			b.ReportAllocs()
			for b.Loop() {
				service.GroupPhotos(photos)
			}
		})
	}
}
