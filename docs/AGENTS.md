# AGENTS.md — Project Governance & AI Pair Programming

> Documento central de governança técnica, arquitetura e colaboração com agentes de IA.
>
> Este documento estabelece as diretrizes que devem ser seguidas por qualquer agente de IA que participe do desenvolvimento deste projeto.
>
> **Regra principal:** Nenhuma implementação deve começar sem que o agente compreenda o contexto, valide os requisitos e alinhe a abordagem técnica com o desenvolvedor. Utilize a skill em `ub-mode` (seção 13) para auxiliar nas decisões de melhores práticas.

---

# 1. Project Overview

## 1.1. Identificação

- **Nome do projeto:** Spot Sense Web App
- **Descrição:** Aplicação web para estimativa de probabilidade de vagas de estacionamento (em vias públicas e estacionamentos privados) e monitoramento de eventos de impacto em uma determinada área geográfica e horário.
- **Objetivo principal:** Auxiliar motoristas a planejar o estacionamento em destinos movimentados ou desconhecidos, apresentando visualização espacial e heurística de vagas prováveis num raio de até 1.000 metros.
- **Público-alvo:** Motoristas urbanos que necessitam encontrar vagas ao se deslocar para regiões movimentadas ou eventos.
- **Status:** PLANNING
- **Responsável técnico:** Naelton

## 1.2. Visão do Produto

Motoristas frequentemente enfrentam estresse, atrasos e desperdício de combustível ao buscar vagas de estacionamento em regiões desconhecidas ou durante eventos com alta demanda.

O **Spot Sense** resolve essa incerteza ao prover um mapa inteligente e estático/probabilístico (sem necessidade de sensores IoT em tempo real na fase MVP) focado no destino informado pelo usuário e no seu entorno de 1 km. A solução combina dados geoespaciais de vias públicas (OpenStreetMap), cadastro de estacionamentos privados e um scraper de eventos locais para calcular um índice de probabilidade de vagas (*Scoring*), exibido de forma visual e intuitiva no mapa.

## 1.3. Escopo

### Funcionalidades principais
- **Busca por Localização (Geocoding):** Conversão de endereços em coordenadas geográficas via Mapbox Geocoding.
- **Visualização em Mapa Dinâmico:** Interface fluida (estilo Waze/Maps) com transição *FlyTo* e renderização de camadas em GPU via Deck.gl.
- **Mapa de Prováveis Vagas (Hexagonal Heatmap):** Renderização de hexágonos de calor indicando a densidade/probabilidade de vagas.
- **Exibição de Estacionamentos Privados:** Marcação de estacionamentos comerciais com capacidade e informações na área de busca.
- **Mapeamento de Eventos Locais:** Exibição de eventos que impactam o tráfego e a disponibilidade de vagas.
- **Cálculo Heurístico de Vagas (Scoring):** Motor que relaciona oferta viária/privada com demanda gerada por eventos e horário.
- **Cache de Alta Performance com Grid Snapping:** Normalização de coordenadas para 3 casas decimais (~100m) e cacheamento no Redis para atender múltiplos usuários simultâneos no mesmo raio geográfico.
- **Workers de Ingestão em Background:** Rotinas em Go para importar dados do OpenStreetMap e coletar eventos regionais periodicamente.

### Fora do escopo (MVP)
- Telemetria de sensores IoT ao vivo em cada vaga de rua.
- Reserva e pagamento de vagas privadas pelo aplicativo.
- Navegação GPS passo-a-passo (turn-by-turn).
- Aplicativo mobile nativo (o MVP é Web responsivo).

### Restrições e premissas
- **Região Geográfica do MVP:** O primeiro MVP é focado exclusivamente em **Fortaleza - CE** (`Bounding Box: [-38.64, -3.89, -38.40, -3.68]`). O autocomplete de endereços e a câmera do mapa devem priorizar e restringir resultados a Fortaleza.
- O raio padrão de consulta espacial é de **150 metros** (~1 a 2 quarteirões imediatos ao redor do ponto de destino).
- O backend deve ser desenvolvido em Go garantindo alta performance e baixo consumo de recursos.
- O banco de dados geoespacial deve ser PostgreSQL com extensão PostGIS e índices espaciais `GIST`.
- As consultas devem ser cacheadas no Redis com estratégia de Grid Snapping.

---

# 2. Architecture & Technical Decisions

