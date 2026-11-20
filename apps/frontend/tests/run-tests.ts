import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import {
  buildSpotsApiUrl,
  buildPhotonAutocompleteUrl,
  isWithinFortaleza,
  parsePhotonFeatures,
} from '../src/services/spotsApi.ts'
import {
  buildCirclePolygon,
  buildOverpassStreetsQuery,
  buildStreetLines,
  clipLineToCircle,
  extractOverpassParkings,
  fromBasemapFeatures,
  fromOverpassElements,
  getProbabilityColor,
  lineLengthMeters,
  mergeAreaParkings,
  summarizeStreetsByName,
} from '../src/composables/useStreetLines.ts'
import type { SpotsResponse } from '../src/types/spots.ts'

const CENTER = { lat: -3.729, lng: -38.523 }
const DEG_PER_METER_LAT = 1 / 111320
const degLngPerMeter = 1 / (111320 * Math.cos((CENTER.lat * Math.PI) / 180))

const baseAssessment: SpotsResponse = {
  grid_coordinates: CENTER,
  radius_meters: 1000,
  score: 0.78,
  probability: 'HIGH',
  total_supply: 255,
  estimated_demand: 55,
  active_events: 0,
  roads: [],
  parkings: [],
  events: [],
  from_cache: false,
}

describe('Autocomplete (Fortaleza MVP) & API', () => {
  it('builds the GET /api/vagas URL with default 150m walking radius', () => {
    const url = buildSpotsApiUrl(-3.728912, -38.523419, undefined, 'http://localhost:8080/api')
    assert.equal(url, 'http://localhost:8080/api/vagas?lat=-3.728912&lng=-38.523419&radius=150')
  })

  it('validates whether coordinates are within Fortaleza bounding box', () => {
    assert.equal(isWithinFortaleza({ lat: -3.7325, lng: -38.5012 }), true)
    assert.equal(isWithinFortaleza({ lat: -23.5505, lng: -46.6333 }), false)
  })

  it('builds Photon autocomplete URL restricted to Fortaleza bbox', () => {
    const url = buildPhotonAutocompleteUrl('Desembargador Moreira')
    assert.ok(url.includes('bbox=-38.64,-3.89,-38.4,-3.68'))
    assert.ok(url.includes('Desembargador%20Moreira'))
  })

  it('parses Photon features and drops results outside Fortaleza', () => {
    const suggestions = parsePhotonFeatures({
      features: [
        { geometry: { coordinates: [-38.5015, -3.7342] }, properties: { osm_id: 1, name: 'Avenida Desembargador Moreira', district: 'Aldeota' } },
        { geometry: { coordinates: [-46.6333, -23.5505] }, properties: { osm_id: 2, name: 'Avenida Paulista' } },
      ],
    })
    assert.equal(suggestions.length, 1)
    assert.equal(suggestions[0].name, 'Avenida Desembargador Moreira')
  })
})

