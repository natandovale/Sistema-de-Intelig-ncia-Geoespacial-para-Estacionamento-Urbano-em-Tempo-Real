package domain_test

import (
	"testing"

	"github.com/spot-sense/backend/internal/domain"
)

func TestNewCoordinates_Valid(t *testing.T) {
	lat := -3.728912
	lng := -38.523419

	coords, err := domain.NewCoordinates(lat, lng)
	if err != nil {
		t.Fatalf("expected no error for valid coordinates, got: %v", err)
	}

	if coords.Latitude() != lat {
		t.Errorf("expected latitude %f, got %f", lat, coords.Latitude())
	}
	if coords.Longitude() != lng {
		t.Errorf("expected longitude %f, got %f", lng, coords.Longitude())
	}
}

func TestNewCoordinates_InvalidLatitude(t *testing.T) {
	testCases := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{"latitude above 90", 90.0001, -38.52},
		{"latitude below -90", -90.0001, -38.52},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewCoordinates(tc.lat, tc.lng)
			if err == nil {
				t.Fatalf("expected error for invalid latitude %f, got nil", tc.lat)
			}
			if err != domain.ErrInvalidLatitude {
				t.Errorf("expected ErrInvalidLatitude, got: %v", err)
			}
		})
	}
}

func TestNewCoordinates_InvalidLongitude(t *testing.T) {
	testCases := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{"longitude above 180", -3.72, 180.0001},
		{"longitude below -180", -3.72, -180.0001},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewCoordinates(tc.lat, tc.lng)
			if err == nil {
				t.Fatalf("expected error for invalid longitude %f, got nil", tc.lng)
			}
			if err != domain.ErrInvalidLongitude {
				t.Errorf("expected ErrInvalidLongitude, got: %v", err)
			}
		})
	}
}

func TestCoordinates_SnapToGrid(t *testing.T) {
	// Dois pontos muito próximos (~10 metros de distância) na Beira Mar
	pontoA, err := domain.NewCoordinates(-3.728912, -38.523419)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pontoB, err := domain.NewCoordinates(-3.728945, -38.523380)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snapA := pontoA.SnapToGrid()
	snapB := pontoB.SnapToGrid()

	expectedLat := -3.729
	expectedLng := -38.523

	if snapA.Latitude() != expectedLat {
		t.Errorf("expected snap latitude %f, got %f", expectedLat, snapA.Latitude())
	}
	if snapA.Longitude() != expectedLng {
		t.Errorf("expected snap longitude %f, got %f", expectedLng, snapA.Longitude())
	}

	// Ambos os pontos devem coincidir no mesmo grid (~100m)
	if snapA != snapB {
		t.Errorf("expected pontoA and pontoB to snap to same coordinates. snapA: %+v, snapB: %+v", snapA, snapB)
	}

	// Teste de normalização em torno do zero
	pontoZero, err := domain.NewCoordinates(0.0001, -0.0001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	snapZero := pontoZero.SnapToGrid()
	if snapZero.Latitude() != 0 || snapZero.Longitude() != 0 {
		t.Errorf("expected (0, 0), got (%f, %f)", snapZero.Latitude(), snapZero.Longitude())
	}
	if pontoZero.GridKey() != "grid:0.000:0.000" {
		t.Errorf("expected grid key 'grid:0.000:0.000', got %q", pontoZero.GridKey())
	}
}

func TestCoordinates_GridKey(t *testing.T) {
	coords, err := domain.NewCoordinates(-3.728912, -38.523419)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedKey := "grid:-3.729:-38.523"
	actualKey := coords.GridKey()

	if actualKey != expectedKey {
		t.Errorf("expected grid key %q, got %q", expectedKey, actualKey)
	}
}
