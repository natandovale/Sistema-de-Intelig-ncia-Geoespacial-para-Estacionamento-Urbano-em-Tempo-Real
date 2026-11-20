import type {
  CoordinatesDTO,
  PrivateParkingDTO,
  ProbabilityLevel,
  RoadSegmentDTO,
  SpotsResponse,
  StreetSegmentLine,
} from '../types/spots'

type LngLat = [number, number]

const METERS_PER_DEG_LAT = 111320
const METERS_PER_SPOT = 11 // ~1 vaga a cada 11m de meio-fio aproveitável
const EVENT_INFLUENCE_METERS = 250

// Classes de via que NÃO são ruas públicas de estacionamento (pedestres, ciclovias, rampas de garagem, links, trilhos)
const EXCLUDED_CLASSES = new Set([
  'path', 'track', 'rail', 'transit', 'ferry', 'aerialway', 'pier', 'bridleway',
  'cycleway', 'footway', 'pedestrian', 'steps', 'busway', 'bus_guideway', 'raceway',
  'construction', 'proposed', 'service', 'driveway', 'parking_aisle', 'alley',
  'sidewalk', 'crossing', 'corridor', 'platform', 'living_street_service',
  'motorway_link', 'trunk_link', 'primary_link', 'secondary_link', 'tertiary_link', 'link',
])

// Vias expressas: estacionamento proibido por natureza
const FORBIDDEN_CLASSES = new Set(['motorway', 'trunk'])

const CLASS_SCORE_MODIFIER: Record<string, number> = {
  minor: 0.1,
  tertiary: 0,
  secondary: -0.06,
  primary: -0.18,
}

export interface RawStreetFeature {
  id?: string | number
  name?: string
  highwayClass: string
  subclass?: string
  serviceType?: string
  isLink?: boolean
  layerId?: string
  parkingForbiddenTag?: boolean
  geometry: { type: string; coordinates: any }
}

export function getProbabilityColor(probability: ProbabilityLevel): string {
  switch (probability) {
    case 'HIGH':
      return '#10b981'
    case 'MEDIUM':
      return '#f59e0b'
    case 'LOW':
    default:
      return '#ef4444'
  }
}

export function classifyScore(score: number): ProbabilityLevel {
  if (score >= 0.7) return 'HIGH'
  if (score >= 0.35) return 'MEDIUM'
  return 'LOW'
}

/** Converte a classe OSM `highway=*` para o esquema OpenMapTiles usado pelo basemap. */
export function normalizeHighwayClass(highway: string): string {
  const h = highway.toLowerCase().trim()
  if (h.endsWith('_link')) return 'link'
  if (['residential', 'unclassified', 'living_street', 'road', 'street'].includes(h)) return 'minor'
  return h
}

function isExcludedStreetFeature(feature: RawStreetFeature): boolean {
  const hw = (feature.highwayClass || '').toLowerCase()
  const sub = (feature.subclass || '').toLowerCase()
  const srv = (feature.serviceType || '').toLowerCase()
  const lid = (feature.layerId || '').toLowerCase()

  if (feature.isLink || hw === 'link' || hw.endsWith('_link')) return true
  if (EXCLUDED_CLASSES.has(hw) || EXCLUDED_CLASSES.has(sub) || EXCLUDED_CLASSES.has(srv)) return true
  if (srv.length > 0) return true // Qualquer via com tag `service=*` (driveway, parking_aisle, alley) é acesso interno
  if (/path|pedestrian|cycle|foot|steps|service|track|rail|transit|ferry|link|casing/i.test(lid)) {
    return true
  }
  return false
}

// ---------- Geometria local (projeção equiretangular centrada no destino) ----------

function cosLat(center: CoordinatesDTO): number {
  return Math.cos((center.lat * Math.PI) / 180)
}

function toLocal(c: LngLat, center: CoordinatesDTO): LngLat {
  return [
    (c[0] - center.lng) * METERS_PER_DEG_LAT * cosLat(center),
    (c[1] - center.lat) * METERS_PER_DEG_LAT,
  ]
}

function toLngLat(p: LngLat, center: CoordinatesDTO): LngLat {
  return [
    center.lng + p[0] / (METERS_PER_DEG_LAT * cosLat(center)),
    center.lat + p[1] / METERS_PER_DEG_LAT,
  ]
}

