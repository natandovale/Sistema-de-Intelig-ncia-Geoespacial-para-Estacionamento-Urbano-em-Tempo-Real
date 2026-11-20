package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	deliveryhttp "github.com/spot-sense/backend/internal/delivery/http"
	"github.com/spot-sense/backend/internal/domain"
	"github.com/spot-sense/backend/internal/service"
)

type stubSpatialRepo struct {
	data domain.SpatialData
	err  error
}

func (s *stubSpatialRepo) FindWithinRadius(_ context.Context, _ domain.Coordinates, _ int) (domain.SpatialData, error) {
	return s.data, s.err
}

func TestGetSpots_Success_Returns200AndJSON(t *testing.T) {
	repo := &stubSpatialRepo{
		data: domain.SpatialData{
			Roads: []domain.RoadSegment{
				{ID: "r1", Name: "Av. Beira Mar", EstimatedSpots: 120, ParkingAllowed: true},
			},
			Parkings: []domain.PrivateParking{
				{ID: "p1", Name: "Beira Mar Parking", Capacity: 80, IsOpen: true},
			},
		},
	}
	svc := service.NewSearchSpotsService(repo, nil)
	router := deliveryhttp.NewRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/vagas?lat=-3.728912&lng=-38.523419&radius=1000", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp deliveryhttp.SpotsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response JSON: %v", err)
	}

	if resp.GridCoordinates.Lat != -3.729 || resp.GridCoordinates.Lng != -38.523 {
		t.Errorf("expected snapped grid coordinates (-3.729, -38.523), got (%f, %f)",
			resp.GridCoordinates.Lat, resp.GridCoordinates.Lng)
	}
	if resp.RadiusMeters != 1000 {
		t.Errorf("expected radius 1000, got %d", resp.RadiusMeters)
	}
	if resp.TotalSupply != 200 {
		t.Errorf("expected total supply 200, got %d", resp.TotalSupply)
	}
	if resp.Probability == "" {
		t.Errorf("expected non-empty probability level")
	}
}

func TestGetSpots_DefaultRadius_WhenOmitted(t *testing.T) {
	repo := &stubSpatialRepo{}
	svc := service.NewSearchSpotsService(repo, nil)
	router := deliveryhttp.NewRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/vagas?lat=-3.7289&lng=-38.5234", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK when radius is omitted, got %d", rec.Code)
	}

	var resp deliveryhttp.SpotsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response JSON: %v", err)
	}

	if resp.RadiusMeters != domain.DefaultRadiusMeters {
		t.Errorf("expected default radius %d, got %d", domain.DefaultRadiusMeters, resp.RadiusMeters)
	}
}

func TestGetSpots_InvalidParameters_Returns400WithStandardError(t *testing.T) {
	repo := &stubSpatialRepo{}
	svc := service.NewSearchSpotsService(repo, nil)
	router := deliveryhttp.NewRouter(svc)

	testCases := []struct {
		name         string
		query        string
		expectedCode string
	}{
		{"missing lat", "?lng=-38.52", "INVALID_COORDINATES"},
		{"non-numeric lat", "?lat=abc&lng=-38.52", "INVALID_COORDINATES"},
		{"out of range lat", "?lat=95.0&lng=-38.52", "INVALID_COORDINATES"},
		{"out of range lng", "?lat=-3.72&lng=-190.0", "INVALID_COORDINATES"},
		{"invalid radius format", "?lat=-3.72&lng=-38.52&radius=xyz", "INVALID_RADIUS"},
		{"radius out of bounds", "?lat=-3.72&lng=-38.52&radius=50", "INVALID_RADIUS"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/vagas"+tc.query, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400 Bad Request, got %d", rec.Code)
			}

			var errResp deliveryhttp.ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("failed to decode error JSON: %v", err)
			}

			if errResp.Error.Code != tc.expectedCode {
				t.Errorf("expected error code %q, got %q", tc.expectedCode, errResp.Error.Code)
			}
			if errResp.Error.Message == "" {
				t.Errorf("expected non-empty error message")
			}
		})
	}
}

func TestHealthz_Returns200(t *testing.T) {
	svc := service.NewSearchSpotsService(&stubSpatialRepo{}, nil)
	router := deliveryhttp.NewRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK on /healthz, got %d", rec.Code)
	}
}
