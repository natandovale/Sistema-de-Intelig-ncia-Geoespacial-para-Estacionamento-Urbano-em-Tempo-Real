package domain

import "time"

// ProbabilityLevel define a classificação semântica da facilidade de estacionamento.
type ProbabilityLevel string

const (
	ProbabilityHigh   ProbabilityLevel = "HIGH"
	ProbabilityMedium ProbabilityLevel = "MEDIUM"
	ProbabilityLow    ProbabilityLevel = "LOW"
)

// RoadSegment representa um trecho de via pública com estimativa de vagas.
type RoadSegment struct {
	ID             string
	Name           string
	EstimatedSpots int
	ParkingAllowed bool
}

// PrivateParking representa um estacionamento comercial privado.
type PrivateParking struct {
	ID       string
	Name     string
	Location Coordinates
	Capacity int
	IsOpen   bool
}

// Event representa um evento local que gera demanda adicional por vagas na região.
type Event struct {
	ID                 string
	Title              string
	Location           Coordinates
	ExpectedAttendance int
	StartTime          time.Time
	EndTime            time.Time
}

// IsActiveAt verifica se o evento impacta a demanda no horário consultado
// (considera eventos já em andamento ou que iniciarão nos próximos 60 minutos).
func (e Event) IsActiveAt(queryTime time.Time) bool {
	windowStart := e.StartTime.Add(-1 * time.Hour)
	return !queryTime.Before(windowStart) && !queryTime.After(e.EndTime)
}
