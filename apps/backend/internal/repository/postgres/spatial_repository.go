package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/spot-sense/backend/internal/domain"
)

const (
	// RoadsWithinRadiusSQL seleciona segmentos viários no raio usando ST_DWithin e índice GIST.
	RoadsWithinRadiusSQL = `
		SELECT id, name, estimated_spots, parking_allowed
		FROM road_segments
		WHERE ST_DWithin(
			geom::geography,
			ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
			$3
		);`

	// ParkingsWithinRadiusSQL seleciona estacionamentos privados no raio usando ST_DWithin.
	ParkingsWithinRadiusSQL = `
		SELECT id, name, ST_Y(geom) AS lat, ST_X(geom) AS lng, capacity, is_open
		FROM private_parkings
		WHERE ST_DWithin(
			geom::geography,
			ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
			$3
		);`

	// EventsWithinRadiusSQL seleciona eventos locais no raio usando ST_DWithin.
	EventsWithinRadiusSQL = `
		SELECT id, title, ST_Y(geom) AS lat, ST_X(geom) AS lng, expected_attendance, start_time, end_time
		FROM events
		WHERE ST_DWithin(
			geom::geography,
			ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
			$3
		);`
)

// PostGISSpatialRepository implementa domain.SpatialRepository consultando o PostgreSQL + PostGIS.
type PostGISSpatialRepository struct {
	db *sql.DB
}

// NewPostGISSpatialRepository cria um repositório espacial conectado ao PostGIS.
func NewPostGISSpatialRepository(db *sql.DB) *PostGISSpatialRepository {
	return &PostGISSpatialRepository{db: db}
}

func (r *PostGISSpatialRepository) FindWithinRadius(ctx context.Context, center domain.Coordinates, radiusMeters int) (domain.SpatialData, error) {
	lng := center.Longitude()
	lat := center.Latitude()

	roads, err := r.queryRoads(ctx, lng, lat, radiusMeters)
	if err != nil {
		return domain.SpatialData{}, err
	}

	parkings, err := r.queryParkings(ctx, lng, lat, radiusMeters)
	if err != nil {
		return domain.SpatialData{}, err
	}

	events, err := r.queryEvents(ctx, lng, lat, radiusMeters)
	if err != nil {
		return domain.SpatialData{}, err
	}

	return domain.SpatialData{
		Roads:    roads,
		Parkings: parkings,
		Events:   events,
	}, nil
}

func (r *PostGISSpatialRepository) queryRoads(ctx context.Context, lng, lat float64, radius int) ([]domain.RoadSegment, error) {
	rows, err := r.db.QueryContext(ctx, RoadsWithinRadiusSQL, lng, lat, radius)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.RoadSegment
	for rows.Next() {
		var seg domain.RoadSegment
		if err := rows.Scan(&seg.ID, &seg.Name, &seg.EstimatedSpots, &seg.ParkingAllowed); err != nil {
			return nil, err
		}
		result = append(result, seg)
	}
	return result, rows.Err()
}

