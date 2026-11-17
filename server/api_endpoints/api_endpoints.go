package apiendpoints

import (
	_ "embed"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/ghodss/yaml"
	kithttp "github.com/go-kit/kit/transport/http"
)

// FeatureRouteFunc registers the routes of a bounded-context / feature module
// onto a router. It mirrors endpointer.HandlerRoutesFunc.
type FeatureRouteFunc func(r *endpointer.Router, opts []kithttp.ServerOption)

//go:embed api_endpoints.yml
var apiEndpointsYAML []byte

var apiEndpoints []fleet.APIEndpoint

var apiEndpointsSet map[string]struct{}

func init() {
	loaded, err := loadAPIEndpoints()
	if err != nil {
		panic(err)
	}
	set := make(map[string]struct{}, len(loaded))
	for _, e := range loaded {
		set[e.Fingerprint()] = struct{}{}
	}
	apiEndpoints = loaded
	apiEndpointsSet = set
}

// GetAPIEndpoints returns a copy of the embedded API endpoints slice.
func GetAPIEndpoints() []fleet.APIEndpoint {
	result := make([]fleet.APIEndpoint, len(apiEndpoints))
	copy(result, apiEndpoints)
	return result
}

// IsInCatalog reports whether the given endpoint fingerprint is in the catalog.
func IsInCatalog(fingerprint string) bool {
	_, ok := apiEndpointsSet[fingerprint]
	return ok
}

// routerProvider is implemented by handlers that wrap the router rather than being one.
type routerProvider interface {
	Router() *endpointer.Router
}

// Validate checks that every endpoint in the embedded catalog is registered on the provided handler or one of the given
// feature routes, reading the routes recorded by the endpointer. Feature routes are registered on a throwaway router so
// callers can validate routes that are declared elsewhere.
func Validate(h http.Handler, featureRoutes ...FeatureRouteFunc) error {
	r, ok := h.(*endpointer.Router)
	if !ok {
		// The API handler may be wrapped by a faster router that keeps the route table for introspection.
		provider, isProvider := h.(routerProvider)
		if !isProvider {
			return fmt.Errorf("expected *endpointer.Router, got %T", h)
		}
		r = provider.Router()
	}

	registered := make(map[string]struct{})
	collect := func(rt *endpointer.Router) {
		for _, route := range rt.Routes() {
			for _, m := range route.Methods {
				registered[fleet.NewAPIEndpointFromTpl(m, route.Path).Fingerprint()] = struct{}{}
			}
		}
	}

	collect(r)

	for _, fr := range featureRoutes {
		if fr == nil {
			continue
		}
		tmp := endpointer.NewRouter()
		fr(tmp, nil)
		collect(tmp)
	}

	var missing []string
	for _, e := range apiEndpoints {
		if _, ok := registered[e.Fingerprint()]; !ok {
			missing = append(missing, e.Method+" "+e.Path)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("the following API endpoints are unknown: %v", missing)
	}

	return nil
}

func loadAPIEndpoints() ([]fleet.APIEndpoint, error) {
	endpoints := make([]fleet.APIEndpoint, 0)

	if err := yaml.Unmarshal(apiEndpointsYAML, &endpoints); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(endpoints))
	for _, e := range endpoints {
		fp := e.Fingerprint()
		if _, ok := seen[fp]; ok {
			panic(fmt.Errorf("duplicate entry (%s, %s)", e.Method, e.Path))
		}
		seen[fp] = struct{}{}
	}
	return endpoints, nil
}
