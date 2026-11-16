export type Coordinates = [longitude: number, latitude: number]

export type Role = 'user' | 'moderator'

export interface Viewer {
  id: string
  name: string
  role: Role
}

export type PhotoStatus = 'published' | 'pending' | 'review' | 'rejected' | 'removed'

export interface Photo {
  id: number
  title: string
  description: string
  place: string
  region?: string
  coordinates: Coordinates
  year: number
  endYear?: number
  imageUrl: string
  sourceName?: string
  sourceUrl?: string
  authorId: string
  authorName: string
  stars: number
  status: PhotoStatus
}

export interface Series {
  id: number
  coordinates: Coordinates
  title: string
  place: string
  photos: Photo[]
  stars: number
}

export type PhotoDraft = Pick<
  Photo,
  'title' | 'description' | 'place' | 'coordinates' | 'year' | 'endYear' |
  'sourceName' | 'sourceUrl' | 'imageUrl'
>
