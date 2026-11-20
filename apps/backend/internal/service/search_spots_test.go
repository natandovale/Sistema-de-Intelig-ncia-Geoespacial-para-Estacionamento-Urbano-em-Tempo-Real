package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/spot-sense/backend/internal/domain"
	"github.com/spot-sense/backend/internal/service"
)

// fakeSpatialRepo simula o repositório PostGIS para testes isolados.
type fakeSpatialRepo struct {
	data       domain.SpatialData
	err        error
	callCount  int
	lastRadius int
	lastCenter domain.Coordinates
}

func (f *fakeSpatialRepo) FindWithinRadius(_ context.Context, center domain.Coordinates, radiusMeters int) (domain.SpatialData, error) {
	f.callCount++
	f.lastCenter = center
	f.lastRadius = radiusMeters
	return f.data, f.err
}

// fakeSpotCache simula o Redis em memória para validar o comportamento de Grid Snapping + Cache.
type fakeSpotCache struct {
	store    map[string]domain.ZoneAssessment
	getCalls int
	setCalls int
}

func newFakeSpotCache() *fakeSpotCache {
	return &fakeSpotCache{
		store: make(map[string]domain.ZoneAssessment),
	}
}

func (c *fakeSpotCache) Get(_ context.Context, key string) (*domain.ZoneAssessment, bool, error) {
	c.getCalls++
	val, ok := c.store[key]
	if !ok {
		return nil, false, nil
	}
	copyVal := val
	return &copyVal, true, nil
}

func (c *fakeSpotCache) Set(_ context.Context, key string, value domain.ZoneAssessment) error {
	c.setCalls++
	c.store[key] = value
	return nil
}

func TestSearchSpots_InvalidRadius_ReturnsError(t *testing.T) {
	repo := &fakeSpatialRepo{}
	cache := newFakeSpotCache()
	svc := service.NewSearchSpotsService(repo, cache)

	coords, _ := domain.NewCoordinates(-3.7289, -38.5234)

	testCases := []struct {
		name   string
		radius int
	}{
		{"radius too small (< 100m)", 50},
		{"radius negative", -10},
		{"radius too large (> 3000m)", 3500},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Execute(context.Background(), service.SearchInput{
				Coordinates: coords,
				Radius:      tc.radius,
				QueryTime:   time.Now(),
			})
			if err != domain.ErrInvalidRadius {
				t.Errorf("expected ErrInvalidRadius for radius %d, got: %v", tc.radius, err)
			}
		})
	}
}

func TestSearchSpots_DefaultRadius_Uses1000Meters(t *testing.T) {
	repo := &fakeSpatialRepo{}
	cache := newFakeSpotCache()
	svc := service.NewSearchSpotsService(repo, cache)

	coords, _ := domain.NewCoordinates(-3.7289, -38.5234)

	res, err := svc.Execute(context.Background(), service.SearchInput{
		Coordinates: coords,
		Radius:      0, // Deve assumir 1000m por padrão
		QueryTime:   time.Date(2026, 11, 18, 14, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.RadiusMeters != domain.DefaultRadiusMeters {
		t.Errorf("expected default radius %d, got %d", domain.DefaultRadiusMeters, res.RadiusMeters)
	}
	if repo.lastRadius != domain.DefaultRadiusMeters {
		t.Errorf("expected repo to be queried with radius %d, got %d", domain.DefaultRadiusMeters, repo.lastRadius)
	}
}

func TestSearchSpots_GridSnappingCacheHit_SkipsDatabaseForSubsequentRequests(t *testing.T) {
	queryTime := time.Date(2026, 11, 18, 3, 0, 0, 0, time.UTC)
	repo := &fakeSpatialRepo{
		data: domain.SpatialData{
			Roads: []domain.RoadSegment{
				{ID: "r1", Name: "Av. Beira Mar", EstimatedSpots: 100, ParkingAllowed: true},
			},
		},
	}
	cache := newFakeSpotCache()
	svc := service.NewSearchSpotsService(repo, cache)

	// Primeira pessoa busca um ponto na Beira Mar -> Cache Miss -> Vai ao banco e salva no cache
	user1Coords, _ := domain.NewCoordinates(-3.728912, -38.523419)
	firstRes, err := svc.Execute(context.Background(), service.SearchInput{
		Coordinates: user1Coords,
		Radius:      1000,
		QueryTime:   queryTime,
	})
	if err != nil {
		t.Fatalf("unexpected error on first search: %v", err)
	}
	if firstRes.FromCache {
		t.Errorf("expected first search to be a Cache Miss (FromCache = false)")
	}
	if repo.callCount != 1 {
		t.Errorf("expected repo callCount = 1, got %d", repo.callCount)
	}

	// Outras 49 pessoas buscam um ponto vizinho (~10m de distância, mesmo Grid Snapping -3.729, -38.523)
	user2Coords, _ := domain.NewCoordinates(-3.728945, -38.523380)
	for i := 0; i < 49; i++ {
		cachedRes, err := svc.Execute(context.Background(), service.SearchInput{
			Coordinates: user2Coords,
			Radius:      1000,
			QueryTime:   queryTime,
		})
		if err != nil {
			t.Fatalf("unexpected error on cached search #%d: %v", i+1, err)
		}
		if !cachedRes.FromCache {
			t.Fatalf("expected search #%d to be served from cache (FromCache = true)", i+1)
		}
		if cachedRes.Score != firstRes.Score {
			t.Errorf("expected cached score %f, got %f", firstRes.Score, cachedRes.Score)
		}
	}

	// O banco PostGIS deve ter sido consultado APENAS 1 vez em 50 requisições!
	if repo.callCount != 1 {
		t.Errorf("expected repo to be called only once for 50 nearby requests, got %d", repo.callCount)
	}
}