export function lineLengthMeters(coords: LngLat[]): number {
  let total = 0
  for (let i = 1; i < coords.length; i++) {
    const [lng1, lat1] = coords[i - 1]
    const [lng2, lat2] = coords[i]
    const dy = (lat2 - lat1) * METERS_PER_DEG_LAT
    const dx = (lng2 - lng1) * METERS_PER_DEG_LAT * Math.cos((((lat1 + lat2) / 2) * Math.PI) / 180)
    total += Math.hypot(dx, dy)
  }
  return total
}

/** Intervalo [t0, t1] ⊂ [0, 1] do segmento a→b que está dentro do círculo de raio r (origem). */
function segmentCircleInterval(a: LngLat, b: LngLat, r: number): [number, number] | null {
  const dx = b[0] - a[0]
  const dy = b[1] - a[1]
  const A = dx * dx + dy * dy
  const B = 2 * (a[0] * dx + a[1] * dy)
  const C = a[0] * a[0] + a[1] * a[1] - r * r

  if (A === 0) return C <= 0 ? [0, 1] : null

  const disc = B * B - 4 * A * C
  if (disc < 0) return null

  const s = Math.sqrt(disc)
  const lo = Math.max(0, (-B - s) / (2 * A))
  const hi = Math.min(1, (-B + s) / (2 * A))
  return lo < hi ? [lo, hi] : null
}

/**
 * Recorta uma polilinha ao perímetro circular (centro + raio em metros), preservando
 * os vértices originais internos e cortando exatamente na borda do perímetro.
 */
export function clipLineToCircle(
  coords: LngLat[],
  center: CoordinatesDTO,
  radiusMeters: number,
): LngLat[][] {
  const pieces: LngLat[][] = []
  let current: LngLat[] = []

  const flush = () => {
    if (current.length >= 2) pieces.push(current)
    current = []
  }

  for (let i = 0; i < coords.length - 1; i++) {
    const a = toLocal(coords[i], center)
    const b = toLocal(coords[i + 1], center)
    const interval = segmentCircleInterval(a, b, radiusMeters)

    if (!interval) {
      flush()
      continue
    }

    const [t0, t1] = interval
    const pointAt = (t: number, original: LngLat): LngLat =>
      t === 0 ? original : toLngLat([a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t], center)

    if (current.length === 0 || t0 > 0) {
      flush()
      current.push(t0 === 0 ? coords[i] : pointAt(t0, coords[i]))
    }
    current.push(t1 === 1 ? coords[i + 1] : pointAt(t1, coords[i]))

    if (t1 < 1) flush()
  }

  flush()
  return pieces
}

export function buildCirclePolygon(center: CoordinatesDTO, radiusMeters: number, steps = 72): LngLat[] {
  const ring: LngLat[] = []
  for (let i = 0; i < steps; i++) {
    const angle = (2 * Math.PI * i) / steps
    ring.push(toLngLat([radiusMeters * Math.cos(angle), radiusMeters * Math.sin(angle)], center))
  }
  ring.push(ring[0])
  return ring
}

// ---------- Construção das linhas de probabilidade por rua ----------

function scoreStreet(
  highwayClass: string,
  allowed: boolean,
  midpoint: LngLat,
  assessment: SpotsResponse,
  center: CoordinatesDTO,
): number {
  if (!allowed) return 0

  let modifier = CLASS_SCORE_MODIFIER[highwayClass] ?? 0
  const mid = toLocal(midpoint, center)

  for (const ev of assessment.events || []) {
    const evLocal = toLocal([ev.location.lng, ev.location.lat], center)
    if (Math.hypot(evLocal[0] - mid[0], evLocal[1] - mid[1]) < EVENT_INFLUENCE_METERS) {
      modifier -= 0.28
    }
  }

  return Number(Math.max(0.05, Math.min(0.98, assessment.score + modifier)).toFixed(2))
}

function extractLines(geometry: RawStreetFeature['geometry']): LngLat[][] {
  if (geometry?.type === 'LineString') return [geometry.coordinates as LngLat[]]
  if (geometry?.type === 'MultiLineString') return geometry.coordinates as LngLat[][]
  return []
}

function dedupKey(piece: LngLat[]): string {
  // Arredonda para ~5 metros (4 casas decimais) para fundir geometrias duplicadas de tiles/camadas sobrepostas
  const r = (n: number) => n.toFixed(4)
  const first = piece[0]
  const last = piece[piece.length - 1]
  const mid = piece[Math.floor(piece.length / 2)]
  const [p, q] =
    first[0] < last[0] || (first[0] === last[0] && first[1] < last[1])
      ? [first, last]
      : [last, first]
  return `${r(p[0])},${r(p[1])}|${r(mid[0])},${r(mid[1])}|${r(q[0])},${r(q[1])}`
}

