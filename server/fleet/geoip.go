package fleet

import (
	"context"
	"errors"
	"log/slog"
	"net"

	"github.com/oschwald/geoip2-golang"
)

var notCityDBError = geoip2.InvalidMethodError{}

type GeoLocation struct {
	CountryISO string    `json:"country_iso" csv:"-"`
	CityName   string    `json:"city_name" csv:"-"`
	Geometry   *Geometry `json:"geometry,omitempty" csv:"-"`
}

type Geometry struct {
	Type        string    `json:"type" csv:"-"`
	Coordinates []float64 `json:"coordinates" csv:"-"`
}

type GeoIP interface {
	Lookup(ctx context.Context, ip string) *GeoLocation
}

type MaxMindGeoIP struct {
	reader *geoip2.Reader
	l      *slog.Logger
}

type NoOpGeoIP struct{}

func (n *NoOpGeoIP) Lookup(ctx context.Context, ip string) *GeoLocation {
	return nil
}

func NewMaxMindGeoIP(logger *slog.Logger, path string) (*MaxMindGeoIP, error) {
	r, err := geoip2.Open(path)
	if err != nil {
		return nil, err
	}
	return &MaxMindGeoIP{reader: r, l: logger}, nil
}

func (m *MaxMindGeoIP) Lookup(ctx context.Context, ip string) *GeoLocation {
	if ip == "" {
		return nil
	}
	// City has location data, so we'll start there first
	parseIP := net.ParseIP(ip)
	if parseIP == nil {
		return nil
	}
	resp, err := m.reader.City(parseIP)
	if err != nil && errors.Is(err, notCityDBError) {
		resp, err := m.reader.Country(parseIP)
		if err != nil {
			m.l.DebugContext(ctx, "failed to lookup location from mmdb file", "err", err)
			return nil
		}
		if resp == nil {
			return nil
		}
		// all we have is country iso, no geometry
		return &GeoLocation{CountryISO: resp.Country.IsoCode}
	}
	if err != nil {
		m.l.DebugContext(ctx, "failed to lookup location from mmdb file", "err", err)
		return nil
	}
	return parseCity(resp)
}

func parseCity(resp *geoip2.City) *GeoLocation {
	if resp == nil {
		return nil
	}
	return &GeoLocation{
		CountryISO: resp.Country.IsoCode,
		CityName:   resp.City.Names["en"], // names is a map of language to city name names["us"] = "New York"
		Geometry: &Geometry{
			Type:        "Point",
			Coordinates: []float64{resp.Location.Latitude, resp.Location.Longitude},
		},
	}
}
