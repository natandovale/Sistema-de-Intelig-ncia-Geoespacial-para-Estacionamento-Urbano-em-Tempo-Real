package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/spot-sense/backend/internal/domain"
)

const (
	defaultRoadEstimatedSpots = 30
	defaultParkingCapacity    = 40
)

type overpassResponse struct {
	Elements []overpassElement `json:"elements"`
}

type overpassElement struct {
	Type string            `json:"type"`
	ID   int64             `json:"id"`
	Lat  float64           `json:"lat"`
	Lon  float64           `json:"lon"`
	Tags map[string]string `json:"tags"`
}

// ParseOverpassResponse converte o payload JSON da Overpass API (OpenStreetMap)
// em listas tipadas de RoadSegment e PrivateParking.
func ParseOverpassResponse(r io.Reader) ([]domain.RoadSegment, []domain.PrivateParking, error) {
	var payload overpassResponse
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, nil, err
	}

	var roads []domain.RoadSegment
	var parkings []domain.PrivateParking

	for _, el := range payload.Elements {
		if el.Tags == nil {
			continue
		}

		// 1. Estacionamentos Privados/Públicos dedicados (amenity=parking)
		if el.Tags["amenity"] == "parking" {
			coords, err := domain.NewCoordinates(el.Lat, el.Lon)
			if err != nil {
				continue
			}
			name := el.Tags["name"]
			if name == "" {
				name = "Estacionamento Mapeado"
			}
			capacity := defaultParkingCapacity
			if capStr, ok := el.Tags["capacity"]; ok {
				if parsed, err := strconv.Atoi(capStr); err == nil && parsed > 0 {
					capacity = parsed
				}
			}
			parkings = append(parkings, domain.PrivateParking{
				ID:       fmt.Sprintf("osm-parking-%d", el.ID),
				Name:     name,
				Location: coords,
				Capacity: capacity,
				IsOpen:   true,
			})
			continue
		}

		// 2. Segmentos Viários (highway=*)
		if _, isHighway := el.Tags["highway"]; isHighway {
			name := el.Tags["name"]
			if name == "" {
				continue
			}

			allowed := true
			if el.Tags["parking:both"] == "no" || el.Tags["parking:lane:both"] == "no" {
				allowed = false
			}

			spots := defaultRoadEstimatedSpots
			if spotsStr, ok := el.Tags["spots:estimated"]; ok {
				if parsed, err := strconv.Atoi(spotsStr); err == nil && parsed >= 0 {
					spots = parsed
				}
			}

			roads = append(roads, domain.RoadSegment{
				ID:             fmt.Sprintf("osm-way-%d", el.ID),
				Name:           name,
				EstimatedSpots: spots,
				ParkingAllowed: allowed,
			})
		}
	}

	return roads, parkings, nil
}

type regionalEventsResponse struct {
	Data []regionalEventItem `json:"data"`
}

type regionalEventItem struct {
	ID                 string  `json:"id"`
	Title              string  `json:"title"`
	Lat                float64 `json:"lat"`
	Lng                float64 `json:"lng"`
	ExpectedAttendance int     `json:"expected_attendance"`
	StartTime          string  `json:"start_time"`
	EndTime            string  `json:"end_time"`
}

// ParseRegionalEventsResponse converte o payload JSON da API de busca de eventos em domain.Event.
func ParseRegionalEventsResponse(r io.Reader) ([]domain.Event, error) {
	var payload regionalEventsResponse
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, err
	}

	var events []domain.Event
	for _, item := range payload.Data {
		if item.ID == "" || item.Title == "" {
			continue
		}
		coords, err := domain.NewCoordinates(item.Lat, item.Lng)
		if err != nil {
			continue
		}
		startTime, err := time.Parse(time.RFC3339, item.StartTime)
		if err != nil {
			continue
		}
		endTime, err := time.Parse(time.RFC3339, item.EndTime)
		if err != nil {
			continue
		}

		events = append(events, domain.Event{
			ID:                 item.ID,
			Title:              item.Title,
			Location:           coords,
			ExpectedAttendance: item.ExpectedAttendance,
			StartTime:          startTime,
			EndTime:            endTime,
		})
	}

	return events, nil
}

// IngestionWorkerService coordena a coleta e persistência idempotente de oferta e demanda espacial.
type IngestionWorkerService struct {
	osmIngestor   domain.OSMIngestor
	eventIngestor domain.EventIngestor
	store         domain.SpatialStore
}

// NewIngestionWorkerService constrói o serviço de worker de ingestão.
func NewIngestionWorkerService(
	osm domain.OSMIngestor,
	events domain.EventIngestor,
	store domain.SpatialStore,
) *IngestionWorkerService {
	return &IngestionWorkerService{
		osmIngestor:   osm,
		eventIngestor: events,
		store:         store,
	}
}

// RunSync executa um ciclo completo e idempotente de ingestão para a região informada.
func (w *IngestionWorkerService) RunSync(ctx context.Context, region string) error {
	if w.osmIngestor != nil {
		roads, parkings, err := w.osmIngestor.FetchSpatialSupply(ctx, region)
		if err != nil {
			return fmt.Errorf("failed to fetch OSM supply: %w", err)
		}
		if err := w.store.UpsertSupply(ctx, roads, parkings); err != nil {
			return fmt.Errorf("failed to upsert spatial supply: %w", err)
		}
	}

	if w.eventIngestor != nil {
		events, err := w.eventIngestor.FetchRegionalEvents(ctx, region)
		if err != nil {
			return fmt.Errorf("failed to fetch regional events: %w", err)
		}
		if err := w.store.UpsertEvents(ctx, events); err != nil {
			return fmt.Errorf("failed to upsert regional events: %w", err)
		}
	}

	return nil
}