/**
 * Recorta todas as ruas públicas reais ao perímetro e calcula comprimento real,
 * vagas estimadas, score e cor de cada trecho, descartando calçadas, ciclovias,
 * rampas de garagem (`service`) e alças de cruzamento (`*_link`).
 */
export function buildStreetLines(
  features: RawStreetFeature[],
  center: CoordinatesDTO,
  radiusMeters: number,
  assessment: SpotsResponse,
): StreetSegmentLine[] {
  const lines: StreetSegmentLine[] = []
  const seen = new Set<string>()

  features.forEach((feature, featureIdx) => {
    if (isExcludedStreetFeature(feature)) return
    const highwayClass = normalizeHighwayClass(feature.highwayClass || 'minor')
    if (EXCLUDED_CLASSES.has(highwayClass)) return

    const allowed = !feature.parkingForbiddenTag && !FORBIDDEN_CLASSES.has(highwayClass)

    for (const rawLine of extractLines(feature.geometry)) {
      for (const piece of clipLineToCircle(rawLine, center, radiusMeters)) {
        const lengthMeters = lineLengthMeters(piece)
        // Ignora micro-fragmentos de esquina (< 12 metros) que poluem visualmente os cruzamentos
        if (lengthMeters < 12) continue

        const key = dedupKey(piece)
        if (seen.has(key)) continue
        seen.add(key)

        const score = scoreStreet(highwayClass, allowed, piece[Math.floor(piece.length / 2)], assessment, center)
        const probability: ProbabilityLevel = allowed ? classifyScore(score) : 'LOW'

        lines.push({
          id: `street-${feature.id ?? featureIdx}-${lines.length}`,
          name: feature.name || '',
          highway: highwayClass,
          coordinates: piece,
          length_meters: Math.round(lengthMeters),
          estimated_spots: allowed ? Math.round(lengthMeters / METERS_PER_SPOT) : 0,
          parking_allowed: allowed,
          score,
          probability,
          color: getProbabilityColor(probability),
        })
      }
    }
  })

  return lines
}

/** Agrupa os trechos por nome de rua (para o painel lateral), somando comprimento e vagas. */
export function summarizeStreetsByName(lines: StreetSegmentLine[], limit = 10): RoadSegmentDTO[] {
  const grouped = new Map<string, RoadSegmentDTO & { length: number }>()

  for (const line of lines) {
    if (!line.name) continue
    const existing = grouped.get(line.name)
    if (existing) {
      existing.estimated_spots += line.estimated_spots
      existing.length += line.length_meters
      existing.parking_allowed = existing.parking_allowed && line.parking_allowed
    } else {
      grouped.set(line.name, {
        id: line.id,
        name: line.name,
        estimated_spots: line.estimated_spots,
        parking_allowed: line.parking_allowed,
        length: line.length_meters,
      })
    }
  }

  return Array.from(grouped.values())
    .sort((a, b) => b.length - a.length)
    .slice(0, limit)
    .map(({ length: _length, ...road }) => road)
}

// ---------- Fonte 1: features vetoriais do próprio basemap ----------

export function fromBasemapFeatures(features: any[]): RawStreetFeature[] {
  if (!Array.isArray(features)) return []
  const extracted: RawStreetFeature[] = []

  for (let i = 0; i < features.length; i++) {
    const f = features[i]
    const geom = f?.geometry
    if (!geom || (geom.type !== 'LineString' && geom.type !== 'MultiLineString')) {
      continue
    }
    const props = f.properties || {}
    const layerId = typeof f.layer?.id === 'string' ? f.layer.id : ''
    const rawClass =
      props.class ||
      props.highway ||
      props.subclass ||
      (layerId.includes('primary') ? 'primary' : 'minor')

    extracted.push({
      id: f.id ?? `${props.name || 'seg'}-${i}`,
      name: props.name || props['name:pt'] || props.name_int || props.ref || '',
      highwayClass: normalizeHighwayClass(String(rawClass)),
      subclass: props.subclass ? String(props.subclass) : undefined,
      serviceType: props.service ? String(props.service) : undefined,
      isLink: props.ramp === 1 || props.ramp === true || String(rawClass).endsWith('_link'),
      layerId,
      geometry: {
        type: geom.type,
        coordinates: geom.coordinates,
      },
    })
  }

  return extracted
}

