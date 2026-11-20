<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  DEFAULT_SEARCH_RADIUS_METERS,
  fetchSpotsAssessment,
  PRESET_PLACES,
  type PlaceSuggestion,
} from './services/spotsApi'
import {
  buildCirclePolygon,
  buildStreetLines,
  extractBasemapParkings,
  fetchOverpassPerimeterData,
  fromBasemapFeatures,
  mergeAreaParkings,
  summarizeStreetsByName,
} from './composables/useStreetLines'
import { useSearchAddress } from './composables/useSearchAddress'
import { useMapController } from './composables/useMapController'
import type {
  CoordinatesDTO,
  PrivateParkingDTO,
  RoadSegmentDTO,
  SpotsResponse,
  StreetSegmentLine,
} from './types/spots'

import SearchBar from './components/SearchBar.vue'
import StreetLegend from './components/StreetLegend.vue'
import IntelligenceDrawer from './components/IntelligenceDrawer.vue'

declare global {
  interface Window {
    maplibregl?: any
  }
}

// 1. Estado da Zona Urbana Avaliada
const loading = ref(false)
const assessment = ref<SpotsResponse | null>(null)
const streetLines = ref<StreetSegmentLine[]>([])
const areaParkings = ref<PrivateParkingDTO[]>([])
const namedStreets = ref<RoadSegmentDTO[]>([])
const streetSource = ref<'basemap' | 'overpass' | 'none'>('none')
const activeCenter = ref<CoordinatesDTO>(PRESET_PLACES[0].coordinates)
const activeRadius = ref<number>(DEFAULT_SEARCH_RADIUS_METERS)

// 2. Controlador do Mapa WebGL (DIP: Inversão de dependência da DOM/MapLibre)
const mapCtrl = useMapController()

// 3. Mecanismo de Busca Dinâmica e Geocoding (SRP: Responsabilidade Única de Autocomplete)
const searchCtrl = useSearchAddress({
  initialPlace: PRESET_PLACES[0],
  onSelect: async (place: PlaceSuggestion) => {
    await loadZoneData(place.coordinates.lat, place.coordinates.lng)
  },
})

// 4. Computeds de Apresentação
const displayedRoads = computed<RoadSegmentDTO[]>(() =>
  namedStreets.value.length > 0 ? namedStreets.value : assessment.value?.roads || [],
)

const totalStreetKm = computed(() =>
  (
    streetLines.value.reduce(
      (sum: number, l: StreetSegmentLine) => sum + l.length_meters,
      0,
    ) / 1000
  ).toFixed(1),
)

// 5. Casos de Uso e Orquestração
async function loadZoneData(lat: number, lng: number, radius = DEFAULT_SEARCH_RADIUS_METERS) {
  loading.value = true
  try {
    const data = await fetchSpotsAssessment(lat, lng, radius)
    assessment.value = data
    await focusPerimeterAndRenderStreets({ lat, lng }, radius, data)
  } catch {
    const mockData: SpotsResponse = {
      grid_coordinates: {
        lat: Number(lat.toFixed(3)),
        lng: Number(lng.toFixed(3)),
      },
      radius_meters: radius,
      score: 0.68,
      probability: 'MEDIUM',
      total_supply: 260,
      estimated_demand: 83,
      active_events: 1,
      roads: [
        { id: 'osm-way-102', name: 'Rua dos Tabajaras', estimated_spots: 60, parking_allowed: true },
        { id: 'osm-way-103', name: 'Avenida Abolição', estimated_spots: 90, parking_allowed: true },
        { id: 'osm-way-104', name: 'Av. Monsenhor Tabosa (Faixa Exclusiva)', estimated_spots: 40, parking_allowed: false },
      ],
      parkings: [],
      events: [],
      from_cache: false,
    }
    assessment.value = mockData
    await focusPerimeterAndRenderStreets({ lat, lng }, radius, mockData)
  } finally {
    loading.value = false
  }
}

function syncBasemapStreetsForActivePerimeter(): number {
  if (!assessment.value) return 0
  const center = activeCenter.value
  const radius = activeRadius.value
  const { roads, names } = mapCtrl.queryAllBasemapStreets()

  const lines = buildStreetLines(fromBasemapFeatures(roads), center, radius, assessment.value)
  if (lines.length > 0) {
    streetLines.value = lines
    const named = buildStreetLines(fromBasemapFeatures(names), center, radius, assessment.value)
    const summarized = summarizeStreetsByName(named.length > 0 ? named : lines)
    if (summarized.length > 0) {
      namedStreets.value = summarized
    }
    streetSource.value = 'basemap'
    mapCtrl.renderStreetLinesOnMap(lines)
  }

  const tileParkings = extractBasemapParkings(mapCtrl.queryBasemapParkingFeatures(), center, radius)
  if (tileParkings.length > 0) {
    areaParkings.value = mergeAreaParkings(center, radius, areaParkings.value, tileParkings)
    mapCtrl.renderParkingMarkersOnMap(areaParkings.value, center)
  }

  return lines.length
}

