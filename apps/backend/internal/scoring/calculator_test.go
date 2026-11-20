package scoring_test

import (
	"testing"
	"time"

	"github.com/spot-sense/backend/internal/domain"
	"github.com/spot-sense/backend/internal/scoring"
)

func TestCalculateScore_NoSupply_ReturnsZero(t *testing.T) {
	queryTime := time.Date(2026, 11, 18, 14, 0, 0, 0, time.UTC)

	result := scoring.Calculate(scoring.Input{
		Roads:     nil,
		Parkings:  nil,
		Events:    nil,
		QueryTime: queryTime,
	})

	if result.Score != 0.0 {
		t.Errorf("expected score 0.0 when no supply exists, got %f", result.Score)
	}
	if result.Probability != domain.ProbabilityLow {
		t.Errorf("expected probability %s, got %s", domain.ProbabilityLow, result.Probability)
	}
}

func TestCalculateScore_NormalSupplyNoEvents_OffPeak_ReturnsHigh(t *testing.T) {
	// 03:00 da manhã (madrugada = ocupação base de 15%)
	queryTime := time.Date(2026, 11, 18, 3, 0, 0, 0, time.UTC)

	roads := []domain.RoadSegment{
		{ID: "r1", Name: "Rua A", EstimatedSpots: 60, ParkingAllowed: true},
	}
	parkings := []domain.PrivateParking{
		{ID: "p1", Name: "Estacionamento Central", Capacity: 40, IsOpen: true},
	}

	result := scoring.Calculate(scoring.Input{
		Roads:     roads,
		Parkings:  parkings,
		Events:    nil,
		QueryTime: queryTime,
	})

	// Oferta = 100, Demanda Base (15%) = 15 -> Score esperado = 0.85 (HIGH)
	if result.Score < 0.70 {
		t.Errorf("expected score >= 0.70, got %f", result.Score)
	}
	if result.Probability != domain.ProbabilityHigh {
		t.Errorf("expected probability %s, got %s", domain.ProbabilityHigh, result.Probability)
	}
	if result.TotalSupply != 100 {
		t.Errorf("expected total supply 100, got %d", result.TotalSupply)
	}
}

func TestCalculateScore_PeakHourNoEvents_ReturnsMedium(t *testing.T) {
	// 14:00 (horário comercial = ocupação base de 50%)
	queryTime := time.Date(2026, 11, 18, 14, 0, 0, 0, time.UTC)

	roads := []domain.RoadSegment{
		{ID: "r1", Name: "Av. Comercial", EstimatedSpots: 100, ParkingAllowed: true},
	}

	result := scoring.Calculate(scoring.Input{
		Roads:     roads,
		Parkings:  nil,
		Events:    nil,
		QueryTime: queryTime,
	})

	if result.Score != 0.50 {
		t.Errorf("expected score 0.50 during business hours without events, got %f", result.Score)
	}
	if result.Probability != domain.ProbabilityMedium {
		t.Errorf("expected probability %s, got %s", domain.ProbabilityMedium, result.Probability)
	}
}

func TestCalculateScore_LargeActiveEvent_DropsScoreToLow(t *testing.T) {
	queryTime := time.Date(2026, 11, 18, 20, 0, 0, 0, time.UTC)

	roads := []domain.RoadSegment{
		{ID: "r1", Name: "Av. Beira Mar", EstimatedSpots: 100, ParkingAllowed: true},
	}
	events := []domain.Event{
		{
			ID:                 "e1",
			Title:              "Show na Orla",
			ExpectedAttendance: 900, // 900 / 3 = 300 veículos de demanda extra
			StartTime:          queryTime.Add(-30 * time.Minute),
			EndTime:            queryTime.Add(2 * time.Hour),
		},
	}

	result := scoring.Calculate(scoring.Input{
		Roads:     roads,
		Parkings:  nil,
		Events:    events,
		QueryTime: queryTime,
	})

	if result.Score != 0.0 {
		t.Errorf("expected score 0.0 due to large active event, got %f", result.Score)
	}
	if result.Probability != domain.ProbabilityLow {
		t.Errorf("expected probability %s, got %s", domain.ProbabilityLow, result.Probability)
	}
}

func TestCalculateScore_InactiveEvent_Ignored(t *testing.T) {
	queryTime := time.Date(2026, 11, 18, 3, 0, 0, 0, time.UTC)

	roads := []domain.RoadSegment{
		{ID: "r1", Name: "Rua A", EstimatedSpots: 100, ParkingAllowed: true},
	}
	events := []domain.Event{
		{
			ID:                 "e1",
			Title:              "Evento de Ontem",
			ExpectedAttendance: 1500,
			StartTime:          queryTime.Add(-24 * time.Hour),
			EndTime:            queryTime.Add(-20 * time.Hour),
		},
	}

	result := scoring.Calculate(scoring.Input{
		Roads:     roads,
		Parkings:  nil,
		Events:    events,
		QueryTime: queryTime,
	})

	// Evento inativo deve ser ignorado, mantendo score alto na madrugada (0.85)
	if result.Probability != domain.ProbabilityHigh {
		t.Errorf("expected inactive event to be ignored (HIGH probability), got %s (score: %f)", result.Probability, result.Score)
	}
}

func TestCalculateScore_ClosedParkingAndForbiddenRoad_Ignored(t *testing.T) {
	queryTime := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)

	roads := []domain.RoadSegment{
		{ID: "r1", Name: "Via Proibida", EstimatedSpots: 50, ParkingAllowed: false},
	}
	parkings := []domain.PrivateParking{
		{ID: "p1", Name: "Estacionamento Fechado", Capacity: 80, IsOpen: false},
	}

	result := scoring.Calculate(scoring.Input{
		Roads:     roads,
		Parkings:  parkings,
		Events:    nil,
		QueryTime: queryTime,
	})

	if result.TotalSupply != 0 {
		t.Errorf("expected total supply 0 when road is forbidden and parking is closed, got %d", result.TotalSupply)
	}
	if result.Score != 0.0 {
		t.Errorf("expected score 0.0, got %f", result.Score)
	}
}
