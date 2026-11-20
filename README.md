# Spot Sense Web App

> Aplicação web para estimativa de probabilidade de vagas de estacionamento (vias públicas e estacionamentos privados) e monitoramento de eventos de impacto em um raio de 1 km ao redor do destino pesquisado.

---

## 🚀 Arquitetura e Tecnologias

- **Backend (`apps/backend`):** Go 1.23+ (Clean Architecture, `net/http`, `log/slog`).
- **Frontend (`apps/frontend`):** Vue 3 (Composition API, TypeScript, Vite) + Mapbox Geocoding + Mapa Vetorial com Camada Hexagonal GeoJSON/Deck.gl.
- **Banco Geoespacial:** PostgreSQL 16 + PostGIS 3.4 com índices espaciais `GIST` e filtro `ST_DWithin`.
- **Cache de Alta Performance:** Redis 7 (protocolo RESP2) e Cache em Memória com **Grid Snapping (~100m / 3 casas decimais)**.
- **Workers de Ingestão (`cmd/worker`):** Coleta idempotente do OpenStreetMap (Overpass API) e Eventos Regionais.

---

## 🛠️ Como Executar o Projeto

### 1. Subir PostGIS e Redis (via WSL Docker)
```bash
wsl docker compose -f docker/docker-compose.yml up -d
```

### 2. Executar o Backend (API REST em Go)
```powershell
cd apps/backend
go run ./cmd/api
```
- Endpoint de busca: `GET http://localhost:8080/api/vagas?lat=-3.728912&lng=-38.523419&radius=1000`
- Healthcheck: `GET http://localhost:8080/healthz`

### 3. Executar os Testes Automatizados (TDD)
```powershell
# Testes do Backend (Go)
cd apps/backend
go test -v ./...

# Testes do Frontend (TypeScript)
node --test apps/frontend/tests/run-tests.ts
```

### 4. Executar o Frontend (Vue 3 + Mapa Interativo)
```powershell
cd apps/frontend
npm install
npm run dev
```
Acesse `http://localhost:5173`.

---

## 📂 Governança e Documentação

- [`docs/architecture/architecture-overview.md`](./docs/architecture/architecture-overview.md) — **Documentação de Arquitetura do Protótipo (Diagramas, Fluxos, Heurística e Geometria).**
- [`docs/architecture/dynamic-address-search.md`](./docs/architecture/dynamic-address-search.md) — **Arquitetura da Busca Dinâmica de Endereços (Geocoding, Autocomplete, BBOX e Anti-Race Condition).**
- [`docs/AGENTS.md`](./docs/AGENTS.md) — Governança técnica, arquitetura e diretrizes XP / TDD / `ub-mode`.
- [`docs/development/development-plan.md`](./docs/development/development-plan.md) — Plano de fases e entregas.
- [`docs/development/progress.md`](./docs/development/progress.md) — Status atual de cada marco do MVP.
- [`docs/development/development-log.md`](./docs/development/development-log.md) — Diário cronológico de acontecimentos técnicos.
