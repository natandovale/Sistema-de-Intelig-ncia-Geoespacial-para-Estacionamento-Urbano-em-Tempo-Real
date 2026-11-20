export type ProbabilityLevel = 'HIGH' | 'MEDIUM' | 'LOW'

export interface CoordinatesDTO {
  lat: number
  lng: number
}

export interface RoadSegmentDTO {
  id: string
  name: string
  estimated_spots: number
  parking_allowed: boolean
}

export interface PrivateParkingDTO {
  id: string
  name: string
  location: CoordinatesDTO
  capacity: number
  is_open: boolean
}

export interface EventDTO {
  id: string
  title: string
  location: CoordinatesDTO
  expected_attendance: number
  start_time: string
  end_time: string
}

export interface SpotsResponse {
  grid_coordinates: CoordinatesDTO
  radius_meters: number
  score: number
  probability: ProbabilityLevel
  total_supply: number
  estimated_demand: number
  active_events: number
  roads: RoadSegmentDTO[]
  parkings: PrivateParkingDTO[]
  events: EventDTO[]
  from_cache: boolean
}

export interface StreetSegmentLine {
  id: string
  name: string
  highway: string
  coordinates: [number, number][] // [lng, lat] LineString GeoJSON recortada ao perímetro
  length_meters: number
  estimated_spots: number
  parking_allowed: boolean
  score: number
  probability: ProbabilityLevel
  color: string
}