async function focusPerimeterAndRenderStreets(
  center: CoordinatesDTO,
  radius: number,
  data: SpotsResponse,
) {
  await mapCtrl.waitForMapReady()

  activeCenter.value = center
  activeRadius.value = radius
  streetSource.value = 'none'

  // Inicializa com estacionamentos retornados pela API no raio
  areaParkings.value = mergeAreaParkings(center, radius, data.parkings || [])
  mapCtrl.renderParkingMarkersOnMap(areaParkings.value, center)

  mapCtrl.updateDestinationMarker(center.lat, center.lng)

  // Renderiza contorno do perímetro
  const ring = buildCirclePolygon(center, radius)
  mapCtrl.renderPerimeterOnMap(ring)

  // Dispara busca Overpass OSM em background para enriquecer vias e estacionamentos
  void fetchOverpassPerimeterData(center.lat, center.lng, radius).then((overpassResult) => {
    if (activeCenter.value.lat !== center.lat || activeCenter.value.lng !== center.lng) {
      return
    }
    if (overpassResult.streets.length > 0) {
      const overpassLines = buildStreetLines(overpassResult.streets, center, radius, data)
      if (overpassLines.length > 0) {
        streetLines.value = overpassLines
        const summarized = summarizeStreetsByName(overpassLines)
        if (summarized.length > 0) {
          namedStreets.value = summarized
        }
        streetSource.value = 'overpass'
        mapCtrl.renderStreetLinesOnMap(overpassLines)
      }
    }
    if (overpassResult.parkings.length > 0) {
      areaParkings.value = mergeAreaParkings(
        center,
        radius,
        overpassResult.parkings,
        areaParkings.value,
      )
      mapCtrl.renderParkingMarkersOnMap(areaParkings.value, center)
    }
  })

  // Enquadra a câmera com inclinação 3D
  mapCtrl.fitBoundsPerimeter(ring)

  // Aguarda fim do voo da câmera e extrai ruas renderizadas no basemap
  await mapCtrl.waitForCameraAndTiles(1800)
  syncBasemapStreetsForActivePerimeter()
}

onMounted(async () => {
  mapCtrl.initMap(searchCtrl.selectedPlace.value.coordinates, {
    onClickPoint: async (clickedCoords: CoordinatesDTO) => {
      const customPlace: PlaceSuggestion = {
        id: `click-${clickedCoords.lat.toFixed(4)}`,
        name: `Ponto no Mapa (${clickedCoords.lat.toFixed(4)}, ${clickedCoords.lng.toFixed(4)})`,
        subtitle: 'Fortaleza - CE',
        category: 'venue',
        coordinates: clickedCoords,
      }
      await searchCtrl.selectDestination(customPlace)
    },
    onIdle: () => {
      if (assessment.value && streetSource.value !== 'overpass') {
        syncBasemapStreetsForActivePerimeter()
      }
    },
  })

  await loadZoneData(
    searchCtrl.selectedPlace.value.coordinates.lat,
    searchCtrl.selectedPlace.value.coordinates.lng,
  )
})
</script>

<template>
  <div class="layout-root">
    <!-- 1. Canvas do Mapa WebGL (3D Pitch 45° / Bearing -12°) -->
    <div :ref="mapCtrl.mapContainer" class="map-canvas"></div>

    <!-- 2. Barra de Busca Dinâmica e Autocomplete (Componente Especializado) -->
    <SearchBar
      v-model:search-query="searchCtrl.searchQuery.value"
      :suggestions="searchCtrl.suggestions.value"
      :show-suggestions="searchCtrl.showSuggestions.value"
      :searching-address="searchCtrl.searchingAddress.value"
      :active-suggestion-index="searchCtrl.activeSuggestionIndex.value"
      :loading="loading"
      :active-radius="activeRadius"
      @search-input="searchCtrl.handleSearchInput"
      @clear="searchCtrl.clearSearch"
      @submit="searchCtrl.submitSearch"
      @select-place="searchCtrl.selectDestination"
      @keydown="searchCtrl.handleKeydown"
      @focus="searchCtrl.showSuggestions.value = true"
      @hover-suggestion="(idx) => searchCtrl.activeSuggestionIndex.value = idx"
    />

    <!-- 3. Legenda Cromática das Vias e Estacionamentos -->
    <StreetLegend :parking-count="areaParkings.length" />

    <!-- 4. Painel Lateral de Inteligência Espacial (Métricas, Vagas, Eventos e Vias) -->
    <IntelligenceDrawer
      v-if="assessment"
      :assessment="assessment"
      :area-parkings="areaParkings"
      :displayed-roads="displayedRoads"
      :total-street-km="totalStreetKm"
      :active-radius="activeRadius"
      :street-source="streetSource"
      :active-center="activeCenter"
      @focus-parking="(pk) => mapCtrl.focusParkingOnMap(pk, areaParkings)"
    />
  </div>
</template>

<style scoped>
.layout-root {
  position: relative;
  width: 100vw;
  height: 100vh;
}

.map-canvas {
  width: 100%;
  height: 100%;
  background: radial-gradient(circle at center, #1e293b 0%, #0f172a 100%);
}

:deep(.parking-map-marker) {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: #0f172a;
  border: 2px solid #38bdf8;
  border-radius: 999px;
  padding: 2px 7px 2px 3px;
  color: #f8fafc;
  font-weight: 700;
  font-size: 0.72rem;
  cursor: pointer;
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.55), 0 0 10px rgba(56, 189, 248, 0.45);
  transition: transform 0.15s ease;
}

:deep(.parking-map-marker:hover) {
  transform: scale(1.12);
}

:deep(.parking-p-badge) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #0284c7;
  color: #fff;
  font-weight: 800;
  font-size: 0.7rem;
}

:deep(.parking-cap-label) {
  color: #38bdf8;
  font-weight: 700;
}
</style>
