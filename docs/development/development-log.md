# Development Log — Spot Sense Web App

> Diário cronológico dos acontecimentos técnicos durante o desenvolvimento da aplicação.

---

## 2026-11-20

### [HIGH] [REFACTOR] Refatoração SOLID do Frontend (`App.vue` Monolítico -> Arquitetura Modular)

**Categoria:** Architecture / Frontend Refactoring  
**Importância:** Alta  
**Horário:** 14:26  

**Problema:**  
O arquivo `App.vue` havia se tornado um componente "Deus" (*God Component*) com 1.368 linhas de código, acumulando múltiplas responsabilidades não relacionadas:
1. Ciclo de vida, eventos de câmera 3D e camadas WebGL do MapLibre GL.
2. Estado de digitação, debounce de 280ms, cancelamento com `AbortController` e navegação por teclado da busca.
3. Carregamento de dados de API, cálculo de métricas e orquestração de chamadas assíncronas do Overpass.
4. Apresentação visual da barra de pesquisa, dropdown, legenda de cores e gaveta lateral de inteligência espacial.
5. Mais de 500 linhas de CSS acopladas em um único bloco `<style scoped>`.

**Solução Aplicando Princípios SOLID:**
- **S (Single Responsibility Principle):**
  - `AddressSearchController.ts`: Serviço de aplicação puro em TypeScript (zero dependência de Vue) responsável pela lógica de debounce, abort controller, navegação por setas/Enter/Escape e geocodificação.
  - `useSearchAddress.ts`: Adaptador de reatividade do Vue que conecta o `AddressSearchController` ao ciclo de vida da aplicação.
  - `useMapController.ts`: Composable dedicado exclusivamente ao controle do mapa WebGL, manipulação de marcadores, popups e renderização de camadas de linhas e polígonos.
  - `SearchBar.vue`: Componente focado unicamente na interface da barra de pesquisa e dropdown de sugestões.
  - `StreetLegend.vue`: Componente focado na legenda de cores das vias e estacionamentos.
  - `IntelligenceDrawer.vue`: Componente responsável por exibir o score, métricas de oferta/demanda, lista de eventos e estacionamentos.
- **O (Open/Closed Principle):**
  - Componentes de UI comunicam-se via eventos (`@submit`, `@select-place`, `@focus-parking`) e propriedades tipadas, permitindo evolução de funcionalidades sem alteração da casca.
- **L (Liskov Substitution Principle):**
  - Interfaces tipadas em `types/spots.ts` (`PlaceSuggestion`, `CoordinatesDTO`, `SpotsResponse`, `PrivateParkingDTO`) garantem intercambiabilidade estrita entre dados reais de produção e mock fallbacks.
- **I (Interface Segregation Principle):**
  - Cada componente filho recebe apenas as propriedades específicas de que necessita, eliminando o acoplamento desnecessário a objetos gigantes.
- **D (Dependency Inversion Principle):**
  - O `App.vue` não manipula diretamente o objeto global `window.maplibregl` ou nós da DOM; ele consome a abstração de alto nível fornecida por `useMapController`.

**Impacto:**  
- `App.vue` reduzido de **1.368 linhas para 219 linhas** (redução de 84% de código acoplado), funcionando puramente como orquestrador declarativo.
- Adicionados 3 novos testes unitários para `AddressSearchController`, elevando a suíte para **20 testes automatizados** passando em 173ms.

**Arquivos criados/afetados:**  
- `apps/frontend/src/services/addressSearchController.ts`  
- `apps/frontend/src/composables/useSearchAddress.ts`  
- `apps/frontend/src/composables/useMapController.ts`  
- `apps/frontend/src/components/SearchBar.vue`  
- `apps/frontend/src/components/StreetLegend.vue`  
- `apps/frontend/src/components/IntelligenceDrawer.vue`  
- `apps/frontend/src/App.vue`  
- `apps/frontend/tests/run-tests.ts`  
- `docs/development/progress.md`  
- `docs/development/development-log.md`  

---

### [MEDIUM] [DOCS] Criação da Documentação Dedicada da Busca Dinâmica de Endereços

**Categoria:** Documentation / Frontend & Geocoding Architecture  
**Importância:** Média  
**Horário:** 14:13  