// ---------- Fonte 2 (complementar/fallback): Overpass API (Ruas + Estacionamentos) ----------

const OVERPASS_ENDPOINTS = [
  'https://overpass-api.de/api/interpreter',
  'https://overpass.kumi.systems/api/interpreter',
]

export const MIN_PARKING_DISCOVERY_RADIUS_METERS = 400
const PARKING_NAME_REGEX = /estacionamento|parking|garagem|multipark|autopark|park\b/i

export function distanceMeters(a: CoordinatesDTO, b: CoordinatesDTO): number {
  const dy = (b.lat - a.lat) * METERS_PER_DEG_LAT
  const dx = (b.lng - a.lng) * METERS_PER_DEG_LAT * Math.cos((((a.lat + b.lat) / 2) * Math.PI) / 180)
  return Math.hypot(dx, dy)
}

export function effectiveParkingRadius(streetRadiusMeters: number): number {
  return Math.max(streetRadiusMeters, MIN_PARKING_DISCOVERY_RADIUS_METERS)
}

export function buildOverpassStreetsQuery(lat: number, lng: number, radiusMeters: number): string {
  const parkRadius = effectiveParkingRadius(radiusMeters)
  // Busca exclusivamente ruas públicas reais (primary, secondary, tertiary, residential, living_street),
  // excluindo rampas de garagem (`service`), calçadas (`footway`), ciclovias (`cycleway`) e alças (`*_link`).
  return `[out:json][timeout:15];(way(around:${radiusMeters},${lat},${lng})["highway"~"^(primary|secondary|tertiary|residential|living_street)$"]["service"!~".*"]["access"!~"^(private|no)$"];nwr(around:${parkRadius},${lat},${lng})["amenity"~"^(parking|parking_entrance|parking_space)$"];nwr(around:${parkRadius},${lat},${lng})["building"="parking"];nwr(around:${parkRadius},${lat},${lng})["name"~"Estacionamento|Parking|Garagem|MultiPark",i];);out center geom;`
}

export function fromOverpassElements(overpassJson: any): RawStreetFeature[] {
  if (!overpassJson || !Array.isArray(overpassJson.elements)) return []

  return overpassJson.elements
    .filter(
      (el: any) =>
        el.type === 'way' &&
        el.tags?.highway &&
        !el.tags?.service &&
        el.tags?.access !== 'private' &&
        el.tags?.access !== 'no' &&
        Array.isArray(el.geometry) &&
        el.geometry.length >= 2,
    )
    .map((el: any) => {
      const tags = el.tags || {}
      return {
        id: el.id,
        name: tags.name,
        highwayClass: normalizeHighwayClass(tags.highway || 'residential'),
        subclass: tags.footway || tags.cycleway || tags.sidewalk,
        serviceType: tags.service,
        isLink: String(tags.highway || '').endsWith('_link'),
        parkingForbiddenTag:
          tags['parking:both'] === 'no' ||
          tags['parking:lane:both'] === 'no' ||
          (tags['parking:right'] === 'no' && tags['parking:left'] === 'no'),
        geometry: {
          type: 'LineString',
          coordinates: el.geometry.map((pt: any) => [Number(pt.lon), Number(pt.lat)]),
        },
      }
    })
}

export function isParkingElement(tags: Record<string, any>): boolean {
  if (!tags) return false
  if (tags.access === 'private' || tags.access === 'no') return false
  if (
    tags.amenity === 'parking' ||
    tags.amenity === 'parking_entrance' ||
    tags.amenity === 'parking_space' ||
    tags.building === 'parking'
  ) {
    return true
  }
  const candidateName = String(tags.name || tags['name:pt'] || tags.operator || '')
  return PARKING_NAME_REGEX.test(candidateName) && !tags.highway
}

/**
 * Extrai estacionamentos reais (`amenity=parking|parking_entrance`, `building=parking` ou POIs com nome de estacionamento)
 * retornados pelo OpenStreetMap (Overpass) dentro do raio de descoberta de estacionamentos.
 */
