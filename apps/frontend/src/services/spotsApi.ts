import type { CoordinatesDTO, SpotsResponse } from '../types/spots'

export type PlaceCategory = 'street' | 'venue' | 'district'

export interface PlaceSuggestion {
  id: string
  name: string
  subtitle: string
  category?: PlaceCategory
  coordinates: CoordinatesDTO
}

export const FORTALEZA_CENTER: CoordinatesDTO = {
  lat: -3.7319,
  lng: -38.5267,
}

export const FORTALEZA_BBOX = {
  minLng: -38.64,
  minLat: -3.89,
  maxLng: -38.4,
  maxLat: -3.68,
}

export const PRESET_PLACES: PlaceSuggestion[] = [
  {
    id: 'beira-mar-meireles',
    name: 'Avenida Beira Mar (Feirinha / Meireles)',
    subtitle: 'Meireles, Fortaleza - CE',
    category: 'street',
    coordinates: { lat: -3.728912, lng: -38.523419 },
  },
  {
    id: 'praca-portugal-aldeota',
    name: 'Praça Portugal & Av. Desembargador Moreira',
    subtitle: 'Aldeota, Fortaleza - CE',
    category: 'venue',
    coordinates: { lat: -3.7336, lng: -38.4979 },
  },
  {
    id: 'iguatemi-bosque',
    name: 'Shopping Iguatemi Bosque',
    subtitle: 'Edson Queiroz, Fortaleza - CE',
    category: 'venue',
    coordinates: { lat: -3.7564, lng: -38.4896 },
  },
  {
    id: 'dragao-do-mar',
    name: 'Centro Dragão do Mar de Arte e Cultura',
    subtitle: 'Praia de Iracema, Fortaleza - CE',
    category: 'venue',
    coordinates: { lat: -3.7219, lng: -38.5215 },
  },
  {
    id: 'riomar-papicu',
    name: 'RioMar Fortaleza',
    subtitle: 'Papicu, Fortaleza - CE',
    category: 'venue',
    coordinates: { lat: -3.7408, lng: -38.4712 },
  },
  {
    id: 'arena-castelao',
    name: 'Arena Castelão (Estádio Plácido Castelo)',
    subtitle: 'Boa Vista / Castelão, Fortaleza - CE',
    category: 'venue',
    coordinates: { lat: -3.8073, lng: -38.5225 },
  },
  {
    id: 'mucuripe-orla',
    name: 'Orla do Mucuripe & Mercado dos Peixes',
    subtitle: 'Mucuripe, Fortaleza - CE',
    category: 'venue',
    coordinates: { lat: -3.7258, lng: -38.4982 },
  },
  {
    id: 'monsenhor-tabosa',
    name: 'Avenida Monsenhor Tabosa',
    subtitle: 'Praia de Iracema, Fortaleza - CE',
    category: 'street',
    coordinates: { lat: -3.7265, lng: -38.518 },
  },
]

export function isWithinFortaleza(coords: CoordinatesDTO): boolean {
  return (
    coords.lat >= FORTALEZA_BBOX.minLat &&
    coords.lat <= FORTALEZA_BBOX.maxLat &&
    coords.lng >= FORTALEZA_BBOX.minLng &&
    coords.lng <= FORTALEZA_BBOX.maxLng
  )
}

export function buildPhotonAutocompleteUrl(query: string): string {
  const bboxStr = `${FORTALEZA_BBOX.minLng},${FORTALEZA_BBOX.minLat},${FORTALEZA_BBOX.maxLng},${FORTALEZA_BBOX.maxLat}`
  const q = encodeURIComponent(`${query.trim()} Fortaleza`)
  return `https://photon.komoot.io/api/?q=${q}&lat=${FORTALEZA_CENTER.lat}&lon=${FORTALEZA_CENTER.lng}&bbox=${bboxStr}&limit=7`
}

export function buildNominatimAutocompleteUrl(query: string): string {
  const viewbox = `${FORTALEZA_BBOX.minLng},${FORTALEZA_BBOX.maxLat},${FORTALEZA_BBOX.maxLng},${FORTALEZA_BBOX.minLat}`
  const q = encodeURIComponent(`${query.trim()}, Fortaleza, Ceará`)
  return `https://nominatim.openstreetmap.org/search?q=${q}&format=jsonv2&addressdetails=1&countrycodes=br&viewbox=${viewbox}&bounded=1&limit=6`
}