**Descrição:**  
Criado o documento de arquitetura dedicado `docs/architecture/dynamic-address-search.md` detalhando:
- O diagrama de sequência end-to-end do fluxo de autocomplete (digitação do usuário $\to$ debounce $\to$ cancelamento de requisição anterior $\to$ consulta $\to$ renderização $\to$ transição de câmera 3D).
- Mecanismo anti-race condition via `AbortController` impedindo inversão temporal de resultados assíncronos.
- Geofencing estrito com Bounding Box de Fortaleza (`minLng: -38.64, minLat: -3.89, maxLng: -38.40, maxLat: -3.68`) e validação residual em código (`isWithinFortaleza`).
- Pipeline de contingência *Dual-Provider* (Photon Komoot API primária com Elasticsearch OSM $\to$ Fallback Nominatim OSM $\to$ Presets locais em memória).
- Deduplicação de nós OSM e categorização visual por ícones (`street`, `venue`, `district`).
- Integração com a câmera 3D inclinada (`pitch: 45`, `bearing: -12`) e trava perimetral `maxBounds`.

**Arquivos afetados:**  
- `docs/architecture/dynamic-address-search.md`  
- `docs/architecture/architecture-overview.md`  
- `README.md`  
- `docs/development/progress.md`  
- `docs/development/development-log.md`  

---

### [MEDIUM] [DOCS] Expansão dos Fundamentos Teóricos de Persistência Geoespacial, WGS 84 e Cache

**Categoria:** Documentation / Geospatial Architecture  
**Importância:** Média  
**Horário:** 14:06  

**Descrição:**  
Adicionada explicação aprofundada na Seção 4 de `docs/architecture/architecture-overview.md` detalhando:
- Contraste teórico entre **B-tree** (1D, ordem linear total) e **R-tree / GiST** (2D/ND, caixas delimitadoras mínimas - MBR).
- Exemplo didático de poda espacial na R-tree (como o nó raiz descarta centenas de milhares de vias da Zona Sul em $O(\log N)$ para consultas na Aldeota antes de rodar qualquer cálculo geodésico).
- Tabela comparativa direta: dimensões, elementos indexados, operadores e aplicação no Spot Sense.
- **Modelo Geodésico Global (SRID 4326 / WGS 84):** Explicação da relação física entre graus e distâncias (analogia da lanterna no centro da Terra, cálculo dos $40.000\text{ km} / 360^\circ \approx 111{,}1\text{ km}$ e tabela de subdivisão decimal até o quarteirão de $111\text{m}$ e o passo de $1\text{m}$), a convergência dos meridianos (tangerina) e como o cruzamento ortogonal unívoco de Latitude e Longitude localiza qualquer ponto na superfície terrestre.
- Distinção entre `GEOMETRY` e `GEOGRAPHY` no PostGIS e a estratégia em dois estágios do `ST_DWithin` contra `ST_Distance`.
- Fundamentos de quantização contínua via *Grid Snapping* (3 casas decimais $\approx 111\text{m}$, dimensões de quarteirão) e padrão *Cache-Aside* com Redis RESP2 e TTL de 10 minutos.

**Arquivos afetados:**  
- `docs/architecture/architecture-overview.md`  
- `docs/development/progress.md`  
- `docs/development/development-log.md`  

---

### [MEDIUM] [DOCS] Criação da Documentação de Arquitetura Completa do Protótipo

**Categoria:** Documentation / Architecture  
**Importância:** Média  
**Horário:** 12:31  

**Descrição:**  
Elaborada a documentação arquitetural detalhada do protótipo em `docs/architecture/architecture-overview.md`, contendo diagramas Mermaid de componentes, fluxo de sequência do *Grid Snapping* + *Cache-Aside* no Redis, fórmula matemática do *Scoring Engine*, pipeline geométrico de recorte circular das ruas em `150m`, descoberta multi-fonte de estacionamentos em `400m` e topologia dos 4 containers Docker.

**Arquivos afetados:**  
- `docs/architecture/architecture-overview.md`  
- `README.md`  

---

### [HIGH] [FIX] Eliminação de Linhas Duplicadas de Multi-Zoom, Calçadas e Acessos Internos de Garagem

**Categoria:** Bugfix / Street Geometry Filtering  
**Importância:** Alta  
**Horário:** 11:05  

**Problema:**  
O mapa desenhava mais linhas do que as vias reais existentes: ruas simples de mão única (ex: *Rua Barbosa de Freitas*, *Rua Canuto de Aguiar*) apareciam com 2 ou 3 linhas paralelas sobre as calçadas, e surgiam ramificações entrando dentro dos quarteirões/edifícios.