export function extractOverpassParkings(
  overpassJson: any,
  center: CoordinatesDTO,
  radiusMeters: number,
): PrivateParkingDTO[] {
  if (!overpassJson || !Array.isArray(overpassJson.elements)) return []

  const maxDist = effectiveParkingRadius(radiusMeters)
  const rawParkings: PrivateParkingDTO[] = []

  for (const el of overpassJson.elements) {
    const tags = el.tags || {}
    if (!isParkingElement(tags)) continue

    let lat = Number(el.lat ?? el.center?.lat)
    let lng = Number(el.lon ?? el.center?.lon)

    if ((Number.isNaN(lat) || Number.isNaN(lng)) && Array.isArray(el.geometry) && el.geometry.length > 0) {
      let sumLat = 0
      let sumLng = 0
      for (const pt of el.geometry) {
        sumLat += Number(pt.lat)
        sumLng += Number(pt.lon)
      }
      lat = sumLat / el.geometry.length
      lng = sumLng / el.geometry.length
    }

    if (Number.isNaN(lat) || Number.isNaN(lng)) continue

    const location: CoordinatesDTO = {
      lat: Number(lat.toFixed(6)),
      lng: Number(lng.toFixed(6)),
    }

    if (distanceMeters(center, location) > maxDist) continue

    let capacity = 40
    if (tags.capacity) {
      const parsed = Number.parseInt(String(tags.capacity), 10)
      if (!Number.isNaN(parsed) && parsed > 0) {
        capacity = parsed
      }
    } else if (Array.isArray(el.geometry) && el.geometry.length >= 3) {
      const polyCoords: LngLat[] = el.geometry.map((pt: any) => [Number(pt.lon), Number(pt.lat)])
      const perim = lineLengthMeters(polyCoords)
      capacity = Math.max(20, Math.min(300, Math.round(perim / 4.5)))
    }

    const name =
      tags.name ||
      tags['name:pt'] ||
      tags.operator ||
      (tags.parking === 'multi-storey' || tags.building === 'parking'
        ? 'Edifício Garagem'
        : tags.parking === 'underground'
          ? 'Estacionamento Subterrâneo'
          : 'Estacionamento')

    rawParkings.push({
      id: `osm-parking-${el.type}-${el.id}`,
      name,
      location,
      capacity,
      is_open: true,
    })
  }

  return mergeAreaParkings(center, maxDist, rawParkings)
}

/**
 * Extrai POIs/polígonos de estacionamento (`class=parking` ou `subclass=parking`)
 * presentes nos vector tiles do próprio basemap dentro do perímetro selecionado.
 */
export function extractBasemapParkings(
  features: any[],
  center: CoordinatesDTO,
  radiusMeters: number,
): PrivateParkingDTO[] {
  if (!Array.isArray(features)) return []
  const found: PrivateParkingDTO[] = []

  for (let i = 0; i < features.length; i++) {
    const f = features[i]
    const props = f?.properties || {}
    const isParking =
      props.class === 'parking' ||
      props.subclass === 'parking' ||
      props.amenity === 'parking'
    if (!isParking) continue

    const geom = f?.geometry
    if (!geom) continue

    let lat = NaN
    let lng = NaN
    if (geom.type === 'Point' && Array.isArray(geom.coordinates)) {
      lng = Number(geom.coordinates[0])
      lat = Number(geom.coordinates[1])
    } else if (geom.type === 'Polygon' && Array.isArray(geom.coordinates?.[0])) {
      const ring = geom.coordinates[0]
      let sLat = 0
      let sLng = 0
      for (const pt of ring) {
        sLng += Number(pt[0])
        sLat += Number(pt[1])
      }
      lat = sLat / ring.length
      lng = sLng / ring.length
    }

    if (Number.isNaN(lat) || Number.isNaN(lng)) continue
    const location = { lat: Number(lat.toFixed(6)), lng: Number(lng.toFixed(6)) }
    if (distanceMeters(center, location) > radiusMeters) continue

    found.push({
      id: `tile-parking-${f.id ?? i}`,
      name: props.name || props['name:pt'] || 'Estacionamento',
      location,
      capacity: 45,
      is_open: true,
    })
  }

  return mergeAreaParkings(center, radiusMeters, found)
}

/**
 * Mescla e deduplica listas de estacionamentos mantendo apenas aqueles
 * estritamente dentro do perímetro (`radiusMeters`) e mesclando pontos a menos de 28m.
 */
