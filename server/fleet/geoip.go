package fleet

import (
	"context"
	"errors"
	"github.com/oschwald/geoip2-golang"
	"net"
)

var notCityDBError = geoip2.InvalidMethodError{}

type GeoLocation struct {
	CountryISO string    `json:"country_iso"`
	Geometry   *Geometry `json:"geometry,omitempty"`
}

type Geometry struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

type GeoIP interface {
	Lookup(ctx context.Context, ip string) *GeoLocation
}

type MaxMindGeoIP struct {
	reader *geoip2.Reader
}

type NoOpGeoIP struct{}

func (n *NoOpGeoIP) Lookup(ctx context.Context, ip string) *GeoLocation {
	return nil
}

func NewMaxMindGeoIP(path string) GeoIP {
	r, err := geoip2.Open(path)
	if err != nil {
		return &NoOpGeoIP{}
	}
	return &MaxMindGeoIP{reader: r}
}

func (m *MaxMindGeoIP) Lookup(ctx context.Context, ip string) *GeoLocation {
	// City has location data, so we'll start there first
	var err error
	parseIP := net.ParseIP(ip)
	resp, err := m.reader.City(parseIP)
	if errors.Is(err, notCityDBError) {
		resp, err := m.reader.Country(parseIP)
		if err != nil {
			return nil
		}
		// all we have is country iso, no geometry
		return &GeoLocation{CountryISO: resp.Country.IsoCode}
	}
	return parseCity(resp)
}

func parseCity(resp *geoip2.City) *GeoLocation {
	return &GeoLocation{
		CountryISO: resp.Country.IsoCode,
		Geometry: &Geometry{
			Type:        "Point",
			Coordinates: makeCoordinates(resp.Location.Latitude, resp.Location.Longitude),
		},
	}
}

func makeCoordinates(lat float64, lon float64) []float64 {
	coords := make([]float64, 2)
	coords[0] = lat
	coords[1] = lon
	return coords
}
