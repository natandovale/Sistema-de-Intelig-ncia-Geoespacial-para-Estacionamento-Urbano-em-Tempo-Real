package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/spot-sense/backend/internal/domain"
	"github.com/spot-sense/backend/internal/service"
)

type fakeSpatialStore struct {
	roads    map[string]domain.RoadSegment
	parkings map[string]domain.PrivateParking
	events   map[string]domain.Event
}

func newFakeSpatialStore() *fakeSpatialStore {
	return &fakeSpatialStore{
		roads:    make(map[string]domain.RoadSegment),
		parkings: make(map[string]domain.PrivateParking),
		events:   make(map[string]domain.Event),
	}
}

func (s *fakeSpatialStore) UpsertSupply(_ context.Context, roads []domain.RoadSegment, parkings []domain.PrivateParking) error {
	for _, r := range roads {
		s.roads[r.ID] = r
	}
	for _, p := range parkings {
		s.parkings[p.ID] = p
	}
	return nil
}

func (s *fakeSpatialStore) UpsertEvents(_ context.Context, events []domain.Event) error {
	for _, e := range events {
		s.events[e.ID] = e
	}
	return nil
}

func TestParseOverpassJSON_ExtractsRoadsAndPrivateParkings(t *testing.T) {
	rawOverpassJSON := `{
		"elements": [
			{
				"type": "way",
				"id": 1001,
				"tags": {
					"highway": "secondary",
					"name": "Avenida Beira Mar",
					"spots:estimated": "80"
				}
			},
			{
				"type": "way",
				"id": 1002,
				"tags": {
					"highway": "primary",
					"name": "Avenida Dom Luís (Corredor Exclusivo)",
					"parking:both": "no"
				}
			},
			{
				"type": "node",
				"id": 2001,
				"lat": -3.7288,
				"lon": -38.5225,
				"tags": {
					"amenity": "parking",
					"name": "Estacionamento Orla",
					"capacity": "110"
				}
			}
		]
	}`

	roads, parkings, err := service.ParseOverpassResponse(strings.NewReader(rawOverpassJSON))
	if err != nil {
		t.Fatalf("unexpected error parsing Overpass JSON: %v", err)
	}

	if len(roads) != 2 {
		t.Fatalf("expected 2 roads, got %d", len(roads))
	}
	if !roads[0].ParkingAllowed || roads[0].EstimatedSpots != 80 {
		t.Errorf("expected first road to allow parking with 80 spots, got %+v", roads[0])
	}
	if roads[1].ParkingAllowed {
		t.Errorf("expected second road with parking:both=no to disallow parking")
	}

	if len(parkings) != 1 {
		t.Fatalf("expected 1 private parking, got %d", len(parkings))
	}
	if parkings[0].Name != "Estacionamento Orla" || parkings[0].Capacity != 110 {
		t.Errorf("unexpected parking data: %+v", parkings[0])
	}
}

func TestParseRegionalEventsJSON_ExtractsValidEvents(t *testing.T) {
	rawEventsJSON := `{
		"data": [
			{
				"id": "sympla-501",
				"title": "Festival de Jazz na Praia",
				"lat": -3.7291,
				"lng": -38.5231,
				"expected_attendance": 600,
				"start_time": "2026-11-18T18:00:00Z",
				"end_time": "2026-11-18T23:00:00Z"
			}
		]
	}`

	events, err := service.ParseRegionalEventsResponse(strings.NewReader(rawEventsJSON))
	if err != nil {
		t.Fatalf("unexpected error parsing events JSON: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != "sympla-501" || events[0].ExpectedAttendance != 600 {
		t.Errorf("unexpected event parsed: %+v", events[0])
	}
}

type stubOSMIngestor struct {
	roads    []domain.RoadSegment
	parkings []domain.PrivateParking
}

func (s *stubOSMIngestor) FetchSpatialSupply(_ context.Context, _ string) ([]domain.RoadSegment, []domain.PrivateParking, error) {
	return s.roads, s.parkings, nil
}

type stubEventIngestor struct {
	events []domain.Event
}

func (s *stubEventIngestor) FetchRegionalEvents(_ context.Context, _ string) ([]domain.Event, error) {
	return s.events, nil
}

func TestIngestionWorkerService_RunSync_IsIdempotent(t *testing.T) {
	coord, _ := domain.NewCoordinates(-3.729, -38.523)
	osm := &stubOSMIngestor{
		roads: []domain.RoadSegment{
			{ID: "osm-1", Name: "Rua A", EstimatedSpots: 40, ParkingAllowed: true},
		},
		parkings: []domain.PrivateParking{
			{ID: "park-1", Name: "Parking A", Location: coord, Capacity: 50, IsOpen: true},
		},
	}
	scraper := &stubEventIngestor{
		events: []domain.Event{
			{ID: "ev-1", Title: "Show Local", Location: coord, ExpectedAttendance: 300},
		},
	}
	store := newFakeSpatialStore()

	worker := service.NewIngestionWorkerService(osm, scraper, store)

	// Executa duas vezes seguidas para comprovar idempotência (Upsert sem duplicar)
	if err := worker.RunSync(context.Background(), "Fortaleza"); err != nil {
		t.Fatalf("first RunSync failed: %v", err)
	}
	if err := worker.RunSync(context.Background(), "Fortaleza"); err != nil {
		t.Fatalf("second RunSync failed: %v", err)
	}

	if len(store.roads) != 1 || len(store.parkings) != 1 || len(store.events) != 1 {
		t.Errorf("expected idempotent store counts (1, 1, 1), got (%d, %d, %d)",
			len(store.roads), len(store.parkings), len(store.events))
	}
}