describe('Street lines clipped to the search perimeter', () => {
  it('keeps a street fully inside the perimeter untouched (same vertices, same length)', () => {
    const street: [number, number][] = [
      [CENTER.lng - 200 * degLngPerMeter, CENTER.lat],
      [CENTER.lng, CENTER.lat + 100 * DEG_PER_METER_LAT],
      [CENTER.lng + 200 * degLngPerMeter, CENTER.lat],
    ]
    const pieces = clipLineToCircle(street, CENTER, 1000)
    assert.equal(pieces.length, 1)
    assert.deepEqual(pieces[0], street)
  })

  it('cuts a street crossing the whole perimeter exactly at the border (length = diameter)', () => {
    const street: [number, number][] = [
      [CENTER.lng - 3000 * degLngPerMeter, CENTER.lat],
      [CENTER.lng + 3000 * degLngPerMeter, CENTER.lat],
    ]
    const pieces = clipLineToCircle(street, CENTER, 1000)
    assert.equal(pieces.length, 1)
    assert.ok(Math.abs(lineLengthMeters(pieces[0]) - 2000) < 2, `length was ${lineLengthMeters(pieces[0])}`)
  })

  it('drops streets completely outside the perimeter', () => {
    const street: [number, number][] = [
      [CENTER.lng - 500 * degLngPerMeter, CENTER.lat + 1500 * DEG_PER_METER_LAT],
      [CENTER.lng + 500 * degLngPerMeter, CENTER.lat + 1500 * DEG_PER_METER_LAT],
    ]
    assert.equal(clipLineToCircle(street, CENTER, 1000).length, 0)
  })

  it('splits a street that leaves and re-enters the perimeter into two pieces', () => {
    const street: [number, number][] = [
      [CENTER.lng - 500 * degLngPerMeter, CENTER.lat],
      [CENTER.lng, CENTER.lat + 2000 * DEG_PER_METER_LAT],
      [CENTER.lng + 500 * degLngPerMeter, CENTER.lat],
    ]
    assert.equal(clipLineToCircle(street, CENTER, 1000).length, 2)
  })

  it('builds a closed circle polygon for the perimeter outline', () => {
    const ring = buildCirclePolygon(CENTER, 1000, 36)
    assert.equal(ring.length, 37)
    assert.deepEqual(ring[0], ring[36])
  })

  it('builds lines from basemap tile features: excludes footpaths, service driveways, links, and dedupes multi-zoom tile duplicates', () => {
    const segment: [number, number][] = [
      [CENTER.lng - 300 * degLngPerMeter, CENTER.lat],
      [CENTER.lng + 300 * degLngPerMeter, CENTER.lat],
    ]
    const raw = fromBasemapFeatures([
      { id: 1, properties: { class: 'minor', name: 'Rua Nunes Valente' }, geometry: { type: 'LineString', coordinates: segment } },
      // Mesma rua repetida no tile vizinho -> deve ser deduplicada
      { id: 1, properties: { class: 'minor', name: 'Rua Nunes Valente' }, geometry: { type: 'LineString', coordinates: segment } },
      // Calçada / path -> excluída
      { id: 2, properties: { class: 'path' }, geometry: { type: 'LineString', coordinates: segment } },
      // Entrada interna de garagem (service=driveway) -> excluída para não criar caminhos falsos dentro de quarteirões
      { id: 4, properties: { class: 'service', service: 'driveway' }, geometry: { type: 'LineString', coordinates: segment } },
      // Alça de cruzamento (primary_link) -> excluída
      { id: 5, properties: { class: 'primary_link', ramp: 1 }, geometry: { type: 'LineString', coordinates: segment } },
      {
        id: 3,
        properties: { class: 'secondary', name: 'Avenida Abolição' },
        geometry: {
          type: 'MultiLineString',
          coordinates: [
            [[CENTER.lng, CENTER.lat + 100 * DEG_PER_METER_LAT], [CENTER.lng, CENTER.lat + 400 * DEG_PER_METER_LAT]],
            [[CENTER.lng, CENTER.lat - 100 * DEG_PER_METER_LAT], [CENTER.lng, CENTER.lat - 400 * DEG_PER_METER_LAT]],
          ],
        },
      },
    ])

    const lines = buildStreetLines(raw, CENTER, 1000, baseAssessment)
    assert.equal(lines.length, 3)
    assert.equal(lines[0].name, 'Rua Nunes Valente')
    assert.ok(Math.abs(lines[0].length_meters - 600) <= 1)
    assert.equal(lines[0].estimated_spots, Math.round(600 / 11))
    assert.equal(lines[0].color, '#10b981')
  })

  it('marks motorways and parking:both=no streets as forbidden (red, 0 spots)', () => {
    const raw = fromOverpassElements({
      elements: [
        {
          type: 'way',
          id: 502,
          tags: { name: 'Avenida Dom Luís', highway: 'primary', 'parking:both': 'no' },
          geometry: [{ lat: CENTER.lat, lon: CENTER.lng }, { lat: CENTER.lat, lon: CENTER.lng + 300 * degLngPerMeter }],
        },
      ],
    })
    const lines = buildStreetLines(raw, CENTER, 1000, baseAssessment)
    assert.equal(lines.length, 1)
    assert.equal(lines[0].parking_allowed, false)
    assert.equal(lines[0].estimated_spots, 0)
    assert.equal(lines[0].color, '#ef4444')
  })

  it('lowers score of streets near an active event', () => {
    const segment: [number, number][] = [
      [CENTER.lng - 50 * degLngPerMeter, CENTER.lat],
      [CENTER.lng + 50 * degLngPerMeter, CENTER.lat],
    ]
    const withEvent: SpotsResponse = {
      ...baseAssessment,
      events: [{ id: 'e1', title: 'Show', location: CENTER, expected_attendance: 900, start_time: '', end_time: '' }],
    }
    const raw = fromBasemapFeatures([{ id: 9, properties: { class: 'minor' }, geometry: { type: 'LineString', coordinates: segment } }])
    const [line] = buildStreetLines(raw, CENTER, 1000, withEvent)
    assert.ok(line.score < baseAssessment.score)
  })

  it('summarizes streets by name summing length and spots', () => {
    const summary = summarizeStreetsByName([
      { id: 'a', name: 'Rua A', highway: 'minor', coordinates: [], length_meters: 300, estimated_spots: 27, parking_allowed: true, score: 0.8, probability: 'HIGH', color: '' },
      { id: 'b', name: 'Rua A', highway: 'minor', coordinates: [], length_meters: 200, estimated_spots: 18, parking_allowed: true, score: 0.8, probability: 'HIGH', color: '' },
      { id: 'c', name: '', highway: 'minor', coordinates: [], length_meters: 900, estimated_spots: 80, parking_allowed: true, score: 0.8, probability: 'HIGH', color: '' },
    ])
    assert.equal(summary.length, 1)
    assert.equal(summary[0].estimated_spots, 45)
  })

  it('builds Overpass fallback query with the full perimeter radius and expanded parking matchers', () => {
    const q = buildOverpassStreetsQuery(-3.729, -38.523, 150)
    assert.ok(q.includes('around:150,-3.729,-38.523'))
    assert.ok(q.includes('around:400,-3.729,-38.523'))
    assert.ok(q.includes('parking_entrance'))
    assert.ok(q.includes('Estacionamento|Parking|Garagem|MultiPark'))
  })

  it('extracts real parkings from Overpass (amenity=parking, building=parking, and named Estacionamento POIs) within discovery radius', () => {
    const overpassWithParkings = {
      elements: [
        {
          type: 'node',
          id: 701,
          lat: CENTER.lat + 0.001,
          lon: CENTER.lng + 0.001,
          tags: {
            amenity: 'parking',
            name: 'Estacionamento Central',
            capacity: '85',
          },
        },
        {
          // Estacionamento cadastrado apenas pelo nome comercial (ex: HVMM Estacionamento) a ~220m
          type: 'node',
          id: 704,
          lat: CENTER.lat + 0.0018,
          lon: CENTER.lng + 0.001,
          tags: {
            name: 'HVMM Estacionamento',
            shop: 'car',
          },
        },
        {
          type: 'way',
          id: 702,
          tags: {
            building: 'parking',
            name: 'Estacionamento Ocean Tower',
          },
          geometry: [
            { lat: CENTER.lat - 0.001, lon: CENTER.lng - 0.001 },
            { lat: CENTER.lat - 0.001, lon: CENTER.lng - 0.0005 },
            { lat: CENTER.lat - 0.0005, lon: CENTER.lng - 0.0005 },
            { lat: CENTER.lat - 0.001, lon: CENTER.lng - 0.001 },
          ],
        },
        {
          // Estacionamento fora do raio -> deve ser descartado
          type: 'node',
          id: 703,
          lat: CENTER.lat + 0.05,
          lon: CENTER.lng + 0.05,
          tags: {
            amenity: 'parking',
            name: 'Estacionamento Distante',
          },
        },
      ],
    }

    const parkings = extractOverpassParkings(overpassWithParkings, CENTER, 150)
    assert.equal(parkings.length, 3)
    const names = parkings.map((p) => p.name)
    assert.ok(names.includes('Estacionamento Central'))
    assert.ok(names.includes('HVMM Estacionamento'))
    assert.ok(names.includes('Estacionamento Ocean Tower'))

    // Se não houver estacionamentos na área selecionada, retorna lista vazia
    const emptyParkings = extractOverpassParkings({ elements: [] }, CENTER, 150)
    assert.equal(emptyParkings.length, 0)
  })

  it('merges and deduplicates area parkings keeping only those within the selected perimeter', () => {
    const merged = mergeAreaParkings(
      CENTER,
      1000,
      [
        {
          id: 'p1',
          name: 'Estacionamento A',
          location: { lat: CENTER.lat + 0.0005, lng: CENTER.lng + 0.0005 },
          capacity: 60,
          is_open: true,
        },
      ],
      [
        // Duplicata muito próxima (< 25m) -> não deve duplicar
        {
          id: 'p1-dup',
          name: 'Estacionamento',
          location: { lat: CENTER.lat + 0.00052, lng: CENTER.lng + 0.00052 },
          capacity: 40,
          is_open: true,
        },
        // Fora do perímetro -> descartado
        {
          id: 'p-far',
          name: 'Estacionamento Longe',
          location: { lat: CENTER.lat + 0.04, lng: CENTER.lng + 0.04 },
          capacity: 100,
          is_open: true,
        },
      ],
    )
    assert.equal(merged.length, 1)
    assert.equal(merged[0].name, 'Estacionamento A')
  })

  it('maps probability levels to semantic colors', () => {
    assert.equal(getProbabilityColor('HIGH'), '#10b981')
    assert.equal(getProbabilityColor('MEDIUM'), '#f59e0b')
    assert.equal(getProbabilityColor('LOW'), '#ef4444')
  })
})

