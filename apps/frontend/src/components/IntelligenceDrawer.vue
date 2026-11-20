<script setup lang="ts">
import { computed } from 'vue'
import type {
  CoordinatesDTO,
  PrivateParkingDTO,
  RoadSegmentDTO,
  SpotsResponse,
} from '../types/spots'
import { distanceMeters, getProbabilityColor } from '../composables/useStreetLines'

interface Props {
  assessment: SpotsResponse
  areaParkings: PrivateParkingDTO[]
  displayedRoads: RoadSegmentDTO[]
  totalStreetKm: string
  activeRadius: number
  streetSource: 'basemap' | 'overpass' | 'none'
  activeCenter: CoordinatesDTO
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'focus-parking', parking: PrivateParkingDTO): void
}>()

const probabilityLabel = computed(() => {
  switch (props.assessment.probability) {
    case 'HIGH':
      return 'Alta Probabilidade de Vagas'
    case 'MEDIUM':
      return 'Probabilidade Moderada'
    case 'LOW':
    default:
      return 'Baixa Disponibilidade (Alta Demanda)'
  }
})

const probabilityBadgeColor = computed(() => {
  return getProbabilityColor(props.assessment.probability)
})
</script>

<template>
  <aside class="intelligence-drawer">
    <!-- Cabeçalho de Score e Probabilidade -->
    <div class="score-header">
      <div class="score-circle" :style="{ borderColor: probabilityBadgeColor }">
        <span class="score-value">{{ Math.round(assessment.score * 100) }}%</span>
        <span class="score-caption">Score</span>
      </div>
      <div class="score-meta">
        <span class="prob-pill" :style="{ backgroundColor: probabilityBadgeColor }">
          {{ probabilityLabel }}
        </span>
        <div class="grid-meta">
          <span>Grid (~100m): <code>{{ assessment.grid_coordinates.lat.toFixed(3) }}, {{ assessment.grid_coordinates.lng.toFixed(3) }}</code></span>
          <span
            class="cache-badge"
            :class="assessment.from_cache ? 'cache-hit' : 'cache-miss'"
          >
            {{ assessment.from_cache ? '⚡ CACHE HIT (RAM)' : '🗄️ POSTGIS QUERY' }}
          </span>
        </div>
      </div>
    </div>

    <!-- Métricas de Oferta vs Demanda -->
    <div class="metrics-grid">
      <div class="metric-card">
        <span class="metric-label">Oferta ({{ activeRadius }} m)</span>
        <strong class="metric-num">{{ assessment.total_supply }} vagas</strong>
      </div>
      <div class="metric-card">
        <span class="metric-label">Demanda Estimada</span>
        <strong class="metric-num">{{ assessment.estimated_demand }} veículos</strong>
      </div>
      <div class="metric-card">
        <span class="metric-label">Ruas no Perímetro</span>
        <strong class="metric-num">{{ totalStreetKm }} km</strong>
      </div>
    </div>

    <!-- Eventos Impactando a Região -->
    <section class="panel-section">
      <h3>🎟️ Eventos Locais no Raio (Demanda)</h3>
      <div v-if="assessment.events.length === 0" class="empty-note">
        Nenhum evento gerador de tráfego detectado agora neste raio.
      </div>
      <div v-for="ev in assessment.events" :key="ev.id" class="item-card event-card">
        <div class="item-title">{{ ev.title }}</div>
        <div class="item-sub">
          Público est.: <strong>{{ ev.expected_attendance }} pessoas</strong> (~{{ Math.round(ev.expected_attendance / 3) }} carros)
        </div>
      </div>
    </section>

    <!-- Estacionamentos Encontrados na Área Selecionada -->
    <section class="panel-section">
      <h3>🅿️ Estacionamentos na Área ({{ areaParkings.length }})</h3>
      <div v-if="areaParkings.length === 0" class="empty-note">
        Nenhum estacionamento encontrado na área selecionada.
      </div>
      <div
        v-for="pk in areaParkings"
        :key="pk.id"
        class="item-card parking-card"
        title="Clique para localizar no mapa"
        @click="emit('focus-parking', pk)"
      >
        <div class="item-row">
          <span class="item-title">{{ pk.name }}</span>
          <span class="status-tag" :class="pk.is_open ? 'tag-open' : 'tag-closed'">
            {{ pk.is_open ? `~${pk.capacity} vagas` : 'Fechado' }}
          </span>
        </div>
        <div class="item-sub">
          A {{ Math.round(distanceMeters(activeCenter, pk.location)) }}m do ponto pesquisado · Clique para ver no mapa
        </div>
      </div>
    </section>

    <!-- Vias Públicas Mapeadas (OSM) -->
    <section class="panel-section">
      <h3>🛣️ Vias no Perímetro ({{ streetSource === 'overpass' ? 'Overpass OSM' : 'Basemap OSM' }})</h3>
      <div v-if="streetSource === 'none'" class="empty-note">
        Não foi possível carregar a malha viária desta região agora.
      </div>
      <div v-for="rd in displayedRoads" :key="rd.id" class="item-card">
        <div class="item-row">
          <span class="item-title">{{ rd.name }}</span>
          <span class="status-tag" :class="rd.parking_allowed ? 'tag-open' : 'tag-closed'">
            {{ rd.parking_allowed ? `~${rd.estimated_spots} vagas` : 'Proibido Estacionar' }}
          </span>
        </div>
      </div>
    </section>
  </aside>