**Causa:**  
1. `querySourceFeatures('transportation')` no MapLibre GL retorna simultaneamente as geometrias em cache de **todos os níveis de zoom carregados (`z12`, `z13`, `z14`, `z15`)**, onde os zooms menores têm coordenadas simplificadas deslocadas em alguns metros, gerando 2 a 3 linhas paralelas para a mesma rua.  
2. Vias de serviço internas (`highway=service`, `driveway`, `parking_aisle`) e alças de conversão (`*_link`) estavam incluídas na query, desenhando linhas em rampas de garagem dentro dos lotes.  
3. A condição anterior `if (overpassLines.length >= streetLines.value.length)` impedia que a topologia limpa do Overpass (1 eixo por rua) substituísse as linhas multiplicadas do basemap.

**Solução:**  
1. Removido `querySourceFeatures` da extração de vias em `App.vue`, utilizando apenas `queryRenderedFeatures` nas camadas de eixo do zoom atual (excluindo `case`/`casing`/`path`/`service`/`link`).  
2. Excluídas todas as vias de acesso interno (`service`, `driveway`, `parking_aisle`, `alley`), calçadas (`sidewalk`, `footway`), ciclovias e alças (`*_link`) tanto no basemap quanto na Overpass API, além de micro-fragmentos `< 12m`.  
3. Priorizada a substituição imediata pela geometria oficial do Overpass assim que `overpassLines.length > 0`.

**Validação:**  
Suite de 17 testes automatizados em `apps/frontend/tests/run-tests.ts` executada com 100% de aprovação.

**Arquivos afetados:**  
- `apps/frontend/src/composables/useStreetLines.ts`  
- `apps/frontend/src/App.vue`  
- `apps/frontend/tests/run-tests.ts`  

---

### [MEDIUM] [IMPROVEMENT] Ampliação do Motor de Descoberta de Estacionamentos (Multi-Tag OSM, Busca Textual Nominal e Raio de Caminhada)

**Categoria:** Improvement / Parking Discovery  
**Importância:** Média  
**Horário:** 10:51  

**Descrição:**  
Aprimorada a lógica de descoberta de estacionamentos para capturar estabelecimentos comerciais que não utilizam apenas a tag `amenity=parking`:  
1. **Multi-Tag & Regex Nominal no Overpass (`buildOverpassStreetsQuery` & `isParkingElement`):** A consulta agora captura `amenity=parking|parking_entrance|parking_space`, `building=parking` e qualquer POI/edifício cujo nome contenha `Estacionamento|Parking|Garagem|MultiPark`.  
2. **Busca Textual Local no Photon (`fetchLocalNamedParkings`):** Dispara busca georreferenciada por `"Estacionamento"` na *bounding box* local do destino pesquisado.  
3. **Raio de Descoberta de Estacionamentos (`400m`):** Mantém o recorte visual das linhas de rua em `150m`, mas busca estacionamentos comerciais num raio de caminhada de até `400m` ao redor do destino (incluindo *Estacionamento Ocean Tower* e *HVMM Estacionamento* na Orla do Mucuripe).

**Arquivos afetados:**  
- `apps/frontend/src/composables/useStreetLines.ts`  
- `apps/frontend/tests/run-tests.ts`  
- `apps/backend/internal/repository/postgres/spatial_repository.go`  

---

### [LOW] [IMPROVEMENT] Ajuste do Perímetro da Área Selecionada para 150 Metros

**Categoria:** Improvement / Domain & Spatial UX  
**Importância:** Baixa  
**Horário:** 09:55  

**Descrição:**  
Reduzido o raio padrão de busca e recorte do perímetro de `350m` para **`150m`** (`DefaultRadiusMeters = 150` e `DEFAULT_SEARCH_RADIUS_METERS = 150`), concentrando a análise e a visualização estritamente no quarteirão do destino e suas vias adjacentes imediatas.

**Arquivos afetados:**  
- `apps/backend/internal/domain/ports.go`  
- `apps/frontend/src/services/spotsApi.ts`  
- `apps/frontend/tests/run-tests.ts`  
- `docs/AGENTS.md`  

---

### [LOW] [IMPROVEMENT] Redução do Raio Padrão da Área Selecionada para 350 Metros (Distância de Caminhada Confortável)

**Categoria:** Improvement / Domain & Spatial UX  
**Importância:** Baixa  
**Horário:** 09:46  