export function parsePhotonFeatures(geojson: any): PlaceSuggestion[] {
  if (!geojson || !Array.isArray(geojson.features)) {
    return []
  }

  const results: PlaceSuggestion[] = []
  const seen = new Set<string>()

  for (const feature of geojson.features) {
    const coordsArr = feature?.geometry?.coordinates
    if (!Array.isArray(coordsArr) || coordsArr.length < 2) continue

    const coordinates: CoordinatesDTO = {
      lng: Number(coordsArr[0]),
      lat: Number(coordsArr[1]),
    }

    if (!isWithinFortaleza(coordinates)) continue

    const props = feature.properties || {}
    const primaryName =
      props.name ||
      [props.street, props.housenumber].filter(Boolean).join(', ') ||
      props.district
    if (!primaryName) continue

    const district = props.district || props.suburb || props.locality || ''
    const streetPart =
      props.street && props.street !== primaryName
        ? `${props.street}${props.housenumber ? `, ${props.housenumber}` : ''}`
        : ''

    const subtitleParts = [streetPart, district, 'Fortaleza - CE'].filter(Boolean)
    const subtitle = subtitleParts.join(' · ')

    const dedupKey = `${primaryName.toLowerCase()}|${district.toLowerCase()}`
    if (seen.has(dedupKey)) continue
    seen.add(dedupKey)

    let category: PlaceCategory = 'venue'
    if (props.osm_key === 'highway' || props.type === 'street') {
      category = 'street'
    } else if (props.type === 'district' || props.type === 'locality') {
      category = 'district'
    }

    results.push({
      id: `photon-${props.osm_id || results.length}-${coordinates.lat.toFixed(4)}`,
      name: primaryName,
      subtitle,
      category,
      coordinates,
    })
  }

  return results
}

export function parseNominatimResults(items: any[]): PlaceSuggestion[] {
  if (!Array.isArray(items)) return []

  const results: PlaceSuggestion[] = []
  const seen = new Set<string>()

  for (const item of items) {
    const coordinates: CoordinatesDTO = {
      lat: Number(item.lat),
      lng: Number(item.lon),
    }
    if (Number.isNaN(coordinates.lat) || !isWithinFortaleza(coordinates)) continue

    const addr = item.address || {}
    const name =
      item.name ||
      addr.road ||
      addr.amenity ||
      addr.suburb ||
      (item.display_name ? item.display_name.split(',')[0] : '')
    if (!name) continue

    const district = addr.suburb || addr.neighbourhood || addr.city_district || ''
    const subtitle = [district, 'Fortaleza - CE'].filter(Boolean).join(' · ')
    const dedupKey = `${name.toLowerCase()}|${district.toLowerCase()}`
    if (seen.has(dedupKey)) continue
    seen.add(dedupKey)

    results.push({
      id: `nom-${item.place_id || results.length}`,
      name,
      subtitle,
      category: item.category === 'highway' ? 'street' : 'venue',
      coordinates,
    })
  }

  return results
}

export const DEFAULT_SEARCH_RADIUS_METERS = 150

export function buildSpotsApiUrl(
  lat: number,
  lng: number,
  radius = DEFAULT_SEARCH_RADIUS_METERS,
  baseUrl = (import.meta as any).env?.VITE_API_BASE_URL || '/api',
): string {
  const cleanBase = baseUrl.replace(/\/+$/, '')
  return `${cleanBase}/vagas?lat=${lat}&lng=${lng}&radius=${radius}`
}

export async function fetchSpotsAssessment(
  lat: number,
  lng: number,
  radius = DEFAULT_SEARCH_RADIUS_METERS,
): Promise<SpotsResponse> {
  const url = buildSpotsApiUrl(lat, lng, radius)
  const response = await fetch(url)
  if (!response.ok) {
    throw new Error(`API error (${response.status})`)
  }
  return (await response.json()) as SpotsResponse
}

export async function geocodeLocation(
  query: string,
  signal?: AbortSignal,
): Promise<PlaceSuggestion[]> {
  const trimmed = query.trim()
  if (!trimmed) {
    return PRESET_PLACES
  }

  const lower = trimmed.toLowerCase()
  const localMatches = PRESET_PLACES.filter(
    (p) =>
      p.name.toLowerCase().includes(lower) ||
      p.subtitle.toLowerCase().includes(lower),
  )

  if (trimmed.length < 2) {
    return localMatches.length > 0 ? localMatches : PRESET_PLACES
  }

  // 1. Tenta Photon API (Search-as-you-type para Fortaleza)
  try {
    const photonRes = await fetch(buildPhotonAutocompleteUrl(trimmed), { signal })
    if (photonRes.ok) {
      const geojson = await photonRes.json()
      const remote = parsePhotonFeatures(geojson)
      if (remote.length > 0) {
        return mergeSuggestions(localMatches, remote)
      }
    }
  } catch (err: any) {
    if (err?.name === 'AbortError') throw err
  }

  // 2. Fallback para Nominatim OpenStreetMap (restrito ao viewbox de Fortaleza)
  try {
    const nomRes = await fetch(buildNominatimAutocompleteUrl(trimmed), {
      signal,
      headers: { 'Accept-Language': 'pt-BR' },
    })
    if (nomRes.ok) {
      const items = await nomRes.json()
      const remote = parseNominatimResults(items)
      if (remote.length > 0) {
        return mergeSuggestions(localMatches, remote)
      }
    }
  } catch (err: any) {
    if (err?.name === 'AbortError') throw err
  }

  return localMatches
}

function mergeSuggestions(
  local: PlaceSuggestion[],
  remote: PlaceSuggestion[],
): PlaceSuggestion[] {
  const seen = new Set<string>()
  const merged: PlaceSuggestion[] = []

  for (const item of [...local, ...remote]) {
    const key = item.name.toLowerCase()
    if (!seen.has(key)) {
      seen.add(key)
      merged.push(item)
    }
  }
  return merged.slice(0, 8)
}
