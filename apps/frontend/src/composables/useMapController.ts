import { ref } from 'vue'
import type {
  CoordinatesDTO,
  PrivateParkingDTO,
  StreetSegmentLine,
} from '../types/spots'
import { distanceMeters } from './useStreetLines'
import { isWithinFortaleza } from '../services/spotsApi'

export function useMapController() {
  const mapContainer = ref<HTMLElement | null>(null)
  let mapInstance: any = null
  let mapLoaded = false
  let destinationMarker: any = null
  let parkingMarkers: any[] = []

  function initMap(
    initialCoords: CoordinatesDTO,
    handlers: {
      onClickPoint?: (coords: CoordinatesDTO) => void
      onIdle?: () => void
    } = {},
  ) {
    if (!mapContainer.value || !window.maplibregl) return

    mapInstance = new window.maplibregl.Map({
      container: mapContainer.value,
      style: 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json',
      center: [initialCoords.lng, initialCoords.lat],
      zoom: 15,
      pitch: 45,
      bearing: -12,
      maxBounds: [
        [-38.68, -3.92], // Sudoeste de Fortaleza
        [-38.38, -3.65], // Nordeste de Fortaleza
      ],
    })

    mapInstance.on('load', () => {
      mapLoaded = true
    })

    mapInstance.on('idle', () => {
      mapLoaded = true
      if (handlers.onIdle) {
        handlers.onIdle()
      }
    })

    mapInstance.on('click', (e: any) => {
      const clickedCoords = { lat: Number(e.lngLat.lat), lng: Number(e.lngLat.lng) }
      if (!isWithinFortaleza(clickedCoords)) return
      if (handlers.onClickPoint) {
        handlers.onClickPoint(clickedCoords)
      }
    })
  }

  function updateDestinationMarker(lat: number, lng: number) {
    if (!mapInstance || !window.maplibregl) return

    if (!destinationMarker) {
      destinationMarker = new window.maplibregl.Marker({ color: '#38bdf8' })
        .setLngLat([lng, lat])
        .addTo(mapInstance)
    } else {
      destinationMarker.setLngLat([lng, lat])
    }
  }

  function clearParkingMarkers() {
    for (const m of parkingMarkers) {
      try {
        m.remove()
      } catch {
        // ignore
      }
    }
    parkingMarkers = []
  }

  function renderParkingMarkersOnMap(
    parkings: PrivateParkingDTO[],
    center: CoordinatesDTO,
  ) {
    clearParkingMarkers()
    if (!mapInstance || !window.maplibregl || parkings.length === 0) return

    for (const pk of parkings) {
      const dist = Math.round(distanceMeters(center, pk.location))
      const el = document.createElement('div')
      el.className = 'parking-map-marker'
      el.innerHTML = `<span class="parking-p-badge">P</span><span class="parking-cap-label">${pk.capacity}</span>`
      el.title = `${pk.name} (~${pk.capacity} vagas · a ${dist}m)`

      const popupHtml = `
        <div style="color:#0f172a;font-family:Inter,sans-serif;padding:4px 2px;min-width:170px;">
          <div style="font-weight:700;font-size:13px;margin-bottom:4px;">🅿️ ${pk.name}</div>
          <div style="font-size:12px;color:#334155;">Capacidade: <strong>~${pk.capacity} vagas</strong></div>
          <div style="font-size:11px;color:#64748b;margin-top:2px;">Distância do destino: <strong>${dist} m</strong></div>
          <div style="font-size:11px;margin-top:4px;font-weight:600;color:${pk.is_open ? '#059669' : '#dc2626'};">
            ${pk.is_open ? '● Aberto na área selecionada' : '● Fechado'}
          </div>
        </div>
      `
      const popup = new window.maplibregl.Popup({ offset: 18, closeButton: false }).setHTML(popupHtml)

      const marker = new window.maplibregl.Marker({ element: el, anchor: 'bottom' })
        .setLngLat([pk.location.lng, pk.location.lat])
        .setPopup(popup)
        .addTo(mapInstance)

      parkingMarkers.push(marker)
    }
  }

  function focusParkingOnMap(
    pk: PrivateParkingDTO,
    parkings: PrivateParkingDTO[],
  ) {
    if (!mapInstance) return
    mapInstance.easeTo({
      center: [pk.location.lng, pk.location.lat],
      duration: 600,
    })
    const idx = parkings.findIndex((p: PrivateParkingDTO) => p.id === pk.id)
    if (idx !== -1 && parkingMarkers[idx]) {
      parkingMarkers[idx].togglePopup()
    }
  }

  function firstSymbolLayerId(): string | undefined {
    return mapInstance?.getStyle().layers.find((l: any) => l.type === 'symbol')?.id
  }

  function renderPerimeterOnMap(ring: [number, number][]) {
    if (!mapInstance) return
    const data = {
      type: 'Feature',
      properties: {},
      geometry: { type: 'Polygon', coordinates: [ring] },
    }
    const sourceId = 'spotsense-perimeter'
    const existing = mapInstance.getSource(sourceId)
    if (existing) {
      existing.setData(data)
      return
    }
    mapInstance.addSource(sourceId, { type: 'geojson', data })
    mapInstance.addLayer(
      {
        id: 'spotsense-perimeter-outline',
        type: 'line',
        source: sourceId,
        paint: {
          'line-color': '#38bdf8',
          'line-width': 1.6,
          'line-opacity': 0.7,
          'line-dasharray': [3, 2],
        },
      },
      firstSymbolLayerId(),
    )
  }

  function renderStreetLinesOnMap(lines: StreetSegmentLine[]) {
    if (!mapInstance) return

    const features = lines.map((line: StreetSegmentLine) => ({
      type: 'Feature',
      properties: {
        id: line.id,
        name: line.name,
        score: line.score,
        color: line.color,
        allowed: line.parking_allowed,
        spots: line.estimated_spots,
      },
      geometry: {
        type: 'LineString',
        coordinates: line.coordinates,
      },
    }))

    const geojsonData = {
      type: 'FeatureCollection',
      features,
    }

    const sourceId = 'spotsense-street-lines'
    const existingSource = mapInstance.getSource(sourceId)
    if (existingSource) {
      existingSource.setData(geojsonData)
      return
    }

    mapInstance.addSource(sourceId, {
      type: 'geojson',
      data: geojsonData,
    })

    const beforeId = firstSymbolLayerId()

    const widthByZoom = (base: number) => [
      'interpolate', ['exponential', 1.5], ['zoom'],
      13, base * 0.5,
      16, base * 1.6,
      18, base * 4,
    ]

    mapInstance.addLayer(
      {
        id: 'spotsense-street-glow',
        type: 'line',
        source: sourceId,
        layout: { 'line-cap': 'round', 'line-join': 'round' },
        paint: {
          'line-color': ['get', 'color'],
          'line-width': widthByZoom(5),
          'line-opacity': 0.28,
          'line-blur': 3,
        },
      },
      beforeId,
    )

    mapInstance.addLayer(
      {
        id: 'spotsense-street-core',
        type: 'line',
        source: sourceId,
        layout: { 'line-cap': 'round', 'line-join': 'round' },
        paint: {
          'line-color': ['get', 'color'],
          'line-width': widthByZoom(2),
          'line-opacity': 0.95,
        },
      },
      beforeId,
    )
  }

  function fitBoundsPerimeter(ring: [number, number][]) {
    if (!mapInstance) return
    const lngs = ring.map((p: [number, number]) => p[0])
    const lats = ring.map((p: [number, number]) => p[1])
    mapInstance.fitBounds(
      [
        [Math.min(...lngs), Math.min(...lats)],
        [Math.max(...lngs), Math.max(...lats)],
      ],
      {
        padding: { top: 90, bottom: 70, left: 430, right: 40 },
        pitch: 45,
        bearing: -12,
        duration: 900,
      },
    )
  }

  function queryAllBasemapStreets(): { roads: any[]; names: any[] } {
    if (!mapInstance) return { roads: [], names: [] }
    const style = mapInstance.getStyle()
    if (!style || !Array.isArray(style.layers)) return { roads: [], names: [] }

    const roadLayerIds = style.layers
      .filter(
        (l: any) =>
          l.type === 'line' &&
          !String(l.id).startsWith('spotsense-') &&
          !/path|pedestrian|cycle|foot|steps|service|track|rail|transit|ferry|link|case|casing|outline|shadow/i.test(
            String(l.id),
          ) &&
          (l['source-layer'] === 'transportation' ||
            l['source-layer'] === 'road' ||
            /road|street|highway|transportation|bridge|tunnel/i.test(String(l.id))),
      )
      .map((l: any) => l.id)

    let renderedRoads: any[] = []
    if (roadLayerIds.length > 0) {
      try {
        renderedRoads = mapInstance.queryRenderedFeatures(undefined, { layers: roadLayerIds }) || []
      } catch {
        renderedRoads = []
      }
    }

    const nameLayerIds = style.layers
      .filter(
        (l: any) =>
          !String(l.id).startsWith('spotsense-') &&
          l['source-layer'] === 'transportation_name',
      )
      .map((l: any) => l.id)

    let renderedNames: any[] = []
    if (nameLayerIds.length > 0) {
      try {
        renderedNames = mapInstance.queryRenderedFeatures(undefined, { layers: nameLayerIds }) || []
      } catch {
        renderedNames = []
      }
    }

    return {
      roads: renderedRoads,
      names: [...renderedNames, ...renderedRoads],
    }
  }

  function queryBasemapParkingFeatures(): any[] {
    if (!mapInstance) return []
    const style = mapInstance.getStyle()
    if (!style || !Array.isArray(style.layers)) return []

    const results: any[] = []
    const transportLayer = style.layers.find((l: any) => l['source-layer'] === 'transportation')
    if (transportLayer) {
      for (const sl of ['poi', 'landuse']) {
        try {
          const feats = mapInstance.querySourceFeatures(transportLayer.source, { sourceLayer: sl }) || []
          results.push(...feats)
        } catch {
          // opcional
        }
      }
    }
    return results
  }

  function waitForCameraAndTiles(timeoutMs = 3500): Promise<void> {
    return new Promise((resolve) => {
      if (!mapInstance) return resolve()
      let settled = false
      const done = () => {
        if (settled) return
        settled = true
        clearTimeout(timer)
        resolve()
      }
      const timer = setTimeout(done, timeoutMs)

      mapInstance.once('moveend', () => {
        if (mapInstance.areTilesLoaded && mapInstance.areTilesLoaded()) {
          setTimeout(done, 120)
        } else {
          mapInstance.once('idle', done)
          mapInstance.triggerRepaint()
        }
      })
    })
  }

  function waitForMapReady(timeoutMs = 1500): Promise<void> {
    if (!mapInstance || mapLoaded || mapInstance.isStyleLoaded()) {
      return Promise.resolve()
    }
    return new Promise((resolve) => {
      let done = false
      const finish = () => {
        if (done) return
        done = true
        mapLoaded = true
        clearTimeout(timer)
        resolve()
      }
      const timer = setTimeout(finish, timeoutMs)
      mapInstance.once('load', finish)
      mapInstance.once('idle', finish)
    })
  }

  return {
    mapContainer,
    initMap,
    updateDestinationMarker,
    clearParkingMarkers,
    renderParkingMarkersOnMap,
    focusParkingOnMap,
    renderPerimeterOnMap,
    renderStreetLinesOnMap,
    fitBoundsPerimeter,
    queryAllBasemapStreets,
    queryBasemapParkingFeatures,
    waitForCameraAndTiles,
    waitForMapReady,
  }
}