**Descrição:**  
Reduzido o raio padrão de busca e visualização do perímetro de `1000m` para **`350m`** (~3 a 4 quarteirões ao redor do destino), focando apenas nas ruas e estacionamentos a uma distância de caminhada confortável do local pesquisado e proporcionando um enquadramento de câmera mais próximo e legível.

**Arquivos afetados:**  
- `apps/backend/internal/domain/ports.go`  
- `apps/frontend/src/services/spotsApi.ts`  
- `apps/frontend/src/App.vue`  
- `apps/frontend/tests/run-tests.ts`  
- `docs/AGENTS.md`  

---

### [HIGH] [FIX] Correção do Botão de Pesquisa Preso em Estado de Carregamento (`...`)

**Categoria:** Bugfix / Frontend UX  
**Importância:** Alta  
**Horário:** 09:36  

**Problema:**  
O botão azul ao lado da barra de busca ficava permanentemente exibindo `...` (`loading = true`), parecendo que o rótulo do botão de pesquisa havia desaparecido.

**Causa:**  
1. A verificação `if (!mapInstance.isStyleLoaded()) await new Promise((resolve) => mapInstance.once('load', resolve))` dentro de `focusPerimeterAndRenderStreets` podia bloquear indefinidamente em buscas subsequentes, pois no MapLibre GL `isStyleLoaded()` retorna `false` enquanto tiles novos estão carregando, mas o evento `'load'` inicial já havia disparado anteriormente.  
2. O `await overpassPromise` mantinha `loading = true` até os servidores públicos da Overpass API responderem.

**Solução:**  
1. Criada a função `waitForMapReady(1500)` com flag `mapLoaded` e timeout de segurança de 1.5s para nunca bloquear em `once('load')`.  
2. Desacoplado o enriquecimento da Overpass API em background com timeout HTTP de 5.5s (`AbortController`), liberando o estado `loading = false` imediatamente após o enquadramento do mapa.  
3. Atualizado o botão para **`🔍 Pesquisar`** conectado à função `submitSearch()`, permitindo pesquisar diretamente o endereço digitado ao clicar no botão ou pressionar `Enter`.

**Validação:**  
Suite de 17 testes automatizados em `apps/frontend/tests/run-tests.ts` executada com 100% de aprovação.

**Arquivos afetados:**  
- `apps/frontend/src/App.vue`  
- `apps/frontend/src/composables/useStreetLines.ts`  

---

### [MEDIUM] [FEATURE] Descoberta e Renderização de Estacionamentos Reais na Área Selecionada

**Categoria:** Feature / Spatial Discovery & Map Markers  
**Importância:** Média  
**Horário:** 09:24  

**Descrição:**  
Implementada via TDD a descoberta, filtragem espacial e exibição visual no mapa de estacionamentos reais (`amenity=parking`) existentes dentro do perímetro selecionado:  
- Expandida a consulta Overpass (`buildOverpassStreetsQuery`) e implementado `extractOverpassParkings`, `extractBasemapParkings` e `mergeAreaParkings` em `useStreetLines.ts` para buscar estacionamentos reais (nós e polígonos OSM + POIs dos tiles vetoriais) estritamente dentro do raio selecionado (`distanceMeters <= radiusMeters`).  
- Removida a geração de estacionamento fictício de fallback no backend (`spatial_repository.go`) e no frontend (`App.vue`), garantindo que estacionamentos só apareçam quando realmente existirem na área selecionada.  
- Adicionados marcadores interativos `P` no mapa com badge de capacidade e popup detalhado (nome, capacidade, distância até o destino e status), além de clique na lista lateral para localizar e abrir o estacionamento no mapa.

**Módulos afetados:**  
- `apps/frontend/src/composables/useStreetLines.ts`  
- `apps/frontend/src/App.vue`  
- `apps/frontend/tests/run-tests.ts`  
- `apps/backend/internal/repository/postgres/spatial_repository.go`  

---

## 2026-11-19

### [LOW] [IMPROVEMENT] Restauração da Perspectiva 3D Inclinada (Pitch 45° / Bearing -12°) no Mapa

**Categoria:** Improvement / Map UX  
**Importância:** Baixa  
**Horário:** 13:19  

**Descrição:**  
Restaurada a inclinação de câmera (`pitch: 45`, `bearing: -12`) na inicialização do mapa e na transição `fitBounds` do perímetro em `App.vue`, preservando a estética de navegação 3D estilo Waze enquanto mantém a extração e recorte exato das vias dentro do círculo de 1 km.

