# Project Progress — Spot Sense Web App

## Current Status

- **Status:** DEVELOPMENT (MVP Feature Complete)
- **Last Update:** 2026-11-20
- **Current Milestone:** MVP Entregue & Documentação Arquitetural Geoespacial Consolidada

---

## Completed

- [x] Análise completa de requisitos e escopo do produto.
- [x] Alinhamento das decisões técnicas (Go + Vue 3, Monorepo, PostGIS, Redis, Scraper regional).
- [x] Definição da governança e preenchimento integral do `docs/AGENTS.md`.
- [x] Elaboração do Plano de Desenvolvimento macro (`docs/development/development-plan.md`).
- [x] Instalação e verificação do ambiente Go (v1.27.4) e Docker via WSL (v29.9.3).
- [x] Setup inicial do monorepo e infraestrutura de dev (`docker/docker-compose.yml`, `apps/backend/go.mod`).
- [x] **Milestone 1.2:** Implementação via TDD do Value Object `Coordinates` e algoritmo de `Grid Snapping` (~100m) com chave de cache para Redis.
- [x] **Milestone 1.3:** Modelagem das entidades de oferta/demanda (`RoadSegment`, `PrivateParking`, `Event`) e motor heurístico de Scoring (`scoring.Calculate`).
- [x] **Milestone 2.1:** Portas de domínio (`SpatialRepository`, `SpotCache`), Caso de Uso `SearchSpotsService` (Cache-Aside + Grid Snapping) e migration SQL do PostGIS com índices espaciais `GIST` (`migrations/001_init_postgis.sql`).
- [x] **Milestone 2.2 & 2.3:** Adaptadores de repositório (`PostGISSpatialRepository` com `ST_DWithin`, `SeedSpatialRepository` com Haversine, `RedisSpotCache` via protocolo RESP2 e `MemorySpotCache` com TTL) e ponto de entrada `cmd/api/main.go`.
- [x] **Milestone 3.1:** Endpoint REST `GET /api/vagas?lat={x}&lng={y}&radius=1000` e `/healthz` com DTOs de transporte e testes de contrato HTTP (`httptest`).
- [x] **Milestone 3.2 & 3.3:** Workers de Ingestão (`ParseOverpassResponse`, `ParseRegionalEventsResponse`, `IngestionWorkerService` idempotente e binário `cmd/worker/main.go`).
- [x] **Milestone 4.1 & 4.2:** Aplicação Frontend Vue 3 + TypeScript + Mapa Vetorial com transição *FlyTo*, barra de busca (Mapbox Geocoding + Presets), malha espacial de hexágonos (`generateHexagonGrid`) e painel de inteligência com indicador de Cache Hit.
- [x] **Refatoração SOLID do Frontend (`App.vue`):** Desacoplamento do componente monolítico de 1368 linhas em módulos coesos e testáveis: `useMapController` (DIP/MapLibre GL), `AddressSearchController` (SRP/Clean Architecture isolado de framework), `useSearchAddress` (Adapter de Reatividade), `SearchBar.vue`, `StreetLegend.vue` e `IntelligenceDrawer.vue`. App.vue reduzido para menos de 200 linhas.
- [x] **Documentação Arquitetural Detalhada:** Adicionado compêndio teórico completo em `docs/architecture/architecture-overview.md` detalhando B-tree vs R-tree, WGS 84 / SRID 4326, PostGIS e Cache, complementado pelo documento de arquitetura dedicada da busca dinâmica de endereços em `docs/architecture/dynamic-address-search.md` (Dual-Provider Fallback Photon + Nominatim, Geofencing estrito com BBOX, prevenção de race condition com AbortController e debounce calibrado).

---

## In Progress

- [ ] Nenhum bloqueio pendente (todas as 4 fases do MVP concluídas).

---

## Next Steps (Evoluções Futuras Pós-MVP)

1. Conectar o worker agendado diretamente à API Overpass de produção para a região escolhida.
2. Ajustar pesos da heurística de Scoring com feedback real dos primeiros motoristas.
3. Adicionar filtro de previsão futura (consultar como estará o local daqui a $N$ horas).
