import type { Coordinates, Photo, PhotoDraft } from '../demo/types'

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

export function toPhotoDraft(
  fields: PhotoFieldsValue, coordinates: Coordinates | null, imageUrl: string,
): PhotoDraft {
  if (!coordinates) throw new Error('Отметьте на карте объект, который виден на фото.')
  return {
    title: fields.title,
    place: fields.place,
    description: fields.description,
    year: fields.year === '' ? Number.NaN : fields.year,
    endYear: fields.endYear === '' ? undefined : fields.endYear,
    coordinates,
    imageUrl,
    sourceName: fields.sourceName,
    sourceUrl: fields.sourceUrl,
  }
}

export function formError(error: unknown): string {
  return error instanceof Error ? error.message : 'Не удалось выполнить действие. Попробуйте ещё раз.'
}

export const photoStatuses: Record<Photo['status'], string> = {
  pending: 'На проверке',
  published: 'На карте',
  review: 'Повторная проверка',
  rejected: 'Не опубликовано',
  removed: 'Удалено автором',
}
