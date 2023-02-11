package update

import "github.com/fleetdm/fleet/v4/server/fleet"

// OrbitConfigFetcher allows fetching Orbit configuration.
type OrbitConfigFetcher interface {
	// GetConfig returns the Orbit configuration.
	GetConfig() (*fleet.OrbitConfig, error)
}
