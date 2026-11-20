package scoring

import (
	"math"
	"time"

	"github.com/spot-sense/backend/internal/domain"
)

const (
	attendeesPerVehicle = 3.0
	highThreshold       = 0.70
	mediumThreshold     = 0.35
)

// Input agrupa os dados espaciais e temporais para o cálculo heurístico.
type Input struct {
	Roads     []domain.RoadSegment
	Parkings  []domain.PrivateParking
	Events    []domain.Event
	QueryTime time.Time
}

// Result contém o resultado consolidado do cálculo de probabilidade de vagas.
type Result struct {
	Score           float64
	Probability     domain.ProbabilityLevel
	TotalSupply     int
	EstimatedDemand int
	ActiveEvents    int
}

// Calculate executa a heurística pura de oferta vs. demanda para estimar a facilidade de estacionar.
func Calculate(input Input) Result {
	supply := calculateSupply(input.Roads, input.Parkings)
	if supply <= 0 {
		return Result{
			Score:       0.0,
			Probability: domain.ProbabilityLow,
			TotalSupply: 0,
		}
	}

	baseDemand := float64(supply) * baseOccupancyRate(input.QueryTime)
	eventDemand, activeEvents := calculateEventDemand(input.Events, input.QueryTime)
	totalDemand := baseDemand + eventDemand

	rawScore := 1.0 - (totalDemand / float64(supply))
	clampedScore := clamp(rawScore, 0.0, 1.0)
	roundedScore := math.Round(clampedScore*100) / 100

	return Result{
		Score:           roundedScore,
		Probability:     classifyProbability(roundedScore),
		TotalSupply:     supply,
		EstimatedDemand: int(math.Round(totalDemand)),
		ActiveEvents:    activeEvents,
	}
}

func calculateSupply(roads []domain.RoadSegment, parkings []domain.PrivateParking) int {
	total := 0
	for _, r := range roads {
		if r.ParkingAllowed && r.EstimatedSpots > 0 {
			total += r.EstimatedSpots
		}
	}
	for _, p := range parkings {
		if p.IsOpen && p.Capacity > 0 {
			total += p.Capacity
		}
	}
	return total
}

func baseOccupancyRate(t time.Time) float64 {
	hour := t.Hour()
	switch {
	case hour >= 0 && hour < 6:
		return 0.15
	case hour >= 8 && hour <= 20:
		return 0.50
	default:
		return 0.30
	}
}

func calculateEventDemand(events []domain.Event, queryTime time.Time) (float64, int) {
	demand := 0.0
	activeCount := 0
	for _, e := range events {
		if e.IsActiveAt(queryTime) && e.ExpectedAttendance > 0 {
			demand += float64(e.ExpectedAttendance) / attendeesPerVehicle
			activeCount++
		}
	}
	return demand, activeCount
}

func classifyProbability(score float64) domain.ProbabilityLevel {
	switch {
	case score >= highThreshold:
		return domain.ProbabilityHigh
	case score >= mediumThreshold:
		return domain.ProbabilityMedium
	default:
		return domain.ProbabilityLow
	}
}

func clamp(val, minVal, maxVal float64) float64 {
	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}