</template>

<style scoped>
.intelligence-drawer {
  position: absolute;
  top: 84px;
  left: 20px;
  bottom: 20px;
  width: 390px;
  z-index: 15;
  background: var(--panel-bg);
  backdrop-filter: blur(14px);
  border: 1px solid var(--border-subtle);
  border-radius: 16px;
  padding: 18px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 16px;
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.5);
}

.score-header {
  display: flex;
  align-items: center;
  gap: 14px;
}

.score-circle {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  border: 4px solid #10b981;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.score-value {
  font-size: 1.35rem;
  font-weight: 800;
}

.score-caption {
  font-size: 0.68rem;
  color: var(--text-secondary);
  text-transform: uppercase;
}

.score-meta {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.prob-pill {
  display: inline-block;
  padding: 4px 10px;
  border-radius: 99px;
  font-size: 0.78rem;
  font-weight: 700;
  color: #0f172a;
  width: fit-content;
}

.grid-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 0.76rem;
  color: var(--text-secondary);
}

.cache-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 6px;
  font-weight: 700;
  font-size: 0.7rem;
  width: fit-content;
}

.cache-hit {
  background: rgba(16, 185, 129, 0.2);
  color: #34d399;
}

.cache-miss {
  background: rgba(56, 189, 248, 0.2);
  color: #38bdf8;
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}

.metric-card {
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-subtle);
  border-radius: 10px;
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.metric-label {
  font-size: 0.7rem;
  color: var(--text-secondary);
}

.metric-num {
  font-size: 0.92rem;
}

.panel-section h3 {
  font-size: 0.82rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.item-card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-subtle);
  border-radius: 10px;
  padding: 10px 12px;
  margin-bottom: 8px;
}

.event-card {
  border-left: 3px solid #f59e0b;
}

.parking-card {
  cursor: pointer;
  border-left: 3px solid #38bdf8;
  transition: background 0.14s ease;
}

.parking-card:hover {
  background: rgba(56, 189, 248, 0.12);
}

.item-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.item-title {
  font-size: 0.86rem;
  font-weight: 600;
}

.item-sub {
  font-size: 0.78rem;
  color: var(--text-secondary);
  margin-top: 4px;
}

.status-tag {
  font-size: 0.72rem;
  padding: 2px 8px;
  border-radius: 99px;
  font-weight: 600;
  white-space: nowrap;
}

.tag-open {
  background: rgba(16, 185, 129, 0.16);
  color: #34d399;
}

.tag-closed {
  background: rgba(239, 68, 68, 0.16);
  color: #f87171;
}

.empty-note {
  font-size: 0.8rem;
  color: var(--text-secondary);
}
</style>