## 2.1. Padrão Arquitetural

- **Arquitetura adotada:** Monolito Modular estruturado em Monorepo, com aplicação dos princípios de Clean Architecture / Ports & Adapters no Backend.
- **Estilo de comunicação:** RESTful HTTP (JSON).
- **Organização:** Monorepo (`apps/backend` e `apps/frontend`).
- **Estratégia de persistência:** PostgreSQL 16 com extensão PostGIS 3.4 para dados geoespaciais; Redis 7 para camada de cache chave-valor com TTL.

### Princípios arquiteturais

1. Separação de responsabilidades.
2. Baixo acoplamento entre módulos.
3. Alta coesão.
4. Inversão de dependência (Domínio isolado de frameworks e drivers).
5. Independência das regras de negócio em relação à infraestrutura.
6. Código testável e extensível (TDD em primeiro lugar).
7. Evitar abstrações prematuras (KISS / YAGNI).

## 2.2. Regras de Dependência

- **Camada de Domínio (`internal/domain`):** Contém entidades puras, objetos de valor e interfaces. Não importa pacotes de infraestrutura, HTTP, PostGIS ou Redis.
- **Camada de Aplicação / Serviços (`internal/service`):** Orquestra casos de uso e heurísticas de scoring, dependendo apenas das interfaces de domínio.
- **Camada de Infraestrutura / Repositórios (`internal/repository`):** Implementa os contratos de acesso a dados (PostGIS via SQL, Redis via cliente Go).
- **Camada de Entrega (`internal/delivery`):** Expõe as APIs REST (`http`) e dispara jobs do cron (`cron`). Depende dos serviços de aplicação.
- **Frontend (`apps/frontend`):** Comunica-se exclusivamente através dos endpoints públicos da API REST e da API do Mapbox.

## 2.3. Architecture Decision Records (ADR)

Decisões arquiteturais relevantes devem ser documentadas em: `docs/architecture/decisions/`.

Decisões iniciais alinhadas:
- **ADR-001:** Adoção de Monorepo com Go (`apps/backend`) e Vue.js 3 (`apps/frontend`).
- **ADR-002:** Utilização de PostGIS com índices GIST e query `ST_DWithin` para isolamento de dados no raio de busca.
- **ADR-003:** Estratégia de Cache com Grid Snapping (3 casas decimais / ~100m) persistido em Redis.
- **ADR-004:** Ingestor de Eventos implementado em Go nativo consumindo APIs públicas de busca de plataformas de eventos.

---

# 3. Technology Stack

## 3.1. Tecnologias

| Camada | Tecnologia | Versão | Justificativa |
|---|---|---|---|
| Backend | Go | 1.23+ | Alta performance, baixo uso de memória, concorrência nativa para workers. |
| Roteador HTTP | `go-chi/chi/v5` | v5.x | Roteador leve, idiomático, compatível 100% com `net/http`. |
| Frontend | Vue.js (Vite + TS) | 3.x | Reatividade moderna, setup rápido e excelente performance. |
| Camada Geoespacial | Mapbox GL JS + Deck.gl | v3.x / v9.x | Renderização de mapas vetoriais e camadas hexagonais aceleradas por WebGL/GPU. |
| Database | PostgreSQL + PostGIS | 16 / 3.4 | Padrão da indústria para operações geoespaciais e índices GIST. |
| Cache | Redis | 7.x | Armazenamento em memória com expiração por TTL para requisições de grid. |
| Testes Backend | `testing` (Go stdlib) + `testify` | - | Testes unitários rápidos e expressivos para TDD. |
| Testes Frontend | Vitest + Vue Test Utils | - | Testes rápidos de componentes e composables. |
| Infraestrutura Dev | Docker + Docker Compose | - | Subida automatizada de PostgreSQL/PostGIS e Redis local. |

---

# 4. Repository Structure

## 4.1. Estrutura de Diretórios (Monorepo)