**Arquivos afetados:**  
- `apps/frontend/src/App.vue`  

---

### [HIGH] [FIX] Correção de Race Condition na Extração das Ruas do Perímetro durante Animação de Câmera

**Categoria:** Bugfix / Map Rendering  
**Importância:** Alta  
**Horário:** 13:06  

**Problema:**  
Ao pesquisar ou clicar em um novo endereço (ex: *Avenida Dom Manuel / Centro*), o círculo pontilhado de 1 km era desenhado no mapa, porém nenhuma linha colorida aparecia sobre as ruas dentro do perímetro.

**Causa:**  
1. A função anterior `waitForMapIdle()` escutava apenas `mapInstance.once('idle')` logo após iniciar `mapInstance.fitBounds(..., { duration: 1200 })`. O evento `idle` disparava no primeiro frame **antes** da câmera terminar de se deslocar da coordenada anterior (*Beira Mar*) até o novo destino (*Centro*).  
2. Como consequência, `querySourceFeatures` lia apenas os tiles da região antiga, e o recorte `clipLineToCircle` descartava 100% das vias por estarem fora do círculo do novo destino.

**Solução:**  
1. Substituída `waitForMapIdle` por `waitForCameraAndTiles`, aguardando explicitamente o evento `moveend` (término do deslocamento da câmera até o novo perímetro) seguido da confirmação de carregamento dos tiles (`areTilesLoaded` / `idle`).  
2. Combinada a leitura via `queryRenderedFeatures` (todas as camadas `type: 'line'` de transporte visíveis no canvas) + `querySourceFeatures`, além de um listener reativo `mapInstance.on('idle')` que sincroniza automaticamente as ruas do perímetro ativo sempre que novos vector tiles terminam de carregar.  
3. Adicionada busca paralela na Overpass API com múltiplos mirrors (`overpass-api.de` e `overpass.kumi.systems`) para enriquecimento das vias sem bloquear a renderização imediata do basemap.

**Impacto:**  
Todas as ruas dentro do círculo de 1 km passam a ser extraídas, recortadas na borda exata do perímetro e pintadas sobre o traçado real das vias assim que a câmera chega ao endereço pesquisado.

**Validação:**  
Suite de 15 testes automatizados em `apps/frontend/tests/run-tests.ts` executada com 100% de aprovação.

**Arquivos afetados:**  
- `apps/frontend/src/App.vue`  
- `apps/frontend/src/composables/useStreetLines.ts`  

---

### [MEDIUM] [IMPROVEMENT] Substituição de Hexágonos por Linhas Luminosas sobre o Traçado Real das Ruas (LineString)

**Categoria:** Improvement / Spatial Visualization  
**Importância:** Média  
**Horário:** 12:49  

**Descrição:**  
Substituída a camada de polígonos hexagonais no mapa por linhas vetoriais GeoJSON (`LineString`) desenhadas exatamente sobre o traçado das ruas no entorno do destino pesquisado (estilo Waze / Google Traffic):  
- Consulta em tempo real da geometria das vias (`out geom;`) via Overpass API para o raio ao redor do endereço selecionado em Fortaleza.  
- Classificação cromática por trecho de via (`#10b981` Verde = Alta chance de vaga; `#f59e0b` Âmbar = Moderada; `#ef4444` Vermelho = Proibido estacionar ou via saturada por evento).  
- Renderização dupla em WebGL (`spotsense-street-glow` com blur e `spotsense-street-core` nítido) e legenda visual no mapa.

**Módulos afetados:**  
- `apps/frontend/src/types/spots.ts`  
- `apps/frontend/src/composables/useHexGrid.ts`  
- `apps/frontend/src/App.vue`  
- `apps/frontend/tests/run-tests.ts`  

---

### [MEDIUM] [FEATURE] Motor de Busca Dinâmica e Autocomplete Restrito a Fortaleza - CE

**Categoria:** Feature / Geocoding & Spatial UX  
**Importância:** Média  
**Horário:** 12:20  

