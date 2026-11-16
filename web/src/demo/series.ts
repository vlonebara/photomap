import type { Coordinates, Photo, Series } from './types'

const radiusMetres = 100
const earthRadiusMetres = 6_371_000
const radians = Math.PI / 180

function distance(from: Coordinates, to: Coordinates) {
  const latitudeDelta = (to[1] - from[1]) * radians
  const longitudeDelta = (to[0] - from[0]) * radians
  const a = Math.sin(latitudeDelta / 2) ** 2
    + Math.cos(from[1] * radians) * Math.cos(to[1] * radians)
      * Math.sin(longitudeDelta / 2) ** 2

  return 2 * earthRadiusMetres * Math.asin(Math.min(1, Math.sqrt(a)))
}

function nearestSeries(groups: Series[], photo: Photo) {
  let nearest: Series | undefined
  let nearestDistance = radiusMetres

  for (const group of groups) {
    const metres = distance(photo.coordinates, group.coordinates)
    if (metres < nearestDistance ||
        (metres === nearestDistance && (!nearest || group.id < nearest.id))) {
      nearest = group
      nearestDistance = metres
    }
  }

  return nearest
}

export function makeSeries(photos: Photo[]): Series[] {
  const published = photos
    .filter(photo => photo.status === 'published')
    .sort((a, b) => b.stars - a.stars || a.id - b.id)
  const groups: Series[] = []

  for (const photo of published) {
    if (!nearestSeries(groups, photo)) {
      groups.push({
        id: photo.id,
        coordinates: photo.coordinates,
        title: photo.place,
        place: photo.region ?? photo.place,
        photos: [],
        stars: 0,
      })
    }
  }

  for (const photo of published) {
    const group = nearestSeries(groups, photo)
    if (group) {
      group.photos.push(photo)
      group.stars += photo.stars
    }
  }

  for (const group of groups) {
    group.photos.sort((a, b) => a.year - b.year || a.id - b.id)
  }

  return groups.sort((a, b) => b.stars - a.stars || a.id - b.id)
}