```text
spot-sense-web-app/
├── apps/
│   ├── backend/
│   │   ├── cmd/
│   │   │   ├── api/                 # Ponto de entrada da API REST
│   │   │   └── worker/              # Ponto de entrada do worker de ingestão
│   │   ├── internal/
│   │   │   ├── domain/              # Entidades, Value Objects e Interfaces
│   │   │   ├── scoring/             # Heurística de cálculo de probabilidade de vagas
│   │   │   ├── service/             # Casos de uso (SearchSpots, Ingestion)
│   │   │   ├── repository/
│   │   │   │   ├── postgres/        # Implementação PostGIS (GIST / ST_DWithin)
│   │   │   │   └── redis/           # Implementação do Cache com Grid Snapping
│   │   │   ├── delivery/
│   │   │   │   ├── http/            # Handlers REST e rotas Chi
│   │   │   │   └── cron/            # Agendamento e triggers de workers
│   │   │   └── platform/            # Config, DB connection, Redis connection, Logger
│   │   ├── tests/
│   │   │   ├── integration/         # Testes com containers reais
│   │   │   └── fixtures/            # GeoJSON e massas de dados de teste
│   │   ├── go.mod
│   │   └── go.sum
│   │
│   └── frontend/
│       ├── src/
│       │   ├── assets/
│       │   ├── components/          # MapView, SearchBar, HexagonLayer, Legend
│       │   ├── composables/         # useMap, useSpotsSearch
│       │   ├── services/            # Cliente HTTP da API Go
│       │   ├── types/               # Tipagens TypeScript dos contratos da API
│       │   ├── App.vue
│       │   └── main.ts
│       ├── package.json
│       └── vite.config.ts
│
├── docs/                            # Governança e documentação técnica
│   ├── AGENTS.md
│   ├── architecture/
│   │   └── decisions/
│   ├── requirements/
│   └── development/
│       ├── development-plan.md
│       ├── progress.md
│       └── development-log.md
│
├── docker/
│   └── docker-compose.yml           # Postgres+PostGIS e Redis para dev
├── .env.example
├── .gitignore
└── README.md
```

## 4.2. Organização de Arquivos e Convenções

- **Entidades de Domínio:** `apps/backend/internal/domain/`
- **Interfaces de Repositório/Serviço:** `apps/backend/internal/domain/`
- **Regras de Negócio & Scoring:** `apps/backend/internal/scoring/`
- **Casos de Uso:** `apps/backend/internal/service/`
- **Implementações de Banco:** `apps/backend/internal/repository/postgres/`
- **Implementação de Cache:** `apps/backend/internal/repository/redis/`
- **Controllers/Handlers:** `apps/backend/internal/delivery/http/`
- **Testes Unitários:** Arquivos `*_test.go` colocados no mesmo pacote do código testado.
- **Testes de Integração:** `apps/backend/tests/integration/`

---

# 5. Environment & Configuration

## 5.1. Variáveis de Ambiente

| Variável | Descrição | Exemplo / Padrão | Obrigatória |
|---|---|---|---|
| `PORT` | Porta HTTP da API Go | `8080` | Sim |
| `DATABASE_URL` | String de conexão PostGIS | `postgres://user:pass@localhost:5432/spotsense?sslmode=disable` | Sim |
| `REDIS_URL` | String de conexão Redis | `redis://localhost:6379/0` | Sim |
| `CACHE_TTL_MINUTES` | Tempo de expiração do cache | `10` | Não (default: 10) |
| `VITE_MAPBOX_TOKEN` | Token público do Mapbox | `pk.eyJ...` | Sim (Frontend) |
| `ENVIRONMENT` | Ambiente de execução | `development` / `production` | Sim |

## 5.2. Convenções

- Nunca versionar arquivos `.env` com dados reais.
- Manter `.env.example` sempre completo e atualizado.
- O backend deve validar e falhar no startup (*Fail Fast*) se variáveis obrigatórias estiverem ausentes.

---

# 6. Domain Rules & Business Logic

## 6.1. Regras de Domínio

1. **Raio de Consulta Espacial:**
   - O raio padrão de consulta é de 1.000 metros ($r = 1000\text{m}$).
   - Consultas com raio superior a 3.000m ou inferior a 100m devem ser rejeitadas pela API.
2. **Grid Snapping de Coordenadas:**
   - As coordenadas de latitude e longitude recebidas no endpoint devem ser arredondadas para 3 casas decimais antes de gerar a chave de cache no Redis:
     $$\text{lat}_{\text{snap}} = \text{round}(\text{lat}, 3), \quad \text{lng}_{\text{snap}} = \text{round}(\text{lng}, 3)$$
   - Chave de cache gerada: `spots:v1:{lat_snap}:{lng_snap}:{radius}`