**Descrição:**  
Implementado via TDD o motor de busca dinâmica (*Search-as-you-type*) focado exclusivamente em **Fortaleza - CE** (`FORTALEZA_BBOX: [-38.64, -3.89, -38.40, -3.68]`):  
- Integração com **Photon OSM API** (`photon.komoot.io/api/`) e fallback **Nominatim** parametrizados com `bbox` e *location bias* no centro de Fortaleza (`-3.7319, -38.5267`).  
- Filtro estrito de coordenadas (`isWithinFortaleza`), deduplicação de logradouros e classificação por categoria (`street`, `venue`, `district`).  
- UX de autocomplete com *Debounce* (280ms), cancelamento de requisições concorrentes via `AbortController`, navegação por teclado (`ArrowUp`, `ArrowDown`, `Enter`, `Escape`), marcador (*Pin*) no destino, clique direto no mapa e trava de `maxBounds` da câmera em Fortaleza.  
- Expansão do repositório espacial no Backend Go para cobrir todos os bairros e polos de Fortaleza (Meireles, Aldeota, Iguatemi, Dragão do Mar, RioMar, Castelão e malha urbana local).

**Módulos afetados:**  
- `apps/frontend/src/services/spotsApi.ts`  
- `apps/frontend/src/App.vue`  
- `apps/frontend/tests/run-tests.ts`  
- `apps/backend/internal/repository/postgres/spatial_repository.go`  
- `apps/backend/internal/repository/postgres/spatial_repository_test.go`  
- `docs/AGENTS.md`  

---

### [HIGH] [FIX] Correção de ERR_CONNECTION_REFUSED causado por encerramento automático da VM do WSL2

**Categoria:** Bugfix / Infrastructure  
**Importância:** Alta  
**Horário:** 11:51  

**Problema:**  
Ao acessar `http://localhost:5173` no navegador do host Windows, a conexão foi recusada (`ERR_CONNECTION_REFUSED`). A inspeção via `wsl docker ps -a` revelou que todos os 4 containers estavam com status `Exited (0)`.

**Causa:**  
No Windows WSL2, quando um comando é executado em modo detached (`wsl docker compose up -d`) e não há nenhuma sessão ou processo ativo mantendo a distribuição WSL aberta, o serviço do Windows suspende automaticamente a VM do WSL após alguns segundos de inatividade (*WSL Idle Timeout*), encerrando o daemon do Docker e parando os containers.

**Solução:**  
1. Adicionada a política `restart: unless-stopped` em todos os serviços (`postgis`, `redis`, `backend`, `frontend`) no `docker/docker-compose.yml`.  
2. Iniciado o `wsl docker compose -f docker/docker-compose.yml up` como processo daemon persistente (`IsDaemon: true`), mantendo a sessão do WSL ativa e os containers acessíveis em `localhost:5173` e `localhost:8080`.

**Impacto:**  
Os containers permanecem ativos continuamente enquanto o daemon estiver em execução, permitindo o acesso pelo navegador do Windows.

**Arquivos afetados:**  
- `docker/docker-compose.yml`  

---

### [MEDIUM] [INFRA] Containerização Completa da Stack (PostGIS, Redis, Backend Go e Frontend Vue 3)

**Categoria:** Infrastructure / Docker  
**Importância:** Média  
**Horário:** 11:47  

**Descrição:**  
Criados os `Dockerfile`s multi-stage para o Backend (`apps/backend/Dockerfile`) e Frontend (`apps/frontend/Dockerfile` com Nginx e proxy reverso `/api/` -> `http://backend:8080/api/`).  
Atualizado o `docker/docker-compose.yml` para orquestrar os 4 containers (`spotsense-postgis`, `spotsense-redis`, `spotsense-backend`, `spotsense-frontend`) com healthchecks e inicialização automática da migration `001_init_postgis.sql`.  
Stack compilada e iniciada com sucesso via WSL Docker Compose.

**Módulos afetados:**  
- `apps/backend/Dockerfile`  
- `apps/frontend/Dockerfile`  
- `apps/frontend/nginx.conf`  
- `docker/docker-compose.yml`  

---

## 2026-11-18

### [MEDIUM] [FEATURE] Implementação do Frontend Vue 3 + Mapa Interativo + Malha Hexagonal

**Categoria:** Feature / Frontend & Spatial UI  
**Importância:** Média  
**Horário:** 12:25  