func (r *PostGISSpatialRepository) queryParkings(ctx context.Context, lng, lat float64, radius int) ([]domain.PrivateParking, error) {
	rows, err := r.db.QueryContext(ctx, ParkingsWithinRadiusSQL, lng, lat, radius)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.PrivateParking
	for rows.Next() {
		var p domain.PrivateParking
		var pLat, pLng float64
		if err := rows.Scan(&p.ID, &p.Name, &pLat, &pLng, &p.Capacity, &p.IsOpen); err != nil {
			return nil, err
		}
		coords, err := domain.NewCoordinates(pLat, pLng)
		if err != nil {
			continue
		}
		p.Location = coords
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *PostGISSpatialRepository) queryEvents(ctx context.Context, lng, lat float64, radius int) ([]domain.Event, error) {
	rows, err := r.db.QueryContext(ctx, EventsWithinRadiusSQL, lng, lat, radius)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Event
	for rows.Next() {
		var ev domain.Event
		var eLat, eLng float64
		if err := rows.Scan(&ev.ID, &ev.Title, &eLat, &eLng, &ev.ExpectedAttendance, &ev.StartTime, &ev.EndTime); err != nil {
			return nil, err
		}
		coords, err := domain.NewCoordinates(eLat, eLng)
		if err != nil {
			continue
		}
		ev.Location = coords
		result = append(result, ev)
	}
	return result, rows.Err()
}

type seededRoad struct {
	segment domain.RoadSegment
	center  domain.Coordinates
}

// SeedSpatialRepository fornece uma base espacial cobrindo Fortaleza com filtro geodésico Haversine.
type SeedSpatialRepository struct {
	roads    []seededRoad
	parkings []domain.PrivateParking
	events   []domain.Event
}

// NewSeedSpatialRepository cria um repositório espacial populado com dados dos principais polos de Fortaleza.
func NewSeedSpatialRepository() *SeedSpatialRepository {
	mustCoord := func(lat, lng float64) domain.Coordinates {
		c, _ := domain.NewCoordinates(lat, lng)
		return c
	}

	now := time.Now()

	return &SeedSpatialRepository{
		roads: []seededRoad{
			// Polo Beira Mar / Meireles
			{
				segment: domain.RoadSegment{ID: "osm-way-101", Name: "Av. Beira Mar (Orla)", EstimatedSpots: 85, ParkingAllowed: true},
				center:  mustCoord(-3.7258, -38.4982),
			},
			{
				segment: domain.RoadSegment{ID: "osm-way-102", Name: "Rua dos Tabajaras (Meireles)", EstimatedSpots: 60, ParkingAllowed: true},
				center:  mustCoord(-3.7290, -38.5230),
			},
			{
				segment: domain.RoadSegment{ID: "osm-way-103", Name: "Av. Abolição", EstimatedSpots: 90, ParkingAllowed: true},
				center:  mustCoord(-3.7285, -38.5210),
			},
			{
				segment: domain.RoadSegment{ID: "osm-way-104", Name: "Av. Monsenhor Tabosa (Faixa Exclusiva)", EstimatedSpots: 40, ParkingAllowed: false},
				center:  mustCoord(-3.7265, -38.5180),
			},
			// Polo Aldeota / Praça Portugal / Desembargador Moreira
			{
				segment: domain.RoadSegment{ID: "osm-way-201", Name: "Av. Desembargador Moreira (Zona Azul)", EstimatedSpots: 75, ParkingAllowed: true},
				center:  mustCoord(-3.7336, -38.4979),
			},
			{
				segment: domain.RoadSegment{ID: "osm-way-202", Name: "Av. Dom Luís (Corredor Exclusivo Ônibus)", EstimatedSpots: 50, ParkingAllowed: false},
				center:  mustCoord(-3.7341, -38.4960),
			},
			{
				segment: domain.RoadSegment{ID: "osm-way-203", Name: "Rua Barbosa de Freitas", EstimatedSpots: 65, ParkingAllowed: true},
				center:  mustCoord(-3.7328, -38.4991),
			},
			// Polo Iguatemi / Edson Queiroz
			{
				segment: domain.RoadSegment{ID: "osm-way-301", Name: "Av. Washington Soares (Marginal)", EstimatedSpots: 55, ParkingAllowed: true},
				center:  mustCoord(-3.7564, -38.4896),
			},
			{
				segment: domain.RoadSegment{ID: "osm-way-302", Name: "Av. Sebastião de Abreu", EstimatedSpots: 45, ParkingAllowed: true},
				center:  mustCoord(-3.7550, -38.4910),
			},
			// Polo Dragão do Mar / Praia de Iracema
			{
				segment: domain.RoadSegment{ID: "osm-way-401", Name: "Rua Dragão do Mar", EstimatedSpots: 50, ParkingAllowed: true},
				center:  mustCoord(-3.7219, -38.5215),
			},
			{
				segment: domain.RoadSegment{ID: "osm-way-402", Name: "Rua Boris", EstimatedSpots: 35, ParkingAllowed: true},
				center:  mustCoord(-3.7212, -38.5222),
			},
			// Polo RioMar / Papicu
			{
				segment: domain.RoadSegment{ID: "osm-way-501", Name: "Rua Des. Lauro Nogueira", EstimatedSpots: 70, ParkingAllowed: true},
				center:  mustCoord(-3.7408, -38.4712),
			},
			// Polo Arena Castelão
			{
				segment: domain.RoadSegment{ID: "osm-way-601", Name: "Av. Alberto Craveiro (Bolsão Lateral)", EstimatedSpots: 95, ParkingAllowed: true},
				center:  mustCoord(-3.8073, -38.5225),
			},
			{
				segment: domain.RoadSegment{ID: "osm-way-602", Name: "Av. Paulino Rocha (Bloqueio AMC em Jogos)", EstimatedSpots: 60, ParkingAllowed: false},
				center:  mustCoord(-3.8085, -38.5210),
			},
		},
		parkings: []domain.PrivateParking{
			{
				ID:       "park-01",
				Name:     "Estacionamento Beira Mar Grill",
				Location: mustCoord(-3.7288, -38.5225),
				Capacity: 110,
				IsOpen:   true,
			},
			{
				ID:       "park-02",
				Name:     "MultiPark Meireles / Mucuripe",
				Location: mustCoord(-3.7270, -38.4995),
				Capacity: 75,
				IsOpen:   true,
			},
			{
				ID:       "park-07",
				Name:     "Estacionamento Ocean Tower",
				Location: mustCoord(-3.7242, -38.4958),
				Capacity: 65,
				IsOpen:   true,
			},
			{
				ID:       "park-08",
				Name:     "HVMM Estacionamento",
				Location: mustCoord(-3.7254, -38.4976),
				Capacity: 50,
				IsOpen:   true,
			},
			{
				ID:       "park-03",
				Name:     "Aldeota Trade Center Parking",
				Location: mustCoord(-3.7339, -38.4972),
				Capacity: 140,
				IsOpen:   true,
			},
			{
				ID:       "park-04",
				Name:     "Estacionamento Iguatemi Bosque (Deck)",
				Location: mustCoord(-3.7568, -38.4890),
				Capacity: 320,
				IsOpen:   true,
			},
			{
				ID:       "park-05",
				Name:     "Estacionamento Dragão do Mar",
				Location: mustCoord(-3.7222, -38.5210),
				Capacity: 90,
				IsOpen:   true,
			},
			{
				ID:       "park-06",
				Name:     "Estacionamento Arena Castelão Sul",
				Location: mustCoord(-3.8078, -38.5228),
				Capacity: 250,
				IsOpen:   true,
			},
		},
		events: []domain.Event{
			{
				ID:                 "sympla-901",
				Title:              "Feirinha & Show Cultural na Orla",
				Location:           mustCoord(-3.7292, -38.5232),
				ExpectedAttendance: 360,
				StartTime:          now.Add(-1 * time.Hour),
				EndTime:            now.Add(3 * time.Hour),
			},
			{
				ID:                 "sympla-902",
				Title:              "Encontro Gastronômico Praça Portugal",
				Location:           mustCoord(-3.7336, -38.4979),
				ExpectedAttendance: 240,
				StartTime:          now.Add(-30 * time.Minute),
				EndTime:            now.Add(4 * time.Hour),
			},
			{
				ID:                 "sympla-903",
				Title:              "Partida Oficial — Arena Castelão",
				Location:           mustCoord(-3.8073, -38.5225),
				ExpectedAttendance: 1200,
				StartTime:          now.Add(-30 * time.Minute),
				EndTime:            now.Add(2 * time.Hour),
			},
		},
	}
}

func isWithinFortalezaBounds(c domain.Coordinates) bool {
	return c.Latitude() >= -3.89 && c.Latitude() <= -3.68 &&
		c.Longitude() >= -38.64 && c.Longitude() <= -38.40
}

func (s *SeedSpatialRepository) FindWithinRadius(_ context.Context, center domain.Coordinates, radiusMeters int) (domain.SpatialData, error) {
	maxDist := float64(radiusMeters)

	var roads []domain.RoadSegment
	for _, r := range s.roads {
		if haversineMeters(center, r.center) <= maxDist {
			roads = append(roads, r.segment)
		}
	}

	// Para estacionamentos privados, considera um raio mínimo de descoberta de 400m
	// (distância de caminhada rápida de 2-3 quadras do destino), mesmo quando o raio de vias é 150m.
	parkingMaxDist := math.Max(maxDist, 400.0)
	var parkings []domain.PrivateParking
	for _, p := range s.parkings {
		if haversineMeters(center, p.Location) <= parkingMaxDist {
			parkings = append(parkings, p)
		}
	}

	var events []domain.Event
	for _, e := range s.events {
		if haversineMeters(center, e.Location) <= maxDist {
			events = append(events, e)
		}
	}

	// Se o endereço pesquisado está dentro de Fortaleza, mas fora dos polos pré-cadastrados,
	// sintetiza apenas a oferta viária local da quadra pesquisada; estacionamentos só são
	// retornados se realmente existirem dentro do raio selecionado.
	if len(roads) == 0 && isWithinFortalezaBounds(center) {
		gridID := center.GridKey()
		roads = []domain.RoadSegment{
			{
				ID:             fmt.Sprintf("osm-local-main-%s", gridID),
				Name:           "Via Principal do Setor (Estacionamento Lateral)",
				EstimatedSpots: 65,
				ParkingAllowed: true,
			},
			{
				ID:             fmt.Sprintf("osm-local-sec-%s", gridID),
				Name:           "Vias Secundárias / Travessas no Raio de 1km",
				EstimatedSpots: 80,
				ParkingAllowed: true,
			},
		}
	}

	return domain.SpatialData{
		Roads:    roads,
		Parkings: parkings,
		Events:   events,
	}, nil
}

func haversineMeters(a, b domain.Coordinates) float64 {
	const earthRadiusMeters = 6371000.0
	lat1 := a.Latitude() * math.Pi / 180.0
	lat2 := b.Latitude() * math.Pi / 180.0
	dLat := (b.Latitude() - a.Latitude()) * math.Pi / 180.0
	dLng := (b.Longitude() - a.Longitude()) * math.Pi / 180.0

	sinLat := math.Sin(dLat / 2)
	sinLng := math.Sin(dLng / 2)
	h := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLng*sinLng
	return 2 * earthRadiusMeters * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}
