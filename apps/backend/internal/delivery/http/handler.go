package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/spot-sense/backend/internal/domain"
	"github.com/spot-sense/backend/internal/service"
)

// CoordinatesDTO representa uma coordenada na camada de transporte JSON.
type CoordinatesDTO struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// RoadSegmentDTO representa um segmento de via pública no JSON da API.
type RoadSegmentDTO struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	EstimatedSpots int    `json:"estimated_spots"`
	ParkingAllowed bool   `json:"parking_allowed"`
}

// PrivateParkingDTO representa um estacionamento comercial no JSON da API.
type PrivateParkingDTO struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Location CoordinatesDTO `json:"location"`
	Capacity int            `json:"capacity"`
	IsOpen   bool           `json:"is_open"`
}

// EventDTO representa um evento ativo ou relevante no JSON da API.
type EventDTO struct {
	ID                 string         `json:"id"`
	Title              string         `json:"title"`
	Location           CoordinatesDTO `json:"location"`
	ExpectedAttendance int            `json:"expected_attendance"`
	StartTime          time.Time      `json:"start_time"`
	EndTime            time.Time      `json:"end_time"`
}

// SpotsResponse define o contrato público do endpoint GET /api/vagas.
type SpotsResponse struct {
	GridCoordinates CoordinatesDTO      `json:"grid_coordinates"`
	RadiusMeters    int                 `json:"radius_meters"`
	Score           float64             `json:"score"`
	Probability     string              `json:"probability"`
	TotalSupply     int                 `json:"total_supply"`
	EstimatedDemand int                 `json:"estimated_demand"`
	ActiveEvents    int                 `json:"active_events"`
	Roads           []RoadSegmentDTO    `json:"roads"`
	Parkings        []PrivateParkingDTO `json:"parkings"`
	Events          []EventDTO          `json:"events"`
	FromCache       bool                `json:"from_cache"`
}

// ErrorDetail representa os detalhes padronizados de erro da API.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse segue o padrão de erro definido no AGENTS.md.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// Handler gerencia os endpoints HTTP de busca de vagas.
type Handler struct {
	searchService *service.SearchSpotsService
}

// NewRouter configura e retorna o http.Handler com as rotas da API.
func NewRouter(searchService *service.SearchSpotsService) http.Handler {
	h := &Handler{searchService: searchService}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/vagas", h.handleGetSpots)
	mux.HandleFunc("GET /healthz", h.handleHealthz)

	return withCORS(mux)
}

func (h *Handler) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleGetSpots(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	latStr := query.Get("lat")
	lngStr := query.Get("lng")
	if latStr == "" || lngStr == "" {
		writeError(w, http.StatusBadRequest, "INVALID_COORDINATES", "Query parameters 'lat' and 'lng' are required.")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_COORDINATES", "Parameter 'lat' must be a valid floating-point number.")
		return
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_COORDINATES", "Parameter 'lng' must be a valid floating-point number.")
		return
	}

	coords, err := domain.NewCoordinates(lat, lng)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_COORDINATES", err.Error())
		return
	}

	radius := 0
	if radiusStr := query.Get("radius"); radiusStr != "" {
		parsedRadius, err := strconv.Atoi(radiusStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_RADIUS", "Parameter 'radius' must be a valid integer.")
			return
		}
		radius = parsedRadius
	}

	assessment, err := h.searchService.Execute(r.Context(), service.SearchInput{
		Coordinates: coords,
		Radius:      radius,
		QueryTime:   time.Now(),
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidRadius) {
			writeError(w, http.StatusBadRequest, "INVALID_RADIUS", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to evaluate parking spots for the requested area.")
		return
	}

	writeJSON(w, http.StatusOK, toSpotsResponse(assessment))
}

func toSpotsResponse(a domain.ZoneAssessment) SpotsResponse {
	roads := make([]RoadSegmentDTO, 0, len(a.Roads))
	for _, r := range a.Roads {
		roads = append(roads, RoadSegmentDTO{
			ID:             r.ID,
			Name:           r.Name,
			EstimatedSpots: r.EstimatedSpots,
			ParkingAllowed: r.ParkingAllowed,
		})
	}

	parkings := make([]PrivateParkingDTO, 0, len(a.Parkings))
	for _, p := range a.Parkings {
		parkings = append(parkings, PrivateParkingDTO{
			ID:   p.ID,
			Name: p.Name,
			Location: CoordinatesDTO{
				Lat: p.Location.Latitude(),
				Lng: p.Location.Longitude(),
			},
			Capacity: p.Capacity,
			IsOpen:   p.IsOpen,
		})
	}

	events := make([]EventDTO, 0, len(a.Events))
	for _, e := range a.Events {
		events = append(events, EventDTO{
			ID:    e.ID,
			Title: e.Title,
			Location: CoordinatesDTO{
				Lat: e.Location.Latitude(),
				Lng: e.Location.Longitude(),
			},
			ExpectedAttendance: e.ExpectedAttendance,
			StartTime:          e.StartTime,
			EndTime:            e.EndTime,
		})
	}

	return SpotsResponse{
		GridCoordinates: CoordinatesDTO{
			Lat: a.GridCoordinates.Latitude(),
			Lng: a.GridCoordinates.Longitude(),
		},
		RadiusMeters:    a.RadiusMeters,
		Score:           a.Score,
		Probability:     string(a.Probability),
		TotalSupply:     a.TotalSupply,
		EstimatedDemand: a.EstimatedDemand,
		ActiveEvents:    a.ActiveEvents,
		Roads:           roads,
		Parkings:        parkings,
		Events:          events,
		FromCache:       a.FromCache,
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