**Descrição:**  
Implementada a Fase 4 do projeto em `apps/frontend` (Vue 3 + TypeScript + Vite):  
- Cliente HTTP tipado (`spotsApi.ts`) consumindo `GET /api/vagas?lat={x}&lng={y}&radius=1000` e integrando com a Mapbox Geocoding API (com fallback para destinos pré-configurados).  
- Gerador espacial de malha hexagonal (`useHexGrid.ts`) que constrói polígonos GeoJSON de 6 lados no raio da tela atual ponderados pela proximidade de estacionamentos privados e eventos locais.  
- Interface principal (`App.vue`) estilo Waze/Google Maps com transição *FlyTo*, renderização de hexágonos coloridos por probabilidade (`HIGH`/`MEDIUM`/`LOW`), badge de status do Grid Snapping (`CACHE HIT` vs `POSTGIS QUERY`) e listagem de vias públicas, estacionamentos e eventos.  
- Suite de testes do Frontend executada e validada com 100% de aprovação.

**Módulos afetados:**  
- `apps/frontend/src/types/spots.ts`  
- `apps/frontend/src/services/spotsApi.ts`  
- `apps/frontend/src/composables/useHexGrid.ts`  
- `apps/frontend/src/App.vue`  
- `apps/frontend/tests/run-tests.ts`  
- `README.md`  

---

### [MEDIUM] [FEATURE] Implementação dos Workers de Ingestão (OpenStreetMap & Eventos Regionais)

**Categoria:** Feature / Background Workers & Ingestion  
**Importância:** Média  
**Horário:** 12:13  

**Descrição:**  
Implementados via TDD os parsers e o serviço de ingestão em background:  
- `ParseOverpassResponse`: extrai segmentos viários (`highway=*` com verificação de `parking:both=no`) e estacionamentos dedicados (`amenity=parking`, `capacity`) do payload JSON do OpenStreetMap.  
- `ParseRegionalEventsResponse`: converte o payload JSON da API de busca de eventos em `domain.Event` validando coordenadas e janela temporal RFC3339.  
- `IngestionWorkerService`: executa ciclos idempotentes (`RunSync`) persistindo via interface `SpatialStore`.  
- Ponto de entrada `cmd/worker/main.go` com suporte a execução única (`--once`) ou agendada (ticker de 6h).

**Módulos afetados:**  
- `apps/backend/internal/domain/ports.go`  
- `apps/backend/internal/service/ingestion.go`  
- `apps/backend/internal/service/ingestion_test.go`  
- `apps/backend/cmd/worker/main.go`  

---

### [MEDIUM] [FEATURE] Implementação dos Repositórios PostGIS (ST_DWithin), Cache Redis (RESP2) e Entrypoint da API

**Categoria:** Feature / Persistence & Infrastructure  
**Importância:** Média  
**Horário:** 12:04  

**Descrição:**  
Implementados via TDD os adaptadores de infraestrutura para persistência e cache:  
- `PostGISSpatialRepository` utilizando `ST_DWithin` e `ST_MakePoint` sobre os índices `GIST`.  
- `SeedSpatialRepository` com cálculo geodésico Haversine em metros para execução local rápida e testes determinísticos.  
- `RedisSpotCache` utilizando protocolo RESP2 nativo (`GET` e `SET ... EX`) com injeção de `DialFunc` testada em memória via `net.Pipe()`, além de `MemorySpotCache` thread-safe com TTL.  
- Ponto de entrada `cmd/api/main.go` com graceful shutdown e logs estruturados (`log/slog`).  
Verificado também o Docker v29.9.3 via WSL (`wsl docker`).

**Módulos afetados:**  
- `apps/backend/internal/domain/coordinates.go`  
- `apps/backend/internal/repository/postgres/spatial_repository.go`  
- `apps/backend/internal/repository/postgres/spatial_repository_test.go`  
- `apps/backend/internal/repository/redis/cache.go`  
- `apps/backend/internal/repository/redis/cache_test.go`  
- `apps/backend/cmd/api/main.go`  

---

### [MEDIUM] [FEATURE] Implementação do Endpoint REST GET /api/vagas e /healthz

**Categoria:** Feature / HTTP Delivery  
**Importância:** Média  
**Horário:** 11:51  

**Descrição:**  
Implementado via TDD o pacote `internal/delivery/http` com o roteador e handler para `GET /api/vagas?lat={x}&lng={y}&radius=1000` e `GET /healthz`.  
A camada aplica *Boundary Discipline* validando e convertendo os query params na fronteira HTTP, separando DTOs de transporte das entidades de domínio e retornando respostas de erro padronizadas (`INVALID_COORDINATES`, `INVALID_RADIUS`).  
Cobertura de testes atingiu **88.5%** no pacote `delivery/http`.

