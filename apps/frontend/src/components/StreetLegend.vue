<script setup lang="ts">
interface Props {
  parkingCount: number
}

defineProps<Props>()
</script>

<template>
  <div class="street-legend">
    <div class="legend-item">
      <span class="legend-line line-high"></span>
      <span>Alta chance de vaga</span>
    </div>
    <div class="legend-item">
      <span class="legend-line line-medium"></span>
      <span>Moderada</span>
    </div>
    <div class="legend-item">
      <span class="legend-line line-low"></span>
      <span>Lotada / Proibido</span>
    </div>
    <div v-if="parkingCount > 0" class="legend-item">
      <span class="legend-p-icon">P</span>
      <span>Estacionamento ({{ parkingCount }})</span>
    </div>
  </div>
</template>

<style scoped>
.street-legend {
  position: absolute;
  right: 22px;
  bottom: 24px;
  z-index: 15;
  background: var(--panel-bg);
  backdrop-filter: blur(12px);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  padding: 10px 16px;
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 0.78rem;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.45);
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.legend-line {
  width: 22px;
  height: 5px;
  border-radius: 99px;
}

.line-high {
  background: #10b981;
  box-shadow: 0 0 8px #10b981;
}

.line-medium {
  background: #f59e0b;
  box-shadow: 0 0 8px #f59e0b;
}

.line-low {
  background: #ef4444;
  box-shadow: 0 0 8px #ef4444;
}

.legend-p-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 6px;
  background: #0284c7;
  color: #fff;
  font-weight: 800;
  font-size: 0.72rem;
  box-shadow: 0 0 8px rgba(56, 189, 248, 0.7);
}
</style>
