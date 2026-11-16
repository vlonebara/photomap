import type { PhotoDraft } from './types'

export function validatePhoto(draft: PhotoDraft): PhotoDraft {
  const title = draft.title.trim()
  const description = draft.description.trim()
  const place = draft.place.trim()
  const { coordinates, year, endYear, imageUrl } = draft
  const sourceUrl = draft.sourceUrl?.trim() || undefined

  if (!title || !place || !description) {
    throw new Error('Заполните название, место и описание.')
  }

  const [longitude, latitude] = coordinates
  if (!Number.isFinite(longitude) || Math.abs(longitude) > 180 ||
      !Number.isFinite(latitude) || Math.abs(latitude) > 90) {
    throw new Error('Выберите точку на карте.')
  }

  const currentYear = new Date().getFullYear()
  if (!Number.isInteger(year) || year < 1 || year > currentYear) {
    throw new Error('Укажите год съёмки.')
  }

  if (endYear !== undefined &&
      (!Number.isInteger(endYear) || endYear < year || endYear > currentYear)) {
    throw new Error('Проверьте конечный год съёмки.')
  }

  if (endYear !== undefined && endYear - year > 50) {
    throw new Error('Разница между годами не должна превышать 50 лет.')
  }

  if (!imageUrl) throw new Error('Прикрепите фотографию.')

  if (sourceUrl) {
    try {
      const protocol = new URL(sourceUrl).protocol
      if (protocol !== 'https:' && protocol !== 'http:') throw new Error()
    } catch {
      throw new Error('Введите ссылку на источник, которая начинается с http:// или https://.')
    }
  }

  return {
    title,
    description,
    place,
    coordinates,
    year,
    endYear,
    imageUrl,
    sourceName: draft.sourceName?.trim() || undefined,
    sourceUrl,
  }
}
