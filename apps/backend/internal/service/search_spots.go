package service

import (
	"context"
	"fmt"
	"time"

	"github.com/spot-sense/backend/internal/domain"
	"github.com/spot-sense/backend/internal/scoring"
)

// SearchInput contém os parâmetros validados para consulta de vagas.
type SearchInput struct {
	Coordinates domain.Coordinates
	Radius      int
	QueryTime   time.Time
}

// SearchSpotsService orquestra a consulta de vagas com estratégia Cache-Aside e Grid Snapping.
type SearchSpotsService struct {
	repo  domain.SpatialRepository
	cache domain.SpotCache
}

// NewSearchSpotsService constrói uma nova instância do serviço de busca.
func NewSearchSpotsService(repo domain.SpatialRepository, cache domain.SpotCache) *SearchSpotsService {
	return &SearchSpotsService{
		repo:  repo,
		cache: cache,
	}
}

// Execute realiza a busca de vagas aplicando Grid Snapping, Cache e Scoring heurístico.
func (s *SearchSpotsService) Execute(ctx context.Context, input SearchInput) (domain.ZoneAssessment, error) {
	radius := input.Radius
	if radius == 0 {
		radius = domain.DefaultRadiusMeters
	}
	if radius < domain.MinRadiusMeters || radius > domain.MaxRadiusMeters {
		return domain.ZoneAssessment{}, domain.ErrInvalidRadius
	}

	snappedCoords := input.Coordinates.SnapToGrid()
	cacheKey := fmt.Sprintf("spots:v1:%s:r:%d", snappedCoords.GridKey(), radius)

	if s.cache != nil {
		cached, found, err := s.cache.Get(ctx, cacheKey)
		if err == nil && found && cached != nil {
			cached.FromCache = true
			return *cached, nil
		}
	}

	spatialData, err := s.repo.FindWithinRadius(ctx, snappedCoords, radius)
	if err != nil {
		return domain.ZoneAssessment{}, err
	}

	queryTime := input.QueryTime
	if queryTime.IsZero() {
		queryTime = time.Now()
	}

	scoreResult := scoring.Calculate(scoring.Input{
		Roads:     spatialData.Roads,
		Parkings:  spatialData.Parkings,
		Events:    spatialData.Events,
		QueryTime: queryTime,
	})

	assessment := domain.ZoneAssessment{
		GridCoordinates: snappedCoords,
		RadiusMeters:    radius,
		Score:           scoreResult.Score,
		Probability:     scoreResult.Probability,
		TotalSupply:     scoreResult.TotalSupply,
		EstimatedDemand: scoreResult.EstimatedDemand,
		ActiveEvents:    scoreResult.ActiveEvents,
		Roads:           spatialData.Roads,
		Parkings:        spatialData.Parkings,
		Events:          spatialData.Events,
		FromCache:       false,
	}

	if s.cache != nil {
		_ = s.cache.Set(ctx, cacheKey, assessment)
	}

	return assessment, nil
}
