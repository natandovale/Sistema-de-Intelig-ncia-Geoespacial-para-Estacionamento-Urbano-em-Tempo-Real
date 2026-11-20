package domain

import (
	"context"
	"errors"
)

const (
	DefaultRadiusMeters = 150
	MinRadiusMeters     = 100
	MaxRadiusMeters     = 3000
)

// ErrInvalidRadius é retornado quando o raio de busca está fora dos limites permitidos (100m..3000m).
var ErrInvalidRadius = errors.New("search radius must be between 100 and 3000 meters")

// SpatialData agrupa as entidades geoespaciais encontradas no raio de busca.
type SpatialData struct {
	Roads    []RoadSegment
	Parkings []PrivateParking
	Events   []Event
}

// ZoneAssessment consolida a avaliação heurística de estacionamento para uma região.
type ZoneAssessment struct {
	GridCoordinates Coordinates      `json:"grid_coordinates"`
	RadiusMeters    int              `json:"radius_meters"`
	Score           float64          `json:"score"`
	Probability     ProbabilityLevel `json:"probability"`
	TotalSupply     int              `json:"total_supply"`
	EstimatedDemand int              `json:"estimated_demand"`
	ActiveEvents    int              `json:"active_events"`
	Roads           []RoadSegment    `json:"roads"`
	Parkings        []PrivateParking `json:"parkings"`
	Events          []Event          `json:"events"`
	FromCache       bool             `json:"from_cache"`
}

// SpatialRepository define a porta de saída para consultas geoespaciais no PostGIS.
type SpatialRepository interface {
	FindWithinRadius(ctx context.Context, center Coordinates, radiusMeters int) (SpatialData, error)
}

// SpotCache define a porta de saída para o cache de avaliações por Grid Snapping no Redis.
type SpotCache interface {
	Get(ctx context.Context, key string) (*ZoneAssessment, bool, error)
	Set(ctx context.Context, key string, value ZoneAssessment) error
}

// OSMIngestor define o contrato para coleta de vias públicas e estacionamentos do OpenStreetMap.
type OSMIngestor interface {
	FetchSpatialSupply(ctx context.Context, region string) ([]RoadSegment, []PrivateParking, error)
}

// EventIngestor define o contrato para coleta de eventos regionais que impactam a demanda.
type EventIngestor interface {
	FetchRegionalEvents(ctx context.Context, region string) ([]Event, error)
}

// SpatialStore define o contrato de persistência idempotente (Upsert) dos dados coletados pelos workers.
type SpatialStore interface {
	UpsertSupply(ctx context.Context, roads []RoadSegment, parkings []PrivateParking) error
	UpsertEvents(ctx context.Context, events []Event) error
}