3. **Scoring de Probabilidade de Vagas:**
   - O score varia entre 0.0 (quase impossível estacionar) e 1.0 (alta facilidade):
     $$\text{Score} = \text{clamp}\left(1.0 - \frac{D}{O}, \; 0.0, \; 1.0\right)$$
   - O cálculo pondera:
     - **Oferta Efetiva ($O$):** Soma das vagas estimadas de vias públicas onde `ParkingAllowed == true` + capacidade dos estacionamentos privados onde `IsOpen == true`. Se $O \le 0$, o Score é $0.0$.
     - **Demanda Base:** Taxa de ocupação por faixa horária (Madrugada `00h-05h`: $15\%$; Comercial/Pico `08h-20h`: $50\%$; Transição: $30\%$).
     - **Demanda Extra / Eventos:** Eventos ativos no horário da consulta (ou iniciando em até 60 min) adicionam $1$ veículo para cada $3$ participantes estimados ($\text{ExpectedAttendance} / 3$).
   - **Classificação (`ProbabilityLevel`):**
     - `HIGH`: $\text{Score} \ge 0.70$
     - `MEDIUM`: $0.35 \le \text{Score} < 0.70$
     - `LOW`: $\text{Score} < 0.35$
4. **Isolamento Espacial no Banco:**
   - Toda consulta no PostGIS deve utilizar `ST_DWithin(geom, ST_SetSRID(ST_MakePoint(lng, lat), 4326), radius_in_meters, use_spheroid = false)` aproveitando o índice `GIST`.

## 6.2. Ingestão em Background

- **OSM Ingestor:** Coleta vias com tags permitidas de estacionamento e estacionamentos cadastrados para alimentar as tabelas espaciais.
- **Event Scraper:** Coleta eventos da região alvo (ex: plataformas de eventos), extraindo nome, data/hora e coordenadas para a tabela `events`.

---

# 7. Code Quality Standards

## 7.1. Princípios

- **XP Practices:** Simple Design, Small Releases, TDD contínuo.
- **ub-mode:** First Make it Work, Then Make it Right, Follow Practice.
- **KISS & YAGNI:** Não adicionar abstrações ou tecnologias desnecessárias sem evidência de necessidade.
- **Fail Fast:** Tratar erros explicitamente em Go; não ocultar falhas nem usar `panic` em tempo de execução normal.

## 7.2. Tratamento de Erros

- Respostas de erro HTTP devem seguir formato JSON padronizado:
  ```json
  {
    "error": {
      "code": "INVALID_COORDINATES",
      "message": "Latitude must be between -90 and 90 degrees."
    }
  }
  ```

---

# 8. Testing Strategy (TDD)

## 8.1. Estrutura de Testes

| Tipo | Escopo | Ferramenta |
|---|---|---|
| Unitários | Cálculo da Heurística de Score, Grid Snapping, DTOs e validações | Go `testing` + `testify` |
| Integração | Geo-queries no PostGIS com dados reais e operações de Cache no Redis | Go `testing` + containers locais |
| HTTP Contract | Handlers REST e respostas esperadas | `net/http/httptest` |
| Frontend | Renderização de componentes e lógica de composables | Vitest + Vue Test Utils |

## 8.2. Ciclo Red-Green-Refactor
- Nenhum algoritmo de negócio (especialmente Grid Snapping e Scoring) deve ser escrito sem que o teste unitário que falha tenha sido implementado previamente.

---

# 9. AI Pair Programming — Extreme Programming

## 9.1. Papel do Agente

O agente atua como parceiro de pair programming:
- Analisa problemas antes de propor soluções.
- Desenvolve em incrementos mínimos e funcionais.
- Mantém o desenvolvedor alinhado antes de qualquer mudança estrutural.
- Não avança etapas sem validação prévia.

## 9.2. Ciclo de Desenvolvimento
Understand -> Discuss -> Plan -> Test First (Red) -> Implement (Green) -> Refactor -> Review.

---

# 10. Documentation Governance

Documentos do projeto:
- `docs/AGENTS.md`: Fonte da verdade sobre arquitetura, convenções e governança.
- `docs/development/development-plan.md`: Plano macro de entregas.
- `docs/development/progress.md`: Estado atual do planejamento e entregas.
- `docs/development/development-log.md`: Diário de bordo cronológico de acontecimentos técnicos.

---

# 11. Definition of Done (DoD)

