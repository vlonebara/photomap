import type { Coordinates, Photo, PhotoInput, PhotoStatus } from '../api/types'

export interface PhotoFieldsValue {
  title: string
  place: string
  description: string
  year: number | ''
  endYear: number | ''
  sourceName: string
  sourceUrl: string
}

export function photoFields(photo?: Photo): PhotoFieldsValue {
  return {
    title: photo?.title ?? '',
    place: photo?.place ?? '',
    description: photo?.description ?? '',
    year: photo?.year ?? '',
    endYear: photo?.endYear ?? '',
    sourceName: photo?.sourceName ?? '',
    sourceUrl: photo?.sourceUrl ?? '',
  }
}

export function toPhotoInput(fields: PhotoFieldsValue, coordinates: Coordinates | null): PhotoInput {
  if (!coordinates) {
    throw new Error('Отметьте на карте объект, который виден на фото.')
  }
  if (fields.year === '') {
    throw new Error('Укажите год съёмки.')
  }
  const [longitude, latitude] = coordinates

  return {
    title: fields.title.trim(),
    place: fields.place.trim(),
    description: fields.description.trim(),
    year: fields.year,
    endYear: fields.endYear === '' ? null : fields.endYear,
    longitude,
    latitude,
    sourceName: fields.sourceName.trim() || null,
    sourceUrl: fields.sourceUrl.trim() || null,
  }
}

// send only changed fields so edits do not overwrite other values
export function photoChanges(input: PhotoInput, original: Photo): Partial<PhotoInput> {
  const previous = toPhotoInput(photoFields(original), original.coordinates)
  const changes = Object.entries(input).filter(([field, value]) => {
    return previous[field as keyof PhotoInput] !== value
  })
  return Object.fromEntries(changes)
}

export const photoStatuses: Record<PhotoStatus, string> = {
  pending: 'На проверке',
  published: 'На карте',
  review: 'Повторная проверка',
  rejected: 'Не опубликовано',
  removed: 'Удалено автором',
}
