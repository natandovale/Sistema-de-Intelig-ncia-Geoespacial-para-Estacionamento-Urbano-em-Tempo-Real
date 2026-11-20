package postgres_test

import (
	"context"
	"strings"
	"testing"

	"github.com/spot-sense/backend/internal/domain"
	"github.com/spot-sense/backend/internal/repository/postgres"
)

func TestBuildSTDWithinQueries_ContainsSTDWithinAndGeographyCast(t *testing.T) {
	roadsSQL := postgres.RoadsWithinRadiusSQL
	parkingsSQL := postgres.ParkingsWithinRadiusSQL
	eventsSQL := postgres.EventsWithinRadiusSQL

	for name, q := range map[string]string{
		"roads":    roadsSQL,
		"parkings": parkingsSQL,
		"events":   eventsSQL,
	} {
		if !strings.Contains(q, "ST_DWithin") {
			t.Errorf("expected %s query to use ST_DWithin, got: %s", name, q)
		}
		if !strings.Contains(q, "ST_MakePoint") {
			t.Errorf("expected %s query to use ST_MakePoint, got: %s", name, q)
		}
	}
}

func TestSeedSpatialRepository_FiltersByDistanceWithinRadius(t *testing.T) {
	repo := postgres.NewSeedSpatialRepository()
	ctx := context.Background()

	// Centro na Av. Beira Mar (Fortaleza): -3.729, -38.523
	center, _ := domain.NewCoordinates(-3.729, -38.523)

	data, err := repo.FindWithinRadius(ctx, center, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data.Roads) == 0 {
		t.Errorf("expected roads within 1000m of Beira Mar")
	}
	if len(data.Parkings) == 0 {
		t.Errorf("expected private parkings within 1000m of Beira Mar")
	}

	// Ponto distante no oceano (sem vias ou estacionamentos num raio de 500m)
	farAway, _ := domain.NewCoordinates(-1.000, -35.000)
	emptyData, err := repo.FindWithinRadius(ctx, farAway, 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(emptyData.Roads) != 0 || len(emptyData.Parkings) != 0 {
		t.Errorf("expected 0 results for far away coordinates, got roads=%d, parkings=%d",
			len(emptyData.Roads), len(emptyData.Parkings))
	}
}

func TestSeedSpatialRepository_CoversAnyFortalezaNeighborhood(t *testing.T) {
	repo := postgres.NewSeedSpatialRepository()
	ctx := context.Background()

	neighborhoods := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{"Aldeota / Praça Portugal", -3.7336, -38.4979},
		{"Shopping Iguatemi Bosque", -3.7564, -38.4896},
		{"Arena Castelão", -3.8073, -38.5225},
		{"Rua Residencial Qualquer em Fortaleza", -3.7620, -38.5350},
	}

	for _, nb := range neighborhoods {
		t.Run(nb.name, func(t *testing.T) {
			center, _ := domain.NewCoordinates(nb.lat, nb.lng)
			data, err := repo.FindWithinRadius(ctx, center, 1000)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", nb.name, err)
			}
			if len(data.Roads) == 0 {
				t.Errorf("expected road segments for %s in Fortaleza, got 0", nb.name)
			}
		})
	}
}

