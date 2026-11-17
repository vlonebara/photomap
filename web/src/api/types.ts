export type Coordinates = [longitude: number, latitude: number]

export interface Viewer {
  id: string
  name: string
  email: string
  role: 'user' | 'moderator'
}

export type PhotoStatus = 'published' | 'pending' | 'review' | 'rejected' | 'removed'

export interface Photo {
  id: number
  title: string
  description: string
  place: string
  coordinates: Coordinates
  year: number
  endYear?: number
  sourceName?: string
  sourceUrl?: string
  imageUrl: string
  thumbnailUrl: string
  authorId: string
  authorName: string
  stars: number
}

export interface PublicPhoto extends Photo {
  seriesId: number
  starredByMe?: boolean
  reportedByMe?: boolean
}

export interface ManagedPhoto extends Photo {
  status: PhotoStatus
}

export interface SeriesSummary {
  id: number
  coordinates: Coordinates
  title: string
  place: string
  photoCount: number
  stars: number
  thumbnailUrl: string
}

export interface SeriesDetail extends SeriesSummary {
  photos: PublicPhoto[]
}

export interface PhotoInput {
  title: string
  description: string
  place: string
  longitude: number
  latitude: number
  year: number
  endYear: number | null
  sourceName: string | null
  sourceUrl: string | null
}

export type ModerationFilter = 'queue' | 'published' | 'rejected'
export type ModerationAction = 'approve' | 'reject' | 'unpublish' | 'restore'
