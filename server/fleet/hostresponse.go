package fleet

import (
	"context"
	"time"
)

// HostResponse is the response struct that contains the full host information
// along with the host online status and the "display text" to be used when
// rendering in the UI.
type HostResponse struct {
	*Host
	Status           HostStatus   `json:"status" csv:"status"`
	DisplayText      string       `json:"display_text" csv:"display_text"`
	DisplayName      string       `json:"display_name" csv:"display_name"`
	Labels           []*Label     `json:"labels,omitempty" csv:"-"`
	Geolocation      *GeoLocation `json:"geolocation,omitempty" csv:"-"`
	CSVDeviceMapping string       `json:"-" db:"-" csv:"device_mapping"`
}

// HostResponseForHost returns a HostResponse from Host with Geolocation.
func HostResponseForHost(ctx context.Context, svc Service, host *Host) *HostResponse {
	hr := HostResponseForHostCheap(host)
	hr.Geolocation = svc.LookupGeoIP(ctx, host.PublicIP)
	return hr
}

// HostResponseForHostCheap returns a new HostResponse from a Host without computing Geolocation.
func HostResponseForHostCheap(host *Host) *HostResponse {
	return &HostResponse{
		Host:        host,
		Status:      host.Status(time.Now()),
		DisplayText: host.Hostname,
		DisplayName: host.DisplayName(),
	}
}

// HostResponsesForHostsCheap returns a HostResponses from Hosts without computing Geolocation.
func HostResponsesForHostsCheap(hosts []Host) []HostResponse {
	hrs := make([]HostResponse, len(hosts))
	for i, h := range hosts {
		h := h
		hrs[i] = *HostResponseForHostCheap(&h)
	}
	return hrs
}
