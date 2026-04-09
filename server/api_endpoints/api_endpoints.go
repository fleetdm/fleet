package apiendpoints

import (
	_ "embed"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/ghodss/yaml"
)

//go:embed api_endpoints.yml
var apiEndpointsYAML []byte

var apiEndpoints = mustGetAPIEndpoints()

// GetAPIEndpoints returns a copy of the embedded API endpoints slice.
func GetAPIEndpoints() []fleet.APIEndpoint {
	result := make([]fleet.APIEndpoint, len(apiEndpoints))
	copy(result, apiEndpoints)
	return result
}

func mustGetAPIEndpoints() []fleet.APIEndpoint {
	endpoints := make([]fleet.APIEndpoint, 0)

	if err := yaml.Unmarshal(apiEndpointsYAML, &endpoints); err != nil {
		panic(fmt.Errorf("failed to parse: %w", err))
	}

	seen := make(map[string]struct{}, len(endpoints))
	for _, e := range endpoints {
		fp := e.Fingerprint()
		if _, ok := seen[fp]; ok {
			panic(fmt.Errorf("duplicate entry (%s, %s)", e.Method, e.Path))
		}
		seen[fp] = struct{}{}
	}
	return endpoints
}