**Módulos afetados:**  
- `apps/backend/internal/delivery/http/handler.go`  
- `apps/backend/internal/delivery/http/handler_test.go`  

---

### [MEDIUM] [FEATURE] Implementação do Caso de Uso SearchSpotsService (Cache-Aside + Grid Snapping) e Migration PostGIS

**Categoria:** Feature / Application & Persistence  
**Importância:** Média  
**Horário:** 11:44  

**Descrição:**  
Definidas as portas de saída no domínio (`SpatialRepository` e `SpotCache`) e implementado via TDD o caso de uso `SearchSpotsService`.  
O serviço aplica validação de raio (100m..3000m, padrão 1000m), Grid Snapping das coordenadas de busca e estratégia Cache-Aside: validado em teste que 50 requisições próximas na mesma micro-região (~100m) consultam o repositório espacial apenas 1 vez, servindo as 49 seguintes diretamente do cache.  
Criada também a migration `001_init_postgis.sql` com extensão PostGIS e índices espaciais `GIST`.

**Módulos afetados:**  
- `apps/backend/internal/domain/ports.go`  
- `apps/backend/internal/service/search_spots.go`  
- `apps/backend/internal/service/search_spots_test.go`  
- `apps/backend/migrations/001_init_postgis.sql`  

---

### [MEDIUM] [FEATURE] Implementação das Entidades de Oferta/Demanda e Motor Heurístico de Scoring

**Categoria:** Feature / Core Domain & Scoring  
**Importância:** Média  
**Horário:** 11:34  

**Descrição:**  
Modeladas as entidades puras de domínio (`RoadSegment`, `PrivateParking`, `Event` e `ProbabilityLevel`) e implementado via TDD o motor heurístico de cálculo de probabilidade de vagas (`scoring.Calculate`).  
A heurística relaciona oferta efetiva (vias permitidas + estacionamentos abertos) contra demanda base por faixa horária e demanda extra gerada por eventos ativos (1 veículo a cada 3 participantes).  
Cobertura de testes atingiu **94.9%** no pacote `scoring` e **88.9%** no pacote `domain`.

**Módulos afetados:**  
- `apps/backend/internal/domain/entities.go`  
- `apps/backend/internal/scoring/calculator.go`  
- `apps/backend/internal/scoring/calculator_test.go`  
- `docs/AGENTS.md`  

---

### [MEDIUM] [FEATURE] Implementação via TDD do Value Object Coordinates e Grid Snapping

**Categoria:** Feature / Core Domain  
**Importância:** Média  
**Horário:** 11:20  

**Descrição:**  
Implementado o Value Object imutável `Coordinates` no domínio com validação de limites geográficos (-90..90 para latitude e -180..180 para longitude).  
Desenvolvido o algoritmo de **Grid Snapping** (~100m) que normaliza latitude e longitude para 3 casas decimais (com proteção contra -0.0) e gera a chave determinística para o Redis (`grid:{lat}:{lng}`).  
Ciclo completo de TDD (Red -> Green -> Refactor) executado com 100% de sucesso.

**Módulos afetados:**  
- `apps/backend/internal/domain/coordinates.go`  
- `apps/backend/internal/domain/coordinates_test.go`  

---

### [LOW] [ENV] Instalação do Ambiente Go no Host

**Categoria:** Environment  
**Importância:** Baixa  
**Horário:** 11:11  

**Descrição:**  
Instalado o compilador Go (versão 1.27.4 windows/amd64) no host do sistema operacional via Windows Package Manager (`winget`). Validação executada com sucesso.

**Módulos afetados:**  
- Ambiente de desenvolvimento local  

---

### [MEDIUM] [DOCS] Inicialização da Governança e Especificação do Projeto

**Categoria:** Architecture / Governance  
**Importância:** Média  
**Horário:** 11:00  

**Descrição:**  
Realizada a estruturação técnica do projeto Spot Sense Web App a partir da descrição inicial do MVP.  
Definidas as decisões de arquitetura (Monorepo, Clean Architecture, Go 1.23+ com Chi, Vue 3, PostGIS com índices GIST, Redis para cache de grid snapping e ingestor de eventos em Go).  
Preenchido o `docs/AGENTS.md` e criados os documentos de suporte `development-plan.md`, `progress.md` e `development-log.md`.

**Módulos afetados:**  
- `docs/AGENTS.md`  
- `docs/development/`  