export function mergeAreaParkings(
  center: CoordinatesDTO,
  radiusMeters: number,
  ...lists: PrivateParkingDTO[][]
): PrivateParkingDTO[] {
  const maxDist = effectiveParkingRadius(radiusMeters)
  const result: PrivateParkingDTO[] = []

  for (const list of lists) {
    for (const candidate of list || []) {
      if (!candidate?.location) continue
      if (distanceMeters(center, candidate.location) > maxDist) continue

      const existingIdx = result.findIndex(
        (p) =>
          distanceMeters(p.location, candidate.location) < 28 ||
          (p.name !== 'Estacionamento' &&
            p.name.toLowerCase() === candidate.name.toLowerCase() &&
            distanceMeters(p.location, candidate.location) < 120),
      )

      if (existingIdx === -1) {
        result.push({ ...candidate })
      } else if (
        result[existingIdx].name === 'Estacionamento' &&
        candidate.name !== 'Estacionamento'
      ) {
        result[existingIdx] = { ...candidate }
      }
    }
  }

  return result.sort((a, b) => distanceMeters(center, a.location) - distanceMeters(center, b.location))
}

/**
 * Busca estacionamentos comerciais por nome ("Estacionamento") na micro-região do destino
 * via Photon GeoJSON API (capturando estabelecimentos nominais próximos).
 */
export async function fetchLocalNamedParkings(
  lat: number,
  lng: number,
  radiusMeters: number,
): Promise<PrivateParkingDTO[]> {
  const maxDist = effectiveParkingRadius(radiusMeters)
  const center = { lat, lng }
  const degDelta = (maxDist / METERS_PER_DEG_LAT) * 1.25
  const bbox = `${(lng - degDelta).toFixed(5)},${(lat - degDelta).toFixed(5)},${(lng + degDelta).toFixed(5)},${(lat + degDelta).toFixed(5)}`
  const url = `https://photon.komoot.io/api/?q=Estacionamento&lat=${lat}&lon=${lng}&bbox=${bbox}&limit=12`

  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), 4500)
  try {
    const res = await fetch(url, { signal: controller.signal })
    if (!res.ok) return []
    const geojson = await res.json()
    if (!Array.isArray(geojson?.features)) return []

    const found: PrivateParkingDTO[] = []
    for (const f of geojson.features) {
      const coords = f?.geometry?.coordinates
      if (!Array.isArray(coords) || coords.length < 2) continue
      const location = { lat: Number(coords[1]), lng: Number(coords[0]) }
      if (distanceMeters(center, location) > maxDist) continue

      const props = f.properties || {}
      const name = props.name || 'Estacionamento'
      found.push({
        id: `photon-park-${props.osm_id || found.length}`,
        name,
        location,
        capacity: 45,
        is_open: true,
      })
    }
    return mergeAreaParkings(center, maxDist, found)
  } catch {
    return []
  } finally {
    clearTimeout(timer)
  }
}

export interface OverpassPerimeterResult {
  streets: RawStreetFeature[]
  parkings: PrivateParkingDTO[]
}

export async function fetchOverpassPerimeterData(
  lat: number,
  lng: number,
  radiusMeters: number,
): Promise<OverpassPerimeterResult> {
  const query = buildOverpassStreetsQuery(lat, lng, radiusMeters)
  const center = { lat, lng }
  const namedParkingsPromise = fetchLocalNamedParkings(lat, lng, radiusMeters)

  for (const endpoint of OVERPASS_ENDPOINTS) {
    const controller = new AbortController()
    const timer = setTimeout(() => controller.abort(), 5500)
    try {
      const res = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8' },
        body: `data=${encodeURIComponent(query)}`,
        signal: controller.signal,
      })
      if (res.ok) {
        const json = await res.json()
        const streets = fromOverpassElements(json)
        const overpassParkings = extractOverpassParkings(json, center, radiusMeters)
        const namedParkings = await namedParkingsPromise
        const parkings = mergeAreaParkings(center, radiusMeters, overpassParkings, namedParkings)
        if (streets.length > 0 || parkings.length > 0) {
          return { streets, parkings }
        }
      }
    } catch {
      // Tenta o próximo mirror Overpass sem travar a UI
    } finally {
      clearTimeout(timer)
    }
  }
  const namedParkings = await namedParkingsPromise
  return { streets: [], parkings: namedParkings }
}

export async function fetchOverpassStreetFeatures(
  lat: number,
  lng: number,
  radiusMeters: number,
): Promise<RawStreetFeature[]> {
  const { streets } = await fetchOverpassPerimeterData(lat, lng, radiusMeters)
  return streets
}
