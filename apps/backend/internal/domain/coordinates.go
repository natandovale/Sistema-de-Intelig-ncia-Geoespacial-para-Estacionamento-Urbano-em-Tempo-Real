package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

var (
	// ErrInvalidLatitude é retornado quando a latitude não está entre -90 e 90 graus.
	ErrInvalidLatitude = errors.New("latitude must be between -90 and 90 degrees")

	// ErrInvalidLongitude é retornado quando a longitude não está entre -180 e 180 graus.
	ErrInvalidLongitude = errors.New("longitude must be between -180 and 180 degrees")
)

// Coordinates representa um ponto geográfico imutável com latitude e longitude validadas.
type Coordinates struct {
	latitude  float64
	longitude float64
}

// NewCoordinates cria e valida uma nova coordenada geográfica.
func NewCoordinates(lat, lng float64) (Coordinates, error) {
	if lat < -90 || lat > 90 {
		return Coordinates{}, ErrInvalidLatitude
	}
	if lng < -180 || lng > 180 {
		return Coordinates{}, ErrInvalidLongitude
	}

	return Coordinates{
		latitude:  lat,
		longitude: lng,
	}, nil
}

// Latitude retorna o valor da latitude.
func (c Coordinates) Latitude() float64 {
	return c.latitude
}

// Longitude retorna o valor da longitude.
func (c Coordinates) Longitude() float64 {
	return c.longitude
}

// SnapToGrid normaliza a coordenada para uma precisão de 3 casas decimais (~100 metros).
// Isso agrupa pontos vizinhos em uma mesma célula de grid para otimização de cache.
func (c Coordinates) SnapToGrid() Coordinates {
	snappedLat := math.Round(c.latitude*1000) / 1000
	snappedLng := math.Round(c.longitude*1000) / 1000

	// Normaliza -0.0 para 0.0 para evitar chaves de cache divergentes
	if snappedLat == 0 {
		snappedLat = 0
	}
	if snappedLng == 0 {
		snappedLng = 0
	}

	return Coordinates{
		latitude:  snappedLat,
		longitude: snappedLng,
	}
}

// GridKey retorna a chave formatada para representação do grid de coordenadas no Redis.
func (c Coordinates) GridKey() string {
	snapped := c.SnapToGrid()
	return fmt.Sprintf("grid:%.3f:%.3f", snapped.latitude, snapped.longitude)
}

type coordinatesJSON struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// MarshalJSON implementa json.Marshaler preservando o encapsulamento dos campos privados.
func (c Coordinates) MarshalJSON() ([]byte, error) {
	return json.Marshal(coordinatesJSON{
		Lat: c.latitude,
		Lng: c.longitude,
	})
}

// UnmarshalJSON implementa json.Unmarshaler validando os limites geográficos na desserialização.
func (c *Coordinates) UnmarshalJSON(data []byte) error {
	var dto coordinatesJSON
	if err := json.Unmarshal(data, &dto); err != nil {
		return err
	}
	parsed, err := NewCoordinates(dto.Lat, dto.Lng)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