describe('AddressSearchController (SOLID / Decoupled Application Service)', async () => {
  const { AddressSearchController } = await import('../src/services/addressSearchController.ts')

  it('initializes with default place and allows clearing search input', () => {
    const search = new AddressSearchController()
    assert.ok(search.searchQuery.length > 0)
    assert.equal(search.showSuggestions, false)

    search.clearSearch()
    assert.equal(search.searchQuery, '')
    assert.equal(search.showSuggestions, true)
    assert.equal(search.activeSuggestionIndex, -1)
  })

  it('selects destination, updates state, and invokes onSelect callback', async () => {
    let selectedId = ''
    const search = new AddressSearchController({
      onSelect: (place) => {
        selectedId = place.id
      },
    })

    const targetPlace = {
      id: 'test-custom-1',
      name: 'Rua Canuto de Aguiar, 1200',
      subtitle: 'Aldeota, Fortaleza - CE',
      coordinates: { lat: -3.732, lng: -38.502 },
    }

    await search.selectDestination(targetPlace)
    assert.equal(search.selectedPlace.name, 'Rua Canuto de Aguiar, 1200')
    assert.equal(search.searchQuery, 'Rua Canuto de Aguiar, 1200')
    assert.equal(search.showSuggestions, false)
    assert.equal(selectedId, 'test-custom-1')
  })

  it('handles keyboard navigation ArrowDown and ArrowUp cycling', () => {
    const search = new AddressSearchController()
    search.showSuggestions = true

    search.handleKeydown({ key: 'ArrowDown', preventDefault: () => {} })
    assert.equal(search.activeSuggestionIndex, 0)

    search.handleKeydown({ key: 'ArrowDown', preventDefault: () => {} })
    assert.equal(search.activeSuggestionIndex, 1)

    search.handleKeydown({ key: 'ArrowUp', preventDefault: () => {} })
    assert.equal(search.activeSuggestionIndex, 0)

    search.handleKeydown({ key: 'Escape', preventDefault: () => {} })
    assert.equal(search.showSuggestions, false)
  })
})
