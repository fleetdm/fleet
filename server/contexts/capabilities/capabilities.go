package capabilities

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type key int

const capabilitiesKey key = 0

// NewContext creates a new context with the given capabilities.
func NewContext(ctx context.Context, r *http.Request) context.Context {
	capabilities := fleet.CapabilityMap{}
	capabilities.PopulateFromString(r.Header.Get(fleet.CapabilitiesHeader))
	return context.WithValue(ctx, capabilitiesKey, capabilities)
}

// FromContext returns the capabilities in the request if present.
func FromContext(ctx context.Context) (fleet.CapabilityMap, bool) {
	v, ok := ctx.Value(capabilitiesKey).(fleet.CapabilityMap)
	return v, ok
}