Uma tarefa somente pode ser considerada concluída quando:
- [ ] Requisitos atendidos e validados com o desenvolvedor.
- [ ] Testes unitários e de integração implementados e passando.
- [ ] Código revisado com zero complexidade desnecessária.
- [ ] Arquitetura respeitada (Clean Architecture + Monorepo).
- [ ] Documentação de progresso (`progress.md`) e diário (`development-log.md`) atualizados.

---

# 12. Initial Instructions for the AI Agent

Ao iniciar qualquer interação:
1. Ler este documento (`docs/AGENTS.md`).
2. Consultar `docs/development/progress.md` e `docs/development/development-log.md`.
3. Não alterar código antes de compreender e alinhar o contexto.
4. Trabalhar sempre no menor incremento funcional possível guiado por testes.

---

# 13. Skills

As skills estarão na pasta `docs/skills`. Se for necessário usar ou o usuário solicitar, utilize somente as que estão dentro dessa pasta.

---

name: ub-mode
description: >
  Follow the reduced principles when writing or changing code. Every change
  follows First Make it Work, then Then Make it Right, then Follow Practice.
  Use when the user runs /ub-mode or asks for the reduced principles.
---

# ub-mode

Every time you are asked to make a change follow:
* First Make it Work.
* Then Make it Right.
* Follow Practice

# First Make it Work

## Laziness Protocol

* One decision lives in one place.
* Ship the smallest edit that solves the problem.

## Foundational Thinking

* Get the data shape right before you write the logic.
* Do first whatever helps every later phase.
* If a concurrent writer would matter, do not share state.

## TDD

* Write the failing test before you change the code.
* See it fail for the reason you expect.
* Make the smallest change that makes it pass.
* If an honest test is impractical, do not write a bad one.

## Redesign From First Principles

* Do not bolt a new requirement onto the old design.
* Ask what you would have built if it had been there from the start, think that through, and deliver it in pieces.

## Attack the Premise

* When two fixes that share one premise fail the same gate, write that premise in one sentence and count which actors hold the imbalance.
* Remove the assignment.
* Do not start another fix until you have done that.

## Subtract Before You Add

* Take complexity out before you add anything.

## Outcome-Oriented Execution

* Aim at the end state you can check.
* A break in the middle is allowed when you planned it and can undo it.

## Build the Lever

* If the work is not obvious at a glance, build the tool that does it.
* Do one unit by hand, then make the tool match that unit.

## Architecture

### Model the Domain

* Put the domain in a structure that makes the illegal states impossible.
* Maintain a glossary of domain terms.
* If the code is already clear and local, leave it.

### Boundary Discipline

* Check the data where it enters.
* Inside, trust it.
* Keep the business logic in a pure function.

### Type System Discipline

* Make the illegal value impossible to write.
* Parse outside data once, where it comes in.
* Do not silence the compiler.

### Make Operations Idempotent

* Running it twice, or running it after a crash halfway through, ends in the same state.

### Separate Before Serializing Shared State

* If they do not need the same mutable thing, give each its own.
* If they do, allow one writer, and enforce that in the structure.

# Then Make it Right

## Fix Root Causes

* Reproduce it.
* Fix the cause.
* A guard that only hides the crash is not a fix.

## Sequence work into verifiable units

* Finish one small unit and check it.
* Then start the next.

## Test Behavior, Not Implementation

* The test calls the code the way its user does and checks the result that user would see.
* If the test would still pass when every function returned nothing, throw it out.

## Crapper

* Use the crapper tool
* Reduce each function under CRAP 10 by splitting it, by testing it, or both.
* Consider whether case statements are worth reducing.

## Dryer

* Use the dryer tool to eliminate duplication that rank 0.8 or above.

## Mutator

* Use the mutator tool
* Cover the uncovered.
* Kill the survivors.
* Delete dead code.

## Refactoring

* Pin what the code does.
* Change the structure.
* Prove the behavior did not change.

# Practice

## Keep a Log

* Keep a log of every action you take and the reason you took it.
* Report that log upon completion.

## Guard the Context Window

* Keep the main thread small.
* Send the bulk elsewhere, and keep the summary.

## Worktree Cleanup

* Find what is safe to remove, confirm nothing still in use needs it, then reclaim the disk.
* Uncommitted work waits for a decision.

## Commit 
* Suggest the description commit.
* Commit often, in small ordered commits.
* Each commit says why, what changed, and how you proved it.
