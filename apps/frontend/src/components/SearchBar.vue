<script setup lang="ts">
import type { PlaceSuggestion } from '../services/spotsApi'

interface Props {
  searchQuery: string
  suggestions: PlaceSuggestion[]
  showSuggestions: boolean
  searchingAddress: boolean
  activeSuggestionIndex: number
  loading: boolean
  activeRadius: number
}

defineProps<Props>()

const emit = defineEmits<{
  (e: 'update:searchQuery', value: string): void
  (e: 'search-input'): void
  (e: 'clear'): void
  (e: 'submit'): void
  (e: 'select-place', place: PlaceSuggestion): void
  (e: 'keydown', event: KeyboardEvent): void
  (e: 'focus'): void
  (e: 'hover-suggestion', index: number): void
}>()

function categoryIcon(category?: string): string {
  switch (category) {
    case 'street':
      return '🛣️'
    case 'district':
      return '🏙️'
    case 'venue':
    default:
      return '📍'
  }
}
</script>

<template>
  <header class="top-search-bar">
    <div class="brand-pill">
      <span class="pulse-dot"></span>
      <strong>Spot Sense</strong>
      <span class="city-tag">Fortaleza · Raio {{ activeRadius }} m</span>
    </div>

    <div class="search-box-wrapper">
      <div class="input-container">
        <span class="search-icon">🔍</span>
        <input
          :value="searchQuery"
          type="text"
          class="search-input"
          placeholder="Digite qualquer rua, bairro ou local em Fortaleza..."
          autocomplete="off"
          @input="emit('update:searchQuery', ($event.target as HTMLInputElement).value); emit('search-input')"
          @focus="emit('focus')"
          @keydown="emit('keydown', $event)"
        />
        <span v-if="searchingAddress" class="searching-spinner">Buscando...</span>
        <button
          v-if="searchQuery"
          type="button"
          class="clear-btn"
          title="Limpar endereço"
          @click="emit('clear')"
        >
          ✕
        </button>
      </div>

      <button
        type="button"
        class="refresh-btn"
        title="Pesquisar endereço em Fortaleza"
        @click="emit('submit')"
      >
        🔍 {{ loading ? 'Buscando...' : 'Pesquisar' }}
      </button>

      <ul v-if="showSuggestions && suggestions.length" class="suggestions-dropdown">
        <li class="dropdown-header">
          <span>{{ searchQuery.trim() ? 'Endereços encontrados em Fortaleza - CE' : 'Polos Populares em Fortaleza - CE' }}</span>
          <span class="kbd-hint">↑↓ Enter</span>
        </li>
        <li
          v-for="(item, idx) in suggestions"
          :key="item.id"
          class="suggestion-item"
          :class="{ 'active-suggestion': idx === activeSuggestionIndex }"
          @mousedown.prevent="emit('select-place', item)"
          @mouseenter="emit('hover-suggestion', idx)"
        >
          <span class="item-icon">{{ categoryIcon(item.category) }}</span>
          <div class="suggestion-body">
            <div class="suggestion-title">{{ item.name }}</div>
            <div class="suggestion-sub">
              {{ item.subtitle }} · ({{ item.coordinates.lat.toFixed(4) }}, {{ item.coordinates.lng.toFixed(4) }})
            </div>
          </div>
        </li>
      </ul>
    </div>
  </header>
</template>

<style scoped>
.top-search-bar {
  position: absolute;
  top: 18px;
  left: 20px;
  right: 20px;
  z-index: 20;
  display: flex;
  align-items: center;
  gap: 14px;
  max-width: 860px;
}

.brand-pill {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--panel-bg);
  backdrop-filter: blur(12px);
  border: 1px solid var(--border-subtle);
  padding: 10px 16px;
  border-radius: 999px;
  white-space: nowrap;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.4);
}

.pulse-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #10b981;
  box-shadow: 0 0 10px #10b981;
}

.city-tag {
  font-size: 0.75rem;
  color: var(--accent);
  background: rgba(56, 189, 248, 0.14);
  padding: 2px 9px;
  border-radius: 99px;
  font-weight: 600;
}

.search-box-wrapper {
  position: relative;
  flex: 1;
  display: flex;
  gap: 8px;
}

.input-container {
  position: relative;
  flex: 1;
  display: flex;
  align-items: center;
  background: var(--panel-bg);
  backdrop-filter: blur(12px);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  padding: 0 12px;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.4);
}

.input-container:focus-within {
  border-color: var(--accent);
}

.search-icon {
  font-size: 0.9rem;
  margin-right: 8px;
  opacity: 0.7;
}

.search-input {
  flex: 1;
  background: transparent;
  border: none;
  color: var(--text-primary);
  padding: 12px 4px;
  font-size: 0.95rem;
  outline: none;
}

.searching-spinner {
  font-size: 0.75rem;
  color: var(--accent);
  margin-right: 8px;
  white-space: nowrap;
}

.clear-btn {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-secondary);
  border: none;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
}

.clear-btn:hover {
  background: rgba(255, 255, 255, 0.18);
  color: #fff;
}

.refresh-btn {
  background: #0284c7;
  color: #fff;
  border: none;
  padding: 0 18px;
  border-radius: 12px;
  font-weight: 600;
  cursor: pointer;
}

.refresh-btn:hover {
  background: #0369a1;
}

.suggestions-dropdown {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  right: 0;
  background: #1e293b;
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  list-style: none;
  overflow: hidden;
  box-shadow: 0 16px 32px rgba(0, 0, 0, 0.55);
  max-height: 360px;
  overflow-y: auto;
}

.dropdown-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 14px;
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-secondary);
  background: rgba(15, 23, 42, 0.6);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.kbd-hint {
  background: rgba(255, 255, 255, 0.08);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 0.68rem;
}

.suggestion-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  cursor: pointer;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  transition: background 0.12s ease;
}

.suggestion-item:hover,
.active-suggestion {
  background: rgba(56, 189, 248, 0.16);
}

.item-icon {
  font-size: 1.1rem;
}

.suggestion-body {
  display: flex;
  flex-direction: column;
}

.suggestion-title {
  font-weight: 600;
  font-size: 0.9rem;
}

.suggestion-sub {
  font-size: 0.78rem;
  color: var(--text-secondary);
}
</style>
